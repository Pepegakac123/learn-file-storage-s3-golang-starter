package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	const maxMemory = 10 << 20 // 10 MB

	r.ParseMultipartForm(maxMemory)

	fileData, fileHeaders, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "No thumbnail file provided", err)
		return
	}

	contentType := fileHeaders.Header.Get("Content-Type")
	parts := strings.Split(contentType, "/")

	if len(parts) != 2 {
		respondWithError(w, http.StatusBadRequest, "Invalid content type", nil)
		return
	}

	mediaType := parts[0]
	mediaExtension := parts[1]

	if mediaType != "image" || (mediaExtension != "jpeg" && mediaExtension != "png") {
		respondWithError(w, http.StatusBadRequest, "Thumbnail must be a JPEG or PNG image", nil)
		return
	}

	filename := fmt.Sprintf("%v.%v", videoIDString, mediaExtension)
	filePath := filepath.Join(cfg.assetsRoot, filename)
	file, err := os.Create(filePath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error creating a file", err)
		return
	}
	_, err = io.Copy(file, fileData)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error saving a file", err)
		return
	}

	videoMetadata, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to retrieve video", err)
		return
	}
	if videoMetadata.ID == uuid.Nil {
		respondWithError(w, http.StatusNotFound, "Video not found", nil)
		return
	}
	if videoMetadata.UserID != userID {
		respondWithError(w, http.StatusForbidden, "You don't have permission to modify this video", nil)
		return
	}

	dataUrl := fmt.Sprintf("http://localhost:%v/assets/%v.%v", cfg.port, videoMetadata.ID, mediaExtension)

	videoMetadata.ThumbnailURL = &dataUrl
	err = cfg.db.UpdateVideo(videoMetadata)
	if err != nil {
		respondWithError(w, 500, "Failed to update video metadata", err)
	}

	respondWithJSON(w, http.StatusOK, videoMetadata)
}

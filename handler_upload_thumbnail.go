package main

import (
	"fmt"
	"io"
	"net/http"

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

	// TODO: implement the upload here
	const maxMemory = 10 << 20 // 10 MB

	r.ParseMultipartForm(maxMemory)

	fileData, fileHeaders, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, 500, "Couldn't get a fileData", err)
	}

	mediaType := fileHeaders.Header.Get("Content-Type")

	thumbnailData, err := io.ReadAll(fileData)
	if err != nil {
		respondWithError(w, 500, "Couldn't get a data of the image", err)

	}

	videoMetadata, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, 500, "Error of the getting videoMetadata from the database", err)

	}
	if videoMetadata.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "The video is not the user property", err)
	}
	thb := thumbnail{
		data:      thumbnailData,
		mediaType: mediaType,
	}

	videoThumbnails[videoMetadata.ID] = thb

	newThbUrl := fmt.Sprintf("http://localhost:%v/api/thumbnails/%v", cfg.port, videoMetadata.ID)

	videoMetadata.ThumbnailURL = &newThbUrl
	err = cfg.db.UpdateVideo(videoMetadata)
	if err != nil {
		respondWithError(w, 500, "There was an error while updating the video in the database", err)
	}

	respondWithJSON(w, http.StatusOK, videoMetadata)
}

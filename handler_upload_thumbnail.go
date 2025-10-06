package main

import (
	"crypto/rand"
	"encoding/base64"
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

	// ⭐ 1. NAJPIERW sprawdź uprawnienia (przed zapisywaniem pliku!)
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

	// ⭐ 2. Teraz parsuj formularz
	const maxMemory = 10 << 20            // 10 MB
	err = r.ParseMultipartForm(maxMemory) // ⭐ Sprawdź błąd!
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form data", err)
		return
	}

	fileData, fileHeaders, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "No thumbnail file provided", err)
		return
	}
	defer fileData.Close()

	// ⭐ 3. Walidacja Content-Type
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

	// ⭐ 4. Generuj losową nazwę
	randomBytes := make([]byte, 32)
	_, err = rand.Read(randomBytes)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error creating random filename", err)
		return
	}
	randomFilename := base64.RawURLEncoding.EncodeToString(randomBytes)

	filename := fmt.Sprintf("%s.%s", randomFilename, mediaExtension)
	filePath := filepath.Join(cfg.assetsRoot, filename)

	// ⭐ 5. Zapisz plik
	file, err := os.Create(filePath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error creating a file", err)
		return
	}
	defer file.Close() // ⭐ WAŻNE!

	_, err = io.Copy(file, fileData)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error saving a file", err)
		return
	}

	// ⭐ 6. Zaktualizuj URL (używaj filename, nie videoID!)
	dataUrl := fmt.Sprintf("http://localhost:%s/assets/%s", cfg.port, filename)
	videoMetadata.ThumbnailURL = &dataUrl

	err = cfg.db.UpdateVideo(videoMetadata)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to update video metadata", err)
		return // ⭐ WAŻNE!
	}

	respondWithJSON(w, http.StatusOK, videoMetadata)
}

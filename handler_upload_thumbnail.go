package main

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime"
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

	const maxMemory = 10 << 20
	r.ParseMultipartForm(maxMemory)

	// "thumbnail" should match the HTML form input name
	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
		return
	}
	defer file.Close()

	mediaType := header.Header.Get("Content-Type")

	mimeType, _, _ := mime.ParseMediaType(mediaType)

	if mimeType != "image/jpeg" && mimeType != "image/png" {
		respondWithError(w, http.StatusUnsupportedMediaType, "Unsupported file type", errors.New("Unsupported file type"))
		return
	}

	video, err := cfg.db.GetVideo(videoID)

	if err != nil {
		respondWithError(w, http.StatusNotFound, "Video not found", err)
		return
	}

	if video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "You're not allowed to edit this video", nil)
		return
	}

	contentType := strings.Split(mediaType, "/")
	extension := contentType[1]
	randName := make([]byte, 32)
	rand.Read(randName)
	fileName := base64.RawURLEncoding.EncodeToString(randName)
	thumbPath := fmt.Sprintf("%s.%s", fileName, extension)
	savePath := filepath.Join(cfg.assetsRoot, thumbPath)

	savedFilePath, err := os.Create(savePath)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error while creating file", err)
		return
	}

	_, err = io.Copy(savedFilePath, file)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error while copying file", err)
		return
	}

	thumbURL := fmt.Sprintf("http://localhost:%s/assets/%s", cfg.port, thumbPath)
	video.ThumbnailURL = &thumbURL

	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to update video", err)
		return
	}

	respondWithJSON(w, http.StatusOK, video)
}

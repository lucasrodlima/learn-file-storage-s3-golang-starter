package main

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/database"
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

	const maxMemory = 10 << 20

	r.ParseMultipartForm(maxMemory)

	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "couldn't parse from file", err)
		return
	}
	defer file.Close()

	mediaType, _, err := mime.ParseMediaType(header.Header.Get("Content-Type"))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "couldn't parse media type", err)
		return
	}
	if mediaType != "image/jpeg" && mediaType != "image/png" {
		fmt.Println(mediaType)
		respondWithError(w, http.StatusBadRequest, "wrong content type", err)
		return
	}

	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "couln't find video", err)
		return
	}
	if video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "can't access video", err)
		return
	}

	filename := fmt.Sprintf("%s.%s", videoIDString, strings.TrimLeft(mediaType, "image/"))

	newFile, err := os.Create(filepath.Join(cfg.assetsRoot, filename))
	defer newFile.Close()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "can't create new thumbnail file", err)
		return
	}

	_, err = io.Copy(newFile, file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "can't copy contents from multiform file", err)
		return
	}

	newUrl := fmt.Sprintf("http://localhost:%s/assets/%s", cfg.port, filename)
	video.ThumbnailURL = &newUrl
	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "couln't save video metadata", err)
		return
	}

	respondWithJSON(w, http.StatusOK, database.Video{
		ID:           video.ID,
		CreatedAt:    video.CreatedAt,
		UpdatedAt:    video.UpdatedAt,
		ThumbnailURL: video.ThumbnailURL,
		VideoURL:     video.VideoURL,
	})
}

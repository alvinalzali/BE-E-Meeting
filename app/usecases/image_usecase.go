package usecases

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type ImageUsecase interface {
	UploadImage(file *multipart.FileHeader, baseURL string) (string, error)
}

type imageUsecase struct {
	imageRepo repositories.ImageRepository
}

func NewImageUsecase(imageRepo repositories.ImageRepository) ImageUsecase {
	return &imageUsecase{imageRepo: imageRepo}
}

func (u *imageUsecase) UploadImage(file *multipart.FileHeader, baseURL string) (string, error) {
	if file == nil {
		return "", &UseCaseError{Code: http.StatusBadRequest, Message: "Failed to upload image"}
	}
	contentType := file.Header.Get("Content-Type")
	if !(strings.HasPrefix(contentType, "image/jpeg") || strings.HasPrefix(contentType, "image/png")) {
		return "", &UseCaseError{Code: http.StatusBadRequest, Message: "Invalid file type"}
	}
	if file.Size > 1024*1024 {
		return "", &UseCaseError{Code: http.StatusBadRequest, Message: "File size is too large"}
	}

	src, err := file.Open()
	if err != nil {
		return "", &UseCaseError{Code: http.StatusInternalServerError, Message: "Failed to open image file"}
	}
	defer src.Close()

	os.MkdirAll("./assets/temp", os.ModePerm)
	ext := filepath.Ext(file.Filename)
	filename := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
	tempPath := filepath.Join("./assets/temp", filename)
	dst, err := os.Create(tempPath)
	if err != nil {
		return "", &UseCaseError{Code: http.StatusInternalServerError, Message: "Failed to save image"}
	}
	defer dst.Close()
	io.Copy(dst, src)

	imageURL := baseURL + "/assets/temp/" + filename
	return imageURL, nil
}

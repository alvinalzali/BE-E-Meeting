package usecases

import (
	"BE-E-MEETING/app/models"
	"BE-E-MEETING/app/repositories"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type RoomUsecase interface {
	CreateRoom(req models.RoomRequest, file *multipart.FileHeader, baseURL string) (string, error)
	GetRooms(name, roomType, capacity, page, pageSize string) ([]models.Room, int, int, error)
	GetRoomByID(id int) (models.Room, error)
	UpdateRoom(id int, room models.RoomRequest) error
	DeleteRoom(id int) error
	GetRoomReservationSchedule(roomID int, date string) ([]models.RoomSchedule, models.Room, error)
}

type roomUsecase struct {
	roomRepo repositories.RoomRepository
}

func NewRoomUsecase(roomRepo repositories.RoomRepository) RoomUsecase {
	return &roomUsecase{roomRepo: roomRepo}
}

func (u *roomUsecase) CreateRoom(req models.RoomRequest, file *multipart.FileHeader, baseURL string) (string, error) {
	imageURL := strings.TrimSpace(req.ImageURL)
	finalFilename := ""
	createdInRooms := false

	if imageURL != "" && strings.Contains(imageURL, "/assets/temp/") {
		tempFilename := filepath.Base(imageURL)
		tempPath := filepath.Join(".", "assets", "temp", tempFilename)

		if _, err := os.Stat(tempPath); err == nil {
			uploadDir := filepath.Join(".", "assets", "rooms")
			if err := os.MkdirAll(uploadDir, 0755); err != nil {
				return "", &UseCaseError{Code: http.StatusInternalServerError, Message: "error creating upload directory"}
			}

			finalFilename = fmt.Sprintf("%d%s", time.Now().UnixNano(), filepath.Ext(tempFilename))
			newPath := filepath.Join(uploadDir, finalFilename)

			if err := os.Rename(tempPath, newPath); err != nil {
				src, err := os.Open(tempPath)
				if err != nil {
					return "", &UseCaseError{Code: http.StatusInternalServerError, Message: "error processing uploaded image"}
				}
				defer src.Close()

				dst, err := os.Create(newPath)
				if err != nil {
					return "", &UseCaseError{Code: http.StatusInternalServerError, Message: "error processing uploaded image"}
				}
				defer dst.Close()

				if _, err = io.Copy(dst, src); err != nil {
					_ = os.Remove(newPath)
					return "", &UseCaseError{Code: http.StatusInternalServerError, Message: "error saving image"}
				}
				_ = os.Remove(tempPath)
			}
			imageURL = fmt.Sprintf("%s/assets/rooms/%s", baseURL, finalFilename)
			createdInRooms = true
		} else {
			imageURL = ""
		}
	}

	if imageURL == "" {
		if file == nil {
			return "", &UseCaseError{Code: http.StatusBadRequest, Message: "image file is required"}
		}

		if file.Size > 1<<20 {
			return "", &UseCaseError{Code: http.StatusBadRequest, Message: "image file size must be less than 1MB"}
		}

		src, err := file.Open()
		if err != nil {
			return "", &UseCaseError{Code: http.StatusInternalServerError, Message: "error opening uploaded file"}
		}
		defer src.Close()

		buf := make([]byte, 512)
		n, _ := src.Read(buf)
		contentType := http.DetectContentType(buf[:n])
		if contentType != "image/jpeg" && contentType != "image/png" {
			return "", &UseCaseError{Code: http.StatusBadRequest, Message: "invalid file type, only JPG/PNG allowed"}
		}
		if _, err := src.Seek(0, 0); err != nil {
			return "", &UseCaseError{Code: http.StatusInternalServerError, Message: "error processing file"}
		}

		uploadDir := filepath.Join(".", "assets", "rooms")
		if err := os.MkdirAll(uploadDir, 0755); err != nil {
			return "", &UseCaseError{Code: http.StatusInternalServerError, Message: "error creating upload directory"}
		}

		finalFilename = fmt.Sprintf("%d%s", time.Now().UnixNano(), filepath.Ext(file.Filename))
		dstPath := filepath.Join(uploadDir, finalFilename)

		dst, err := os.Create(dstPath)
		if err != nil {
			return "", &UseCaseError{Code: http.StatusInternalServerError, Message: "error creating destination file"}
		}
		defer dst.Close()

		if _, err = io.Copy(dst, src); err != nil {
			_ = os.Remove(dstPath)
			return "", &UseCaseError{Code: http.StatusInternalServerError, Message: "error saving file"}
		}

		imageURL = fmt.Sprintf("%s/assets/rooms/%s", baseURL, finalFilename)
		createdInRooms = true
	}

	err := u.roomRepo.CreateRoom(req, imageURL)
	if err != nil {
		if createdInRooms && finalFilename != "" {
			_ = os.Remove(filepath.Join(".", "assets", "rooms", finalFilename))
		}
		return "", &UseCaseError{Code: http.StatusInternalServerError, Message: "error saving room data"}
	}
	return imageURL, nil
}

func (u *roomUsecase) GetRooms(name, roomType, capacity, page, pageSize string) ([]models.Room, int, int, error) {
	if roomType != "" && roomType != "small" && roomType != "medium" && roomType != "large" {
		return nil, 0, 0, &UseCaseError{Code: http.StatusBadRequest, Message: "room type is not valid"}
	}
	rooms, totalData, err := u.roomRepo.GetRooms(name, roomType, capacity)
	if err != nil {
		return nil, 0, 0, err
	}
	return rooms, totalData, 0, nil
}

func (u *roomUsecase) GetRoomByID(id int) (models.Room, error) {
	return u.roomRepo.GetRoomByID(id)
}

func (u *roomUsecase) UpdateRoom(id int, room models.RoomRequest) error {
	if room.Type != "small" && room.Type != "medium" && room.Type != "large" || room.Capacity <= 0 {
		return &UseCaseError{Code: http.StatusBadRequest, Message: "room type is not valid / capacity must be larger more than 0"}
	}
	rowsAffected, err := u.roomRepo.UpdateRoom(id, room)
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return &UseCaseError{Code: http.StatusNotFound, Message: "room not found"}
	}
	return nil
}

func (u *roomUsecase) DeleteRoom(id int) error {
	rowsAffected, err := u.roomRepo.DeleteRoom(id)
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return &UseCaseError{Code: http.StatusNotFound, Message: "room not found"}
	}
	return nil
}

func (u *roomUsecase) GetRoomReservationSchedule(roomID int, date string) ([]models.RoomSchedule, models.Room, error) {
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}

	exists, err := u.roomRepo.CheckRoomExists(roomID)
	if err != nil {
		return nil, models.Room{}, err
	}
	if !exists {
		return nil, models.Room{}, &UseCaseError{Code: http.StatusNotFound, Message: "room not found"}
	}

	schedules, err := u.roomRepo.GetRoomReservationSchedule(roomID, date)
	if err != nil {
		return nil, models.Room{}, err
	}

	room, err := u.roomRepo.GetRoomByID(roomID)
	if err != nil {
		return nil, models.Room{}, err
	}

	return schedules, room, nil
}

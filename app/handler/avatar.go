package handler

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/labstack/echo/v4"
)

func HandleAvatarUpdate(c echo.Context, userID int, oldAvatarURL, newAvatarURL string) (string, error) {

	log.Printf("[INFO] [handleAvatarUpdate] user_id=%d avatar update requested", userID)

	//jika avatar tidak berubah, langsung return file lama
	if newAvatarURL == "" || newAvatarURL == oldAvatarURL {
		return oldAvatarURL, nil
	}

	fileName := filepath.Base(newAvatarURL)

	tempPath := filepath.Join("./assets/temp", fileName)
	finalPath := filepath.Join("./assets/image/users", fileName)

	log.Printf("[DEBUG] [handleAvatarUpdate] user_id=%d temp=%s final=%s",
		userID, tempPath, finalPath,
	)

	//cek folder final ada
	_ = os.MkdirAll("./assets/image/users", os.ModePerm)

	// Cek file temp ada
	if _, err := os.Stat(tempPath); err != nil {
		log.Printf("[ERROR] [handleAvatarUpdate] Temp file not found user_id=%d file=%s err=%v",
			userID, tempPath, err,
		)
		return oldAvatarURL, fmt.Errorf("temp file not found")
	}

	// Pindahkan file temp ke final
	err := os.Rename(tempPath, finalPath)
	if err != nil {
		log.Printf("[ERROR] [handleAvatarUpdate] Failed moving file user_id=%d src=%s dest=%s err=%v",
			userID, tempPath, finalPath, err,
		)
		return oldAvatarURL, err
	}

	log.Printf("[INFO] [handleAvatarUpdate] Avatar moved user_id=%d new_file=%s",
		userID, finalPath,
	)

	// url final
	baseURL := c.Scheme() + "://" + c.Request().Host
	finalURL := baseURL + "/assets/image/users/" + fileName

	// hapus file lama
	if oldAvatarURL != "" && !strings.Contains(oldAvatarURL, "default") {

		oldFile := filepath.Base(oldAvatarURL)
		oldPath := filepath.Join("./assets/image/users", oldFile)

		log.Printf("[INFO] [handleAvatarUpdate] Removing old avatar user_id=%d file=%s",
			userID, oldPath,
		)

		if err := os.Remove(oldPath); err != nil {

			log.Printf("[WARN] [handleAvatarUpdate] Failed removing old avatar user_id=%d file=%s",
				userID, oldPath,
			)

			// rollback ketika gagal dan hapus avatar baru
			_ = os.Remove(finalPath)

			return oldAvatarURL, fmt.Errorf("rollback: failed removing old avatar")
		}
	}

	log.Printf("[INFO] [handleAvatarUpdate] Avatar updated user_id=%d new_url=%s",
		userID, finalURL,
	)

	return finalURL, nil

}

func HandleRoomImageUpdate(c echo.Context, roomID int, oldAvatarURL, newAvatarURL string) (string, error) {

	log.Printf("[INFO] [handleRoomImageUpdate] room_id=%d avatar update requested", roomID)

	//jika avatar tidak berubah, langsung return file lama
	if newAvatarURL == "" || newAvatarURL == oldAvatarURL {
		return oldAvatarURL, nil
	}

	fileName := filepath.Base(newAvatarURL)

	tempPath := filepath.Join("./assets/temp", fileName)
	finalPath := filepath.Join("./assets/image/rooms", fileName)

	log.Printf("[DEBUG] [handleRoomImageUpdate] room_id=%d temp=%s final=%s",
		roomID, tempPath, finalPath,
	)

	//cek folder final ada
	_ = os.MkdirAll("./assets/image/rooms", os.ModePerm)

	// Cek file temp ada
	if _, err := os.Stat(tempPath); err != nil {
		log.Printf("[ERROR] [handleRoomImageUpdate] Temp file not found room_id=%d file=%s err=%v",
			roomID, tempPath, err,
		)
		return oldAvatarURL, fmt.Errorf("temp file not found")
	}

	// Pindahkan file temp ke final
	err := os.Rename(tempPath, finalPath)
	if err != nil {
		log.Printf("[ERROR] [handleRoomImageUpdate] Failed moving file room_id=%d src=%s dest=%s err=%v",
			roomID, tempPath, finalPath, err,
		)
		return oldAvatarURL, err
	}

	log.Printf("[INFO] [handleRoomImageUpdate] Avatar moved room_id=%d new_file=%s",
		roomID, finalPath,
	)

	// url final
	baseURL := c.Scheme() + "://" + c.Request().Host
	finalURL := baseURL + "/assets/image/rooms/" + fileName

	// hapus file lama
	if oldAvatarURL != "" && !strings.Contains(oldAvatarURL, "default") {

		oldFile := filepath.Base(oldAvatarURL)
		oldPath := filepath.Join("./assets/image/rooms", oldFile)

		log.Printf("[INFO] [handleRoomImageUpdate] Removing old avatar room_id=%d file=%s",
			roomID, oldPath,
		)

		if err := os.Remove(oldPath); err != nil {

			log.Printf("[WARN] [handleRoomImageUpdate] Failed removing old avatar room_id=%d file=%s",
				roomID, oldPath,
			)

			// rollback ketika gagal dan hapus avatar baru
			_ = os.Remove(finalPath)

			return oldAvatarURL, fmt.Errorf("rollback: failed removing old avatar")
		}
	}

	log.Printf("[INFO] [handleRoomImageUpdate] Avatar updated room_id=%d new_url=%s",
		roomID, finalURL,
	)

	return finalURL, nil

}

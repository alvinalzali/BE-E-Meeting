package handler

// mentahan untuk fix image lama, temp dan ke assets

// import (
// 	"fmt"
// 	"log"
// 	"os"
// 	"path/filepath"
// 	"strings"

// 	"github.com/labstack/echo/v4"
// )

// func HandleAvatarUpdate(c echo.Context, userID int, oldAvatarURL, newAvatarURL string) (string, error) {

// 	log.Printf("[INFO] [handleAvatarUpdate] user_id=%d avatar update requested", userID)

// 	//jika avatar tidak berubah, langsung return file lama
// 	if newAvatarURL == "" || newAvatarURL == oldAvatarURL {
// 		return oldAvatarURL, nil
// 	}

// 	fileName := filepath.Base(newAvatarURL)

// 	tempPath := filepath.Join("./assets/temp", fileName)
// 	finalPath := filepath.Join("./assets/image/users", fileName)

// 	log.Printf("[DEBUG] [handleAvatarUpdate] user_id=%d temp=%s final=%s",
// 		userID, tempPath, finalPath,
// 	)

// 	//cek folder final ada
// 	_ = os.MkdirAll("./assets/image/users", os.ModePerm)

// 	// Cek file temp ada
// 	if _, err := os.Stat(tempPath); err != nil {
// 		log.Printf("[ERROR] [handleAvatarUpdate] Temp file not found user_id=%d file=%s err=%v",
// 			userID, tempPath, err,
// 		)
// 		return oldAvatarURL, fmt.Errorf("temp file not found")
// 	}

// 	// Pindahkan file temp ke final
// 	err := os.Rename(tempPath, finalPath)
// 	if err != nil {
// 		log.Printf("[ERROR] [handleAvatarUpdate] Failed moving file user_id=%d src=%s dest=%s err=%v",
// 			userID, tempPath, finalPath, err,
// 		)
// 		return oldAvatarURL, err
// 	}

// 	log.Printf("[INFO] [handleAvatarUpdate] Avatar moved user_id=%d new_file=%s",
// 		userID, finalPath,
// 	)

// 	// url final
// 	baseURL := c.Scheme() + "://" + c.Request().Host
// 	finalURL := baseURL + "/assets/image/users/" + fileName

// 	// hapus file lama
// 	if oldAvatarURL != "" && !strings.Contains(oldAvatarURL, "default") {

// 		oldFile := filepath.Base(oldAvatarURL)
// 		oldPath := filepath.Join("./assets/image/users", oldFile)

// 		log.Printf("[INFO] [handleAvatarUpdate] Removing old avatar user_id=%d file=%s",
// 			userID, oldPath,
// 		)

// 		if err := os.Remove(oldPath); err != nil {

// 			log.Printf("[WARN] [handleAvatarUpdate] Failed removing old avatar user_id=%d file=%s",
// 				userID, oldPath,
// 			)

// 			// rollback ketika gagal dan hapus avatar baru
// 			_ = os.Remove(finalPath)

// 			return oldAvatarURL, fmt.Errorf("rollback: failed removing old avatar")
// 		}
// 	}

// 	log.Printf("[INFO] [handleAvatarUpdate] Avatar updated user_id=%d new_url=%s",
// 		userID, finalURL,
// 	)

// 	return finalURL, nil

// }

// func HandleRoomImageCreate(c echo.Context, tempURL string, DefaultRoomURL string) (string, error) {

// 	// Jika kosong / default, langsung return
// 	if tempURL == "" || strings.Contains(tempURL, "default") {
// 		log.Println("[INFO] Room image default, skip move")
// 		return DefaultRoomURL, nil
// 	}

// 	// jika kosong, dan contain assets/image/rooms, langsung return
// 	if strings.Contains(tempURL, "assets/image/rooms") {
// 		return tempURL, nil
// 	}

// 	fileName := filepath.Base(tempURL)

// 	tempPath := filepath.Join("./assets/temp", fileName)
// 	finalPath := filepath.Join("./assets/image/rooms", fileName)

// 	log.Printf("[INFO] CreateRoom moving image %s → %s\n", tempPath, finalPath)

// 	// Cek file temp
// 	if _, err := os.Stat(tempPath); err != nil {
// 		log.Printf("[ERROR] Temp room image not found: %s", tempPath)
// 		return DefaultRoomURL, fmt.Errorf("temp image not found")
// 	}

// 	// Buat folder final
// 	_ = os.MkdirAll("./assets/image/rooms", os.ModePerm)

// 	// Pindahkan file
// 	if err := os.Rename(tempPath, finalPath); err != nil {
// 		log.Printf("[ERROR] Failed move room image: %v", err)
// 		return DefaultRoomURL, err
// 	}

// 	// Buat URL final
// 	baseURL := c.Scheme() + "://" + c.Request().Host
// 	finalURL := baseURL + "/assets/image/rooms/" + fileName

// 	log.Println("[INFO] Room image success:", finalURL)

// 	return finalURL, nil
// }

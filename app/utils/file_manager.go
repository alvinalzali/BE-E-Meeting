package utils

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// ProcessImageMove memindahkan file fisik dari temp ke permanent folder
func ProcessImageMove(oldFullURL, newFullURL, baseURL, targetFolder string) (string, error) {
	// 1. Validasi: Jika URL kosong atau sama persis dengan yang lama, return aja
	if newFullURL == "" || newFullURL == oldFullURL {
		return oldFullURL, nil
	}

	// 2. Validasi: Jika URL tidak mengandung "assets/temp", berarti bukan file baru dari upload
	// (Bisa jadi user mengirim URL gambar lama yang memang sudah permanent)
	if !strings.Contains(newFullURL, "assets/temp") {
		return newFullURL, nil
	}

	// --- LOGIC PINDAH FILE ---

	// Ambil nama file dari URL (misal: "17675.jpeg")
	fileName := filepath.Base(newFullURL)

	// Tentukan path asal (temp) dan tujuan (permanent)
	// Asumsi struktur folder project kamu: root/assets/temp dan root/assets/image/users
	tempPath := filepath.Join("assets", "temp", fileName)
	finalDir := filepath.Join("assets", "image", targetFolder) // targetFolder = "users" atau "rooms"
	finalPath := filepath.Join(finalDir, fileName)

	// Cek apakah file fisik ada di folder temp?
	if _, err := os.Stat(tempPath); err != nil {
		log.Printf("[ERROR] File temp tidak ditemukan: %s", tempPath)
		// Jika file fisik gak ada, kembalikan error (agar user tau upload gagal/expired)
		return oldFullURL, fmt.Errorf("temp file not found on server")
	}

	// Buat folder tujuan jika belum ada
	if err := os.MkdirAll(finalDir, os.ModePerm); err != nil {
		return oldFullURL, fmt.Errorf("failed to create directory: %v", err)
	}

	// Pindahkan file (Rename / Move)
	if err := os.Rename(tempPath, finalPath); err != nil {
		log.Printf("[ERROR] Gagal memindahkan file: %v", err)
		return oldFullURL, fmt.Errorf("failed to move file")
	}

	// --- LOGIC HAPUS FILE LAMA ---

	// Hapus file lama fisik jika ada, dan BUKAN default
	if oldFullURL != "" && !strings.Contains(oldFullURL, "default") {
		oldFileName := filepath.Base(oldFullURL)
		oldFilePath := filepath.Join(finalDir, oldFileName)
		_ = os.Remove(oldFilePath) // Ignore error kalau gagal hapus, yang penting file baru aman
	}

	// --- RETURN URL BARU ---

	// Construct URL baru: http://localhost:8080/assets/image/users/namafile.jpeg
	// Pastikan tidak ada double slash
	cleanBaseURL := strings.TrimRight(baseURL, "/")
	finalURL := fmt.Sprintf("%s/assets/image/%s/%s", cleanBaseURL, targetFolder, fileName)

	return finalURL, nil
}

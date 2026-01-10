package handler

import (
	"fmt"
	"net/http"
	"net/url"

	"BE-E-Meeting/app/usecases"

	"github.com/labstack/echo/v4"
)

type AuthHandler struct {
	usecase usecases.AuthUsecase
}

func NewAuthHandler(usecase usecases.AuthUsecase) *AuthHandler {
	return &AuthHandler{usecase: usecase}
}

// handler oauth

func (h *AuthHandler) GoogleLogin(c echo.Context) error {
	url, _ := h.usecase.GetGoogleLoginURL()
	return c.Redirect(http.StatusTemporaryRedirect, url)
}

func (h *AuthHandler) GoogleCallback(c echo.Context) error {
	// 1. Cek State
	state := c.QueryParam("state")
	if state != "random-secret-state" {
		return c.JSON(http.StatusUnauthorized, echo.Map{"error": "state invalid"})
	}

	// 2. Ambil Code
	code := c.QueryParam("code")
	if code == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "code missing"})
	}

	// [FIX] Decode URL (Ubah %2F menjadi /)
	// Ini penting karena kadang Google mengirim karakter spesial
	decodedCode, err := url.QueryUnescape(code)
	if err != nil {
		// Jika gagal decode, pakai code asli
		decodedCode = code
	}

	// [DEBUG] Print ke terminal Visual Studio Code kamu
	fmt.Println("--- DEBUG OAUTH ---")
	fmt.Println("Code Raw:", code)
	fmt.Println("Code Decoded:", decodedCode)

	// 3. Proses ke Usecase (Pakai decodedCode)
	token, err := h.usecase.ProcessGoogleLogin(decodedCode)
	if err != nil {
		// [DEBUG] Print error asli dari Google ke terminal
		fmt.Printf("Error Usecase: %v\n", err)

		// Return error ke browser agar terlihat
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error":  "Gagal Login Google",
			"detail": err.Error(),
		})
	}

	// 4. Sukses
	return c.JSON(http.StatusOK, echo.Map{
		"message": "Login Google Berhasil",
		"token":   token,
	})
}

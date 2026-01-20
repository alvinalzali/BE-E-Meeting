package usecases

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	// untuk random password
	"BE-E-Meeting/app/entities"
	"BE-E-Meeting/app/middleware"
	"BE-E-Meeting/app/repositories"

	"golang.org/x/oauth2"
)

type AuthUsecase interface {
	GetGoogleLoginURL() (string, error)
	ProcessGoogleLogin(code string) (string, error)
}

type authUsecase struct {
	userRepo     repositories.UserRepository
	googleConfig *oauth2.Config
}

// NewAuthUsecase: Constructor untuk inject repository ke usecase
func NewAuthUsecase(userRepo repositories.UserRepository, cfg *oauth2.Config) AuthUsecase {
	return &authUsecase{userRepo: userRepo, googleConfig: cfg}
}

// Implement OAuth
func (u *authUsecase) GetGoogleLoginURL() (string, error) {
	return u.googleConfig.AuthCodeURL("random-secret-state"), nil
}

func (u *authUsecase) ProcessGoogleLogin(code string) (string, error) {
	token, err := u.googleConfig.Exchange(context.Background(), code)
	if err != nil {
		return "", err
	}

	// ambil data user dari API google
	resp, err := http.Get("https://www.googleapis.com/oauth2/v2/userinfo?access_token=" + token.AccessToken)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var googleUser entities.GoogleUserInfo
	err = json.NewDecoder(resp.Body).Decode(&googleUser)
	if err != nil {
		return "", err
	}

	// cek user di database (auto register)
	user, _, err := u.userRepo.GetByEmail(googleUser.Email)
	if err != nil {
		// Jika user belum ada, buat baru
		newUser := entities.User{
			Name:      googleUser.Name,
			Email:     googleUser.Email,
			Username:  googleUser.Email,                // Email jadi username
			Password:  "GOOGLE_OAUTH_" + googleUser.ID, // Password kosong (karena OAuth)
			Role:      "user",
			AvatarURL: googleUser.Picture,
		}
		if errCreate := u.userRepo.Create(newUser, ""); errCreate != nil {
			return "", errCreate
		}
		// Ambil lagi user yang baru dibuat (untuk dapat ID)
		user, _, _ = u.userRepo.GetByEmail(googleUser.Email)
	}

	// 4. Generate JWT
	// Pastikan field user.Id sesuai dengan struct entities.GetUser kamu (Id vs ID)
	userIDToken, err := strconv.Atoi(user.Id)
	if err != nil {
		// Handle jika ID ternyata bukan angka (misal UUID)
		return "", err
	}

	return middleware.GenerateToken(userIDToken, user.Username, user.Role)
}

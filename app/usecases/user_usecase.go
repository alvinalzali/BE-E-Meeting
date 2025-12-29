package usecases

import (
	"BE-E-Meeting/app/entities"
	"BE-E-Meeting/app/repositories"
	"errors"
	"os"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// Interface UserUsecase: Daftar menu logic yang bisa dipanggil oleh Handler
type UserUsecase interface {
	Register(user entities.User) error
	Login(username string, password string) (string, string, string, error) // return: accessToken, refreshToken, userID
	GetProfile(id int) (entities.GetUser, error)
}

type userUsecase struct {
	userRepo repositories.UserRepository
}

// NewUserUsecase: Constructor untuk inject repository ke usecase
func NewUserUsecase(userRepo repositories.UserRepository) UserUsecase {
	return &userUsecase{userRepo: userRepo}
}

// --- 1. REGISTER LOGIC ---
func (u *userUsecase) Register(user entities.User) error {
	// A. Validasi Password (Logic dari main.go)
	if !isValidPassword(user.Password) {
		return errors.New("password must contain at least one uppercase letter, one lowercase letter, one number, and one special character")
	}

	// B. Hash Password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user.Password = string(hashedPassword) // Timpa password asli dengan hash

	// C. Set Default Avatar
	// (Sebaiknya URL ini dari .env, tapi kita samakan dulu dengan main.go)
	defaultAvatar := "http://localhost:8080/assets/default/default_profile.jpg"

	// D. Panggil Repository
	return u.userRepo.Create(user, defaultAvatar)
}

// --- 2. LOGIN LOGIC ---
func (u *userUsecase) Login(inputUsername string, password string) (string, string, string, error) {
	var user entities.GetUser
	var storedHash string
	var err error

	// A. Cek apakah login pakai Email atau Username
	if isEmail(inputUsername) {
		user, storedHash, err = u.userRepo.GetByEmail(inputUsername)
	} else {
		user, storedHash, err = u.userRepo.GetByUsername(inputUsername)
	}

	if err != nil {
		// Samarkan error database, return invalid credentials agar aman
		return "", "", "", errors.New("invalid credentials")
	}

	// B. Cek Password Hash
	err = bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(password))
	if err != nil {
		return "", "", "", errors.New("invalid credentials")
	}

	// C. Generate Token (Access & Refresh)
	accessToken, err := generateToken(user.Username, user.Role, 500*time.Minute)
	if err != nil {
		return "", "", "", err
	}

	refreshToken, err := generateToken(user.Username, user.Role, 7*24*time.Hour)
	if err != nil {
		return "", "", "", err
	}

	// Return UserID (struct GetUser kamu pake ID string, jadi langsung return saja)
	return accessToken, refreshToken, user.Id, nil
}

// --- 3. GET PROFILE LOGIC ---
func (u *userUsecase) GetProfile(id int) (entities.GetUser, error) {
	user, err := u.userRepo.GetByID(id)
	if err != nil {
		return user, err
	}

	// Business Logic: Pastikan avatar tidak kosong
	if user.Avatar_url == "" {
		user.Avatar_url = "http://localhost:8080/assets/default/default_profile.jpg"
	}

	return user, nil
}

// ==========================================
// HELPER FUNCTIONS (Private / Helper Logic)
// ==========================================

func isValidPassword(password string) bool {
	var hasUpper, hasLower, hasNumber, hasSpecial bool
	for _, char := range password {
		switch {
		case 'A' <= char && char <= 'Z':
			hasUpper = true
		case 'a' <= char && char <= 'z':
			hasLower = true
		case '0' <= char && char <= '9':
			hasNumber = true
		case (char >= 33 && char <= 47) || (char >= 58 && char <= 64) || (char >= 91 && char <= 96) || (char >= 123 && char <= 126):
			hasSpecial = true
		}
	}
	return hasUpper && hasLower && hasNumber && hasSpecial
}

func isEmail(input string) bool {
	for _, char := range input {
		if char == '@' {
			return true
		}
	}
	return false
}

func generateToken(username, role string, duration time.Duration) (string, error) {
	// Ambil secret dari ENV
	secretStr := os.Getenv("jwt_secret")
	if secretStr == "" {
		secretStr = os.Getenv("secret_key") // Jaga-jaga nama variabel beda
	}
	jwtSecret := []byte(secretStr)

	claims := &entities.Claims{
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(duration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

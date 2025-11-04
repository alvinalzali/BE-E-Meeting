package usecases

import (
	"BE-E-MEETING/app/models"
	"BE-E-MEETING/app/repositories"
	"golang.org/x/crypto/bcrypt"
	"github.com/golang-jwt/jwt/v5"
	"os"
	"time"
)

type UserUsecase interface {
	Login(loginData models.Login) (string, string, string, error)
	RegisterUser(newUser models.User) error
	PasswordReset(resetReq models.ResetRequest) (string, error)
	PasswordResetId(id, newPassword, confirmPassword string) error
	GetUserByID(id int) (models.GetUser, error)
	UpdateUserByID(id int, user models.UpdateUser, baseURL string) (models.UpdateUser, error)
}

type userUsecase struct {
	userRepo  repositories.UserRepository
	jwtSecret []byte
}

func NewUserUsecase(userRepo repositories.UserRepository) UserUsecase {
	return &userUsecase{
		userRepo:  userRepo,
		jwtSecret: []byte(os.Getenv("secret_key")),
	}
}

func (u *userUsecase) Login(loginData models.Login) (string, string, string, error) {
	storedUsername, storedPasswordHash, err := u.userRepo.GetUserByUsernameOrEmail(loginData.Username)
	if err != nil {
		return "", "", "", err
	}

	err = bcrypt.CompareHashAndPassword([]byte(storedPasswordHash), []byte(loginData.Password))
	if err != nil {
		return "", "", "", err
	}

	role, err := u.userRepo.GetUserRole(storedUsername)
	if err != nil {
		return "", "", "", err
	}

	accessToken, err := generateAccessToken(storedUsername, role, u.jwtSecret)
	if err != nil {
		return "", "", "", err
	}

	refreshToken, err := generateRefreshToken(storedUsername, role, u.jwtSecret)
	if err != nil {
		return "", "", "", err
	}

	userID, err := u.userRepo.GetUserID(storedUsername)
	if err != nil {
		return "", "", "", err
	}

	return accessToken, refreshToken, userID, nil
}

func (u *userUsecase) RegisterUser(newUser models.User) error {
	if !isValidPassword(newUser.Password) {
		return &UseCaseError{Code: 400, Message: "Password must contain at least one uppercase letter, one lowercase letter, one number, and one special character"}
	}

	hashedPassword, err := hashPassword(newUser.Password)
	if err != nil {
		return err
	}
	status := "active"
	return u.userRepo.RegisterUser(newUser, hashedPassword, status)
}

func (u *userUsecase) PasswordReset(resetReq models.ResetRequest) (string, error) {
	storedEmail, err := u.userRepo.GetUserEmail(resetReq.Email)
	if err != nil {
		return "", err
	}

	resetToken, err := generateResetToken(storedEmail, u.jwtSecret)
	if err != nil {
		return "", err
	}

	return resetToken, nil
}

func (u *userUsecase) PasswordResetId(id, newPassword, confirmPassword string) error {
	_, err := jwt.Parse(id, func(token *jwt.Token) (interface{}, error) {
		return u.jwtSecret, nil
	})
	if err != nil {
		return &UseCaseError{Code: 400, Message: "Invalid token"}
	}
	if newPassword != confirmPassword {
		return &UseCaseError{Code: 400, Message: "New password and confirm password do not match"}
	}
	if !isValidPassword(newPassword) {
		return &UseCaseError{Code: 400, Message: "Password must contain at least one uppercase letter, one lowercase letter, one number, and one special character"}
	}
	hashedPassword, err := hashPassword(newPassword)
	if err != nil {
		return err
	}
	return u.userRepo.UpdateUserPassword(hashedPassword, id)
}

func (u *userUsecase) GetUserByID(id int) (models.GetUser, error) {
	user, err := u.userRepo.GetUserByID(id)
	if err != nil {
		return models.GetUser{}, err
	}
	if user.AvatarURL == "" {
		user.AvatarURL = os.Getenv("DefaultAvatarURL")
	}
	return user, nil
}

func (u *userUsecase) UpdateUserByID(id int, user models.UpdateUser, baseURL string) (models.UpdateUser, error) {
	user.UpdatedAt = time.Now().Format(time.RFC3339)
	currentUser, err := u.userRepo.GetCurrentUser(id)
	if err != nil {
		return models.UpdateUser{}, err
	}

	if user.Username != "" && user.Username != currentUser.Username {
		exists, err := u.userRepo.CheckUsernameExists(user.Username, id)
		if err != nil {
			return models.UpdateUser{}, err
		}
		if exists {
			user.Username = currentUser.Username
		}
	} else {
		user.Username = currentUser.Username
	}

	if user.Email != "" && user.Email != currentUser.Email {
		exists, err := u.userRepo.CheckEmailExists(user.Email, id)
		if err != nil {
			return models.UpdateUser{}, err
		}
		if exists {
			user.Email = currentUser.Email
		}
	} else {
		user.Email = currentUser.Email
	}

	if user.AvatarURL != "" {
		user.AvatarURL = baseURL + "/assets/image/users/" + user.AvatarURL
	} else {
		user.AvatarURL = currentUser.AvatarURL
	}

	err = u.userRepo.UpdateUser(id, user)
	return user, err
}

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
		case (char >= 33 && char <= 47) || (char >= 58 && char <= 64) ||
			(char >= 91 && char <= 96) || (char >= 123 && char <= 126):
			hasSpecial = true
		}
	}
	return hasUpper && hasLower && hasNumber && hasSpecial
}

func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func generateAccessToken(username string, role string, secret []byte) (string, error) {
	claims := &models.Claims{
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(500 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

func generateRefreshToken(username string, role string, secret []byte) (string, error) {
	claims := &models.Claims{
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

func generateResetToken(email string, secret []byte) (string, error) {
	claims := &models.Claims{
		Username: email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(30 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

type UseCaseError struct {
	Code    int
	Message string
}

func (e *UseCaseError) Error() string {
	return e.Message
}

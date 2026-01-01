package handler

import (
	"net/http"
	"strconv"

	"BE-E-Meeting/app/entities"
	"BE-E-Meeting/app/usecases"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

type UserHandler struct {
	usecase usecases.UserUsecase
}

// NewUserHandler menghubungkan Handler dengan Usecase
func NewUserHandler(usecase usecases.UserUsecase) *UserHandler {
	return &UserHandler{usecase: usecase}
}

// Register godoc
// @Summary Register a new user
// @Description Register a new user
// @Tags Auth
// @Accept json
// @Produce json
// @Param user body entities.User true "User object"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /register [post]
// 1. REGISTER HANDLER
func (h *UserHandler) Register(c echo.Context) error {
	var newUser entities.User

	// Bind input JSON ke struct
	if err := c.Bind(&newUser); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Bad Request"})
	}

	// Validasi input (menggunakan validator yang ada di main.go)
	if err := c.Validate(&newUser); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Validation Error"})
	}

	// Panggil Usecase untuk proses bisnis
	err := h.usecase.Register(newUser)
	if err != nil {
		// Jika error dari validasi password di usecase
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, echo.Map{"message": "User registered successfully"})
}

// Login godoc
// @Summary Login user
// @Description Login user
// @Tags Auth
// @Accept json
// @Produce json
// @Param loginData body entities.Login true "Login data"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /login [post]
// 2. LOGIN HANDLER
func (h *UserHandler) Login(c echo.Context) error {
	var loginData entities.Login

	if err := c.Bind(&loginData); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Login Failed"})
	}

	// Panggil Usecase
	accessToken, refreshToken, userID, err := h.usecase.Login(loginData.Username, loginData.Password)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, echo.Map{"error": err.Error()})
	}

	// Set Header Response
	c.Response().Header().Set("Authorization", "Bearer "+accessToken)
	c.Response().Header().Set("Refresh-Token", "Bearer "+refreshToken)
	c.Response().Header().Set("id", userID)

	return c.JSON(http.StatusOK, echo.Map{
		"message":      "Login successful",
		"accessToken":  accessToken,
		"refreshToken": refreshToken,
		"id":           userID,
	})
}

// GetProfile godoc
// @Summary Get user by ID
// @Description Retrieve user details by user ID
// @Tags User
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /users/{id} [get]
// 3. GET PROFILE HANDLER
func (h *UserHandler) GetProfile(c echo.Context) error {
	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid ID"})
	}

	// --- Logic Authorization (Cek apakah yang akses adalah pemilik akun) ---
	userToken := c.Get("user").(*jwt.Token)
	claims := userToken.Claims.(jwt.MapClaims)
	usernameFromToken := claims["username"].(string)

	// Panggil Usecase untuk ambil data user
	user, err := h.usecase.GetProfile(id)
	if err != nil {
		return c.JSON(http.StatusNotFound, echo.Map{"error": "User not found"})
	}

	// Validasi tambahan: Cek apakah data yang diambil punya username sama dengan token
	if user.Username != usernameFromToken {
		return c.JSON(http.StatusForbidden, echo.Map{"error": "You are not allowed to access this profile"})
	}

	return c.JSON(http.StatusOK, echo.Map{
		"data":    user,
		"message": "User retrieved successfully",
	})
}

// UpdateUser godoc
// @Summary Update user by ID
// @Description Update user details by user ID
// @Tags User
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Param user body entities.UpdateUser true "User object"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /users/{id} [put]
func (h *UserHandler) UpdateUser(c echo.Context) error {
	// 1. Ambil ID dari param
	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid ID"})
	}

	// 2. Authorization (Cek Token vs ID)
	userToken := c.Get("user").(*jwt.Token)
	claims := userToken.Claims.(jwt.MapClaims)
	usernameFromToken := claims["username"].(string)

	// (Opsional) Ambil data user dulu buat cek kepemilikan
	currentUser, err := h.usecase.GetProfile(id)
	if err != nil {
		return c.JSON(http.StatusNotFound, echo.Map{"error": "User not found"})
	}
	if currentUser.Username != usernameFromToken {
		return c.JSON(http.StatusForbidden, echo.Map{"error": "Unauthorized update"})
	}

	// 3. Bind Input
	var input entities.UpdateUser
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid input"})
	}

	// 4. Panggil Usecase
	updatedUser, err := h.usecase.UpdateUser(id, input)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, echo.Map{
		"message": "User updated successfully",
		"data":    updatedUser,
	})
}

// RequestPasswordReset godoc
// @Summary Request password reset
// @Description Request a password reset token to be sent to the user's email
// @Tags User
// @Accept json
// @Produce json
// @Param user body entities.ResetRequest true "Email data"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /password/reset [post]
func (h *UserHandler) RequestPasswordReset(c echo.Context) error {
	var req entities.ResetRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid format"})
	}

	// Panggil Usecase
	token, err := h.usecase.RequestPasswordReset(req.Email)
	if err != nil {
		return c.JSON(http.StatusNotFound, echo.Map{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, echo.Map{
		"message": "Update Password Success!",
		"token":   token,
	})
}

// ResetPassword godoc
// @Summary Reset user password
// @Description Reset user password using a valid reset token
// @Tags User
// @Accept json
// @Produce json
// @Param id path string true "Reset Token (JWT)"
// @Param user body entities.PasswordConfirmReset true "Password data"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Router /password/reset/{id} [put]
func (h *UserHandler) ResetPassword(c echo.Context) error {
	// Di sini 'id' parameter sebenarnya adalah Token JWT
	token := c.Param("id")

	var req entities.PasswordConfirmReset
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid format"})
	}

	err := h.usecase.ResetPassword(token, req.NewPassword, req.ConfirmPassword)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, echo.Map{"message": "Password reset successfully"})
}

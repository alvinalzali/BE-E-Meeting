package handlers

import (
	"BE-E-MEETING/app/models"
	"BE-E-MEETING/app/usecases"
	"github.com/labstack/echo/v4"
	"net/http"
	"strconv"
)

type UserHandler struct {
	userUsecase usecases.UserUsecase
}

func NewUserHandler(userUsecase usecases.UserUsecase) *UserHandler {
	return &UserHandler{userUsecase: userUsecase}
}

// Login godoc
// @Summary Login a user
// @Description Login a user
// @Tags User
// @Accept json
// @Produce json
// @Param user body models.Login true "User credentials"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /login [post]
func (h *UserHandler) Login(c echo.Context) error {
	var loginData models.Login
	if err := c.Bind(&loginData); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid Input"})
	}
	accessToken, refreshToken, userID, err := h.userUsecase.Login(loginData)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, echo.Map{"error": "Invalid Credentials"})
	}
	c.Response().Header().Set("Authorization", "Bearer "+accessToken)
	c.Response().Header().Set("Refresh-Token", "Bearer "+refreshToken)
	c.Response().Header().Set("id", userID)
	return c.JSON(http.StatusOK, echo.Map{"message": "Login successful", "accessToken": accessToken, "refreshToken": refreshToken})
}

// RegisterUser godoc
// @Summary Register a new user
// @Description Register a new user
// @Tags User
// @Accept json
// @Produce json
// @Param user body models.User true "User object to be registered"
// @Success 201 {object} models.User
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /register [post]
func (h *UserHandler) RegisterUser(c echo.Context) error {
	var newUser models.User
	if err := c.Bind(&newUser); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid Input"})
	}
	if err := c.Validate(&newUser); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Validation Error"})
	}
	err := h.userUsecase.RegisterUser(newUser)
	if err != nil {
		if e, ok := err.(*usecases.UseCaseError); ok {
			return c.JSON(e.Code, echo.Map{"error": e.Message})
		}
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Internal Server Error"})
	}
	return c.JSON(http.StatusOK, echo.Map{"message": "User registered successfully"})
}

// PasswordReset godoc
// @Summary Request password reset
// @Description Request a password reset token to be sent to the user's email
// @Tags User
// @Accept json
// @Produce json
// @Param email body models.ResetRequest true "Email"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /password/reset [post]
func (h *UserHandler) PasswordReset(c echo.Context) error {
	var resetReq models.ResetRequest
	if err := c.Bind(&resetReq); err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}
	if err := c.Validate(&resetReq); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Validation Error"})
	}
	resetToken, err := h.userUsecase.PasswordReset(resetReq)
	if err != nil {
		return c.JSON(http.StatusNotFound, echo.Map{"error": "Email not found"})
	}
	return c.JSON(http.StatusOK, echo.Map{"message": "Update Password Success!", "token": resetToken})
}

// PasswordResetId godoc
// @Summary Reset user password
// @Description Reset user password using a valid reset token
// @Tags User
// @Accept json
// @Produce json
// @Param id path string true "Reset Token"
// @Param password body models.PasswordConfirmReset true "New Password and Confirm Password"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /password/reset/{id} [put]
func (h *UserHandler) PasswordResetId(c echo.Context) error {
	id := c.Param("id")
	var passReset models.PasswordConfirmReset
	if err := c.Bind(&passReset); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid Input"})
	}
	if err := c.Validate(&passReset); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Validation Error"})
	}
	err := h.userUsecase.PasswordResetId(id, passReset.NewPassword, passReset.ConfirmPassword)
	if err != nil {
		if e, ok := err.(*usecases.UseCaseError); ok {
			return c.JSON(e.Code, echo.Map{"error": e.Message})
		}
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Internal Server Error"})
	}
	return c.JSON(http.StatusOK, echo.Map{"message": "Password reset successfully"})
}

// GetUserByID godoc
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
// @Security     BearerAuth
// @Router /users/{id} [get]
func (h *UserHandler) GetUserByID(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid ID"})
	}
	user, err := h.userUsecase.GetUserByID(id)
	if err != nil {
		return c.JSON(http.StatusNotFound, echo.Map{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, echo.Map{"data": user, "message": "User retrieved successfully"})
}

// UpdateUserByID godoc
// @Summary Update user by ID
// @Description Update user details by user ID
// @Tags User
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Param user body models.UpdateUser true "User object to be updated"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security     BearerAuth
// @Router /users/{id} [put]
func (h *UserHandler) UpdateUserByID(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid ID"})
	}
	var user models.UpdateUser
	if err := c.Bind(&user); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid request body"})
	}
	baseURL := c.Scheme() + "://" + c.Request().Host
	updatedUser, err := h.userUsecase.UpdateUserByID(id, user, baseURL)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Database error"})
	}
	return c.JSON(http.StatusOK, echo.Map{"message": "User updated successfully", "data": updatedUser})
}

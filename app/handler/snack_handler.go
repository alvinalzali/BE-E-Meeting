package handler

import (
	"net/http"

	"BE-E-Meeting/app/usecases"

	"github.com/labstack/echo/v4"
)

type SnackHandler struct {
	usecase usecases.SnackUsecase
}

func NewSnackHandler(usecase usecases.SnackUsecase) *SnackHandler {
	return &SnackHandler{usecase: usecase}
}

// GetSnacks godoc
// @Summary Get all snacks
// @Description Get all snacks
// @Tags Snack
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /snacks [get]
func (h *SnackHandler) GetSnacks(c echo.Context) error {
	snacks, err := h.usecase.GetAll()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"message": "internal server error"})
	}

	return c.JSON(http.StatusOK, echo.Map{
		"message": "success",
		"data":    snacks,
	})
}

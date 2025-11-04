package handlers

import (
	"BE-E-MEETING/app/usecases"
	"github.com/labstack/echo/v4"
	"net/http"
)

type SnackHandler struct {
	snackUsecase usecases.SnackUsecase
}

func NewSnackHandler(snackUsecase usecases.SnackUsecase) *SnackHandler {
	return &SnackHandler{snackUsecase: snackUsecase}
}

// GetSnacks godoc
// @Summary Get all snacks
// @Description Get all snacks
// @Tags Snack
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Router /snacks [get]
func (h *SnackHandler) GetSnacks(c echo.Context) error {
	snacks, err := h.snackUsecase.GetSnacks()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"message": "internal server error"})
	}

	return c.JSON(http.StatusOK, echo.Map{
		"message": "success",
		"data":    snacks,
	})
}

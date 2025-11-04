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

func (h *SnackHandler) RegisterRoutes(e *echo.Group) {
	e.GET("/snacks", h.GetSnacks)
}

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

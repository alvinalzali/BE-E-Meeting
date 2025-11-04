package handlers

import (
	"BE-E-MEETING/app/usecases"
	"github.com/labstack/echo/v4"
	"net/http"
)

type DashboardHandler struct {
	dashboardUsecase usecases.DashboardUsecase
}

func NewDashboardHandler(dashboardUsecase usecases.DashboardUsecase) *DashboardHandler {
	return &DashboardHandler{dashboardUsecase: dashboardUsecase}
}

func (h *DashboardHandler) RegisterRoutes(e *echo.Group) {
	e.GET("/dashboard", h.GetDashboard)
}

func (h *DashboardHandler) GetDashboard(c echo.Context) error {
	startDate := c.QueryParam("startDate")
	endDate := c.QueryParam("endDate")

	response, err := h.dashboardUsecase.GetDashboard(startDate, endDate)
	if err != nil {
		if e, ok := err.(*usecases.UseCaseError); ok {
			return c.JSON(e.Code, echo.Map{"message": e.Message})
		}
		return c.JSON(http.StatusInternalServerError, echo.Map{"message": "internal server error"})
	}
	return c.JSON(http.StatusOK, response)
}

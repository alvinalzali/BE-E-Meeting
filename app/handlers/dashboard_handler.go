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

// GetDashboard godoc
// @Summary Get dashboard analytics
// @Description Get analytics data for paid transactions within date range
// @Tags Dashboard
// @Accept json
// @Produce json
// @Param startDate query string true "Start date (YYYY-MM-DD)"
// @Param endDate query string true "End date (YYYY-MM-DD)"
// @Success 200 {object} models.DashboardResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /dashboard [get]
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

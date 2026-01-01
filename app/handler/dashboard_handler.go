package handler

import (
	"net/http"

	"BE-E-Meeting/app/usecases"

	"github.com/labstack/echo/v4"
)

type DashboardHandler struct {
	usecase usecases.DashboardUsecase
}

func NewDashboardHandler(usecase usecases.DashboardUsecase) *DashboardHandler {
	return &DashboardHandler{usecase: usecase}
}

// GetDashboard godoc
// @Summary Get dashboard analytics
// @Description Get analytics data for paid transactions within date range
// @Tags Dashboard
// @Accept json
// @Produce json
// @Param startDate query string false "Start date (YYYY-MM-DD)"
// @Param endDate query string false "End date (YYYY-MM-DD)"
// @Success 200 {object} entities.DashboardResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /dashboard [get]
func (h *DashboardHandler) GetDashboard(c echo.Context) error {
	startDate := c.QueryParam("startDate")
	endDate := c.QueryParam("endDate")

	res, err := h.usecase.GetDashboard(startDate, endDate)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": err.Error()})
	}

	return c.JSON(http.StatusOK, res)
}

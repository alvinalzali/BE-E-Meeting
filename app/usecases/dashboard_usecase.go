package usecases

import (
	"BE-E-MEETING/app/models"
	"BE-E-MEETING/app/repositories"
	"net/http"
	"time"
)

type DashboardUsecase interface {
	GetDashboard(startDate, endDate string) (models.DashboardResponse, error)
}

type dashboardUsecase struct {
	dashboardRepo repositories.DashboardRepository
}

func NewDashboardUsecase(dashboardRepo repositories.DashboardRepository) DashboardUsecase {
	return &dashboardUsecase{dashboardRepo: dashboardRepo}
}

func (u *dashboardUsecase) GetDashboard(startDate, endDate string) (models.DashboardResponse, error) {
	if startDate == "" || endDate == "" {
		return models.DashboardResponse{}, &UseCaseError{Code: http.StatusBadRequest, Message: "start date and end date are required"}
	}
	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return models.DashboardResponse{}, &UseCaseError{Code: http.StatusBadRequest, Message: "invalid start date format, use YYYY-MM-DD"}
	}
	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		return models.DashboardResponse{}, &UseCaseError{Code: http.StatusBadRequest, Message: "invalid end date format, use YYYY-MM-DD"}
	}
	if start.After(end) {
		return models.DashboardResponse{}, &UseCaseError{Code: http.StatusBadRequest, Message: "start date must be smaller than end date"}
	}

	totalRoom, err := u.dashboardRepo.GetTotalRooms()
	if err != nil {
		return models.DashboardResponse{}, &UseCaseError{Code: http.StatusInternalServerError, Message: "internal server error"}
	}

	totalVisitor, totalReservation, totalOmzet, err := u.dashboardRepo.GetDashboardData(start, end)
	if err != nil {
		return models.DashboardResponse{}, &UseCaseError{Code: http.StatusInternalServerError, Message: "internal server error"}
	}

	rooms, err := u.dashboardRepo.GetRoomStats(start, end, totalReservation)
	if err != nil {
		return models.DashboardResponse{}, &UseCaseError{Code: http.StatusInternalServerError, Message: "internal server error"}
	}

	response := models.DashboardResponse{
		Message: "get dashboard data success",
	}
	response.Data.TotalRoom = totalRoom
	response.Data.TotalVisitor = totalVisitor
	response.Data.TotalReservation = totalReservation
	response.Data.TotalOmzet = totalOmzet
	response.Data.Rooms = rooms

	return response, nil
}

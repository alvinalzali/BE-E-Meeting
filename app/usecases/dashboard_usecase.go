package usecases

import (
	"errors"
	"time"

	"BE-E-Meeting/app/entities"
	"BE-E-Meeting/app/repositories"
)

type DashboardUsecase interface {
	GetDashboard(startDateStr, endDateStr string) (entities.DashboardResponse, error)
}

type dashboardUsecase struct {
	dashboardRepo repositories.DashboardRepository
}

func NewDashboardUsecase(dashboardRepo repositories.DashboardRepository) DashboardUsecase {
	return &dashboardUsecase{dashboardRepo: dashboardRepo}
}

func (u *dashboardUsecase) GetDashboard(startDateStr, endDateStr string) (entities.DashboardResponse, error) {
	var start, end time.Time
	var err error

	if startDateStr != "" {
		start, err = time.Parse("2006-01-02", startDateStr)
		if err != nil {
			return entities.DashboardResponse{}, errors.New("invalid startDate format, use YYYY-MM-DD")
		}
	}

	if endDateStr != "" {
		end, err = time.Parse("2006-01-02", endDateStr)
		if err != nil {
			return entities.DashboardResponse{}, errors.New("invalid endDate format, use YYYY-MM-DD")
		}
	}

	if !start.IsZero() && !end.IsZero() && start.After(end) {
		return entities.DashboardResponse{}, errors.New("start date must be smaller than end date")
	}

	// Panggil Repository yang mengembalikan entities.DashboardData
	data, err := u.dashboardRepo.GetDashboardData(start, end)
	if err != nil {
		return entities.DashboardResponse{}, err
	}

	// Bungkus ke dalam Response utama
	return entities.DashboardResponse{
		Message: "get dashboard data success",
		Data:    data,
	}, nil
}

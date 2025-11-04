package usecases

import (
	"BE-E-MEETING/app/models"
	"BE-E-MEETING/app/repositories"
	"database/sql"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"
)

type ReservationUsecase interface {
	CalculateReservation(roomID, snackID int, startTimeStr, endTimeStr string, participant int, name, phoneNumber, company string) (models.CalculateReservationResponse, error)
	CreateReservation(req models.ReservationRequestBody) error
	GetReservationHistory(startDate, endDate, roomType, status string, page, pageSize int) (models.ReservationHistoryResponse, error)
	GetReservationByID(id int) (models.ReservationByIDResponse, error)
	UpdateReservationStatus(req models.UpdateReservationRequest) error
	GetReservationSchedules(startDate, endDate string, page, pageSize int) (models.ScheduleResponse, error)
}

type reservationUsecase struct {
	reservationRepo repositories.ReservationRepository
	db              *sql.DB
}

func NewReservationUsecase(reservationRepo repositories.ReservationRepository, db *sql.DB) ReservationUsecase {
	return &reservationUsecase{reservationRepo: reservationRepo, db: db}
}

func (u *reservationUsecase) CalculateReservation(roomID, snackID int, startTimeStr, endTimeStr string, participant int, name, phoneNumber, company string) (models.CalculateReservationResponse, error) {
	if roomID == 0 || startTimeStr == "" || endTimeStr == "" {
		return models.CalculateReservationResponse{}, &UseCaseError{Code: http.StatusBadRequest, Message: "missing required parameters"}
	}

	startTime, err := time.Parse(time.RFC3339, startTimeStr)
	if err != nil {
		return models.CalculateReservationResponse{}, &UseCaseError{Code: http.StatusBadRequest, Message: "invalid startTime format (must be RFC3339)"}
	}
	endTime, err := time.Parse(time.RFC3339, endTimeStr)
	if err != nil {
		return models.CalculateReservationResponse{}, &UseCaseError{Code: http.StatusBadRequest, Message: "invalid endTime format (must be RFC3339)"}
	}

	room, err := u.reservationRepo.GetRoomForCalculation(roomID)
	if err != nil {
		return models.CalculateReservationResponse{}, &UseCaseError{Code: http.StatusNotFound, Message: "room not found"}
	}

	snack, err := u.reservationRepo.GetSnackForCalculation(snackID)
	if err != nil {
		return models.CalculateReservationResponse{}, &UseCaseError{Code: http.StatusNotFound, Message: "snack not found"}
	}

	conflict, err := u.reservationRepo.CheckBookingConflict(roomID, startTime, endTime)
	if err != nil {
		return models.CalculateReservationResponse{}, &UseCaseError{Code: http.StatusInternalServerError, Message: "internal server error"}
	}
	if conflict {
		return models.CalculateReservationResponse{}, &UseCaseError{Code: http.StatusBadRequest, Message: "booking conflict"}
	}

	durationMinutes := int(endTime.Sub(startTime).Minutes())
	durationHours := float64(durationMinutes) / 60.0
	subTotalRoom := room.PricePerHour * durationHours
	subTotalSnack := snack.Price * float64(participant)
	total := subTotalRoom + subTotalSnack

	roomDetail := models.RoomCalculationDetail{
		Name:          room.Name,
		PricePerHour:  room.PricePerHour,
		ImageURL:      room.PictureURL,
		Capacity:      room.Capacity,
		Type:          room.RoomType,
		SubTotalSnack: subTotalSnack,
		SubTotalRoom:  subTotalRoom,
		StartTime:     startTime,
		EndTime:       endTime,
		Duration:      durationMinutes,
		Participant:   participant,
		Snack: models.Snack{
			ID:       snack.ID,
			Name:     snack.Name,
			Unit:     snack.Unit,
			Price:    snack.Price,
			Category: snack.Category,
		},
	}

	response := models.CalculateReservationResponse{
		Message: "success",
		Data: models.CalculateReservationData{
			Rooms:         []models.RoomCalculationDetail{roomDetail},
			PersonalData:  models.PersonalData{Name: name, PhoneNumber: phoneNumber, Company: company},
			SubTotalRoom:  subTotalRoom,
			SubTotalSnack: subTotalSnack,
			Total:         total,
		},
	}
	return response, nil
}

func (u *reservationUsecase) CreateReservation(req models.ReservationRequestBody) error {
	if req.UserID <= 0 || req.Name == "" || req.PhoneNumber == "" || req.Company == "" || len(req.Rooms) == 0 {
		return &UseCaseError{Code: http.StatusBadRequest, Message: "invalid request format"}
	}

	for _, room := range req.Rooms {
		if room.StartTime.IsZero() || room.EndTime.IsZero() {
			return &UseCaseError{Code: http.StatusBadRequest, Message: "invalid start or end time"}
		}
	}

	for _, room := range req.Rooms {
		conflict, err := u.reservationRepo.CheckBookingConflict(room.ID, room.StartTime, room.EndTime)
		if err != nil {
			return &UseCaseError{Code: http.StatusInternalServerError, Message: "internal server error"}
		}
		if conflict {
			return &UseCaseError{Code: http.StatusBadRequest, Message: fmt.Sprintf("Room %d has already been booked for that time range", room.ID)}
		}
	}

	tx, err := u.db.Begin()
	if err != nil {
		return &UseCaseError{Code: http.StatusInternalServerError, Message: "internal server error"}
	}
	defer tx.Rollback()

	reservationID, err := u.reservationRepo.CreateReservation(tx, req)
	if err != nil {
		return &UseCaseError{Code: http.StatusInternalServerError, Message: "internal server error"}
	}

	var subtotalSnack float64
	var subtotalRoom float64
	var totalParticipants int
	var durationMinute int

	for _, room := range req.Rooms {
		roomTable, err := u.reservationRepo.GetRoomForCalculation(room.ID)
		if err != nil {
			return &UseCaseError{Code: http.StatusInternalServerError, Message: "internal server error"}
		}

		snackTable, err := u.reservationRepo.GetSnackForCalculation(room.SnackID)
		if err != nil {
			return &UseCaseError{Code: http.StatusInternalServerError, Message: "internal server error"}
		}

		durationMinute = int(room.EndTime.Sub(room.StartTime).Minutes())
		totalRoom := (float64(durationMinute) / 60.0) * roomTable.PricePerHour
		totalSnack := float64(room.Participant) * snackTable.Price
		subtotalRoom += totalRoom
		subtotalSnack += totalSnack
		totalParticipants += room.Participant

		err = u.reservationRepo.CreateReservationDetail(tx, reservationID, room, roomTable, snackTable, totalRoom, totalSnack, durationMinute)
		if err != nil {
			return &UseCaseError{Code: http.StatusInternalServerError, Message: "internal server error"}
		}
	}
	total := subtotalRoom + subtotalSnack
	err = u.reservationRepo.UpdateReservationTotals(tx, reservationID, subtotalRoom, subtotalSnack, total, durationMinute, totalParticipants, req.AddSnack)
	if err != nil {
		return &UseCaseError{Code: http.StatusInternalServerError, Message: "internal server error"}
	}

	return tx.Commit()
}

func (u *reservationUsecase) GetReservationHistory(startDate, endDate, roomType, status string, page, pageSize int) (models.ReservationHistoryResponse, error) {
	validTypes := map[string]bool{
		"small": true, "medium": true, "large": true,
	}
	if !validTypes[strings.ToLower(roomType)] {
		return models.ReservationHistoryResponse{}, &UseCaseError{Code: http.StatusBadRequest, Message: "room type is not valid"}
	}

	histories, totalData, err := u.reservationRepo.GetReservationHistory(startDate, endDate, roomType, status, page, pageSize)
	if err != nil {
		return models.ReservationHistoryResponse{}, &UseCaseError{Code: http.StatusInternalServerError, Message: "internal server error"}
	}

	for i, h := range histories {
		rooms, err := u.reservationRepo.GetReservationRooms(h.ID)
		if err != nil {
			return models.ReservationHistoryResponse{}, &UseCaseError{Code: http.StatusInternalServerError, Message: "internal server error"}
		}
		histories[i].Rooms = rooms
	}

	totalPage := int(math.Ceil(float64(totalData) / float64(pageSize)))
	if len(histories) == 0 {
		return models.ReservationHistoryResponse{}, &UseCaseError{Code: http.StatusNotFound, Message: "url not found"}
	}

	return models.ReservationHistoryResponse{
		Message:   "Reservation history fetched successfully",
		Data:      histories,
		Page:      page,
		PageSize:  pageSize,
		TotalPage: totalPage,
		TotalData: totalData,
	}, nil
}

func (u *reservationUsecase) GetReservationByID(id int) (models.ReservationByIDResponse, error) {
	data, err := u.reservationRepo.GetReservationByID(id)
	if err != nil {
		if err == sql.ErrNoRows {
			return models.ReservationByIDResponse{}, &UseCaseError{Code: http.StatusNotFound, Message: "url not found"}
		}
		return models.ReservationByIDResponse{}, &UseCaseError{Code: http.StatusInternalServerError, Message: "internal server error"}
	}

	rooms, err := u.reservationRepo.GetReservationDetails(id)
	if err != nil {
		return models.ReservationByIDResponse{}, &UseCaseError{Code: http.StatusInternalServerError, Message: "internal server error"}
	}
	data.Rooms = rooms

	return models.ReservationByIDResponse{
		Message: "success",
		Data:    data,
	}, nil
}

func (u *reservationUsecase) UpdateReservationStatus(req models.UpdateReservationRequest) error {
	req.Status = strings.TrimSpace(req.Status)
	if req.Status == "" {
		return &UseCaseError{Code: http.StatusBadRequest, Message: "bad request"}
	}
	if req.Status != "booked" && req.Status != "cancel" && req.Status != "paid" {
		return &UseCaseError{Code: http.StatusBadRequest, Message: "bad request"}
	}
	if req.ReservationID == 0 {
		return &UseCaseError{Code: http.StatusBadRequest, Message: "bad request"}
	}

	currentStatus, err := u.reservationRepo.GetCurrentReservationStatus(req.ReservationID)
	if err != nil {
		if err == sql.ErrNoRows {
			return &UseCaseError{Code: http.StatusNotFound, Message: "url not found"}
		}
		return &UseCaseError{Code: http.StatusInternalServerError, Message: "internal server error"}
	}

	switch currentStatus {
	case "booked":
		if req.Status != "paid" && req.Status != "cancel" {
			return &UseCaseError{Code: http.StatusBadRequest, Message: "from booked status can only change to paid or cancel"}
		}
	case "paid":
		if req.Status != "cancel" {
			return &UseCaseError{Code: http.StatusBadRequest, Message: "from paid status can only change to cancel"}
		}
	case "cancel":
		return &UseCaseError{Code: http.StatusBadRequest, Message: "canceled reservation cannot be changed"}
	}
	if currentStatus == req.Status {
		return &UseCaseError{Code: http.StatusBadRequest, Message: "new status must be different from current status"}
	}

	return u.reservationRepo.UpdateReservationStatus(req.ReservationID, req.Status)
}

func (u *reservationUsecase) GetReservationSchedules(startDate, endDate string, page, pageSize int) (models.ScheduleResponse, error) {
	if startDate == "" || endDate == "" {
		return models.ScheduleResponse{}, &UseCaseError{Code: http.StatusBadRequest, Message: "start date and end date are required"}
	}
	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return models.ScheduleResponse{}, &UseCaseError{Code: http.StatusBadRequest, Message: "invalid start date format, use YYYY-MM-DD"}
	}
	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		return models.ScheduleResponse{}, &UseCaseError{Code: http.StatusBadRequest, Message: "invalid end date format, use YYYY-MM-DD"}
	}
	if start.After(end) {
		return models.ScheduleResponse{}, &UseCaseError{Code: http.StatusBadRequest, Message: "start date must be before end date"}
	}

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}

	schedules, totalData, err := u.reservationRepo.GetReservationSchedules(start, end, page, pageSize)
	if err != nil {
		return models.ScheduleResponse{}, &UseCaseError{Code: http.StatusInternalServerError, Message: "internal server error"}
	}
	totalPages := (totalData + pageSize - 1) / pageSize

	return models.ScheduleResponse{
		Message:   "success",
		Data:      schedules,
		Page:      page,
		PageSize:  pageSize,
		TotalPage: totalPages,
		TotalData: totalData,
	}, nil
}

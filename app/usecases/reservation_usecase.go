package usecases

import (
	"errors"
	"fmt"
	"time"

	"BE-E-Meeting/app/entities"
	"BE-E-Meeting/app/repositories"
)

type ReservationUsecase interface {
	Calculate(req entities.ReservationRequest) (entities.CalculateReservationData, error)
	Create(req entities.ReservationRequest) error
	GetHistory(userID int, startDate, endDate, roomType, status string, page, pageSize int) (entities.ReservationHistoryResponse, error)
	GetByID(id int) (entities.ReservationDetailResponse, error)
	UpdateStatus(id, userID int, status string, userRole string) error
	GetUserIDByUsername(username string) (int, error)
	GetSchedules(startDate, endDate string, page, pageSize int) (entities.ScheduleResponse, error)
	GetRoomSchedule(roomID int, start, end time.Time) (map[string]interface{}, error)
}

type reservationUsecase struct {
	resRepo   repositories.ReservationRepository
	roomRepo  repositories.RoomRepository
	snackRepo repositories.SnackRepository
}

func NewReservationUsecase(resRepo repositories.ReservationRepository, roomRepo repositories.RoomRepository, snackRepo repositories.SnackRepository) ReservationUsecase {
	return &reservationUsecase{
		resRepo:   resRepo,
		roomRepo:  roomRepo,
		snackRepo: snackRepo,
	}
}

// 1. Calculate
func (u *reservationUsecase) Calculate(req entities.ReservationRequest) (entities.CalculateReservationData, error) {
	var result entities.CalculateReservationData
	if len(req.Rooms) == 0 {
		return result, errors.New("rooms cannot be empty")
	}

	for _, reqRoom := range req.Rooms {
		// [PENTING] Akses field ID (sesuai entities/room.go)
		room, err := u.roomRepo.GetByID(reqRoom.ID)
		if err != nil {
			return result, errors.New("room not found")
		}

		snackPrice := 0.0
		var snackData *entities.Snack
		if reqRoom.AddSnack && reqRoom.SnackID > 0 {
			snack, err := u.snackRepo.GetByID(reqRoom.SnackID)
			if err != nil {
				return result, errors.New("snack not found")
			}
			snackPrice = snack.Price
			snackData = &entities.Snack{ID: snack.ID, Name: snack.Name, Price: snack.Price}
		}

		available, err := u.resRepo.CheckAvailability(reqRoom.ID, reqRoom.StartTime, reqRoom.EndTime)
		if err != nil {
			return result, err
		}
		if !available {
			return result, errors.New("booking schedule conflict")
		}

		durationMins := int(reqRoom.EndTime.Sub(reqRoom.StartTime).Minutes())
		durationHours := float64(durationMins) / 60.0
		subTotalRoom := room.PricePerHour * durationHours
		subTotalSnack := snackPrice * float64(reqRoom.Participant)

		result.SubTotalRoom += subTotalRoom
		result.SubTotalSnack += subTotalSnack
		result.Rooms = append(result.Rooms, entities.RoomCalculationDetail{
			Name: room.Name, PricePerHour: room.PricePerHour, ImageURL: room.PictureURL,
			SubTotalRoom: subTotalRoom, SubTotalSnack: subTotalSnack,
			StartTime: reqRoom.StartTime, EndTime: reqRoom.EndTime,
			Duration: durationMins, Participant: reqRoom.Participant,
			Snack: snackData,
		})
	}
	result.Total = result.SubTotalRoom + result.SubTotalSnack

	return result, nil
}

// 2. Create
func (u *reservationUsecase) Create(req entities.ReservationRequest) error {
	// Cek Availability Semua Room
	for _, r := range req.Rooms {
		// [PENTING] Akses r.ID
		avail, err := u.resRepo.CheckAvailability(r.ID, r.StartTime, r.EndTime)
		if err != nil {
			return err
		}
		if !avail {
			return fmt.Errorf("room %d is already booked", r.ID)
		}
	}

	var resData entities.ReservationData
	var detData []entities.ReservationDetailData

	resData.UserID = req.UserID
	resData.ContactName = req.Name
	resData.ContactPhone = req.PhoneNumber
	resData.ContactCompany = req.Company
	resData.Note = req.Notes

	// hitung total participants dari penjumlahan room participants
	calculatedTotalParticipants := 0
	totalGlobal := 0.0

	for _, r := range req.Rooms {
		roomDB, err := u.roomRepo.GetByID(r.ID)
		if err != nil {
			return err
		}

		snackPrice := 0.0
		snackName := ""
		snackID := 0
		if r.AddSnack && r.SnackID > 0 {
			snackDB, err := u.snackRepo.GetByID(r.SnackID)
			if err != nil {
				return errors.New("snack not found")
			}
			snackPrice = snackDB.Price
			snackName = snackDB.Name
			snackID = snackDB.ID
		}

		durMins := int(r.EndTime.Sub(r.StartTime).Minutes())
		priceRoom := (float64(durMins) / 60.0) * roomDB.PricePerHour
		priceSnack := float64(r.Participant) * snackPrice

		totalGlobal += (priceRoom + priceSnack)
		resData.SubTotalRoom += priceRoom
		resData.SubTotalSnack += priceSnack
		calculatedTotalParticipants += r.Participant

		detData = append(detData, entities.ReservationDetailData{
			RoomID: roomDB.ID, RoomName: roomDB.Name, RoomPrice: roomDB.PricePerHour,
			SnackID: snackID, SnackName: snackName, SnackPrice: snackPrice,
			DurationMinute: durMins, TotalParticipants: r.Participant,
			TotalRoom: priceRoom, TotalSnack: priceSnack,
			StartAt: r.StartTime, EndAt: r.EndTime,
		})
	}
	resData.Total = totalGlobal
	resData.TotalParticipants = calculatedTotalParticipants

	return u.resRepo.Create(resData, detData)
}

// 3. Get History
func (u *reservationUsecase) GetHistory(userID int, startDate, endDate, roomType, status string, page, pageSize int) (entities.ReservationHistoryResponse, error) {
	offset := (page - 1) * pageSize
	data, total, err := u.resRepo.GetHistory(userID, startDate, endDate, roomType, status, pageSize, offset)

	return entities.ReservationHistoryResponse{
		Message:   "success",
		Data:      data,
		TotalData: total,
	}, err
}

// 4. Get By ID
func (u *reservationUsecase) GetByID(id int) (entities.ReservationDetailResponse, error) {
	data, err := u.resRepo.GetByID(id)
	return entities.ReservationDetailResponse{
		Message: "success",
		Data:    data,
	}, err
}

func (u *reservationUsecase) GetUserIDByUsername(username string) (int, error) {
	return u.resRepo.GetUserIDByUsername(username)
}

func (u *reservationUsecase) UpdateStatus(id, userID int, status string, userRole string) error {
	currentData, err := u.resRepo.GetByID(id)
	if err != nil {
		return err
	}
	if currentData.Status == "cancel" {
		return errors.New("cannot update canceled reservation")
	}
	return u.resRepo.UpdateStatus(id, status)
}

func (u *reservationUsecase) GetSchedules(startDate, endDate string, page, pageSize int) (entities.ScheduleResponse, error) {
	offset := (page - 1) * pageSize
	data, total, err := u.resRepo.GetSchedules(startDate, endDate, pageSize, offset)

	return entities.ScheduleResponse{
		Message:   "success",
		Data:      data,
		TotalData: total,
	}, err
}

func (u *reservationUsecase) GetRoomSchedule(roomID int, start, end time.Time) (map[string]interface{}, error) {
	room, err := u.roomRepo.GetByID(roomID)
	if err != nil {
		return nil, errors.New("room not found")
	}
	schedules, err := u.resRepo.GetReservationsByRoomID(roomID, start, end)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"room":      room,
		"schedules": schedules,
		"date":      start.Format("2006-01-02"),
	}, nil
}

package usecases

import (
	"errors"
	"fmt"
	"time"

	"BE-E-Meeting/app/entities"
	"BE-E-Meeting/app/repositories"
)

type ReservationUsecase interface {
	Calculate(req entities.ReservationRequestBody) (entities.CalculateReservationData, error)
	Create(req entities.ReservationRequestBody) error
	GetHistory(userID int, userRole string, startDate, endDate, roomType, status string, page, pageSize int) (entities.ReservationHistoryResponse, error)
	GetByID(id int) (entities.ReservationByIDResponse, error)
	UpdateStatus(id, userID int, status string, userRole string) error
	GetSchedules(startDate, endDate string, page, pageSize int) (entities.ScheduleResponse, error)
	GetUserIDByUsername(username string) (int, error)
	GetRoomSchedule(roomID int, start, end time.Time) (map[string]interface{}, error) // <--- BARU
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
func (u *reservationUsecase) Calculate(req entities.ReservationRequestBody) (entities.CalculateReservationData, error) {
	var result entities.CalculateReservationData
	if len(req.Rooms) == 0 {
		return result, errors.New("rooms cannot be empty")
	}
	reqRoom := req.Rooms[0]

	room, err := u.roomRepo.GetByID(reqRoom.ID)
	if err != nil {
		return result, errors.New("room not found")
	}

	snackPrice := 0.0
	snackName := ""
	if reqRoom.AddSnack && reqRoom.SnackID > 0 {
		snack, err := u.snackRepo.GetByID(reqRoom.SnackID)
		if err != nil {
			return result, errors.New("snack not found")
		}
		snackPrice = snack.Price
		snackName = snack.Name
	}

	available, err := u.resRepo.CheckAvailability(reqRoom.ID, reqRoom.StartTime, reqRoom.EndTime)
	if err != nil {
		return result, err
	}
	if !available {
		return result, errors.New("booking bentrok")
	}

	durationMins := int(reqRoom.EndTime.Sub(reqRoom.StartTime).Minutes())
	durationHours := float64(durationMins) / 60.0
	subTotalRoom := room.PricePerHour * durationHours
	subTotalSnack := snackPrice * float64(reqRoom.Participant)

	result.SubTotalRoom = subTotalRoom
	result.SubTotalSnack = subTotalSnack
	result.Total = subTotalRoom + subTotalSnack
	result.Rooms = append(result.Rooms, entities.RoomCalculationDetail{
		Name: room.Name, PricePerHour: room.PricePerHour, ImageURL: room.PictureURL,
		SubTotalRoom: subTotalRoom, SubTotalSnack: subTotalSnack,
		StartTime: reqRoom.StartTime, EndTime: reqRoom.EndTime,
		Snack: entities.Snack{ID: reqRoom.SnackID, Name: snackName, Price: snackPrice},
	})

	return result, nil
}

// 2. Create
func (u *reservationUsecase) Create(req entities.ReservationRequestBody) error {
	for _, r := range req.Rooms {
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

	totalGlobal := 0.0

	for _, r := range req.Rooms {
		roomDB, err := u.roomRepo.GetByID(r.ID)
		if err != nil {
			return err
		}

		snackPrice := 0.0
		snackName := ""
		if r.AddSnack && r.SnackID > 0 {
			snackDB, err := u.snackRepo.GetByID(r.SnackID)
			if err != nil {
				return errors.New("snack not found")
			}
			snackPrice = snackDB.Price
			snackName = snackDB.Name
		}

		durMins := int(r.EndTime.Sub(r.StartTime).Minutes())
		priceRoom := (float64(durMins) / 60.0) * roomDB.PricePerHour
		priceSnack := float64(r.Participant) * snackPrice

		totalGlobal += (priceRoom + priceSnack)
		resData.SubTotalRoom += priceRoom
		resData.SubTotalSnack += priceSnack
		resData.TotalParticipants += r.Participant

		detData = append(detData, entities.ReservationDetailData{
			RoomID: roomDB.ID, RoomName: roomDB.Name, RoomPrice: roomDB.PricePerHour,
			SnackID: r.SnackID, SnackName: snackName, SnackPrice: snackPrice,
			DurationMinute: durMins, TotalParticipants: r.Participant,
			TotalRoom: priceRoom, TotalSnack: priceSnack,
			StartAt: r.StartTime, EndAt: r.EndTime,
		})
	}
	resData.Total = totalGlobal

	return u.resRepo.Create(resData, detData)
}

func (u *reservationUsecase) GetHistory(userID int, userRole, startDate, endDate, roomType, status string, page, pageSize int) (entities.ReservationHistoryResponse, error) {
	offset := (page - 1) * pageSize
	data, total, err := u.resRepo.GetHistory(userID, userRole, startDate, endDate, roomType, status, pageSize, offset)
	return entities.ReservationHistoryResponse{Data: data, TotalData: total}, err
}

func (u *reservationUsecase) GetByID(id int) (entities.ReservationByIDResponse, error) {
	data, err := u.resRepo.GetByID(id)
	return entities.ReservationByIDResponse{Data: data}, err
}

func (u *reservationUsecase) UpdateStatus(id, userID int, status string, userRole string) error {
	targetID := id
	if targetID == 0 {
		var err error
		targetID, err = u.resRepo.GetLatestReservationIDByUserID(userID)
		if err != nil {
			return errors.New("reservation not found")
		}
	}

	currentData, err := u.resRepo.GetByID(targetID)
	if err != nil {
		return err
	}
	curStatus := currentData.Status

	switch curStatus {
	case "booked":
		if status != "paid" && status != "cancel" {
			return errors.New("invalid status transition")
		}
	case "paid":
		if status != "cancel" {
			return errors.New("can only cancel paid reservation")
		}
	case "cancel":
		return errors.New("cannot change canceled reservation")
	}

	return u.resRepo.UpdateStatus(targetID, status)
}

func (u *reservationUsecase) GetSchedules(startDate, endDate string, page, pageSize int) (entities.ScheduleResponse, error) {
	offset := (page - 1) * pageSize
	data, total, err := u.resRepo.GetSchedules(startDate, endDate, pageSize, offset)
	return entities.ScheduleResponse{Data: data, TotalData: total}, err
}

func (u *reservationUsecase) GetUserIDByUsername(username string) (int, error) {
	return u.resRepo.GetUserIDByUsername(username)
}

func (u *reservationUsecase) GetRoomSchedule(roomID int, start, end time.Time) (map[string]interface{}, error) {
	// 1. Ambil Data Room dulu
	room, err := u.roomRepo.GetByID(roomID)
	if err != nil {
		return nil, errors.New("room not found")
	}

	// 2. Ambil Schedule
	schedules, err := u.resRepo.GetReservationsByRoomID(roomID, start, end)
	if err != nil {
		return nil, err
	}

	// 3. Gabungkan Format Response
	return map[string]interface{}{
		"room":      room,
		"schedules": schedules,
		"date":      start.Format("2006-01-02"), // Info tanggal awal filter
	}, nil
}

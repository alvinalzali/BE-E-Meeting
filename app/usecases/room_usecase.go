package usecases

import (
	"BE-E-Meeting/app/entities"
	"BE-E-Meeting/app/repositories"
	"errors"
)

type RoomUsecase interface {
	Create(room entities.RoomRequest) error
	GetAll(name, roomType, capacity string, page, pageSize int) ([]entities.Room, int, int, error) // Data, TotalPage, TotalData
	GetByID(id int) (entities.Room, error)
	Update(id int, room entities.RoomRequest) error
	Delete(id int) error
}

type roomUsecase struct {
	roomRepo repositories.RoomRepository
}

func NewRoomUsecase(roomRepo repositories.RoomRepository) RoomUsecase {
	return &roomUsecase{roomRepo: roomRepo}
}

func (u *roomUsecase) Create(room entities.RoomRequest) error {
	// Validasi Input (Business Logic)
	if room.Name == "" || room.Type == "" || room.Capacity <= 0 || room.PricePerHour <= 0 {
		return errors.New("invalid room data")
	}
	return u.roomRepo.Create(room)
}

func (u *roomUsecase) GetAll(name, roomType, capacity string, page, pageSize int) ([]entities.Room, int, int, error) {
	// Validasi Filter
	if roomType != "" && roomType != "small" && roomType != "medium" && roomType != "large" {
		return nil, 0, 0, errors.New("room type is not valid")
	}

	// Default Pagination
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	offset := (page - 1) * pageSize

	rooms, totalData, err := u.roomRepo.GetAll(name, roomType, capacity, pageSize, offset)
	if err != nil {
		return nil, 0, 0, err
	}

	totalPage := (totalData + pageSize - 1) / pageSize
	return rooms, totalPage, totalData, nil
}

func (u *roomUsecase) GetByID(id int) (entities.Room, error) {
	return u.roomRepo.GetByID(id)
}

func (u *roomUsecase) Update(id int, room entities.RoomRequest) error {
	// Validasi
	if room.Type != "small" && room.Type != "medium" && room.Type != "large" || room.Capacity <= 0 {
		return errors.New("room type is not valid / capacity must be larger more than 0")
	}
	if room.PricePerHour <= 0 {
		return errors.New("price per hour must be larger more than 0")
	}

	rowsAffected, err := u.roomRepo.Update(id, room)
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errors.New("room not found")
	}

	return nil
}

func (u *roomUsecase) Delete(id int) error {
	rowsAffected, err := u.roomRepo.Delete(id)
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errors.New("room not found")
	}

	return nil
}

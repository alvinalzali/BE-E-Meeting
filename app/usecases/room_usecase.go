package usecases

import (
	"BE-E-Meeting/app/entities"
	"BE-E-Meeting/app/repositories"
	"BE-E-Meeting/app/utils" // <--- Jangan lupa import utils
	"errors"
)

type RoomUsecase interface {
	// Tambahkan parameter baseURL
	Create(room entities.RoomRequest, baseURL string) (entities.RoomRequest, error)
	GetAll(name, roomType, capacity string, page, pageSize int) ([]entities.Room, int, int, error)
	GetByID(id int) (entities.Room, error)
	// Tambahkan parameter baseURL
	Update(id int, room entities.RoomRequest, baseURL string) (entities.RoomRequest, error)
	Delete(id int) error
}

type roomUsecase struct {
	roomRepo repositories.RoomRepository
}

func NewRoomUsecase(roomRepo repositories.RoomRepository) RoomUsecase {
	return &roomUsecase{roomRepo: roomRepo}
}

// Update Signature: Tambah baseURL
func (u *roomUsecase) Create(room entities.RoomRequest, baseURL string) (entities.RoomRequest, error) { // <--- Ubah Return
	// 1. Validasi
	if room.Name == "" || room.Type == "" || room.Capacity <= 0 || room.PricePerHour <= 0 {
		return room, errors.New("invalid room data")
	}

	// 2. Logic Gambar
	if room.ImageURL != "" {
		newImageURL, err := utils.ProcessImageMove("", room.ImageURL, baseURL, "rooms")
		if err != nil {
			return room, err
		}
		// Update variable local 'room'
		room.ImageURL = newImageURL
	} else {
		// Optional default
		// room.ImageURL = baseURL + "/assets/default/default_room.jpg"
	}

	// Simpan ke DB
	err := u.roomRepo.Create(room)
	if err != nil {
		return room, err
	}

	// KEMBALIKAN 'room' YANG SUDAH DIUPDATE URL-NYA
	return room, nil
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

// Update Signature: Tambah baseURL
func (u *roomUsecase) Update(id int, room entities.RoomRequest, baseURL string) (entities.RoomRequest, error) { // <--- Ubah Return
	// Validasi
	if room.Type != "small" && room.Type != "medium" && room.Type != "large" || room.Capacity <= 0 {
		return room, errors.New("room type is not valid / capacity must be larger more than 0")
	}
	if room.PricePerHour <= 0 {
		return room, errors.New("price per hour must be larger more than 0")
	}

	oldRoom, err := u.roomRepo.GetByID(id)
	if err != nil {
		return room, errors.New("room not found")
	}

	// Logic Gambar
	if room.ImageURL != "" {
		newImageURL, err := utils.ProcessImageMove(oldRoom.PictureURL, room.ImageURL, baseURL, "rooms")
		if err != nil {
			return room, err
		}
		room.ImageURL = newImageURL
	} else {
		room.ImageURL = oldRoom.PictureURL
	}

	rowsAffected, err := u.roomRepo.Update(id, room)
	if err != nil {
		return room, err
	}
	if rowsAffected == 0 {
		return room, errors.New("room not found")
	}

	// Kembalikan room yang baru
	return room, nil
}

func (u *roomUsecase) Delete(id int) error {
	// Opsional: Sebelum delete DB, kamu bisa ambil data dulu untuk hapus gambarnya dari storage
	// oldRoom, _ := u.roomRepo.GetByID(id)

	rowsAffected, err := u.roomRepo.Delete(id)
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errors.New("room not found")
	}

	// Opsional: Hapus file fisik
	// utils.DeleteFile(oldRoom.ImageURL)

	return nil
}

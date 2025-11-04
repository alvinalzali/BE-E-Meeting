package repositories

import (
	"BE-E-MEETING/app/models"
	"database/sql"
	"fmt"
)

type RoomRepository interface {
	CreateRoom(room models.RoomRequest, imageURL string) error
	GetRooms(name, roomType, capacity string) ([]models.Room, int, error)
	GetRoomByID(id int) (models.Room, error)
	UpdateRoom(id int, room models.RoomRequest) (int64, error)
	DeleteRoom(id int) (int64, error)
	GetRoomReservationSchedule(roomID int, dateFilter string) ([]models.RoomSchedule, error)
	CheckRoomExists(roomID int) (bool, error)
}

type roomRepository struct {
	db *sql.DB
}

func NewRoomRepository(db *sql.DB) RoomRepository {
	return &roomRepository{db: db}
}

func (r *roomRepository) CreateRoom(room models.RoomRequest, imageURL string) error {
	query := `
        INSERT INTO rooms (name, room_type, capacity, price_per_hour, picture_url, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
    `
	_, err := r.db.Exec(query, room.Name, room.Type, room.Capacity, room.PricePerHour, imageURL)
	return err
}

func (r *roomRepository) GetRooms(name, roomType, capacity string) ([]models.Room, int, error) {
	query := `
        SELECT id, name, room_type, capacity, price_per_hour, picture_url, created_at, updated_at
        FROM rooms
        WHERE 1=1
    `
	var args []interface{}
	argIndex := 1

	if name != "" {
		query += fmt.Sprintf(" AND LOWER(name) LIKE LOWER($%d)", argIndex)
		args = append(args, "%"+name+"%")
		argIndex++
	}
	if roomType != "" {
		query += fmt.Sprintf(" AND room_type = $%d", argIndex)
		args = append(args, roomType)
		argIndex++
	}
	if capacity != "" {
		query += fmt.Sprintf(" AND capacity >= $%d", argIndex)
		args = append(args, capacity)
		argIndex++
	}

	countQuery := "SELECT COUNT(*) FROM (" + query + ") AS total"
	var totalData int
	err := r.db.QueryRow(countQuery, args...).Scan(&totalData)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var rooms []models.Room
	for rows.Next() {
		var room models.Room
		if err := rows.Scan(&room.ID, &room.Name, &room.RoomType, &room.Capacity, &room.PricePerHour, &room.PictureURL, &room.CreatedAt, &room.UpdatedAt); err != nil {
			return nil, 0, err
		}
		rooms = append(rooms, room)
	}
	return rooms, totalData, nil
}

func (r *roomRepository) GetRoomByID(id int) (models.Room, error) {
	query := `
        SELECT id, name, room_type, capacity, price_per_hour, picture_url, created_at, updated_at
        FROM rooms WHERE id = $1
    `
	var room models.Room
	err := r.db.QueryRow(query, id).Scan(
		&room.ID, &room.Name, &room.RoomType, &room.Capacity, &room.PricePerHour,
		&room.PictureURL, &room.CreatedAt, &room.UpdatedAt,
	)
	return room, err
}

func (r *roomRepository) UpdateRoom(id int, room models.RoomRequest) (int64, error) {
	query := `
        UPDATE rooms
        SET name=$1, room_type=$2, capacity=$3, price_per_hour=$4, picture_url=$5, updated_at=NOW()
        WHERE id=$6
    `
	res, err := r.db.Exec(query, room.Name, room.Type, room.Capacity, room.PricePerHour, room.ImageURL, id)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (r *roomRepository) DeleteRoom(id int) (int64, error) {
	query := `DELETE FROM rooms WHERE id=$1`
	res, err := r.db.Exec(query, id)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (r *roomRepository) GetRoomReservationSchedule(roomID int, dateFilter string) ([]models.RoomSchedule, error) {
	query := `
        SELECT
            rd.id,
            rd.start_at,
            rd.end_at,
            r.status_reservation,
            rd.total_participants
        FROM reservation_details rd
        JOIN reservations r ON rd.reservation_id = r.id
        WHERE rd.room_id = $1
        AND DATE(rd.start_at) = DATE($2)
        ORDER BY rd.start_at ASC
    `

	rows, err := r.db.Query(query, roomID, dateFilter)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	schedules := []models.RoomSchedule{}
	for rows.Next() {
		var schedule models.RoomSchedule
		err := rows.Scan(
			&schedule.ID,
			&schedule.StartTime,
			&schedule.EndTime,
			&schedule.Status,
			&schedule.TotalParticipant,
		)
		if err != nil {
			return nil, err
		}
		schedules = append(schedules, schedule)
	}
	return schedules, nil
}

func (r *roomRepository) CheckRoomExists(roomID int) (bool, error) {
	var roomExists bool
	err := r.db.QueryRow("SELECT EXISTS(SELECT 1 FROM rooms WHERE id = $1)", roomID).Scan(&roomExists)
	return roomExists, err
}

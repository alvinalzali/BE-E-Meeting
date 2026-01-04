package repositories

import (
	"database/sql"
	"fmt"

	"BE-E-Meeting/app/entities"
)

type RoomRepository interface {
	Create(room entities.RoomRequest) error
	GetAll(name, roomType, capacity string, limit, offset int) ([]entities.Room, int, error) // Return data + totalCount
	GetByID(id int) (entities.Room, error)
	Update(id int, room entities.RoomRequest) (int64, error) // Return rowsAffected
	Delete(id int) (int64, error)                            // Return rowsAffected
}

type roomRepository struct {
	db *sql.DB
}

func NewRoomRepository(db *sql.DB) RoomRepository {
	return &roomRepository{db: db}
}

// 1. Create
func (r *roomRepository) Create(room entities.RoomRequest) error {
	query := `
        INSERT INTO rooms (name, room_type, capacity, price_per_hour, picture_url, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
    `
	_, err := r.db.Exec(query, room.Name, room.Type, room.Capacity, room.PricePerHour, room.ImageURL)
	return err
}

// 2. GetAll (Dengan Filter & Pagination)
func (r *roomRepository) GetAll(name, roomType, capacity string, limit, offset int) ([]entities.Room, int, error) {
	// Query Dasar
	query := `SELECT id, name, room_type, capacity, price_per_hour, picture_url, created_at, updated_at FROM rooms WHERE 1=1`
	countQuery := `SELECT COUNT(*) FROM rooms WHERE 1=1`

	var args []interface{}
	argIndex := 1

	// Filter Logic
	if name != "" {
		filter := fmt.Sprintf(" AND LOWER(name) LIKE LOWER($%d)", argIndex)
		query += filter
		countQuery += filter
		args = append(args, "%"+name+"%")
		argIndex++
	}
	if roomType != "" {
		filter := fmt.Sprintf(" AND room_type = $%d", argIndex)
		query += filter
		countQuery += filter
		args = append(args, roomType)
		argIndex++
	}
	if capacity != "" {
		filter := fmt.Sprintf(" AND capacity >= $%d", argIndex)
		query += filter
		countQuery += filter
		args = append(args, capacity)
		argIndex++
	}

	// Hitung Total Data dulu
	var totalData int
	err := r.db.QueryRow(countQuery, args...).Scan(&totalData)
	if err != nil {
		return nil, 0, err
	}

	// Tambah Limit Offset ke Query Utama
	query += fmt.Sprintf(" ORDER BY id ASC LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, limit, offset)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var rooms []entities.Room
	for rows.Next() {
		var rm entities.Room
		var createdAt, updatedAt sql.NullTime // Handle null time handling

		if err := rows.Scan(&rm.ID, &rm.Name, &rm.RoomType, &rm.Capacity, &rm.PricePerHour, &rm.PictureURL, &createdAt, &updatedAt); err != nil {
			return nil, 0, err
		}

		if createdAt.Valid {
			rm.CreatedAt = createdAt.Time
		}
		if updatedAt.Valid {
			rm.UpdatedAt = updatedAt.Time
		}

		rooms = append(rooms, rm)
	}

	return rooms, totalData, nil
}

// 3. GetByID
func (r *roomRepository) GetByID(id int) (entities.Room, error) {
	query := `SELECT id, name, room_type, capacity, price_per_hour, picture_url, created_at, updated_at FROM rooms WHERE id = $1`
	var rm entities.Room
	var createdAt, updatedAt sql.NullTime

	err := r.db.QueryRow(query, id).Scan(&rm.ID, &rm.Name, &rm.RoomType, &rm.Capacity, &rm.PricePerHour, &rm.PictureURL, &createdAt, &updatedAt)

	if createdAt.Valid {
		rm.CreatedAt = createdAt.Time
	}
	if updatedAt.Valid {
		rm.UpdatedAt = updatedAt.Time
	}

	return rm, err
}

// 4. Update
func (r *roomRepository) Update(id int, room entities.RoomRequest) (int64, error) {
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

// 5. Delete
func (r *roomRepository) Delete(id int) (int64, error) {
	query := `DELETE FROM rooms WHERE id=$1`
	res, err := r.db.Exec(query, id)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

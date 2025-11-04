package repositories

import (
	"BE-E-MEETING/app/models"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type ReservationRepository interface {
	GetRoomForCalculation(roomID int) (models.Room, error)
	GetSnackForCalculation(snackID int) (models.Snack, error)
	CheckBookingConflict(roomID int, startTime, endTime time.Time) (bool, error)
	CreateReservation(tx *sql.Tx, req models.ReservationRequestBody) (int, error)
	CreateReservationDetail(tx *sql.Tx, reservationID int, room models.RoomReservationRequest, roomTable models.Room, snackTable models.Snack, totalRoom, totalSnack float64, durationMinute int) error
	UpdateReservationTotals(tx *sql.Tx, reservationID int, subtotalRoom, subtotalSnack, total float64, durationMinute, totalParticipants int, addSnack bool) error
	GetReservationHistory(startDate, endDate, roomType, status string, page, pageSize int) ([]models.ReservationHistoryData, int, error)
	GetReservationRooms(reservationID int) ([]models.ReservationHistoryRoomDetail, error)
	GetReservationByID(id int) (models.ReservationByIDData, error)
	GetReservationDetails(id int) ([]models.RoomInfo, error)
	UpdateReservationStatus(reservationID int, status string) error
	GetCurrentReservationStatus(reservationID int) (string, error)
	GetReservationSchedules(start, end time.Time, page, pageSize int) ([]models.RoomScheduleInfo, int, error)
}

type reservationRepository struct {
	db *sql.DB
}

func NewReservationRepository(db *sql.DB) ReservationRepository {
	return &reservationRepository{db: db}
}

func (r *reservationRepository) GetRoomForCalculation(roomID int) (models.Room, error) {
	var room models.Room
	err := r.db.QueryRow(`
		SELECT id, name, room_type, capacity, price_per_hour, picture_url, created_at, updated_at
		FROM rooms WHERE id = $1
	`, roomID).Scan(&room.ID, &room.Name, &room.RoomType, &room.Capacity, &room.PricePerHour, &room.PictureURL, &room.CreatedAt, &room.UpdatedAt)
	return room, err
}

func (r *reservationRepository) GetSnackForCalculation(snackID int) (models.Snack, error) {
	var snack models.Snack
	err := r.db.QueryRow(`
		SELECT id, name, unit, price, category
		FROM snacks WHERE id = $1
	`, snackID).Scan(&snack.ID, &snack.Name, &snack.Unit, &snack.Price, &snack.Category)
	return snack, err
}

func (r *reservationRepository) CheckBookingConflict(roomID int, startTime, endTime time.Time) (bool, error) {
	var existing int
	err := r.db.QueryRow(`
		SELECT COUNT(*)
		FROM reservation_details
		WHERE room_id = $1
		AND (
			(start_at, end_at) OVERLAPS ($2, $3)
		)
	`, roomID, startTime, endTime).Scan(&existing)
	return existing > 0, err
}

func (r *reservationRepository) CreateReservation(tx *sql.Tx, req models.ReservationRequestBody) (int, error) {
	var reservationID int
	err := tx.QueryRow(`
		INSERT INTO reservations (
			user_id, contact_name, contact_phone, contact_company,
			note, status_reservation, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, 'booked', NOW(), NOW())
		RETURNING id
	`, req.UserID, req.Name, req.PhoneNumber, req.Company, req.Notes).Scan(&reservationID)
	return reservationID, err
}

func (r *reservationRepository) CreateReservationDetail(tx *sql.Tx, reservationID int, room models.RoomReservationRequest, roomTable models.Room, snackTable models.Snack, totalRoom, totalSnack float64, durationMinute int) error {
	_, err := tx.Exec(`
		INSERT INTO reservation_details (
			reservation_id,
			room_id, room_name, room_price,
			snack_id, snack_name, snack_price,
			duration_minute, total_participants,
			total_room, total_snack,
			start_at, end_at,
			created_at, updated_at
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,NOW(),NOW())
	`,
		reservationID,
		room.ID, roomTable.Name, roomTable.PricePerHour,
		room.SnackID, snackTable.Name, snackTable.Price,
		durationMinute, room.Participant,
		totalRoom, totalSnack,
		room.StartTime, room.EndTime,
	)
	return err
}

func (r *reservationRepository) UpdateReservationTotals(tx *sql.Tx, reservationID int, subtotalRoom, subtotalSnack, total float64, durationMinute, totalParticipants int, addSnack bool) error {
	_, err := tx.Exec(`
		UPDATE reservations
		SET subtotal_room = $1,
			subtotal_snack = $2,
			duration_minute = $3,
			total = $4,
			total_participants = $5,
			add_snack = $6,
			updated_at = NOW()
		WHERE id = $7
	`, subtotalRoom, subtotalSnack, durationMinute, total, totalParticipants, addSnack, reservationID)
	return err
}

func (r *reservationRepository) GetReservationHistory(startDate, endDate, roomType, status string, page, pageSize int) ([]models.ReservationHistoryData, int, error) {
	query := `
	SELECT
		r.id, r.contact_name, r.contact_phone, r.contact_company,
		COALESCE(SUM(rd.snack_price),0) AS sub_total_snack,
		COALESCE(SUM(rd.room_price),0) AS sub_total_room,
		COALESCE(SUM(rd.snack_price + rd.room_price),0) AS total,
		r.status_reservation, r.created_at, r.updated_at
	FROM reservations r
	JOIN reservation_details rd ON rd.reservation_id = r.id
	JOIN rooms rm ON rm.id = rd.room_id
	WHERE 1=1
	`
	args := []interface{}{}
	argIdx := 1

	if startDate != "" {
		query += fmt.Sprintf(" AND r.created_at >= $%d", argIdx)
		args = append(args, startDate)
		argIdx++
	}
	if endDate != "" {
		query += fmt.Sprintf(" AND r.created_at <= $%d", argIdx)
		args = append(args, endDate)
		argIdx++
	}
	if roomType != "" {
		query += fmt.Sprintf(" AND rm.room_type = $%d", argIdx)
		args = append(args, roomType)
		argIdx++
	}
	if status != "" {
		query += fmt.Sprintf(" AND r.status_reservation = $%d", argIdx)
		args = append(args, status)
		argIdx++
	}

	countQuery := `
		SELECT COUNT(DISTINCT r.id)
		FROM reservations r
		JOIN reservation_details rd ON rd.reservation_id = r.id
		JOIN rooms rm ON rm.id = rd.room_id
		WHERE 1=1
	` + query[strings.Index(query, "WHERE"):]
	var totalData int
	err := r.db.QueryRow(countQuery, args...).Scan(&totalData)
	if err != nil {
		return nil, 0, err
	}

	query += `
	GROUP BY r.id, r.contact_name, r.contact_phone, r.contact_company, r.status_reservation, r.created_at, r.updated_at
	ORDER BY r.created_at DESC
	LIMIT $%d OFFSET $%d
	`
	args = append(args, pageSize, (page-1)*pageSize)
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var histories []models.ReservationHistoryData
	for rows.Next() {
		var h models.ReservationHistoryData
		err := rows.Scan(
			&h.ID, &h.Name, &h.PhoneNumber, &h.Company,
			&h.SubTotalSnack, &h.SubTotalRoom, &h.Total,
			&h.Status, &h.CreatedAt, &h.UpdatedAt,
		)
		if err != nil {
			return nil, 0, err
		}
		histories = append(histories, h)
	}
	return histories, totalData, nil
}

func (r *reservationRepository) GetReservationRooms(reservationID int) ([]models.ReservationHistoryRoomDetail, error) {
	roomRows, err := r.db.Query(`
			SELECT rm.id, rm.price_per_hour, rm.name, rm.room_type,
				COALESCE(rd.room_price,0), COALESCE(rd.snack_price,0)
			FROM reservation_details rd
			JOIN rooms rm ON rm.id = rd.room_id
			WHERE rd.reservation_id = $1
		`, reservationID)
	if err != nil {
		return nil, err
	}
	defer roomRows.Close()

	var rooms []models.ReservationHistoryRoomDetail
	for roomRows.Next() {
		var room models.ReservationHistoryRoomDetail
		err := roomRows.Scan(
			&room.ID, &room.Price, &room.Name, &room.Type,
			&room.TotalRoom, &room.TotalSnack,
		)
		if err != nil {
			return nil, err
		}
		rooms = append(rooms, room)
	}
	return rooms, nil
}

func (r *reservationRepository) GetReservationByID(id int) (models.ReservationByIDData, error) {
	var data models.ReservationByIDData
	err := r.db.QueryRow(`
        SELECT contact_name, contact_phone, contact_company,
               COALESCE(subtotal_snack, 0) as subtotal_snack,
               COALESCE(subtotal_room, 0) as subtotal_room,
               COALESCE(total, 0) as total,
               COALESCE(status_reservation::text, '') as status_reservation
        FROM reservations
        WHERE id = $1
    `, id).Scan(&data.PersonalData.Name, &data.PersonalData.PhoneNumber, &data.PersonalData.Company, &data.SubTotalSnack, &data.SubTotalRoom, &data.Total, &data.Status)
	return data, err
}

func (r *reservationRepository) GetReservationDetails(id int) ([]models.RoomInfo, error) {
	rows, err := r.db.Query(`
        SELECT
            COALESCE(r.name, '') as room_name,
            COALESCE(r.price_per_hour, 0) as price_per_hour,
            COALESCE(r.picture_url, '') as image_url,
            COALESCE(r.capacity, 0) as capacity,
            COALESCE(r.room_type::text, 'small') as room_type,
            COALESCE(rd.total_snack, 0) as total_snack,
            COALESCE(rd.total_room, 0) as total_room,
            rd.start_at,
            rd.end_at,
            COALESCE(rd.duration_minute, 0) as duration,
            COALESCE(rd.total_participants, 0) as participant,
            s.id as snack_id,
			COALESCE(s.name, '') as snack_name,
            COALESCE(s.unit::text, '') as snack_unit,
            COALESCE(s.price, 0) as snack_price,
            COALESCE(s.category::text, '') as snack_category
        FROM reservation_details rd
        LEFT JOIN rooms r ON rd.room_id = r.id
        LEFT JOIN snacks s ON rd.snack_id = s.id
        WHERE rd.reservation_id = $1
    `, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rooms []models.RoomInfo
	for rows.Next() {
		var room models.RoomInfo
		var snack models.Snack
		var startAt, endAt sql.NullTime

		err := rows.Scan(
			&room.Name, &room.PricePerHour, &room.ImageURL, &room.Capacity, &room.Type,
			&room.TotalSnack, &room.TotalRoom, &startAt, &endAt, &room.Duration, &room.Participant,
			&snack.ID, &snack.Name, &snack.Unit, &snack.Price, &snack.Category,
		)
		if err != nil {
			return nil, err
		}

		if startAt.Valid {
			room.StartTime = startAt.Time.Format(time.RFC3339)
		}
		if endAt.Valid {
			room.EndTime = endAt.Time.Format(time.RFC3339)
		}

		if snack.ID > 0 {
			room.Snack = &snack
		}
		rooms = append(rooms, room)
	}
	return rooms, nil
}

func (r *reservationRepository) UpdateReservationStatus(reservationID int, status string) error {
	_, err := r.db.Exec(`UPDATE reservations SET status_reservation=$1::status_reservation WHERE id=$2`,
		status, reservationID)
	return err
}

func (r *reservationRepository) GetCurrentReservationStatus(reservationID int) (string, error) {
	var currentStatus sql.NullString
	err := r.db.QueryRow(`SELECT status_reservation FROM reservations WHERE id=$1`, reservationID).Scan(&currentStatus)
	return currentStatus.String, err
}

func (r *reservationRepository) GetReservationSchedules(start, end time.Time, page, pageSize int) ([]models.RoomScheduleInfo, int, error) {
	offset := (page - 1) * pageSize
	var totalData int
	countQuery := `
        SELECT COUNT(DISTINCT rd.room_id)
        FROM reservation_details rd
        JOIN reservations r ON rd.reservation_id = r.id
        WHERE DATE(rd.start_at) BETWEEN $1 AND $2
    `
	err := r.db.QueryRow(countQuery, start, end).Scan(&totalData)
	if err != nil {
		return nil, 0, err
	}

	query := `
        WITH RoomReservations AS (
            SELECT DISTINCT rd.room_id
            FROM reservation_details rd
            WHERE DATE(rd.start_at) BETWEEN $1 AND $2
            LIMIT $3 OFFSET $4
        )
        SELECT
            r.id,
            r.name AS room_name,
            res.contact_company,
            rd.start_at,
            rd.end_at,
            CASE
                WHEN rd.end_at < NOW() THEN 'Done'
                WHEN rd.start_at <= NOW() AND rd.end_at >= NOW() THEN 'In Progress'
                ELSE 'Up Coming'
            END as status
        FROM RoomReservations rr
        JOIN rooms r ON rr.room_id = r.id
        LEFT JOIN reservation_details rd ON r.id = rd.room_id
        LEFT JOIN reservations res ON rd.reservation_id = res.id
        WHERE DATE(rd.start_at) BETWEEN $1 AND $2
        ORDER BY r.id, rd.start_at
    `

	rows, err := r.db.Query(query, start, end, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	scheduleMap := make(map[string]*models.RoomScheduleInfo)
	for rows.Next() {
		var (
			roomID, roomName   string
			companyName        sql.NullString
			startTime, endTime time.Time
			status             string
		)

		err := rows.Scan(&roomID, &roomName, &companyName, &startTime, &endTime, &status)
		if err != nil {
			return nil, 0, err
		}

		if _, exists := scheduleMap[roomID]; !exists {
			scheduleMap[roomID] = &models.RoomScheduleInfo{
				ID:          roomID,
				RoomName:    roomName,
				CompanyName: companyName.String,
				Schedules:   make([]models.Schedule, 0),
			}
		}

		scheduleMap[roomID].Schedules = append(scheduleMap[roomID].Schedules, models.Schedule{
			StartTime: startTime.Format(time.RFC3339),
			EndTime:   endTime.Format(time.RFC3339),
			Status:    status,
		})
	}
	schedules := make([]models.RoomScheduleInfo, 0, len(scheduleMap))
	for _, schedule := range scheduleMap {
		schedules = append(schedules, *schedule)
	}
	return schedules, totalData, nil
}

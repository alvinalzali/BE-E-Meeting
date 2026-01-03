package repositories

import (
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"BE-E-Meeting/app/entities"
)

type ReservationRepository interface {
	CheckAvailability(roomID int, startTime, endTime time.Time) (bool, error)
	Create(reservation entities.ReservationData, details []entities.ReservationDetailData) error
	GetHistory(userID int, userRole, startDate, endDate, roomType, status string, limit, offset int) ([]entities.ReservationHistoryData, int, error)
	GetByID(id int) (entities.ReservationByIDData, error)
	UpdateStatus(id int, status string) error
	GetSchedules(startDate, endDate string, limit, offset int) ([]entities.RoomScheduleInfo, int, error)
	GetUserIDByUsername(username string) (int, error)
	GetLatestReservationIDByUserID(userID int) (int, error)
	GetReservationsByRoomID(roomID int, start, end time.Time) ([]entities.RoomSchedule, error) // <--- BARU
}

type reservationRepository struct {
	db *sql.DB
}

func NewReservationRepository(db *sql.DB) ReservationRepository {
	return &reservationRepository{db: db}
}

// 1. Cek Availability
func (r *reservationRepository) CheckAvailability(roomID int, startTime, endTime time.Time) (bool, error) {
	var existing int
	query := `
		SELECT COUNT(*) 
		FROM reservation_details 
		WHERE room_id = $1
		AND (start_at, end_at) OVERLAPS ($2, $3)
	`
	err := r.db.QueryRow(query, roomID, startTime, endTime).Scan(&existing)
	if err != nil {
		return false, err
	}
	return existing == 0, nil
}

// 2. Create (Transaction)
func (r *reservationRepository) Create(res entities.ReservationData, details []entities.ReservationDetailData) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			tx.Rollback()
		} else {
			tx.Commit()
		}
	}()

	var reservationID int
	queryHeader := `
		INSERT INTO reservations (
			user_id, contact_name, contact_phone, contact_company,
			note, status_reservation, subtotal_room, subtotal_snack, 
			total, duration_minute, total_participants, add_snack, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, 'booked', $6, $7, $8, $9, $10, $11, NOW(), NOW())
		RETURNING id
	`
	err = tx.QueryRow(queryHeader,
		res.UserID, res.ContactName, res.ContactPhone, res.ContactCompany, res.Note,
		res.SubTotalRoom, res.SubTotalSnack, res.Total, res.DurationMinute, res.TotalParticipants, res.AddSnack,
	).Scan(&reservationID)

	if err != nil {
		return err
	}

	queryDetail := `
		INSERT INTO reservation_details (
			reservation_id, room_id, room_name, room_price,
			snack_id, snack_name, snack_price,
			duration_minute, total_participants,
			total_room, total_snack,
			start_at, end_at,
			created_at, updated_at
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,NOW(),NOW())
	`

	for _, d := range details {
		_, err = tx.Exec(queryDetail,
			reservationID,
			d.RoomID, d.RoomName, d.RoomPrice,
			d.SnackID, d.SnackName, d.SnackPrice,
			d.DurationMinute, d.TotalParticipants,
			d.TotalRoom, d.TotalSnack,
			d.StartAt, d.EndAt,
		)
		if err != nil {
			return err
		}
	}

	return nil
}

// 3. Get History (Dengan Pagination & Real Count)
func (r *reservationRepository) GetHistory(userID int, userRole, startDate, endDate, roomType, status string, limit, offset int) ([]entities.ReservationHistoryData, int, error) {

	filterSQL := " WHERE 1=1"
	var args []interface{}
	argIdx := 1

	if userRole == "user" {
		filterSQL += fmt.Sprintf(" AND r.user_id = $%d", argIdx)
		args = append(args, userID)
		argIdx++
	}
	if startDate != "" {
		filterSQL += fmt.Sprintf(" AND r.created_at >= $%d", argIdx)
		args = append(args, startDate)
		argIdx++
	}
	if endDate != "" {
		filterSQL += fmt.Sprintf(" AND r.created_at <= $%d", argIdx)
		args = append(args, endDate)
		argIdx++
	}
	if roomType != "" {
		filterSQL += fmt.Sprintf(" AND rm.room_type = $%d", argIdx)
		args = append(args, roomType)
		argIdx++
	}
	if status != "" {
		filterSQL += fmt.Sprintf(" AND r.status_reservation = $%d", argIdx)
		args = append(args, status)
		argIdx++
	}

	// Count Query
	countQuery := `
		SELECT COUNT(DISTINCT r.id) 
		FROM reservations r
		JOIN reservation_details rd ON rd.reservation_id = r.id
		JOIN rooms rm ON rm.id = rd.room_id
	` + filterSQL

	var totalData int
	err := r.db.QueryRow(countQuery, args...).Scan(&totalData)
	if err != nil {
		return nil, 0, err
	}

	// Main Query
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
	` + filterSQL + fmt.Sprintf(`
		GROUP BY r.id, r.contact_name, r.contact_phone, r.contact_company, r.status_reservation, r.created_at, r.updated_at
		ORDER BY r.created_at DESC
		LIMIT $%d OFFSET $%d
	`, argIdx, argIdx+1)

	args = append(args, limit, offset)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var histories []entities.ReservationHistoryData
	for rows.Next() {
		var h entities.ReservationHistoryData
		if err := rows.Scan(&h.ID, &h.Name, &h.PhoneNumber, &h.Company, &h.SubTotalSnack, &h.SubTotalRoom, &h.Total, &h.Status, &h.CreatedAt, &h.UpdatedAt); err != nil {
			return nil, 0, err
		}

		roomRows, _ := r.db.Query(`
			SELECT 
				rm.id, rm.price_per_hour, rm.name, rm.room_type, 
				COALESCE(rd.room_price,0), COALESCE(rd.snack_price,0) 
			FROM reservation_details rd 
			JOIN rooms rm ON rm.id = rd.room_id 
			WHERE rd.reservation_id = $1
		`, h.ID)

		for roomRows.Next() {
			var rd entities.ReservationHistoryRoomDetail
			roomRows.Scan(&rd.ID, &rd.Price, &rd.Name, &rd.Type, &rd.TotalRoom, &rd.TotalSnack)
			h.Rooms = append(h.Rooms, rd)
		}
		roomRows.Close()

		histories = append(histories, h)
	}

	return histories, totalData, nil
}

// 4. Get By ID
func (r *reservationRepository) GetByID(id int) (entities.ReservationByIDData, error) {
	var data entities.ReservationByIDData
	var status sql.NullString

	err := r.db.QueryRow(`
		SELECT contact_name, contact_phone, contact_company, COALESCE(subtotal_room, 0), COALESCE(subtotal_snack, 0), COALESCE(total, 0), COALESCE(status_reservation::text, '')
		FROM reservations WHERE id = $1
	`, id).Scan(&data.PersonalData.Name, &data.PersonalData.PhoneNumber, &data.PersonalData.Company, &data.SubTotalRoom, &data.SubTotalSnack, &data.Total, &status)

	if err != nil {
		return data, err
	}
	data.Status = status.String

	rows, err := r.db.Query(`
		SELECT COALESCE(r.name,''), COALESCE(r.price_per_hour,0), COALESCE(r.picture_url,''), COALESCE(r.capacity,0), COALESCE(r.room_type::text,'small'),
			   COALESCE(rd.total_snack,0), COALESCE(rd.total_room,0), rd.start_at, rd.end_at, COALESCE(rd.duration_minute,0), COALESCE(rd.total_participants,0),
			   s.id, COALESCE(s.name,''), COALESCE(s.unit::text,''), COALESCE(s.price,0), COALESCE(s.category::text,'')
		FROM reservation_details rd
		LEFT JOIN rooms r ON rd.room_id = r.id
		LEFT JOIN snacks s ON rd.snack_id = s.id
		WHERE rd.reservation_id = $1
	`, id)
	if err != nil {
		return data, err
	}
	defer rows.Close()

	for rows.Next() {
		var room entities.RoomInfo
		var snack entities.Snack
		var startAt, endAt sql.NullTime
		rows.Scan(&room.Name, &room.PricePerHour, &room.ImageURL, &room.Capacity, &room.Type, &room.TotalSnack, &room.TotalRoom, &startAt, &endAt, &room.Duration, &room.Participant, &snack.ID, &snack.Name, &snack.Unit, &snack.Price, &snack.Category)

		if startAt.Valid {
			room.StartTime = startAt.Time.Format(time.RFC3339)
		}
		if endAt.Valid {
			room.EndTime = endAt.Time.Format(time.RFC3339)
		}
		if snack.ID > 0 {
			room.Snack = &snack
		}

		data.Rooms = append(data.Rooms, room)
	}

	return data, nil
}

// 5. Update Status
func (r *reservationRepository) UpdateStatus(id int, status string) error {
	_, err := r.db.Exec(`UPDATE reservations SET status_reservation=$1::status_reservation WHERE id=$2`, status, id)
	return err
}

// 6. Get Schedules
func (r *reservationRepository) GetSchedules(startDate, endDate string, limit, offset int) ([]entities.RoomScheduleInfo, int, error) {
	filterSQL := " WHERE 1=1 "
	args := []interface{}{}
	argIdx := 1
	if startDate != "" {
		filterSQL += fmt.Sprintf(" AND DATE(rd.start_at) >= $%d", argIdx)
		args = append(args, startDate)
		argIdx++
	}
	if endDate != "" {
		filterSQL += fmt.Sprintf(" AND DATE(rd.start_at) <= $%d", argIdx)
		args = append(args, endDate)
		argIdx++
	}

	countQuery := `
		SELECT COUNT(DISTINCT rd.room_id)
		FROM reservation_details rd
		LEFT JOIN reservations res ON rd.reservation_id = res.id
	` + filterSQL

	var totalData int
	err := r.db.QueryRow(countQuery, args...).Scan(&totalData)
	if err != nil {
		return nil, 0, err
	}

	query := `
		SELECT r.id, r.name, res.contact_company, rd.start_at, rd.end_at,
			CASE WHEN rd.end_at < NOW() THEN 'Done' WHEN rd.start_at <= NOW() AND rd.end_at >= NOW() THEN 'In Progress' ELSE 'Up Coming' END as status
		FROM reservation_details rd
		JOIN rooms r ON r.id = rd.room_id
		LEFT JOIN reservations res ON rd.reservation_id = res.id
	` + filterSQL + fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIdx, argIdx+1)

	args = append(args, limit, offset)
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	scheduleMap := make(map[int]*entities.RoomScheduleInfo)
	for rows.Next() {
		var roomID int
		var roomName string
		var comp sql.NullString
		var start, end time.Time
		var st string
		rows.Scan(&roomID, &roomName, &comp, &start, &end, &st)

		if _, exists := scheduleMap[roomID]; !exists {
			scheduleMap[roomID] = &entities.RoomScheduleInfo{ID: strconv.Itoa(roomID), RoomName: roomName, CompanyName: comp.String}
		}
		scheduleMap[roomID].Schedules = append(scheduleMap[roomID].Schedules, entities.Schedule{StartTime: start.Format(time.RFC3339), EndTime: end.Format(time.RFC3339), Status: st})
	}

	var results []entities.RoomScheduleInfo
	for _, v := range scheduleMap {
		results = append(results, *v)
	}

	return results, totalData, nil
}

// 7. Helpers
func (r *reservationRepository) GetUserIDByUsername(username string) (int, error) {
	var id int
	err := r.db.QueryRow("SELECT id FROM users WHERE username = $1", username).Scan(&id)
	return id, err
}

func (r *reservationRepository) GetLatestReservationIDByUserID(userID int) (int, error) {
	var id int
	err := r.db.QueryRow(`SELECT id FROM reservations WHERE user_id=$1 ORDER BY created_at DESC LIMIT 1`, userID).Scan(&id)
	return id, err
}

func (r *reservationRepository) GetReservationsByRoomID(roomID int, start, end time.Time) ([]entities.RoomSchedule, error) {
	var rows *sql.Rows
	var err error

	// query jadwal yang bentrok
	query := `
        SELECT rd.id, rd.start_at, rd.end_at, r.status_reservation, rd.total_participants 
        FROM reservation_details rd 
        JOIN reservations r ON rd.reservation_id = r.id 
        WHERE rd.room_id = $1 
        AND (rd.start_at, rd.end_at) OVERLAPS ($2, $3) 
        ORDER BY rd.start_at ASC
    `
	rows, err = r.db.Query(query, roomID, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var schedules []entities.RoomSchedule
	for rows.Next() {
		var s entities.RoomSchedule
		var startAt, endAt sql.NullTime
		var status sql.NullString
		var p sql.NullInt64

		rows.Scan(&s.ID, &startAt, &endAt, &status, &p)

		if startAt.Valid {
			s.StartTime = startAt.Time
		}
		if endAt.Valid {
			s.EndTime = endAt.Time
		}
		if status.Valid {
			s.Status = status.String
		}
		if p.Valid {
			s.TotalParticipant = int(p.Int64)
		}

		schedules = append(schedules, s)
	}
	return schedules, nil
}

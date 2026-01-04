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
	GetHistory(userID int, startDate, endDate, roomType, status string, limit, offset int) ([]entities.ReservationHistoryData, int, error)
	GetByID(id int) (entities.ReservationHistoryData, error)
	UpdateStatus(id int, status string) error
	GetUserIDByUsername(username string) (int, error)
	GetSchedules(startDate, endDate string, limit, offset int) ([]entities.RoomScheduleInfo, int, error)
	GetReservationsByRoomID(roomID int, start, end time.Time) ([]entities.RoomSchedule, error)
}

type reservationRepository struct {
	db *sql.DB
}

func NewReservationRepository(db *sql.DB) ReservationRepository {
	return &reservationRepository{db: db}
}

// 1. Availability
func (r *reservationRepository) CheckAvailability(roomID int, startTime, endTime time.Time) (bool, error) {
	var existing int
	query := `SELECT COUNT(*) FROM reservation_details WHERE room_id = $1 AND (start_at, end_at) OVERLAPS ($2, $3)`
	err := r.db.QueryRow(query, roomID, startTime, endTime).Scan(&existing)
	return existing == 0, err
}

// 2. Create
func (r *reservationRepository) Create(res entities.ReservationData, details []entities.ReservationDetailData) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var reservationID int
	queryHeader := `
		INSERT INTO reservations (user_id, contact_name, contact_phone, contact_company, note, status_reservation, subtotal_room, subtotal_snack, total, duration_minute, total_participants, add_snack, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, 'booked', $6, $7, $8, 0, $9, $10, NOW(), NOW()) RETURNING id`

	// Perhatikan mapping $ nya
	err = tx.QueryRow(queryHeader,
		res.UserID, res.ContactName, res.ContactPhone, res.ContactCompany, res.Note,
		res.SubTotalRoom, res.SubTotalSnack, res.Total, res.TotalParticipants, res.AddSnack,
	).Scan(&reservationID)

	if err != nil {
		return err
	}

	queryDetail := `
		INSERT INTO reservation_details (reservation_id, room_id, room_name, room_price, snack_id, snack_name, snack_price, duration_minute, total_participants, total_room, total_snack, start_at, end_at, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,NOW(),NOW())`

	for _, d := range details {
		_, err = tx.Exec(queryDetail, reservationID, d.RoomID, d.RoomName, d.RoomPrice, d.SnackID, d.SnackName, d.SnackPrice, d.DurationMinute, d.TotalParticipants, d.TotalRoom, d.TotalSnack, d.StartAt, d.EndAt)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

// 3. Get History
func (r *reservationRepository) GetHistory(userID int, startDate, endDate, roomType, status string, limit, offset int) ([]entities.ReservationHistoryData, int, error) {
	// Query Count (Perbaikan: rm.type -> rm.room_type)
	countQuery := `
		SELECT COUNT(DISTINCT r.id) 
		FROM reservations r 
		JOIN reservation_details rd ON r.id = rd.reservation_id
		JOIN rooms rm ON rd.room_id = rm.id 
		WHERE 1=1 `

	// Query Data (Perbaikan: rm.type -> rm.room_type)
	query := `
		SELECT 
			r.id, r.contact_name, r.contact_phone, r.contact_company, 
			r.subtotal_snack, r.subtotal_room, r.total, r.status_reservation, r.created_at,
			rd.room_id, rm.name, rm.room_type, rm.price_per_hour, rd.total_room, rd.total_snack
		FROM reservations r
		JOIN reservation_details rd ON r.id = rd.reservation_id
		JOIN rooms rm ON rd.room_id = rm.id
		WHERE 1=1 `

	var args []interface{}
	argCount := 1

	// Filter Logic
	if userID != 0 {
		filter := fmt.Sprintf(" AND r.user_id = $%d", argCount)
		countQuery += filter
		query += filter
		args = append(args, userID)
		argCount++
	}
	if startDate != "" && endDate != "" {
		filter := fmt.Sprintf(" AND DATE(rd.start_at) >= $%d AND DATE(rd.end_at) <= $%d", argCount, argCount+1)
		countQuery += filter
		query += filter
		args = append(args, startDate, endDate)
		argCount += 2
	}

	// Perbaikan Filter Room Type (rm.type -> rm.room_type)
	if roomType != "" {
		filter := fmt.Sprintf(" AND rm.room_type = $%d", argCount)
		countQuery += filter
		query += filter
		args = append(args, roomType)
		argCount++
	}

	if status != "" {
		filter := fmt.Sprintf(" AND r.status_reservation = $%d", argCount)
		countQuery += filter
		query += filter
		args = append(args, status)
		argCount++
	}

	var totalData int
	err := r.db.QueryRow(countQuery, args...).Scan(&totalData)
	if err != nil {
		return nil, 0, err
	}

	query += fmt.Sprintf(" ORDER BY r.created_at DESC LIMIT $%d OFFSET $%d", argCount, argCount+1)
	args = append(args, limit, offset)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	resultMap := make(map[int]*entities.ReservationHistoryData)
	var order []int

	for rows.Next() {
		var resID int
		var name, phone, company, stat string
		var subSnack, subRoom, total float64
		var createdAt time.Time
		var roomID int
		var roomName, rType string
		var rPrice, rTotal, rSnackTotal float64

		err := rows.Scan(&resID, &name, &phone, &company, &subSnack, &subRoom, &total, &stat, &createdAt,
			&roomID, &roomName, &rType, &rPrice, &rTotal, &rSnackTotal)
		if err != nil {
			return nil, 0, err
		}

		if _, exists := resultMap[resID]; !exists {
			resultMap[resID] = &entities.ReservationHistoryData{
				ID: resID, Name: name, PhoneNumber: phone, Company: company,
				SubTotalSnack: subSnack, SubTotalRoom: subRoom, Total: total, Status: stat,
				CreatedAt: createdAt,
				Rooms:     []entities.ReservationRoomDetail{},
			}
			order = append(order, resID)
		}

		resultMap[resID].Rooms = append(resultMap[resID].Rooms, entities.ReservationRoomDetail{
			ID: roomID, Name: roomName, Type: rType, Price: rPrice, TotalRoom: rTotal, TotalSnack: rSnackTotal,
		})
	}

	var finalResult []entities.ReservationHistoryData
	for _, id := range order {
		finalResult = append(finalResult, *resultMap[id])
	}

	return finalResult, totalData, nil
}

// 4. Get By ID
func (r *reservationRepository) GetByID(id int) (entities.ReservationHistoryData, error) {
	var data entities.ReservationHistoryData

	queryHeader := `
		SELECT id, contact_name, contact_phone, contact_company, subtotal_snack, subtotal_room, total, status_reservation, created_at 
		FROM reservations WHERE id = $1`

	err := r.db.QueryRow(queryHeader, id).Scan(
		&data.ID, &data.Name, &data.PhoneNumber, &data.Company,
		&data.SubTotalSnack, &data.SubTotalRoom, &data.Total, &data.Status, &data.CreatedAt,
	)
	if err != nil {
		return data, err
	}

	queryDetails := `
		SELECT rd.room_id, r.name, r.room_type, r.price_per_hour, rd.total_room, rd.total_snack
		FROM reservation_details rd
		JOIN rooms r ON rd.room_id = r.id
		WHERE rd.reservation_id = $1`

	rows, err := r.db.Query(queryDetails, id)
	if err != nil {
		return data, err
	}
	defer rows.Close()

	for rows.Next() {
		var room entities.ReservationRoomDetail
		rows.Scan(&room.ID, &room.Name, &room.Type, &room.Price, &room.TotalRoom, &room.TotalSnack)
		data.Rooms = append(data.Rooms, room)
	}

	return data, nil
}

func (r *reservationRepository) GetUserIDByUsername(username string) (int, error) {
	var id int
	err := r.db.QueryRow("SELECT id FROM users WHERE username = $1", username).Scan(&id)
	return id, err
}

func (r *reservationRepository) UpdateStatus(id int, status string) error {
	_, err := r.db.Exec(`UPDATE reservations SET status_reservation=$1 WHERE id=$2`, status, id)
	return err
}

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
			CASE 
				WHEN rd.end_at < NOW() THEN 'Done' 
				WHEN rd.start_at <= NOW() AND rd.end_at >= NOW() THEN 'In Progress' 
				ELSE 'Up Coming' 
			END as status
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
			scheduleMap[roomID] = &entities.RoomScheduleInfo{
				ID:          strconv.Itoa(roomID),
				RoomName:    roomName,
				CompanyName: comp.String,
			}
		}
		scheduleMap[roomID].Schedules = append(scheduleMap[roomID].Schedules, entities.Schedule{
			StartTime: start.Format(time.RFC3339),
			EndTime:   end.Format(time.RFC3339),
			Status:    st,
		})
	}

	var results []entities.RoomScheduleInfo
	for _, v := range scheduleMap {
		results = append(results, *v)
	}

	return results, totalData, nil
}

func (r *reservationRepository) GetReservationsByRoomID(roomID int, start, end time.Time) ([]entities.RoomSchedule, error) {
	query := `
		SELECT rd.id, rd.start_at, rd.end_at, r.status_reservation, rd.total_participants 
		FROM reservation_details rd 
		JOIN reservations r ON rd.reservation_id = r.id 
		WHERE rd.room_id = $1 
		AND (rd.start_at, rd.end_at) OVERLAPS ($2, $3) 
		ORDER BY rd.start_at ASC
	`
	rows, err := r.db.Query(query, roomID, start, end)
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

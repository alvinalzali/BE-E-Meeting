package repositories

import (
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"BE-E-Meeting/app/entities"
)

type DashboardRepository interface {
	GetDashboardData(startDate, endDate time.Time) (entities.DashboardData, error)
}

type dashboardRepository struct {
	db *sql.DB
}

func NewDashboardRepository(db *sql.DB) DashboardRepository {
	return &dashboardRepository{db: db}
}

func (r *dashboardRepository) GetDashboardData(startDate, endDate time.Time) (entities.DashboardData, error) {
	// 1. Inisialisasi slice agar tidak return null di JSON
	result := entities.DashboardData{
		Rooms: []entities.DashboardRoom{},
	}

	// ------------------------------------------
	// A. HITUNG TOTAL ROOM
	// ------------------------------------------
	err := r.db.QueryRow(`SELECT COUNT(*) FROM rooms`).Scan(&result.TotalRoom)
	if err != nil {
		return result, err
	}

	// ------------------------------------------
	// B. BANGUN FILTER (Tanggal & Status Paid)
	// ------------------------------------------
	// Filter ini hanya akan ditempelkan pada tabel RESERVASI, bukan pada tabel ROOMS
	// agar rooms yang tidak laku tetap muncul di list.

	filterConditions := " WHERE res.status_reservation = 'paid' "
	var args []interface{}
	argIdx := 1

	if !startDate.IsZero() {
		filterConditions += fmt.Sprintf(" AND DATE(rd.start_at) >= $%d", argIdx)
		args = append(args, startDate)
		argIdx++
	}
	if !endDate.IsZero() {
		filterConditions += fmt.Sprintf(" AND DATE(rd.end_at) <= $%d", argIdx)
		args = append(args, endDate)
		argIdx++
	}

	// ------------------------------------------
	// C. HITUNG TOTAL STATS (Visitor, Reservation, Omzet)
	// ------------------------------------------
	// Query ini menggunakan INNER JOIN karena kita memang hanya mau menghitung yang ada transaksinya
	totalsQuery := `
		SELECT 
			COALESCE(SUM(rd.total_participants), 0),
			COUNT(DISTINCT res.id),
			COALESCE(SUM(res.total), 0)
		FROM reservations res
		JOIN reservation_details rd ON res.id = rd.reservation_id
	` + filterConditions

	err = r.db.QueryRow(totalsQuery, args...).Scan(&result.TotalVisitor, &result.TotalReservation, &result.TotalOmzet)
	if err != nil {
		return result, err
	}

	// ------------------------------------------
	// D. HITUNG ROOM STATS (Per Ruangan)
	// ------------------------------------------
	// Teknik: Kita filter dulu reservasinya di dalam subquery (FilteredRes),
	// baru kita LEFT JOIN ke tabel rooms. Ini menjamin semua room tetap muncul.

	roomQuery := `
		SELECT 
			r.id,
			r.name,
			COALESCE(SUM(FilteredRes.total), 0) AS omzet,
			
			-- Rumus Persentase: (Jumlah Reservasi Room Ini / Total Reservasi Global) * 100
			CASE 
				WHEN $` + strconv.Itoa(argIdx) + ` = 0 THEN 0
				ELSE (COUNT(DISTINCT FilteredRes.reservation_id)::float / $` + strconv.Itoa(argIdx) + `::float) * 100
			END AS percentage_of_usage

		FROM rooms r
		LEFT JOIN (
			SELECT res.id as reservation_id, res.total, rd.room_id
			FROM reservations res
			JOIN reservation_details rd ON res.id = rd.reservation_id
			` + filterConditions + `
		) FilteredRes ON r.id = FilteredRes.room_id
		
		GROUP BY r.id, r.name
		ORDER BY omzet DESC
	`

	// Tambahkan TotalReservation ke args untuk pembagi rumus persentase
	roomArgs := append(args, result.TotalReservation)

	rows, err := r.db.Query(roomQuery, roomArgs...)
	if err != nil {
		return result, err
	}
	defer rows.Close()

	for rows.Next() {
		var room entities.DashboardRoom
		if err := rows.Scan(&room.ID, &room.Name, &room.Omzet, &room.PercentageOfUsage); err != nil {
			return result, err
		}
		result.Rooms = append(result.Rooms, room)
	}

	return result, nil
}

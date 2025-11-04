package repositories

import (
	"BE-E-MEETING/app/models"
	"database/sql"
	"time"
)

type DashboardRepository interface {
	GetTotalRooms() (int, error)
	GetDashboardData(start, end time.Time) (int, int, float64, error)
	GetRoomStats(start, end time.Time, totalReservation int) ([]models.DashboardRoom, error)
}

type dashboardRepository struct {
	db *sql.DB
}

func NewDashboardRepository(db *sql.DB) DashboardRepository {
	return &dashboardRepository{db: db}
}

func (r *dashboardRepository) GetTotalRooms() (int, error) {
	var totalRoom int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM rooms`).Scan(&totalRoom)
	return totalRoom, err
}

func (r *dashboardRepository) GetDashboardData(start, end time.Time) (int, int, float64, error) {
	var totalVisitor, totalReservation int
	var totalOmzet float64
	err := r.db.QueryRow(`
        SELECT
            COALESCE(SUM(rd.total_participants), 0) as total_visitors,
            COUNT(DISTINCT r.id) as total_reservations,
            COALESCE(SUM(r.total), 0) as total_omzet
        FROM reservations r
        JOIN reservation_details rd ON r.id = rd.reservation_id
        WHERE r.status_reservation = 'paid'
        AND DATE(r.created_at) BETWEEN $1 AND $2
    `, start, end).Scan(&totalVisitor, &totalReservation, &totalOmzet)
	return totalVisitor, totalReservation, totalOmzet, err
}

func (r *dashboardRepository) GetRoomStats(start, end time.Time, totalReservation int) ([]models.DashboardRoom, error) {
	rows, err := r.db.Query(`
        WITH RoomStats AS (
            SELECT
                r.id,
                r.name,
                COALESCE(SUM(res.total), 0) as omzet,
                COUNT(DISTINCT res.id) as reservation_count
            FROM rooms r
            LEFT JOIN reservation_details rd ON r.id = rd.room_id
            LEFT JOIN reservations res ON rd.reservation_id = res.id
                AND res.status_reservation = 'paid'
                AND DATE(res.created_at) BETWEEN $1 AND $2
            GROUP BY r.id, r.name
        )
        SELECT
            id,
            name,
            omzet,
            CASE
                WHEN $3 = 0 THEN 0
                ELSE (reservation_count::float / $3::float) * 100
            END as percentage_of_usage
        FROM RoomStats
        ORDER BY omzet DESC
    `, start, end, totalReservation)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rooms []models.DashboardRoom
	for rows.Next() {
		var room models.DashboardRoom
		err := rows.Scan(&room.ID, &room.Name, &room.Omzet, &room.PercentageOfUsage)
		if err != nil {
			return nil, err
		}
		rooms = append(rooms, room)
	}
	return rooms, nil
}

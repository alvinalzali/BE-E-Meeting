package entities

import "time"

// route GET /reservations/schedules
type Schedule struct {
	StartTime string `json:"startTime"`
	EndTime   string `json:"endTime"`
	Status    string `json:"status"`
}

type RoomScheduleInfo struct {
	ID          string     `json:"id"`
	RoomName    string     `json:"roomName"`
	CompanyName string     `json:"companyName"`
	Schedules   []Schedule `json:"schedules"`
}

type ScheduleResponse struct {
	Message   string             `json:"message"`
	Data      []RoomScheduleInfo `json:"data"`
	Page      int                `json:"page"`
	PageSize  int                `json:"pageSize"`
	TotalPage int                `json:"totalPage"`
	TotalData int                `json:"totalData"`
}

// route GET /dashboard
type DashboardRoom struct {
	ID                int     `json:"id"`
	Name              string  `json:"name"`
	Omzet             float64 `json:"omzet"`
	PercentageOfUsage float64 `json:"percentageOfUsage"`
}

// --- PERUBAHAN DISINI: Kita buat struct terpisah untuk Data ---
type DashboardData struct {
	TotalRoom        int             `json:"totalRoom"`
	TotalVisitor     int             `json:"totalVisitor"`
	TotalReservation int             `json:"totalReservation"`
	TotalOmzet       float64         `json:"totalOmzet"`
	Rooms            []DashboardRoom `json:"rooms"`
}

type DashboardResponse struct {
	Message string        `json:"message"`
	Data    DashboardData `json:"data"` // Menggunakan struct bernama
}

// Struct untuk legacy / other usage
type RoomSchedule struct {
	ID               int       `json:"id"`
	StartTime        time.Time `json:"startTime"`
	EndTime          time.Time `json:"endTime"`
	Status           string    `json:"status"`
	TotalParticipant int       `json:"totalParticipant"`
}

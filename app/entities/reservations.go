package entities

import (
	"time"
)

// ==========================================
// 1. REQUEST MODELS
// ==========================================

type ReservationRequest struct {
	UserID      int    `json:"userID"` // Diisi token
	Name        string `json:"name" validate:"required"`
	PhoneNumber string `json:"phoneNumber" validate:"required"`
	Company     string `json:"company" validate:"required"`
	Notes       string `json:"notes"`
	// TotalParticipants bisa dihitung dari jumlah peserta per room,
	// tapi jika butuh data global, kita simpan disini.
	TotalParticipants int                      `json:"totalParticipants"`
	Rooms             []RoomReservationRequest `json:"rooms" validate:"required,min=1"`
}

type UpdateReservationRequest struct {
	ReservationID int    `json:"reservation_id" validate:"required"`
	Status        string `json:"status" validate:"required,oneof=booked cancel paid"`
}

// ==========================================
// 2. RESPONSE MODELS
// ==========================================

// --- A. Calculation Response ---
type CalculateReservationData struct {
	Rooms         []RoomCalculationDetail `json:"rooms"`
	SubTotalRoom  float64                 `json:"subTotalRoom"`
	SubTotalSnack float64                 `json:"subTotalSnack"`
	Total         float64                 `json:"total"`
}

type RoomCalculationDetail struct {
	Name          string    `json:"name"`
	PricePerHour  float64   `json:"pricePerHour"`
	ImageURL      string    `json:"imageURL"`
	SubTotalSnack float64   `json:"subTotalSnack"`
	SubTotalRoom  float64   `json:"subTotalRoom"`
	StartTime     time.Time `json:"startTime"`
	EndTime       time.Time `json:"endTime"`
	Duration      int       `json:"duration"` // menit
	Participant   int       `json:"participant"`
	Snack         *Snack    `json:"snack"`
}

// --- B. History & Detail Response ---

type ReservationHistoryResponse struct {
	Message   string                   `json:"message"`
	Data      []ReservationHistoryData `json:"data"`
	Page      int                      `json:"page"`
	PageSize  int                      `json:"pageSize"`
	TotalPage int                      `json:"totalPage"`
	TotalData int                      `json:"totalData"`
}

type ReservationDetailResponse struct {
	Message string                 `json:"message"`
	Data    ReservationHistoryData `json:"data"`
}

type ReservationHistoryData struct {
	ID            int       `json:"id"`
	Name          string    `json:"name"`
	PhoneNumber   string    `json:"phoneNumber"`
	Company       string    `json:"company"`
	SubTotalSnack float64   `json:"subTotalSnack"`
	SubTotalRoom  float64   `json:"subTotalRoom"`
	Total         float64   `json:"total"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"createdAt"`
	// Kita gunakan struct khusus untuk response history agar aman dari format RoomInfo
	Rooms []ReservationRoomDetail `json:"rooms"`
}

type ReservationRoomDetail struct {
	ID         int     `json:"id"`
	Name       string  `json:"name"`
	Type       string  `json:"type"`
	Price      float64 `json:"price"`
	TotalRoom  float64 `json:"totalRoom"`
	TotalSnack float64 `json:"totalSnack"`
}

// --- C. Schedule Response ---

type ScheduleResponse struct {
	Message   string             `json:"message"`
	Data      []RoomScheduleInfo `json:"data"`
	TotalData int                `json:"totalData"`
}

type RoomScheduleInfo struct {
	ID          string     `json:"id"`
	RoomName    string     `json:"roomName"`
	CompanyName string     `json:"companyName"`
	Schedules   []Schedule `json:"schedules"`
}

type Schedule struct {
	StartTime string `json:"startTime"`
	EndTime   string `json:"endTime"`
	Status    string `json:"status"`
}

type RoomSchedule struct {
	ID               int       `json:"id"`
	StartTime        time.Time `json:"startTime"`
	EndTime          time.Time `json:"endTime"`
	Status           string    `json:"status"`
	TotalParticipant int       `json:"totalParticipant"`
}

// ==========================================
// 3. REPOSITORY DTOs
// ==========================================

type ReservationData struct {
	ID                int
	UserID            int
	ContactName       string
	ContactPhone      string
	ContactCompany    string
	Note              string
	StatusReservation string
	SubTotalRoom      float64
	SubTotalSnack     float64
	Total             float64
	TotalParticipants int
	AddSnack          bool
	DurationMinute    int // Field ini penting untuk Repository
}

type ReservationDetailData struct {
	ReservationID     int
	RoomID            int
	RoomName          string
	RoomPrice         float64
	SnackID           int
	SnackName         string
	SnackPrice        float64
	DurationMinute    int
	TotalParticipants int
	TotalRoom         float64
	TotalSnack        float64
	StartAt           time.Time
	EndAt             time.Time
}

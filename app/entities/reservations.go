package entities

import (
	"time"
)

type ReservationRequestBody struct {
	UserID            int                      `json:"userID"`
	Name              string                   `json:"name"`
	PhoneNumber       string                   `json:"phoneNumber"`
	Company           string                   `json:"company"`
	Notes             string                   `json:"notes"`
	TotalParticipants int                      `json:"totalParticipants"` // total keseluruhan peserta
	AddSnack          bool                     `json:"addSnack"`          // apakah reservasi ini melibatkan snack
	Rooms             []RoomReservationRequest `json:"rooms"`
}

// Response struct history
type HistoryResponse struct {
	Message string               `json:"message"`
	Data    []ReservationHistory `json:"data"`
}

// Data struct h
type ReservationHistory struct {
	ID            int     `json:"id"`
	Name          string  `json:"name"`
	PhoneNumber   float64 `json:"phoneNumber"`
	Company       string  `json:"company"`
	SubTotalSnack float64 `json:"subTotalSnack"`
	SubTotalRoom  float64 `json:"subTotalRoom"`
	GrandTotal    float64 `json:"grandTotal"`
	Type          string  `json:"type"`
	Status        string  `json:"status"`
	CreatedAt     string  `json:"createdAt"`
}

// Struct Reservation History :
// Untuk respons utama
type ReservationHistoryResponse struct {
	Message   string                   `json:"message"`
	Data      []ReservationHistoryData `json:"data"`
	Page      int                      `json:"page"`
	PageSize  int                      `json:"pageSize"`
	TotalPage int                      `json:"totalPage"`
	TotalData int                      `json:"totalData"`
}

// Room detail dalam response perhitungan reservasi
type RoomCalculationDetail struct {
	Name          string    `json:"name"`
	PricePerHour  float64   `json:"pricePerHour"`
	ImageURL      string    `json:"imageURL"`
	Capacity      int       `json:"capacity"`
	Type          string    `json:"type"`
	SubTotalSnack float64   `json:"subTotalSnack"`
	SubTotalRoom  float64   `json:"subTotalRoom"`
	StartTime     time.Time `json:"startTime"`
	EndTime       time.Time `json:"endTime"`
	Duration      int       `json:"duration"`
	Participant   int       `json:"participant"`
	Snack         Snack     `json:"snack"`
}

// Data personal yang disertakan pada reservasi
type PersonalData struct {
	Name        string `json:"name"`
	PhoneNumber string `json:"phoneNumber"`
	Company     string `json:"company"`
}

type CalculateReservationResponse struct {
	Message string                   `json:"message"`
	Data    CalculateReservationData `json:"data"`
}

type CalculateReservationData struct {
	Rooms         []RoomCalculationDetail `json:"rooms"`
	PersonalData  PersonalData            `json:"personalData"`
	SubTotalRoom  float64                 `json:"subTotalRoom"`
	SubTotalSnack float64                 `json:"subTotalSnack"`
	Total         float64                 `json:"total"`
}

// Data utama per reservation
type ReservationHistoryData struct {
	ID            int                            `json:"id"`
	Name          string                         `json:"name"`
	PhoneNumber   string                         `json:"phoneNumber"`
	Company       string                         `json:"company"`
	SubTotalSnack float64                        `json:"subTotalSnack"`
	SubTotalRoom  float64                        `json:"subTotalRoom"`
	Total         float64                        `json:"total"`
	Status        string                         `json:"status"`
	CreatedAt     time.Time                      `json:"createdAt"`
	UpdatedAt     *time.Time                     `json:"updatedAt"` // <--- Ganti jadi Pointer Time
	Rooms         []ReservationHistoryRoomDetail `json:"rooms"`
}

// Detail room di dalam reservation
type ReservationHistoryRoomDetail struct {
	ID         int     `json:"id"`
	Price      float64 `json:"price"`
	Name       string  `json:"name"`
	Type       string  `json:"type"`
	TotalRoom  float64 `json:"totalRoom"`
	TotalSnack float64 `json:"totalSnack"`
}

type UpdateReservationRequest struct {
	ReservationID int    `json:"reservation_id" validate:"required"`
	Status        string `json:"status" validate:"required,oneof=booked cancel paid"`
}

// Add this response types
type ReservationByIDData struct {
	Rooms         []RoomInfo   `json:"rooms"`
	PersonalData  PersonalData `json:"personalData"`
	SubTotalSnack float64      `json:"subTotalSnack"`
	SubTotalRoom  float64      `json:"subTotalRoom"`
	Total         float64      `json:"total"`
	Status        string       `json:"status"`
}

type ReservationByIDResponse struct {
	Message string              `json:"message"`
	Data    ReservationByIDData `json:"data"`
}

// Struct ini mewakili baris data di tabel 'reservations'
type ReservationData struct {
	UserID            int
	ContactName       string
	ContactPhone      string
	ContactCompany    string
	Note              string
	StatusReservation string
	SubTotalRoom      float64
	SubTotalSnack     float64
	Total             float64
	DurationMinute    int
	TotalParticipants int
	AddSnack          bool
}

// Struct ini mewakili baris data di tabel 'reservation_details'
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

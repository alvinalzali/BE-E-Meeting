package entities

import (
	"time"
)

// Request body untuk endpoint rooms
type RoomRequest struct {
	Name         string  `json:"name"`
	Type         string  `json:"type"`
	Capacity     int     `json:"capacity"`
	PricePerHour float64 `json:"pricePerHour"`
	ImageURL     string  `json:"imageURL"`
}

// Response struct untuk rooms
type Room struct {
	ID           int       `json:"id"`
	Name         string    `json:"name"`
	RoomType     string    `json:"type"`
	Capacity     int       `json:"capacity"`
	PricePerHour float64   `json:"pricePerHour"`
	PictureURL   string    `json:"imageURL"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type RoomReservationRequest struct {
	ID          int       `json:"roomID"`
	StartTime   time.Time `json:"startTime"`
	EndTime     time.Time `json:"endTime"`
	Participant int       `json:"participant"`
	SnackID     int       `json:"snackID"`
	AddSnack    bool      `json:"addSnack"`
}

type RoomInfo struct {
	Name         string  `json:"name"`
	PricePerHour float64 `json:"pricePerHour"`
	ImageURL     string  `json:"imageURL"`
	Capacity     int     `json:"capacity"`
	Type         string  `json:"type"`
	TotalSnack   float64 `json:"totalSnack"`
	TotalRoom    float64 `json:"totalRoom"`
	StartTime    string  `json:"startTime"`
	EndTime      string  `json:"endTime"`
	Duration     int     `json:"duration"`
	Participant  int     `json:"participant"`
	Snack        *Snack  `json:"snack,omitempty"`
}

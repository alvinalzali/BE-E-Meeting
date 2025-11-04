package models

import "time"

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

// route GET /rooms/:id/reservation
type RoomSchedule struct {
	ID               int       `json:"id"`
	StartTime        time.Time `json:"startTime"`
	EndTime          time.Time `json:"endTime"`
	Status           string    `json:"status"`
	TotalParticipant int       `json:"totalParticipant"`
}

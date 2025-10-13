package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Model
type Room struct {
	ID           uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Name         string    `gorm:"size:100;not null" json:"name"`
	PricePerHour float64   `gorm:"type:numeric(12,2);not null" json:"pricePerHour"`
	ImageURL     string    `gorm:"size:255;column:image_url" json:"imageURL"`
	Capacity     int       `gorm:"not null" json:"capacity"`
	RoomType     string    `gorm:"size:20;not null;column:room_type" json:"type"`
	CreatedAt    time.Time `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime" json:"updatedAt"`
}

type Snack struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Name      string    `gorm:"size:100;not null" json:"name"`
	Category  string    `gorm:"size:50;not null" json:"category"`
	Unit      string    `gorm:"size:20;not null" json:"unit"`
	Price     float64   `gorm:"type:numeric(12,2);not null" json:"price"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updatedAt"`
}

type ReservationDetail struct {
	ID                uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	ReservationID     int       `json:"reservation_id"`
	RoomID            int       `json:"room_id"`
	RoomName          string    `gorm:"size:100;not null" json:"room_name"`
	RoomPrice         float64   `gorm:"type:numeric(12,2);not null" json:"room_price"`
	SnackID           int       `json:"snack_id"`
	SnackName         string    `gorm:"size:100;not null" json:"snack_name"`
	SnackPrice        float64   `gorm:"type:numeric(12,2);not null" json:"snack_price"`
	DurationMinute    int       `json:"duration_minute"`
	TotalParticipants int       `json:"total_participants"`
	StartAt           time.Time `json:"start_at"`
	EndAt             time.Time `json:"end_at"`
	TotalSnack        float64   `gorm:"type:numeric(14,2)" json:"total_snack"`
	TotalRoom         float64   `gorm:"type:numeric(14,2)" json:"total_room"`
	CreatedAt         time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt         time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

var validRoomTypes = map[string]bool{
	"small":  true,
	"medium": true,
	"large":  true,
}

func initDB() *gorm.DB {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("[error] DATABASE_URL is not set in environment")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("[error] connect db: %v", err)
	}
	if err := db.AutoMigrate(&Room{}, &Snack{}, &ReservationDetail{}); err != nil {
		log.Fatalf("[error] migrate: %v", err)
	}
	fmt.Println("✅ Database connected")
	return db
}

func main() {
	db := initDB()
	seedSnacks(db)

	e := echo.New()

	// auth middleware + inject db into context
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			auth := strings.TrimSpace(c.Request().Header.Get("Authorization"))
			log.Printf("Authorization header: %q", auth)
			if auth == "" {
				return c.JSON(http.StatusUnauthorized, map[string]string{"message": "unauthorized"})
			}
			c.Set("db", db)
			return next(c)
		}
	})

	// custom JSON 404
	e.HTTPErrorHandler = func(err error, c echo.Context) {
		if he, ok := err.(*echo.HTTPError); ok && he.Code == http.StatusNotFound {
			_ = c.JSON(http.StatusNotFound, map[string]string{"message": "url not found"})
			return
		}
		c.Echo().DefaultHTTPErrorHandler(err, c)
	}

	// routes
	e.POST("/rooms", createRoom)
	e.GET("/rooms", getRooms)
	e.PUT("/rooms/:id", updateRoom)
	e.DELETE("/rooms/:id", deleteRoom)
	e.GET("/snacks", getSnacks)
	e.GET("/reservation-details", getReservationDetails)

	addr := "localhost:8080"
	fmt.Println("listening on", addr)
	e.Logger.Fatal(e.Start(addr))
}

// Create Room - POST /rooms
func createRoom(c echo.Context) error {
	db := c.Get("db").(*gorm.DB)

	var req Room
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"message": "invalid request"})
	}

	req.RoomType = strings.TrimSpace(req.RoomType)
	req.Name = strings.TrimSpace(req.Name)

	if !validRoomTypes[req.RoomType] {
		return c.JSON(http.StatusBadRequest, map[string]string{"message": "room type is not valid"})
	}
	if req.Capacity <= 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"message": "capacity must be larger more than 0"})
	}
	if req.Name == "" || req.PricePerHour <= 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"message": "invalid request"})
	}

	if err := db.Create(&req).Error; err != nil {
		log.Printf("[error] create room: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"message": "internal server error"})
	}

	return c.JSON(http.StatusCreated, map[string]string{"message": "create room success"})
}

// Get Rooms - GET /rooms?name=&type=&capacity=&page=1&pageSize=20
func getRooms(c echo.Context) error {
	db := c.Get("db").(*gorm.DB)
	var rooms []Room

	name := strings.TrimSpace(c.QueryParam("name"))
	rtype := strings.TrimSpace(c.QueryParam("type"))
	capacityStr := strings.TrimSpace(c.QueryParam("capacity"))
	pageStr := c.QueryParam("page")
	pageSizeStr := c.QueryParam("pageSize")

	if rtype != "" && !validRoomTypes[rtype] {
		return c.JSON(http.StatusBadRequest, map[string]string{"message": "room type is not valid"})
	}

	page := 1
	if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
		page = p
	}
	pageSize := 20
	if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 {
		pageSize = ps
	}

	query := db.Model(&Room{})
	if name != "" {
		query = query.Where("name ILIKE ?", "%"+name+"%")
	}
	if rtype != "" {
		query = query.Where("room_type = ?", rtype)
	}
	if capacityStr != "" {
		if capFilter, err := strconv.Atoi(capacityStr); err == nil {
			query = query.Where("capacity >= ?", capFilter)
		}
	}

	var totalData int64
	if err := query.Count(&totalData).Error; err != nil {
		log.Printf("[error] count rooms: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"message": "internal server error"})
	}
	totalPage := (int(totalData) + pageSize - 1) / pageSize

	if err := query.Offset((page - 1) * pageSize).Limit(pageSize).Order("id asc").Find(&rooms).Error; err != nil {
		log.Printf("[error] find rooms: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"message": "internal server error"})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message":   "get rooms success",
		"data":      rooms,
		"page":      page,
		"pageSize":  pageSize,
		"totalPage": totalPage,
		"totalData": totalData,
	})
}

// Update Room - PUT /rooms/:id
func updateRoom(c echo.Context) error {
	db := c.Get("db").(*gorm.DB)
	id := c.Param("id")

	var room Room
	if err := db.First(&room, id).Error; err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"message": "url not found"})
	}

	var req Room
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"message": "invalid request"})
	}

	req.RoomType = strings.TrimSpace(req.RoomType)
	if !validRoomTypes[req.RoomType] {
		return c.JSON(http.StatusBadRequest, map[string]string{"message": "room type is not valid"})
	}
	if req.Capacity <= 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"message": "capacity must be larger more than 0"})
	}

	room.Name = req.Name
	room.PricePerHour = req.PricePerHour
	room.ImageURL = req.ImageURL
	room.Capacity = req.Capacity
	room.RoomType = req.RoomType

	if err := db.Save(&room).Error; err != nil {
		log.Printf("[error] update room: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"message": "internal server error"})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "update room success"})
}

// Delete Room - DELETE /rooms/:id
func deleteRoom(c echo.Context) error {
	db := c.Get("db").(*gorm.DB)
	id := c.Param("id")

	var room Room
	if err := db.First(&room, id).Error; err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"message": "url not found"})
	}

	var count int64
	if err := db.Table("reservation_details").Where("room_id = ?", id).Count(&count).Error; err != nil {
		log.Printf("[error] check reservation: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"message": "internal server error"})
	}
	if count > 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"message": "cannot delete rooms. room has reservation"})
	}

	if err := db.Delete(&room).Error; err != nil {
		log.Printf("[error] delete room: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"message": "internal server error"})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "delete room success"})
}

// Get Snacks - GET /snacks
func getSnacks(c echo.Context) error {
	db := c.Get("db").(*gorm.DB)
	var snacks []Snack

	if err := db.Find(&snacks).Error; err != nil {
		log.Printf("[error] get snacks: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"message": "internal server error"})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "get snacks success",
		"data":    snacks,
	})
}

// Get Reservation Details - GET /reservation-details
func getReservationDetails(c echo.Context) error {
	db := c.Get("db").(*gorm.DB)
	var details []ReservationDetail

	if err := db.Find(&details).Error; err != nil {
		log.Printf("[error] get reservation_details: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"message": "internal server error"})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "get reservation details success",
		"data":    details,
	})
}

// Seed data snack awal (hanya sekali)
func seedSnacks(db *gorm.DB) {
	var count int64
	db.Model(&Snack{}).Count(&count)
	if count > 0 {
		return
	}
	snacks := []Snack{
		{Name: "Snack A", Category: "food", Unit: "box", Price: 10000},
		{Name: "Snack B", Category: "drink", Unit: "person", Price: 5000},
	}
	for _, snack := range snacks {
		db.Create(&snack)
	}
}

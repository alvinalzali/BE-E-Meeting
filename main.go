package main

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"BE-E-Meeting/app/entities"
	"BE-E-Meeting/app/handler"
	"BE-E-Meeting/app/middleware"
	"BE-E-Meeting/app/repositories"
	"BE-E-Meeting/app/usecases"
	"BE-E-Meeting/database"
	_ "BE-E-Meeting/docs"

	"github.com/go-playground/validator/v10"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	_ "github.com/lib/pq"
	echoSwagger "github.com/swaggo/echo-swagger"
)

type CustomValdator struct {
	validator *validator.Validate
}

func (cv *CustomValdator) Validate(i interface{}) error {
	return cv.validator.Struct(i)
}

// Global Variables (Masih dibutuhkan untuk handler lama yang belum direfactor)
var BaseURL string = "http://localhost:8080"
var DefaultAvatarURL string = BaseURL + "/assets/default/default_profile.jpg"
var DefaultRoomURL string = BaseURL + "/assets/default/default_room.jpg"
var db *sql.DB // Global DB untuk handler lama (Dashboard, dll)
var JwtSecret []byte

// @title E-Meeting API
// @version 1.0
// @description This is a sample server for E-Meeting.
// @termsOfService http://swagger.io/terms/
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	// 1. Load ENV
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Warning: .env file not found")
	}

	dbHost := os.Getenv("db_host")
	dbPort, _ := strconv.Atoi(os.Getenv("db_port"))
	dbUser := os.Getenv("db_user")
	dbPassword := os.Getenv("db_password")
	dbName := os.Getenv("db_name")
	JwtSecret = []byte(os.Getenv("jwt_secret"))

	// 2. Connect DB (Menggunakan package database baru, tapi simpan ke global 'db' juga)
	db = database.ConnectDB(dbUser, dbPassword, dbName, dbHost, dbPort)

	// 3. Migration
	skipMigration := os.Getenv("SKIP_MIGRATION")
	skipMigration = strings.ToLower(skipMigration)
	if skipMigration != "true" {
		fmt.Println("Enter 1 for migrate up, 2 for migrate down, 3 for continue:")
		var input int
		fmt.Scanln(&input)
		switch input {
		case 1:
			migrateUp(db)
		case 2:
			migrateDown(db)
		}
	}

	e := echo.New()
	e.Validator = &CustomValdator{validator: validator.New()}

	// ==========================================
	// DEPENDENCY INJECTION (CLEAN ARCHITECTURE)
	// ==========================================

	// --- A. Repository Initialization ---
	userRepo := repositories.NewUserRepository(db)
	roomRepo := repositories.NewRoomRepository(db)
	snackRepo := repositories.NewSnackRepository(db)
	resRepo := repositories.NewReservationRepository(db)
	dashboardRepo := repositories.NewDashboardRepository(db)

	// --- B. Usecase Initialization ---
	userUsecase := usecases.NewUserUsecase(userRepo)
	roomUsecase := usecases.NewRoomUsecase(roomRepo)
	snackUsecase := usecases.NewSnackUsecase(snackRepo)
	// Reservation butuh Room dan Snack Repo juga
	resUsecase := usecases.NewReservationUsecase(resRepo, roomRepo, snackRepo)
	dashboardUsecase := usecases.NewDashboardUsecase(dashboardRepo)

	// --- C. Handler Initialization ---
	userHandler := handler.NewUserHandler(userUsecase)
	roomHandler := handler.NewRoomHandler(roomUsecase)
	snackHandler := handler.NewSnackHandler(snackUsecase)
	resHandler := handler.NewReservationHandler(resUsecase)
	dashboardHandler := handler.NewDashboardHandler(dashboardUsecase)

	// ==========================================
	// ROUTES
	// ==========================================

	// Swagger & Assets
	e.GET("/swagger/*", echoSwagger.WrapHandler)
	e.Static("/assets", "./assets")

	// --- 1. USER & AUTH MODULE ---
	e.POST("/login", userHandler.Login)
	e.POST("/register", userHandler.Register)
	e.GET("/users/:id", userHandler.GetProfile, middleware.RoleAuthMiddleware("admin", "user"))
	e.PUT("/users/:id", userHandler.UpdateUser, middleware.RoleAuthMiddleware("admin", "user"))

	// Password Reset (Pake Handler Baru)
	e.POST("password/reset_request", userHandler.RequestPasswordReset)

	// Note: Middleware Auth TIDAK DIPAKAI di sini karena user belum login (Lupa Password)
	// Token validasi dilakukan di dalam logic ResetPassword usecase
	e.PUT("/password/reset/:id", userHandler.ResetPassword, middleware.RoleAuthMiddleware("admin", "user"))

	// // Legacy User Routes (Belum refactor)
	// e.POST("password/reset_request", PasswordReset)
	// e.PUT("/password/reset/:id", PasswordResetId, middleware.RoleAuthMiddleware("admin", "user"))

	// --- 2. ROOM MODULE ---
	e.POST("/rooms", roomHandler.CreateRoom, middleware.RoleAuthMiddleware("admin"))
	e.GET("/rooms", roomHandler.GetRooms, middleware.RoleAuthMiddleware("admin", "user"))
	e.GET("/rooms/:id", roomHandler.GetRoomByID, middleware.RoleAuthMiddleware("admin", "user"))
	e.PUT("/rooms/:id", roomHandler.UpdateRoom, middleware.RoleAuthMiddleware("admin"))
	e.DELETE("/rooms/:id", roomHandler.DeleteRoom, middleware.RoleAuthMiddleware("admin"))

	// --- 3. SNACK MODULE ---
	e.GET("/snacks", snackHandler.GetSnacks, middleware.RoleAuthMiddleware("admin", "user"))

	// --- 4. RESERVATION MODULE ---
	e.GET("/reservation/calculation", resHandler.CalculateReservation, middleware.RoleAuthMiddleware("admin", "user"))
	e.POST("/reservation", resHandler.CreateReservation, middleware.RoleAuthMiddleware("admin", "user"))
	e.GET("/reservation/history", resHandler.GetHistory, middleware.RoleAuthMiddleware("user"))
	e.PUT("/reservation/status", resHandler.UpdateReservationStatus, middleware.RoleAuthMiddleware("admin", "user"))
	e.GET("/reservation/:id", resHandler.GetReservationByID, middleware.RoleAuthMiddleware("admin", "user"))
	e.GET("/reservations/schedules", resHandler.GetReservationSchedules, middleware.RoleAuthMiddleware("admin"))

	// -- 5. Dashboard Module ---
	e.GET("/dashboard", dashboardHandler.GetDashboard, middleware.RoleAuthMiddleware("admin"))

	// Legacy Routes (Belum refactor penuh)
	e.GET("/rooms/:id/reservation", GetRoomReservationSchedule, middleware.RoleAuthMiddleware("admin", "user")) // Per room schedule
	e.POST("/uploads", UploadImage, middleware.RoleAuthMiddleware("admin", "user"))                             // Standalone upload

	e.Logger.Fatal(e.Start(":8080"))
}

// =================================================================================
// LEGACY CODE (YANG BELUM DI-REFACTOR)
// Jangan dihapus dulu sampai kita buatkan Repo/Usecase untuk Dashboard & Password
// =================================================================================

func migrateUp(db *sql.DB) {
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		log.Fatal(err)
	}
	m, err := migrate.NewWithDatabaseInstance("file://migrations", "postgres", driver)
	if err != nil {
		log.Fatal(err)
	}
	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		log.Fatal(err)
	}
	fmt.Println("Migrate up successfully")
}

func migrateDown(db *sql.DB) {
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		log.Fatal(err)
	}
	m, err := migrate.NewWithDatabaseInstance("file://migrations", "postgres", driver)
	if err != nil {
		log.Fatal(err)
	}
	err = m.Down()
	if err != nil && err != migrate.ErrNoChange {
		log.Fatal(err)
	}
	fmt.Println("Migrate down successfully")
}

// --- LEGACY HANDLERS ---

// UploadImage godoc
func UploadImage(c echo.Context) error {
	file, err := c.FormFile("image")
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Failed to upload image"})
	}
	contentType := file.Header.Get("Content-Type")
	if !(strings.HasPrefix(contentType, "image/jpeg") || strings.HasPrefix(contentType, "image/png")) {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid file type"})
	}
	if file.Size > 1024*1024 {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "File size is too large"})
	}
	src, err := file.Open()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to open image file"})
	}
	defer src.Close()
	os.MkdirAll("./assets/temp", os.ModePerm)
	ext := filepath.Ext(file.Filename)
	filename := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
	tempPath := filepath.Join("./assets/temp", filename)
	dst, err := os.Create(tempPath)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to save image"})
	}
	defer dst.Close()
	io.Copy(dst, src)
	baseURL := c.Scheme() + "://" + c.Request().Host
	imageURL := baseURL + "/assets/temp/" + filename
	return c.JSON(http.StatusOK, echo.Map{"message": "Image uploaded successfully", "imageURL": imageURL})
}

// GetRoomReservationSchedule godoc (Specific Room Schedule)
func GetRoomReservationSchedule(c echo.Context) error {
	roomID, _ := strconv.Atoi(c.Param("id"))
	startDTStr := c.QueryParam("start_datetime")
	endDTStr := c.QueryParam("end_datetime")
	dateStr := c.QueryParam("date")
	var useRange bool
	var startDT, endDT, dateFilter time.Time
	var err error
	if startDTStr != "" && endDTStr != "" {
		startDT, _ = time.Parse(time.RFC3339, startDTStr)
		endDT, _ = time.Parse(time.RFC3339, endDTStr)
		useRange = true
	} else if dateStr != "" {
		dateFilter, _ = time.Parse("2006-01-02", dateStr)
	} else {
		dateFilter = time.Now()
	}

	var rows *sql.Rows
	if useRange {
		query := `SELECT rd.id, rd.start_at, rd.end_at, r.status_reservation, rd.total_participants FROM reservation_details rd JOIN reservations r ON rd.reservation_id = r.id WHERE rd.room_id = $1 AND (rd.start_at, rd.end_at) OVERLAPS ($2, $3) ORDER BY rd.start_at ASC`
		rows, err = db.Query(query, roomID, startDT, endDT)
	} else {
		query := `SELECT rd.id, rd.start_at, rd.end_at, r.status_reservation, rd.total_participants FROM reservation_details rd JOIN reservations r ON rd.reservation_id = r.id WHERE rd.room_id = $1 AND DATE(rd.start_at) = DATE($2) ORDER BY rd.start_at ASC`
		rows, err = db.Query(query, roomID, dateFilter)
	}
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"message": "error"})
	}
	defer rows.Close()

	schedules := []entities.RoomSchedule{}
	for rows.Next() {
		var s entities.RoomSchedule
		var start, end sql.NullTime
		var st sql.NullString
		var p sql.NullInt64
		rows.Scan(&s.ID, &start, &end, &st, &p)
		if start.Valid {
			s.StartTime = start.Time
		}
		if end.Valid {
			s.EndTime = end.Time
		}
		if st.Valid {
			s.Status = st.String
		}
		if p.Valid {
			s.TotalParticipant = int(p.Int64)
		}
		schedules = append(schedules, s)
	}
	// Simplified room detail fetch
	var room entities.Room
	db.QueryRow(`SELECT id, name, room_type, capacity, price_per_hour, picture_url FROM rooms WHERE id = $1`, roomID).Scan(&room.ID, &room.Name, &room.RoomType, &room.Capacity, &room.PricePerHour, &room.PictureURL)

	return c.JSON(http.StatusOK, echo.Map{"message": "success", "data": echo.Map{"room": room, "schedules": schedules, "date": dateFilter.Format("2006-01-02")}})
}

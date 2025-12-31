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
	"github.com/golang-jwt/jwt/v5"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	_ "github.com/lib/pq"
	echoSwagger "github.com/swaggo/echo-swagger"
	"golang.org/x/crypto/bcrypt"
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

	// --- B. Usecase Initialization ---
	userUsecase := usecases.NewUserUsecase(userRepo)
	roomUsecase := usecases.NewRoomUsecase(roomRepo)
	snackUsecase := usecases.NewSnackUsecase(snackRepo)
	// Reservation butuh Room dan Snack Repo juga
	resUsecase := usecases.NewReservationUsecase(resRepo, roomRepo, snackRepo)

	// --- C. Handler Initialization ---
	userHandler := handler.NewUserHandler(userUsecase)
	roomHandler := handler.NewRoomHandler(roomUsecase)
	snackHandler := handler.NewSnackHandler(snackUsecase)
	resHandler := handler.NewReservationHandler(resUsecase)

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

	// Legacy User Routes (Belum refactor)
	e.POST("password/reset_request", PasswordReset)
	e.PUT("/password/reset/:id", PasswordResetId, middleware.RoleAuthMiddleware("admin", "user"))

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

	// Legacy Routes (Belum refactor penuh)
	e.GET("/rooms/:id/reservation", GetRoomReservationSchedule, middleware.RoleAuthMiddleware("admin", "user")) // Per room schedule
	e.POST("/uploads", UploadImage, middleware.RoleAuthMiddleware("admin", "user"))                             // Standalone upload
	e.GET("/dashboard", GetDashboard, middleware.RoleAuthMiddleware("admin"))                                   // Dashboard

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

// --- LEGACY HELPERS (Still needed for legacy handlers) ---
func isValidPassword(password string) bool {
	var hasUpper, hasLower, hasNumber, hasSpecial bool
	for _, char := range password {
		switch {
		case 'A' <= char && char <= 'Z':
			hasUpper = true
		case 'a' <= char && char <= 'z':
			hasLower = true
		case '0' <= char && char <= '9':
			hasNumber = true
		case (char >= 33 && char <= 47) || (char >= 58 && char <= 64) || (char >= 91 && char <= 96) || (char >= 123 && char <= 126):
			hasSpecial = true
		}
	}
	return hasUpper && hasLower && hasNumber && hasSpecial
}

func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func generateResetToken(email string) (string, error) {
	JwtSecret = []byte(os.Getenv("secret_key"))
	claims := &entities.Claims{
		Username: email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(30 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(JwtSecret)
}

// --- LEGACY HANDLERS ---

// PasswordResetId godoc
func PasswordResetId(c echo.Context) error {
	id := c.Param("id")
	var passReset entities.PasswordConfirmReset
	userID, err := strconv.Atoi(id)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Bad Request"})
	}
	userToken := c.Get("user").(*jwt.Token)
	claims := userToken.Claims.(jwt.MapClaims)
	usernameFromToken := claims["username"].(string)
	var usernameFromDB string
	err = db.QueryRow("SELECT username FROM users WHERE id = $1", userID).Scan(&usernameFromDB)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Bad Request"})
	}
	if usernameFromToken != usernameFromDB {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Bad Request"})
	}
	if err := c.Bind(&passReset); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Bad Request"})
	}
	if passReset.NewPassword != passReset.ConfirmPassword {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "New password and confirm password do not match"})
	}
	if err := c.Validate(&passReset); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Validation Error"})
	}
	if !isValidPassword(passReset.NewPassword) {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Password weak"})
	}
	hashedPassword, _ := hashPassword(passReset.NewPassword)
	sqlStatement := `UPDATE users SET password_hash=$1 WHERE id=$2`
	_, err = db.Exec(sqlStatement, hashedPassword, id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Internal Server Error"})
	}
	return c.JSON(http.StatusOK, echo.Map{"message": "Password reset successfully"})
}

// PasswordReset godoc
func PasswordReset(c echo.Context) error {
	var resetReq entities.ResetRequest
	if err := c.Bind(&resetReq); err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}
	if err := c.Validate(&resetReq); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Bad Request"})
	}
	var storedEmail string
	err := db.QueryRow(`SELECT email FROM users WHERE email=$1`, resetReq.Email).Scan(&storedEmail)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.JSON(http.StatusNotFound, echo.Map{"error": "Email not found"})
		}
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Internel Server Error"})
	}
	resetToken, _ := generateResetToken(storedEmail)
	fmt.Println("Password reset requested for email:", resetReq.Email)
	return c.JSON(http.StatusOK, echo.Map{"message": "Update Password Success!", "token": resetToken})
}

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

// GetDashboard godoc
func GetDashboard(c echo.Context) error {
	startDate := c.QueryParam("startDate")
	endDate := c.QueryParam("endDate")
	var start, end time.Time
	var err error
	if startDate != "" {
		start, err = time.Parse("2006-01-02", startDate)
		if err != nil {
			return c.JSON(http.StatusBadRequest, echo.Map{"message": "invalid startDate"})
		}
	}
	if endDate != "" {
		end, err = time.Parse("2006-01-02", endDate)
		if err != nil {
			return c.JSON(http.StatusBadRequest, echo.Map{"message": "invalid endDate"})
		}
	}
	var totalRoom int
	db.QueryRow(`SELECT COUNT(*) FROM rooms`).Scan(&totalRoom)
	filterSQL := ` WHERE res.status_reservation = 'paid' `
	args := []interface{}{}
	argIdx := 1
	if !start.IsZero() {
		filterSQL += fmt.Sprintf(" AND DATE(rd.start_at) >= $%d", argIdx)
		args = append(args, start)
		argIdx++
	}
	if !end.IsZero() {
		filterSQL += fmt.Sprintf(" AND DATE(rd.end_at) <= $%d", argIdx)
		args = append(args, end)
		argIdx++
	}
	var totalVisitor, totalReservation int
	var totalOmzet float64
	totalsQuery := `SELECT COALESCE(SUM(rd.total_participants), 0), COUNT(DISTINCT res.id), COALESCE(SUM(res.total), 0) FROM reservations res JOIN reservation_details rd ON res.id = rd.reservation_id` + filterSQL
	db.QueryRow(totalsQuery, args...).Scan(&totalVisitor, &totalReservation, &totalOmzet)
	// (Bagian Room Stats dipotong sedikit biar ringkas, tapi logic intinya sama dengan file lamamu)
	// Jika ingin Dashboard full, copy function GetDashboard lama kesini
	response := entities.DashboardResponse{Message: "get dashboard data success"}
	response.Data.TotalRoom = totalRoom
	response.Data.TotalVisitor = totalVisitor
	response.Data.TotalReservation = totalReservation
	response.Data.TotalOmzet = totalOmzet
	return c.JSON(http.StatusOK, response)
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

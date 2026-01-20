package main

import (
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"strings"

	"BE-E-Meeting/app/config"
	"BE-E-Meeting/app/handler"
	"BE-E-Meeting/app/middleware"
	"BE-E-Meeting/app/repositories"
	"BE-E-Meeting/app/usecases"
	"BE-E-Meeting/database"
	_ "BE-E-Meeting/docs"

	"github.com/go-playground/validator/v10"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	_ "github.com/lib/pq"
	echoSwagger "github.com/swaggo/echo-swagger"
)

// Custom Validator Wrapper
type CustomValdator struct {
	validator *validator.Validate
}

func (cv *CustomValdator) Validate(i interface{}) error {
	return cv.validator.Struct(i)
}

// Global variable yang mungkin masih dipanggil helper (bisa dipindah ke config nanti)
var BaseURL string = "http://localhost:8080"
var DefaultAvatarURL string = BaseURL + "/assets/default/default_profile.jpg"
var DefaultRoomURL string = BaseURL + "/assets/default/default_room.jpg"
var db *sql.DB

// @title E-Meeting API
// @version 1.0
// @description This is a sample server for E-Meeting.
// @termsOfService http://swagger.io/terms/
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	// 1. Load ENV
	godotenv.Load()

	dbHost := os.Getenv("db_host")
	dbPort, _ := strconv.Atoi(os.Getenv("db_port"))
	dbUser := os.Getenv("db_user")
	dbPassword := os.Getenv("db_password")
	dbName := os.Getenv("db_name")

	// 2. Database Connection
	db = database.ConnectDB(dbUser, dbPassword, dbName, dbHost, dbPort)

	// 3. Migration Check
	checkMigration := os.Getenv("SKIP_MIGRATION")
	checkMigration = strings.ToLower(checkMigration)
	if checkMigration != "true" {
		fmt.Println("Enter 1 for migrate up, 2 for migrate down, 3 for continue:")
		var input int
		fmt.Scanln(&input)
		switch input {
		case 1:
			database.MigrateUp(db)
		case 2:
			database.MigrateDown(db)
		}
	}

	e := echo.New()
	e.Validator = &CustomValdator{validator: validator.New()}

	// ==========================================
	// DEPENDENCY INJECTION (WIRING)
	// ==========================================

	// Google Oauth
	googleConfig := config.LoadGoogleConfig()

	// Repositories
	userRepo := repositories.NewUserRepository(db)
	roomRepo := repositories.NewRoomRepository(db)
	snackRepo := repositories.NewSnackRepository(db)
	resRepo := repositories.NewReservationRepository(db)
	dashboardRepo := repositories.NewDashboardRepository(db)

	// Usecases
	userUsecase := usecases.NewUserUsecase(userRepo)
	roomUsecase := usecases.NewRoomUsecase(roomRepo)
	snackUsecase := usecases.NewSnackUsecase(snackRepo)
	resUsecase := usecases.NewReservationUsecase(resRepo, roomRepo, snackRepo)
	dashboardUsecase := usecases.NewDashboardUsecase(dashboardRepo)
	authUsecase := usecases.NewAuthUsecase(userRepo, googleConfig)

	// Handlers
	userHandler := handler.NewUserHandler(userUsecase)
	roomHandler := handler.NewRoomHandler(roomUsecase)
	snackHandler := handler.NewSnackHandler(snackUsecase)
	resHandler := handler.NewReservationHandler(resUsecase)
	dashboardHandler := handler.NewDashboardHandler(dashboardUsecase)
	fileHandler := handler.NewFileHandler()
	authHandler := handler.NewAuthHandler(authUsecase)

	// ==========================================
	// ROUTES
	// ==========================================

	// Swagger & Assets
	e.GET("/swagger/*", echoSwagger.WrapHandler)
	e.Static("/assets", "./assets")

	// --- AUTH & USER ---
	e.POST("/login", userHandler.Login)
	e.POST("/register", userHandler.Register)
	e.POST("password/reset_request", userHandler.RequestPasswordReset)
	e.PUT("/password/reset/:id", userHandler.ResetPassword)
	e.GET("/users/:id", userHandler.GetProfile, middleware.RoleAuthMiddleware("admin", "user"))
	e.PUT("/users/:id", userHandler.UpdateUser, middleware.RoleAuthMiddleware("admin", "user"))

	// google auth
	e.GET("/auth/google/login", authHandler.GoogleLogin)
	e.GET("/auth/google/callback", authHandler.GoogleCallback)

	// --- ROOM ---
	e.POST("/rooms", roomHandler.CreateRoom, middleware.RoleAuthMiddleware("admin"))
	e.GET("/rooms", roomHandler.GetRooms, middleware.RoleAuthMiddleware("admin", "user"))
	e.GET("/rooms/:id", roomHandler.GetRoomByID, middleware.RoleAuthMiddleware("admin", "user"))
	e.PUT("/rooms/:id", roomHandler.UpdateRoom, middleware.RoleAuthMiddleware("admin"))
	e.DELETE("/rooms/:id", roomHandler.DeleteRoom, middleware.RoleAuthMiddleware("admin"))
	// Endpoint Legacy yang sudah dipindah ke Reservation Handler:
	e.GET("/rooms/:id/reservation", resHandler.GetRoomReservationSchedule, middleware.RoleAuthMiddleware("admin", "user"))

	// --- SNACK ---
	e.GET("/snacks", snackHandler.GetSnacks, middleware.RoleAuthMiddleware("admin", "user"))

	// --- RESERVATION ---
	e.GET("/reservation/calculation", resHandler.CalculateReservation, middleware.RoleAuthMiddleware("admin", "user"))
	e.POST("/reservation", resHandler.CreateReservation, middleware.RoleAuthMiddleware("admin", "user"))
	e.GET("/reservation/history", resHandler.GetHistory, middleware.RoleAuthMiddleware("user"))
	e.PUT("/reservation/status", resHandler.UpdateReservationStatus, middleware.RoleAuthMiddleware("admin", "user"))
	e.GET("/reservation/:id", resHandler.GetReservationByID, middleware.RoleAuthMiddleware("admin", "user"))
	e.GET("/reservations/schedules", resHandler.GetReservationSchedules, middleware.RoleAuthMiddleware("admin"))

	// --- DASHBOARD ---
	e.GET("/dashboard", dashboardHandler.GetDashboard, middleware.RoleAuthMiddleware("admin"))

	// --- UTILS (FILE UPLOAD) ---
	e.POST("/uploads", fileHandler.UploadImage, middleware.RoleAuthMiddleware("admin", "user"))

	// Start Server
	e.Logger.Fatal(e.Start(":8080"))
}

package main

import (
	"BE-E-MEETING/app/handlers"
	"BE-E-MEETING/app/repositories"
	"BE-E-MEETING/app/usecases"
	"BE-E-MEETING/config"
	"BE-E-MEETING/middleware"
	"BE-E-MEETING/pkg/database"
	"BE-E-MEETING/server"
	"fmt"
	"github.com/labstack/echo/v4"
	"log"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize database
	db, err := database.NewPostgresDatabase(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Initialize server
	echoServer := server.NewEchoServer(cfg)
	e := echoServer.GetEcho()

	// Initialize repositories
	userRepo := repositories.NewUserRepository(db.GetDB())
	roomRepo := repositories.NewRoomRepository(db.GetDB())
	snackRepo := repositories.NewSnackRepository(db.GetDB())
	reservationRepo := repositories.NewReservationRepository(db.GetDB())
	dashboardRepo := repositories.NewDashboardRepository(db.GetDB())
	imageRepo := repositories.NewImageRepository()

	// Initialize use cases
	userUsecase := usecases.NewUserUsecase(userRepo)
	roomUsecase := usecases.NewRoomUsecase(roomRepo)
	snackUsecase := usecases.NewSnackUsecase(snackRepo)
	reservationUsecase := usecases.NewReservationUsecase(reservationRepo, db.GetDB())
	dashboardUsecase := usecases.NewDashboardUsecase(dashboardRepo)
	imageUsecase := usecases.NewImageUsecase(imageRepo)

	// Initialize handlers
	userHandler := handlers.NewUserHandler(userUsecase)
	roomHandler := handlers.NewRoomHandler(roomUsecase)
	snackHandler := handlers.NewSnackHandler(snackUsecase)
	reservationHandler := handlers.NewReservationHandler(reservationUsecase)
	dashboardHandler := handlers.NewDashboardHandler(dashboardUsecase)
	imageHandler := handlers.NewImageHandler(imageUsecase)

	// Register routes
	userHandler.RegisterRoutes(e)

	authGroup := e.Group("")
	authGroup.Use(middleware.JWTAuth(cfg))
	roomHandler.RegisterRoutes(authGroup)
	snackHandler.RegisterRoutes(authGroup)
	reservationHandler.RegisterRoutes(authGroup)
	dashboardHandler.RegisterRoutes(authGroup)
	imageHandler.RegisterRoutes(authGroup)

	// Start server
	fmt.Println("Server starting on :8080")
	if err := echoServer.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

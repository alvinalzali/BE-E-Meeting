package app

import (
	"BE-E-MEETING/app/handlers"
	"github.com/labstack/echo/v4"
)

func RegisterRoutes(
	e *echo.Echo,
	userHandler *handlers.UserHandler,
	roomHandler *handlers.RoomHandler,
	snackHandler *handlers.SnackHandler,
	reservationHandler *handlers.ReservationHandler,
	dashboardHandler *handlers.DashboardHandler,
	imageHandler *handlers.ImageHandler,
	authMiddleware echo.MiddlewareFunc,
) {
	// User routes
	e.POST("/login", userHandler.Login)
	e.POST("/register", userHandler.RegisterUser)
	e.POST("password/reset_request", userHandler.PasswordReset)
	e.PUT("/password/reset/:id", userHandler.PasswordResetId)

	userGroup := e.Group("/users")
	userGroup.Use(authMiddleware)
	userGroup.GET("/:id", userHandler.GetUserByID)
	userGroup.PUT("/:id", userHandler.UpdateUserByID)

	// Room routes
	authGroup := e.Group("")
	authGroup.Use(authMiddleware)
	authGroup.POST("/rooms", roomHandler.CreateRoom)
	authGroup.GET("/rooms", roomHandler.GetRooms)
	authGroup.GET("/rooms/:id", roomHandler.GetRoomByID)
	authGroup.PUT("/rooms/:id", roomHandler.UpdateRoom)
	authGroup.DELETE("/rooms/:id", roomHandler.DeleteRoom)
	authGroup.GET("/rooms/:id/reservation", roomHandler.GetRoomReservationSchedule)

	// Snack routes
	authGroup.GET("/snacks", snackHandler.GetSnacks)

	// Reservation routes
	authGroup.GET("/reservation/calculation", reservationHandler.CalculateReservation)
	authGroup.POST("/reservation", reservationHandler.CreateReservation)
	authGroup.GET("/reservation/history", reservationHandler.GetReservationHistory)
	authGroup.POST("/reservation/status", reservationHandler.UpdateReservationStatus)
	authGroup.GET("/reservation/:id", reservationHandler.GetReservationByID)
	authGroup.GET("/reservations/schedules", reservationHandler.GetReservationSchedules)

	// Dashboard routes
	authGroup.GET("/dashboard", dashboardHandler.GetDashboard)

	// Image routes
	authGroup.POST("/uploads", imageHandler.UploadImage)
}

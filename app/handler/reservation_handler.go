package handler

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"BE-E-Meeting/app/entities"
	"BE-E-Meeting/app/usecases"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

type ReservationHandler struct {
	usecase usecases.ReservationUsecase
}

func NewReservationHandler(usecase usecases.ReservationUsecase) *ReservationHandler {
	return &ReservationHandler{usecase: usecase}
}

// CalculateReservation godoc
// @Summary Calculate reservation
// @Description Calculate reservation price details
// @Tags Reservation
// @Produce json
// @Param room_id query string true "Room ID"
// @Param snack_id query string true "Snack ID (0 if none)"
// @Param startTime query string true "Start Time (RFC3339)"
// @Param endTime query string true "End Time (RFC3339)"
// @Param participant query string true "Participant Count"
// @Security BearerAuth
// @Router /reservation/calculation [get]
func (h *ReservationHandler) CalculateReservation(c echo.Context) error {
	roomID, _ := strconv.Atoi(c.QueryParam("room_id"))
	startTimeStr := c.QueryParam("startTime")
	endTimeStr := c.QueryParam("endTime")

	startTime, _ := time.Parse(time.RFC3339, startTimeStr)
	endTime, _ := time.Parse(time.RFC3339, endTimeStr)

	participant, _ := strconv.Atoi(c.QueryParam("participant"))
	snackID, _ := strconv.Atoi(c.QueryParam("snack_id"))
	addSnack := false
	if snackID > 0 {
		addSnack = true
	}

	// Menggunakan struct yang sudah diperbaiki (RoomReservationRequest)
	req := entities.ReservationRequestBody{
		Rooms: []entities.RoomReservationRequest{{
			ID: roomID, StartTime: startTime, EndTime: endTime,
			Participant: participant, AddSnack: addSnack, SnackID: snackID,
		}},
	}

	res, err := h.usecase.Calculate(req)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": err.Error()})
	}
	return c.JSON(http.StatusOK, echo.Map{"message": "success", "data": res})
}

// CreateReservation godoc
// @Summary Create a new reservation
// @Description Create a new reservation transaction
// @Tags Reservation
// @Accept json
// @Produce json
// @Param request body entities.ReservationRequestBody true "Reservation request body"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /reservation [post]
func (h *ReservationHandler) CreateReservation(c echo.Context) error {
	var req entities.ReservationRequestBody
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "invalid format"})
	}

	// Ambil User ID dari Token (Supaya tidak dimanipulasi di body)
	userToken := c.Get("user").(*jwt.Token)
	claims := userToken.Claims.(jwt.MapClaims)
	username := claims["username"].(string)

	userID, err := h.usecase.GetUserIDByUsername(username)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, echo.Map{"message": "user not found"})
	}
	req.UserID = userID

	err = h.usecase.Create(req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"message": err.Error()})
	}
	return c.JSON(http.StatusOK, echo.Map{"message": "reservation created successfully"})
}

// GetHistory godoc
// @Summary Get meeting reservation history
// @Description Retrieve meeting reservation history filtered by filters
// @Tags Reservation
// @Produce json
// @Param startDate query string false "Start Date (YYYY-MM-DD)"
// @Param endDate query string false "End Date (YYYY-MM-DD)"
// @Param type query string false "Room Type (small/medium/large)"
// @Param status query string false "Status (booked/paid/cancel)"
// @Param page query int false "Page number"
// @Param pageSize query int false "Page size"
// @Success 200 {object} entities.ReservationHistoryResponse
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /reservation/history [get]
func (h *ReservationHandler) GetHistory(c echo.Context) error {
	userToken := c.Get("user").(*jwt.Token)
	claims := userToken.Claims.(jwt.MapClaims)
	username := claims["username"].(string)

	userID, err := h.usecase.GetUserIDByUsername(username)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, echo.Map{"message": "user not found"})
	}

	// Simple Role Check
	userRole := "user"
	if roleStr, ok := claims["role"].(string); ok {
		if strings.Contains(roleStr, "admin") {
			userRole = "admin"
		}
	} else if roles, ok := claims["role"].([]interface{}); ok {
		for _, r := range roles {
			if r.(string) == "admin" {
				userRole = "admin"
				break
			}
		}
	}

	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(c.QueryParam("pageSize"))
	if pageSize < 1 {
		pageSize = 10
	}

	startDate := c.QueryParam("startDate")
	endDate := c.QueryParam("endDate")
	roomType := c.QueryParam("type")
	status := c.QueryParam("status")

	res, err := h.usecase.GetHistory(userID, userRole, startDate, endDate, roomType, status, page, pageSize)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"message": err.Error()})
	}

	return c.JSON(http.StatusOK, res)
}

// GetReservationByID godoc
// @Summary Detail reservation by ID
// @Description Get full reservation detail (master + reservation details) by reservation ID
// @Tags Reservation
// @Produce json
// @Param id path int true "Reservation ID"
// @Success 200 {object} entities.ReservationByIDResponse
// @Failure 404 {object} map[string]string
// @Security BearerAuth
// @Router /reservation/{id} [get]
func (h *ReservationHandler) GetReservationByID(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "invalid id"})
	}

	res, err := h.usecase.GetByID(id)
	if err != nil {
		return c.JSON(http.StatusNotFound, echo.Map{"message": "reservation not found"})
	}
	return c.JSON(http.StatusOK, res)
}

// UpdateReservationStatus godoc
// @Summary Update reservation status
// @Description Update status of a reservation (booked/cancel/paid)
// @Tags Reservation
// @Accept json
// @Produce json
// @Param reservation body entities.UpdateReservationRequest true "Reservation details"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Security BearerAuth
// @Router /reservation/status [put]
func (h *ReservationHandler) UpdateReservationStatus(c echo.Context) error {
	var req entities.UpdateReservationRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "invalid format"})
	}

	// Ambil ID dari token
	userToken := c.Get("user").(*jwt.Token)
	claims := userToken.Claims.(jwt.MapClaims)
	username := claims["username"].(string)
	userID, _ := h.usecase.GetUserIDByUsername(username)

	err := h.usecase.UpdateStatus(req.ReservationID, userID, req.Status, "user") // Role bisa disesuaikan
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": err.Error()})
	}

	return c.JSON(http.StatusOK, echo.Map{"message": "update status success"})
}

// GetReservationSchedules godoc
// @Summary Get reservation schedules
// @Description Get all reservation schedules between date range with pagination
// @Tags Reservation
// @Accept json
// @Produce json
// @Param startDate query string true "Start date (YYYY-MM-DD)"
// @Param endDate query string true "End date (YYYY-MM-DD)"
// @Param page query int false "Page number (default: 1)"
// @Param pageSize query int false "Page size (default: 10)"
// @Success 200 {object} entities.ScheduleResponse
// @Security BearerAuth
// @Router /reservations/schedules [get]
func (h *ReservationHandler) GetReservationSchedules(c echo.Context) error {
	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(c.QueryParam("pageSize"))
	if pageSize < 1 {
		pageSize = 10
	}

	startDate := c.QueryParam("startDate")
	endDate := c.QueryParam("endDate")

	res, err := h.usecase.GetSchedules(startDate, endDate, page, pageSize)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"message": err.Error()})
	}
	return c.JSON(http.StatusOK, res)
}

// GetRoomReservationSchedule godoc
// @Summary Get room reservation schedule
// @Description Get all reservations for a specific room
// @Tags Room
// @Produce json
// @Param id path string true "Room ID"
// @Param date query string false "Date filter (YYYY-MM-DD)"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /rooms/{id}/reservation [get]
func (h *ReservationHandler) GetRoomReservationSchedule(c echo.Context) error {
	roomID, _ := strconv.Atoi(c.Param("id"))
	startDTStr := c.QueryParam("start_datetime")
	endDTStr := c.QueryParam("end_datetime")
	dateStr := c.QueryParam("date")

	var startDT, endDT time.Time
	var err error

	// Logic parsing tanggal (sama seperti di main.go lama)
	if startDTStr != "" && endDTStr != "" {
		startDT, _ = time.Parse(time.RFC3339, startDTStr)
		endDT, _ = time.Parse(time.RFC3339, endDTStr)
	} else if dateStr != "" {
		startDT, err = time.Parse("2006-01-02", dateStr)
		if err != nil {
			return c.JSON(http.StatusBadRequest, echo.Map{"message": "invalid date"})
		}
		endDT = startDT.Add(24 * time.Hour)
	} else {
		startDT = time.Now()
		endDT = startDT.Add(24 * time.Hour)
	}

	res, err := h.usecase.GetRoomSchedule(roomID, startDT, endDT)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"message": err.Error()})
	}

	return c.JSON(http.StatusOK, echo.Map{"message": "success", "data": res})
}

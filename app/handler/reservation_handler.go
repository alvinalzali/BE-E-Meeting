package handler

import (
	"net/http"
	"strconv"
	"time"

	"BE-E-Meeting/app/entities"
	"BE-E-Meeting/app/middleware"
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
// @Summary Calculate reservation price
// @Description Simulate price calculation before booking to get total cost
// @Tags Reservation
// @Produce json
// @Param room_id query int true "Room ID"
// @Param snack_id query int false "Snack ID (0 if none)"
// @Param startTime query string true "Start Time (RFC3339 format: 2025-10-20T09:00:00Z)"
// @Param endTime query string true "End Time (RFC3339 format: 2025-10-20T11:00:00Z)"
// @Param participant query int true "Participant Count"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
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

	// [PENTING] Menggunakan entities.RoomReservationRequest dari room.go
	req := entities.ReservationRequest{
		Rooms: []entities.RoomReservationRequest{{
			ID:          roomID,
			StartTime:   startTime,
			EndTime:     endTime,
			SnackID:     snackID,
			Participant: participant,
			AddSnack:    addSnack,
		}},
		TotalParticipants: participant,
	}

	res, err := h.usecase.Calculate(req)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": err.Error()})
	}
	return c.JSON(http.StatusOK, echo.Map{"message": "success", "data": res})
}

// CreateReservation godoc
// @Summary Create a new reservation
// @Description Create a new reservation transaction (Booking)
// @Tags Reservation
// @Accept json
// @Produce json
// @Param request body entities.ReservationRequest true "Reservation Data"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /reservation [post]
func (h *ReservationHandler) CreateReservation(c echo.Context) error {
	var req entities.ReservationRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "invalid format"})
	}

	// Ambil User ID dari Token
	userToken := c.Get("user").(*jwt.Token)
	claims := userToken.Claims.(jwt.MapClaims)

	if username, ok := claims["username"].(string); ok {
		userID, err := h.usecase.GetUserIDByUsername(username)
		if err != nil {
			return c.JSON(http.StatusUnauthorized, echo.Map{"message": "user not found"})
		}
		req.UserID = userID
	} else {
		req.UserID = middleware.ExtractTokenUserID(c)
	}

	err := h.usecase.Create(req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"message": err.Error()})
	}
	return c.JSON(http.StatusOK, echo.Map{"message": "reservation created successfully"})
}

// GetHistory godoc
// @Summary Get reservation history
// @Description Get reservation history with filters and pagination
// @Tags Reservation
// @Produce json
// @Param startDate query string false "Start Date (YYYY-MM-DD)"
// @Param endDate query string false "End Date (YYYY-MM-DD)"
// @Param type query string false "Room Type (small/medium/large)"
// @Param status query string false "Status (booked/paid/cancel)"
// @Param page query int false "Page number (default: 1)"
// @Param pageSize query int false "Page size (default: 10)"
// @Success 200 {object} entities.ReservationHistoryResponse
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /reservation/history [get]
func (h *ReservationHandler) GetHistory(c echo.Context) error {
	startDate := c.QueryParam("startDate")
	endDate := c.QueryParam("endDate")
	roomType := c.QueryParam("type")
	status := c.QueryParam("status")

	page, _ := strconv.Atoi(c.QueryParam("page"))
	pageSize, _ := strconv.Atoi(c.QueryParam("pageSize"))

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	userID := middleware.ExtractTokenUserID(c)

	response, err := h.usecase.GetHistory(userID, startDate, endDate, roomType, status, page, pageSize)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"message": err.Error()})
	}

	response.Page = page
	response.PageSize = pageSize
	if response.TotalData > 0 {
		response.TotalPage = (response.TotalData + pageSize - 1) / pageSize
	}

	return c.JSON(http.StatusOK, response)
}

// GetReservationByID godoc
// @Summary Get reservation detail
// @Description Get full detail of a reservation by ID
// @Tags Reservation
// @Produce json
// @Param id path int true "Reservation ID"
// @Success 200 {object} entities.ReservationDetailResponse
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
// @Description Update status (booked -> paid/cancel)
// @Tags Reservation
// @Accept json
// @Produce json
// @Param body body entities.UpdateReservationRequest true "Status Update"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Security BearerAuth
// @Router /reservation/status [put]
func (h *ReservationHandler) UpdateReservationStatus(c echo.Context) error {
	var req entities.UpdateReservationRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "invalid format"})
	}

	userID := middleware.ExtractTokenUserID(c)

	err := h.usecase.UpdateStatus(req.ReservationID, userID, req.Status, "user")
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": err.Error()})
	}

	return c.JSON(http.StatusOK, echo.Map{"message": "update status success"})
}

// GetReservationSchedules godoc
// @Summary Get all schedules
// @Description Get reservation schedules for all rooms (Admin Dashboard)
// @Tags Reservation
// @Produce json
// @Param startDate query string false "Start Date (YYYY-MM-DD)"
// @Param endDate query string false "End Date (YYYY-MM-DD)"
// @Param page query int false "Page number (default: 1)"
// @Param pageSize query int false "Page size (default: 10)"
// @Success 200 {object} entities.ScheduleResponse
// @Failure 500 {object} map[string]string
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
// @Summary Get specific room schedule
// @Description Get schedule list for a specific room
// @Tags Room
// @Produce json
// @Param id path int true "Room ID"
// @Param date query string false "Date Filter (YYYY-MM-DD)"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /rooms/{id}/reservation [get]
func (h *ReservationHandler) GetRoomReservationSchedule(c echo.Context) error {
	roomID, _ := strconv.Atoi(c.Param("id"))
	dateStr := c.QueryParam("date")

	var startDT, endDT time.Time
	var err error

	if dateStr != "" {
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

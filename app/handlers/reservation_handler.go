package handlers

import (
	"BE-E-MEETING/app/models"
	"BE-E-MEETING/app/usecases"
	"github.com/labstack/echo/v4"
	"net/http"
	"strconv"
)

type ReservationHandler struct {
	reservationUsecase usecases.ReservationUsecase
}

func NewReservationHandler(reservationUsecase usecases.ReservationUsecase) *ReservationHandler {
	return &ReservationHandler{reservationUsecase: reservationUsecase}
}

// CalculateReservation godoc
// @Summary Calculate reservation
// @Description Calculate reservation
// @Tags Reservation
// @Produce json
// @Param room_id query string true "Room ID"
// @Param snack_id query string true "Snack ID"
// @Param startTime query string true "Start Time"
// @Param endTime query string true "End Time"
// @Param participant query string true "Participant"
// @Param user_id query string true "User ID"
// @Param name query string true "Name"
// @Param phoneNumber query string true "Phone Number"
// @Param company query string true "Company"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /reservation/calculation [get]
func (h *ReservationHandler) CalculateReservation(c echo.Context) error {
	roomID, _ := strconv.Atoi(c.QueryParam("room_id"))
	snackID, _ := strconv.Atoi(c.QueryParam("snack_id"))
	startTimeStr := c.QueryParam("startTime")
	endTimeStr := c.QueryParam("endTime")
	participant, _ := strconv.Atoi(c.QueryParam("participant"))
	name := c.QueryParam("name")
	phoneNumber := c.QueryParam("phoneNumber")
	company := c.QueryParam("company")

	response, err := h.reservationUsecase.CalculateReservation(roomID, snackID, startTimeStr, endTimeStr, participant, name, phoneNumber, company)
	if err != nil {
		if e, ok := err.(*usecases.UseCaseError); ok {
			return c.JSON(e.Code, echo.Map{"message": e.Message})
		}
		return c.JSON(http.StatusInternalServerError, echo.Map{"message": "internal server error"})
	}
	return c.JSON(http.StatusOK, response)
}

// CreateReservation godoc
// @Summary Create a new reservation
// @Description Create a new reservation
// @Tags Reservation
// @Accept json
// @Produce json
// @Param request body models.ReservationRequestBody true "Reservation request body"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /reservation [post]
func (h *ReservationHandler) CreateReservation(c echo.Context) error {
	var req models.ReservationRequestBody
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "invalid request format"})
	}

	err := h.reservationUsecase.CreateReservation(req)
	if err != nil {
		if e, ok := err.(*usecases.UseCaseError); ok {
			return c.JSON(e.Code, echo.Map{"message": e.Message})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"message": "internal server error"})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "reservation created successfully",
	})
}

// GetReservationHistory godoc
// @Summary Get meeting reservation history
// @Description Retrieve meeting reservation history filtered by user_id, room_id, or date.
// @Tags Reservation
// @Param user_id query string false "User ID"
// @Param room_id query string false "Room ID"
// @Param date query string false "Date (YYYY-MM-DD)"
// @Produce json
// @Success 200 {object} map[string]interface{} "History retrieved successfully"
// @Failure 400 {object} map[string]interface{} "Invalid query parameter"
// @Failure 500 {object} map[string]interface{} "Failed to retrieve history"
// @Router /history [get]
func (h *ReservationHandler) GetReservationHistory(c echo.Context) error {
	startDate := c.QueryParam("startDate")
	endDate := c.QueryParam("endDate")
	roomType := c.QueryParam("type")
	status := c.QueryParam("status")
	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page <= 0 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(c.QueryParam("pageSize"))
	if pageSize <= 0 {
		pageSize = 10
	}

	response, err := h.reservationUsecase.GetReservationHistory(startDate, endDate, roomType, status, page, pageSize)
	if err != nil {
		if e, ok := err.(*usecases.UseCaseError); ok {
			return c.JSON(e.Code, echo.Map{"message": e.Message})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"message": "internal server error"})
	}
	return c.JSON(http.StatusOK, response)
}

// GetReservationByID godoc
// @Summary Detail reservation by ID
// @Description Get full reservation detail (master + reservation details) by reservation ID
// @Tags Reservation
// @Produce json
// @Param id path int true "Reservation ID"
// @Success 200 {object} models.ReservationByIDResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /reservation/{id} [get]
func (h *ReservationHandler) GetReservationByID(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "invalid reservation id"})
	}

	response, err := h.reservationUsecase.GetReservationByID(id)
	if err != nil {
		if e, ok := err.(*usecases.UseCaseError); ok {
			return c.JSON(e.Code, echo.Map{"message": e.Message})
		}
		return c.JSON(http.StatusInternalServerError, echo.Map{"message": "internal server error"})
	}
	return c.JSON(http.StatusOK, response)
}

// UpdateReservationStatus godoc
// @Summary Update reservation status
// @Description Update status of a reservation (booked/canceled/paid)
// @Tags Reservation
// @Accept json
// @Produce json
// @Param request body models.UpdateReservationRequest true "Status update request"
// @Success 200 {object} map[string]string "message: update status success"
// @Failure 400 {object} map[string]string "message: bad request/reservation already canceled/paid"
// @Failure 401 {object} map[string]string "message: unauthorized"
// @Failure 404 {object} map[string]string "message: url not found"
// @Failure 500 {object} map[string]string "message: internal server error"
// @Router /reservation/status [post]
func (h *ReservationHandler) UpdateReservationStatus(c echo.Context) error {
	var req models.UpdateReservationRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "invalid request format"})
	}

	err := h.reservationUsecase.UpdateReservationStatus(req)
	if err != nil {
		if e, ok := err.(*usecases.UseCaseError); ok {
			return c.JSON(e.Code, echo.Map{"message": e.Message})
		}
		return c.JSON(http.StatusInternalServerError, echo.Map{"message": "internal server error"})
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
// @Success 200 {object} models.ScheduleResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /reservations/schedules [get]
func (h *ReservationHandler) GetReservationSchedules(c echo.Context) error {
	startDate := c.QueryParam("startDate")
	endDate := c.QueryParam("endDate")
	page, _ := strconv.Atoi(c.QueryParam("page"))
	pageSize, _ := strconv.Atoi(c.QueryParam("pageSize"))

	response, err := h.reservationUsecase.GetReservationSchedules(startDate, endDate, page, pageSize)
	if err != nil {
		if e, ok := err.(*usecases.UseCaseError); ok {
			return c.JSON(e.Code, echo.Map{"message": e.Message})
		}
		return c.JSON(http.StatusInternalServerError, echo.Map{"message": "internal server error"})
	}
	return c.JSON(http.StatusOK, response)
}

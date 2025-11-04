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

func (h *ReservationHandler) RegisterRoutes(e *echo.Group) {
	e.GET("/reservation/calculation", h.CalculateReservation)
	e.POST("/reservation", h.CreateReservation)
	e.GET("/reservation/history", h.GetReservationHistory)
	e.POST("/reservation/status", h.UpdateReservationStatus)
	e.GET("/reservation/:id", h.GetReservationByID)
	e.GET("/reservations/schedules", h.GetReservationSchedules)
}

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

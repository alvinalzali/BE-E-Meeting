package handlers

import (
	"BE-E-MEETING/app/models"
	"BE-E-MEETING/app/usecases"
	"github.com/labstack/echo/v4"
	"math"
	"net/http"
	"strconv"
)

type RoomHandler struct {
	roomUsecase usecases.RoomUsecase
}

func NewRoomHandler(roomUsecase usecases.RoomUsecase) *RoomHandler {
	return &RoomHandler{roomUsecase: roomUsecase}
}

// CreateRoom godoc
// @Summary Create a new room
// @Description Create a new room with image validation (JPG/PNG ≤1MB)
// @Tags Room
// @Accept multipart/form-data
// @Produce json
// @Param name formData string true "Room name"
// @Param type formData string true "Room type (small/medium/large)"
// @Param capacity formData int true "Room capacity"
// @Param pricePerHour formData number true "Price per hour"
// @Param image formData file true "Room image (JPG/PNG ≤1MB)"
// @Success 201 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /rooms [post]
func (h *RoomHandler) CreateRoom(c echo.Context) error {
	var req models.RoomRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "invalid request format"})
	}

	if req.Name == "" || req.Type == "" || req.Capacity <= 0 || req.PricePerHour <= 0 {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "invalid room data"})
	}

	file, err := c.FormFile("image")
	if err != nil && err != http.ErrMissingFile {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "image file is required"})
	}

	baseURL := c.Scheme() + "://" + c.Request().Host
	imageURL, err := h.roomUsecase.CreateRoom(req, file, baseURL)
	if err != nil {
		if e, ok := err.(*usecases.UseCaseError); ok {
			return c.JSON(e.Code, echo.Map{"message": e.Message})
		}
		return c.JSON(http.StatusInternalServerError, echo.Map{"message": "internal server error"})
	}

	return c.JSON(http.StatusCreated, echo.Map{"message": "room created successfully", "imageURL": imageURL})
}

// GetRooms godoc
// @Summary Get a list of rooms
// @Description Get a list of rooms
// @Tags Room
// @Produce json
// @Param name query string false "Room name"
// @Param type query string false "Room type"
// @Param capacity query string false "Room capacity"
// @Param page query int false "Page number"
// @Param pageSize query int false "Page size"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /rooms [get]
func (h *RoomHandler) GetRooms(c echo.Context) error {
	name := c.QueryParam("name")
	roomType := c.QueryParam("type")
	capacity := c.QueryParam("capacity")
	pageParam := c.QueryParam("page")
	pageSizeParam := c.QueryParam("pageSize")

	page := 1
	pageSize := 10
	if p, err := strconv.Atoi(pageParam); err == nil && p > 0 {
		page = p
	}
	if ps, err := strconv.Atoi(pageSizeParam); err == nil && ps > 0 {
		pageSize = ps
	}

	rooms, totalData, _, err := h.roomUsecase.GetRooms(name, roomType, capacity, pageParam, pageSizeParam)
	if err != nil {
		if e, ok := err.(*usecases.UseCaseError); ok {
			return c.JSON(e.Code, echo.Map{"message": e.Message})
		}
		return c.JSON(http.StatusInternalServerError, echo.Map{"message": "internal server error"})
	}

	totalPage := int(math.Ceil(float64(totalData) / float64(pageSize)))
	return c.JSON(http.StatusOK, echo.Map{
		"message":   "success",
		"data":      rooms,
		"page":      page,
		"pageSize":  pageSize,
		"totalPage": totalPage,
		"totalData": totalData,
	})
}

// GetRoomByID godoc
// @Summary Get a room by ID
// @Description Get a room by ID
// @Tags Room
// @Produce json
// @Param id path string true "Room ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /rooms/{id} [get]
func (h *RoomHandler) GetRoomByID(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "invalid room id"})
	}

	room, err := h.roomUsecase.GetRoomByID(id)
	if err != nil {
		return c.JSON(http.StatusNotFound, echo.Map{"message": "room not found"})
	}

	return c.JSON(http.StatusOK, echo.Map{
		"message": "success",
		"data":    room,
	})
}

// UpdateRoom godoc
// @Summary Update a room by ID
// @Description Update a room by ID
// @Tags Room
// @Accept json
// @Produce json
// @Param id path string true "Room ID"
// @Param room body models.RoomRequest true "Room details"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /rooms/{id} [put]
func (h *RoomHandler) UpdateRoom(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "invalid room id"})
	}

	var req models.RoomRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "invalid request format"})
	}

	err = h.roomUsecase.UpdateRoom(id, req)
	if err != nil {
		if e, ok := err.(*usecases.UseCaseError); ok {
			return c.JSON(e.Code, echo.Map{"message": e.Message})
		}
		return c.JSON(http.StatusInternalServerError, echo.Map{"message": "internal server error"})
	}

	return c.JSON(http.StatusOK, echo.Map{"message": "room updated successfully"})
}

// DeleteRoom godoc
// @Summary Delete a room by ID
// @Description Delete a room by ID
// @Tags Room
// @Produce json
// @Param id path string true "Room ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /rooms/{id} [delete]
func (h *RoomHandler) DeleteRoom(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "invalid room id"})
	}

	err = h.roomUsecase.DeleteRoom(id)
	if err != nil {
		if e, ok := err.(*usecases.UseCaseError); ok {
			return c.JSON(e.Code, echo.Map{"message": e.Message})
		}
		return c.JSON(http.StatusInternalServerError, echo.Map{"message": "internal server error"})
	}

	return c.JSON(http.StatusOK, echo.Map{"message": "delete room success"})
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
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /rooms/{id}/reservation [get]
func (h *RoomHandler) GetRoomReservationSchedule(c echo.Context) error {
	roomID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "invalid room id"})
	}
	date := c.QueryParam("date")
	schedules, room, err := h.roomUsecase.GetRoomReservationSchedule(roomID, date)
	if err != nil {
		if e, ok := err.(*usecases.UseCaseError); ok {
			return c.JSON(e.Code, echo.Map{"message": e.Message})
		}
		return c.JSON(http.StatusInternalServerError, echo.Map{"message": "internal server error"})
	}

	return c.JSON(http.StatusOK, echo.Map{
		"message": "success",
		"data": echo.Map{
			"room":      room,
			"schedules": schedules,
			"date":      date,
		},
	})
}

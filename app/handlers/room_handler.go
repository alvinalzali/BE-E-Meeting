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

func (h *RoomHandler) RegisterRoutes(e *echo.Group) {
	e.POST("/rooms", h.CreateRoom)
	e.GET("/rooms", h.GetRooms)
	e.GET("/rooms/:id", h.GetRoomByID)
	e.PUT("/rooms/:id", h.UpdateRoom)
	e.DELETE("/rooms/:id", h.DeleteRoom)
	e.GET("/rooms/:id/reservation", h.GetRoomReservationSchedule)
}

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

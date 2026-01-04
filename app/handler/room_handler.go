package handler

import (
	"net/http"
	"strconv"

	"BE-E-Meeting/app/entities"
	"BE-E-Meeting/app/usecases"

	"github.com/labstack/echo/v4"
)

// Helper variable (Bisa dihapus jika logic default image dipindah ke Usecase)
var DefaultRoomURL = "http://localhost:8080/assets/default/default_room.jpg"

type RoomHandler struct {
	usecase usecases.RoomUsecase
}

func NewRoomHandler(usecase usecases.RoomUsecase) *RoomHandler {
	return &RoomHandler{usecase: usecase}
}

// CreateRoom godoc
// @Summary Create a new room
// @Description Create a new room with image validation (JPG/PNG ≤1MB)
// @Tags Room
// @Accept json
// @Produce json
// @Param room body entities.RoomRequest true "Room Data"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /rooms [post]
func (h *RoomHandler) CreateRoom(c echo.Context) error {
	var req entities.RoomRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "invalid request format"})
	}

	baseURL := c.Scheme() + "://" + c.Request().Host

	// TANGKAP DATA BALIKAN (updatedRoom)
	updatedRoom, err := h.usecase.Create(req, baseURL)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": err.Error()})
	}

	return c.JSON(http.StatusCreated, echo.Map{
		"message": "room created successfully",
		"data":    updatedRoom, // <--- Pakai updatedRoom, JANGAN req
	})
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
// @Security BearerAuth
// @Router /rooms [get]
func (h *RoomHandler) GetRooms(c echo.Context) error {
	name := c.QueryParam("name")
	roomType := c.QueryParam("type")
	capacity := c.QueryParam("capacity")
	page, _ := strconv.Atoi(c.QueryParam("page"))
	pageSize, _ := strconv.Atoi(c.QueryParam("pageSize"))

	rooms, totalPage, totalData, err := h.usecase.GetAll(name, roomType, capacity, page, pageSize)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": err.Error()})
	}

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
// @Security BearerAuth
// @Router /rooms/{id} [get]
func (h *RoomHandler) GetRoomByID(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "invalid room id"})
	}

	room, err := h.usecase.GetByID(id)
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
// @Param room body entities.RoomRequest true "Room details"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /rooms/{id} [put]
func (h *RoomHandler) UpdateRoom(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "invalid room id"})
	}

	var req entities.RoomRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "invalid request format"})
	}

	baseURL := c.Scheme() + "://" + c.Request().Host

	// TANGKAP DATA BALIKAN
	updatedRoom, err := h.usecase.Update(id, req, baseURL)
	if err != nil {
		return c.JSON(http.StatusNotFound, echo.Map{"message": err.Error()})
	}

	return c.JSON(http.StatusOK, echo.Map{
		"message": "room updated successfully",
		"data":    updatedRoom, // <--- Pakai updatedRoom
	})
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
// @Security BearerAuth
// @Router /rooms/{id} [delete]
func (h *RoomHandler) DeleteRoom(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "invalid room id"})
	}

	err = h.usecase.Delete(id)
	if err != nil {
		return c.JSON(http.StatusNotFound, echo.Map{"message": err.Error()})
	}

	return c.JSON(http.StatusOK, echo.Map{"message": "delete room success"})
}

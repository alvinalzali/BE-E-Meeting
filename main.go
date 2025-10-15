package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/go-playground/validator/v10"
	jwt "github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"

	_ "github.com/lib/pq"
)

// -------------------- STRUCT --------------------

// Validator untuk validasi struct request
type CustomValdator struct {
	validator *validator.Validate
}

func (cv *CustomValdator) Validate(i interface{}) error {
	return cv.validator.Struct(i)
}

// Untuk login menggunakan username/email
type Login struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

// Untuk registrasi user baru
type User struct {
	Username string `json:"username" validate:"required"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
	Name     string `json:"name" validate:"required"`
}

// Untuk JWT claims
type Claims struct {
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// Request body untuk endpoint rooms
type RoomRequest struct {
	Name         string  `json:"name"`
	Type         string  `json:"type"`
	Capacity     int     `json:"capacity"`
	PricePerHour float64 `json:"pricePerHour"`
	ImageURL     string  `json:"imageURL"`
}

// Response struct untuk rooms
type Room struct {
	ID           int       `json:"id"`
	Name         string    `json:"name"`
	RoomType     string    `json:"type"`
	Capacity     int       `json:"capacity"`
	PricePerHour float64   `json:"pricePerHour"`
	PictureURL   string    `json:"imageURL"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// Response struct untuk snacks
type Snack struct {
	ID       int     `json:"id"`
	Name     string  `json:"name"`
	Unit     string  `json:"unit"`
	Price    float64 `json:"price"`
	Category string  `json:"category"`
}

// -------------------- GLOBAL VAR --------------------
var db *sql.DB
var JwtSecret []byte

// -------------------- MAIN --------------------
func main() {
	// Load .env file
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// Ambil konfigurasi database dari environment variable
	dbHost := os.Getenv("DB_HOST")
	dbPort, err := strconv.Atoi(os.Getenv("DB_PORT"))
	if err != nil {
		log.Fatal("Invalid DB_PORT value in .env")
	}
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")
	serverPort := os.Getenv("SERVER_PORT")
	if serverPort == "" {
		serverPort = "8080"
	}

	// Koneksi ke database
	db = connectDB(dbUser, dbPassword, dbName, dbHost, dbPort)

	e := echo.New()
	e.Validator = &CustomValdator{validator: validator.New()}

	// -------------------- ERROR HANDLER --------------------
	// Custom error handler sesuai kontrak API
	e.HTTPErrorHandler = func(err error, c echo.Context) {
		if he, ok := err.(*echo.HTTPError); ok {
			switch he.Code {
			case http.StatusNotFound:
				c.JSON(http.StatusNotFound, echo.Map{"message": "url not found"})
			case http.StatusUnauthorized:
				c.JSON(http.StatusUnauthorized, echo.Map{"message": "unauthorized"})
			default:
				c.JSON(http.StatusInternalServerError, echo.Map{"message": "internal server error"})
			}
		} else {
			c.JSON(http.StatusInternalServerError, echo.Map{"message": "internal server error"})
		}
	}

	// -------------------- ROUTES --------------------
	e.POST("/login", login)
	e.POST("/register", registerUser)

	e.POST("/rooms", CreateRoom)
	e.GET("/rooms", GetRooms)
	e.GET("/rooms/:id", GetRoomByID)
	e.PUT("/rooms/:id", UpdateRoom)
	e.DELETE("/rooms/:id", DeleteRoom)
	e.GET("/snacks", GetSnacks)

	fmt.Println("Server running on port", serverPort)
	e.Logger.Fatal(e.Start(":" + serverPort))
}

// -------------------- DB CONNECTION --------------------
// Fungsi koneksi ke database PostgreSQL
func connectDB(username, password, dbname, host string, port int) *sql.DB {
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port, username, password, dbname)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}
	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Connected to database successfully")
	return db
}

// -------------------- AUTH HANDLER --------------------

// Handler login user
func login(c echo.Context) error {
	var loginData Login

	if err := c.Bind(&loginData); err != nil {
		// error code 400
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Login Failed"}) // "Invalid Input"
	}
	// cek username apakah email atau username
	var sqlStatement string
	if isEmail(loginData.Username) {
		sqlStatement = `SELECT username, password_hash FROM users WHERE email=$1`
	} else {
		sqlStatement = `SELECT username, password_hash FROM users WHERE username=$1`
	}

	var storedUsername, storedPasswordHash string
	err := db.QueryRow(sqlStatement, loginData.Username).Scan(&storedUsername, &storedPasswordHash)
	if err != nil {
		if err == sql.ErrNoRows {
			// error 401
			return c.JSON(http.StatusUnauthorized, echo.Map{"error": "Invalid Credentials"}) // "User Not Found"
		}
		// error 500
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Unknown Error"}) // "Database Error"
	}

	// hash the provided password and compare with stored hash
	err = bcrypt.CompareHashAndPassword([]byte(storedPasswordHash), []byte(loginData.Password))
	if err != nil {
		// error 401
		return c.JSON(http.StatusUnauthorized, echo.Map{"error": "Invalid Credentials"}) // "Invalid Password"
	}

	// ambil role dari db
	var role string
	err = db.QueryRow(`SELECT role FROM users WHERE username=$1`, storedUsername).Scan(&role)
	if err != nil {
		// error 500
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Unknown Error"}) // "Database Error"
	}

	// generate JWT access token
	token, err := generateAccessToken(storedUsername, role)
	if err != nil {
		// error 500
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Unknown Error"}) // "Token Generation Failed"
	}

	// generate JWT refresh token
	refreshToken, err := generateRefreshToken(storedUsername, role)
	if err != nil {
		// error 500
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Unknown Error"}) // "Token Generation Failed"
	}

	// return token di header
	c.Response().Header().Set("Authorization", "Bearer "+token)
	c.Response().Header().Set("Refresh-Token", "Bearer "+refreshToken)

	return c.JSON(http.StatusOK, echo.Map{"message": "Login successful", "accessToken": token, "refreshToken": refreshToken})
}

// Cek apakah input berupa email
func isEmail(input string) bool {
	for _, char := range input {
		if char == '@' {
			return true
		}
	}
	return false
}

// -------------------- JWT TOKEN GENERATOR --------------------

// Generate JWT access token
func generateAccessToken(username string, role string) (string, error) {
	JwtSecret = []byte(os.Getenv("SECRET_KEY"))
	claims := &Claims{
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(JwtSecret)
}

// Generate JWT refresh token
func generateRefreshToken(username string, role string) (string, error) {
	JwtSecret = []byte(os.Getenv("SECRET_KEY"))
	claims := &Claims{
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(JwtSecret)
}

// Handler registrasi user baru
func registerUser(c echo.Context) error {
	var newUser User
	if err := c.Bind(&newUser); err != nil {
		// error code 400
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Bad Request"}) // "Invalid Input"
	}

	if err := c.Validate(&newUser); err != nil {
		// error code 400
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Bad Request"}) // "Validation Error"
	}

	// insert variable default, Enum status, role, lang
	status := "active"

	// hash password
	hashedPassword, err := hashPassword(newUser.Password)
	if err != nil {
		// error 500
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Internal Server Error"}) // "Password Hashing Failed"
	}

	// insert to db
	sqlStatement := `INSERT INTO users (username, email, password_hash, name, status) VALUES ($1, $2, $3, $4, $5)`
	_, err = db.Exec(sqlStatement, newUser.Username, newUser.Email, hashedPassword, newUser.Name, status)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Database Error"})
	}

	return c.JSON(http.StatusOK, echo.Map{"message": "User registered successfully"})
}

// Hash password dengan bcrypt
func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// -------------------- HANDLER ROOMS --------------------

// (POST /rooms) - Tambah ruangan baru
func CreateRoom(c echo.Context) error {
	var req RoomRequest
	if err := c.Bind(&req); err != nil {
		// error jika format request tidak sesuai
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "invalid request format"})
	}

	// validasi tipe dan kapasitas ruangan
	if req.Type != "small" && req.Type != "medium" && req.Type != "large" || req.Capacity <= 0 {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"message": "room type is not valid / capacity must be larger more than 0",
		})
	}

	query := `
        INSERT INTO rooms (name, room_type, capacity, price_per_hour, picture_url, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
    `
	_, err := db.Exec(query, req.Name, req.Type, req.Capacity, req.PricePerHour, req.ImageURL)
	if err != nil {
		log.Println("CreateRoom DB insert error:", err)
		return c.JSON(http.StatusInternalServerError, echo.Map{"message": "internal server error"})
	}

	return c.JSON(http.StatusCreated, echo.Map{"message": "room created successfully"})
}

// (GET /rooms) - List ruangan
func GetRooms(c echo.Context) error {
	name := c.QueryParam("name")
	roomType := c.QueryParam("type")
	capacityParam := c.QueryParam("capacity")
	pageParam := c.QueryParam("page")
	pageSizeParam := c.QueryParam("pageSize")

	// validasi tipe ruangan
	if roomType != "" && roomType != "small" && roomType != "medium" && roomType != "large" {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "room type is not valid"})
	}

	page := 1
	pageSize := 10
	if p, err := strconv.Atoi(pageParam); err == nil && p > 0 {
		page = p
	}
	if ps, err := strconv.Atoi(pageSizeParam); err == nil && ps > 0 {
		pageSize = ps
	}
	offset := (page - 1) * pageSize

	query := `
        SELECT id, name, room_type, capacity, price_per_hour, picture_url, created_at, updated_at
        FROM rooms
        WHERE 1=1
    `
	var args []interface{}
	argIndex := 1

	if name != "" {
		query += fmt.Sprintf(" AND LOWER(name) LIKE LOWER($%d)", argIndex)
		args = append(args, "%"+name+"%")
		argIndex++
	}
	if roomType != "" {
		query += fmt.Sprintf(" AND room_type = $%d", argIndex)
		args = append(args, roomType)
		argIndex++
	}
	if capacityParam != "" {
		if capVal, err := strconv.Atoi(capacityParam); err == nil {
			query += fmt.Sprintf(" AND capacity >= $%d", argIndex)
			args = append(args, capVal)
			argIndex++
		}
	}

	countQuery := "SELECT COUNT(*) FROM (" + query + ") AS total"
	var totalData int
	err := db.QueryRow(countQuery, args...).Scan(&totalData)
	if err != nil {
		log.Println("Count query error:", err)
		return c.JSON(http.StatusInternalServerError, echo.Map{"message": "internal server error"})
	}

	query += fmt.Sprintf(" ORDER BY id ASC LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, pageSize, offset)

	rows, err := db.Query(query, args...)
	if err != nil {
		log.Println("GetRooms query error:", err)
		return c.JSON(http.StatusInternalServerError, echo.Map{"message": "internal server error"})
	}
	defer rows.Close()

	var rooms []Room
	for rows.Next() {
		var r Room
		if err := rows.Scan(&r.ID, &r.Name, &r.RoomType, &r.Capacity, &r.PricePerHour, &r.PictureURL, &r.CreatedAt, &r.UpdatedAt); err != nil {
			log.Println("GetRooms scan error:", err)
			return c.JSON(http.StatusInternalServerError, echo.Map{"message": "internal server error"})
		}
		rooms = append(rooms, r)
	}

	totalPage := (totalData + pageSize - 1) / pageSize
	return c.JSON(http.StatusOK, echo.Map{
		"message":   "success",
		"data":      rooms,
		"page":      page,
		"pageSize":  pageSize,
		"totalPage": totalPage,
		"totalData": totalData,
	})
}

// (GET /rooms/:id) - Detail ruangan
func GetRoomByID(c echo.Context) error {
	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "invalid room id"})
	}

	query := `
        SELECT id, name, room_type, capacity, price_per_hour, picture_url, created_at, updated_at
        FROM rooms WHERE id = $1
    `
	var r Room
	err = db.QueryRow(query, id).Scan(
		&r.ID, &r.Name, &r.RoomType, &r.Capacity, &r.PricePerHour,
		&r.PictureURL, &r.CreatedAt, &r.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return c.JSON(http.StatusNotFound, echo.Map{"message": "room not found"})
	} else if err != nil {
		log.Println("GetRoomByID DB error:", err)
		return c.JSON(http.StatusInternalServerError, echo.Map{"message": "internal server error"})
	}

	return c.JSON(http.StatusOK, echo.Map{
		"message": "success",
		"data":    r,
	})
}

// (PUT /rooms/:id) - Update ruangan
func UpdateRoom(c echo.Context) error {
	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "invalid room id"})
	}

	var req RoomRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "invalid request format"})
	}

	if req.Type != "small" && req.Type != "medium" && req.Type != "large" || req.Capacity <= 0 {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"message": "room type is not valid / capacity must be larger more than 0",
		})
	}

	query := `
        UPDATE rooms 
        SET name=$1, room_type=$2, capacity=$3, price_per_hour=$4, picture_url=$5, updated_at=NOW()
        WHERE id=$6
    `
	res, err := db.Exec(query, req.Name, req.Type, req.Capacity, req.PricePerHour, req.ImageURL, id)
	if err != nil {
		log.Println("UpdateRoom DB error:", err)
		return c.JSON(http.StatusInternalServerError, echo.Map{"message": "internal server error"})
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return c.JSON(http.StatusNotFound, echo.Map{"message": "room not found"})
	}

	return c.JSON(http.StatusOK, echo.Map{"message": "room updated successfully"})
}

// (DELETE /rooms/:id) - Hapus ruangan
func DeleteRoom(c echo.Context) error {
	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "invalid room id"})
	}

	query := `DELETE FROM rooms WHERE id=$1`
	res, err := db.Exec(query, id)
	if err != nil {
		log.Println("DeleteRoom DB error:", err)
		return c.JSON(http.StatusInternalServerError, echo.Map{"message": "internal server error"})
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return c.JSON(http.StatusNotFound, echo.Map{"message": "room not found"})
	}

	return c.JSON(http.StatusOK, echo.Map{"message": "delete room success"})
}

// -------------------- HANDLER SNACKS --------------------

// (GET /snacks) - List snack
func GetSnacks(c echo.Context) error {
	rows, err := db.Query(`SELECT id, name, unit, price, category FROM snacks ORDER BY id ASC`)
	if err != nil {
		log.Println("DB query error:", err)
		return c.JSON(http.StatusInternalServerError, echo.Map{"message": "internal server error"})
	}
	defer rows.Close()

	var snacks []Snack
	for rows.Next() {
		var s Snack
		if err := rows.Scan(&s.ID, &s.Name, &s.Unit, &s.Price, &s.Category); err != nil {
			log.Println("Scan error:", err)
			return c.JSON(http.StatusInternalServerError, echo.Map{"message": "internal server error"})
		}
		snacks = append(snacks, s)
	}

	return c.JSON(http.StatusOK, echo.Map{
		"message": "success",
		"data":    snacks,
	})
}

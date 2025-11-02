package main

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	_ "BE-E-MEETING/docs"

	"github.com/go-playground/validator/v10"
	jwt "github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"

	_ "github.com/lib/pq"
	echoSwagger "github.com/swaggo/echo-swagger"
)

type CustomValdator struct {
	validator *validator.Validate
}

func (cv *CustomValdator) Validate(i interface{}) error {
	return cv.validator.Struct(i)
}

type Login struct {
	//login using username or email
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

type User struct {
	Username string `json:"username" validate:"required"`
	Email    string `json:"email" validate:"required,email"`
	// password harus ada angka, huruf besar, huruf kecil, dan simbol
	Password string `json:"password" validate:"required"`
	Name     string `json:"name" validate:"required"`
}

type getUser struct {
	Created_at string `json:"createdAt"`
	Email      string `json:"email"`
	Id         string `json:"id"`
	Avatar_url string `json:"imageURL"`
	Lang       string `json:"language"`
	Role       string `json:"role"`
	Status     string `json:"status"`
	Updated_at string `json:"updatedAt"`
	Username   string `json:"username"`
	Name       string `json:"name"`
}

type updateUser struct {
	Email      string `json:"email" validate:"omitempty,email"`
	Avatar_url string `json:"imageURL" validate:"omitempty,url"`
	Lang       string `json:"language" validate:"omitempty,oneof=en id"`
	Role       string `json:"role" validate:"omitempty,oneof=admin user"`
	Status     string `json:"status" validate:"omitempty,oneof=active inactive"`
	Username   string `json:"username" validate:"omitempty"`
	Name       string `json:"name" validate:"omitempty"`
	Updated_at string `json:"updatedAt"`
}

type Claims struct {
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

type ResetRequest struct {
	Email string `json:"email" validate:"required,email"`
}

type PasswordConfirmReset struct {
	ConfirmPassword string `json:"confirm_password" validate:"required"`
	NewPassword     string `json:"new_password" validate:"required"`
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

// struct response CalculateReservation
// Room detail dalam response perhitungan reservasi
type RoomCalculationDetail struct {
	Name          string    `json:"name"`
	PricePerHour  float64   `json:"pricePerHour"`
	ImageURL      string    `json:"imageURL"`
	Capacity      int       `json:"capacity"`
	Type          string    `json:"type"`
	SubTotalSnack float64   `json:"subTotalSnack"`
	SubTotalRoom  float64   `json:"subTotalRoom"`
	StartTime     time.Time `json:"startTime"`
	EndTime       time.Time `json:"endTime"`
	Duration      int       `json:"duration"`
	Participant   int       `json:"participant"`
	Snack         Snack     `json:"snack"`
}

// Data personal yang disertakan pada reservasi
type PersonalData struct {
	Name        string `json:"name"`
	PhoneNumber string `json:"phoneNumber"`
	Company     string `json:"company"`
}

type CalculateReservationResponse struct {
	Message string                   `json:"message"`
	Data    CalculateReservationData `json:"data"`
}

type CalculateReservationData struct {
	Rooms         []RoomCalculationDetail `json:"rooms"`
	PersonalData  PersonalData            `json:"personalData"`
	SubTotalRoom  float64                 `json:"subTotalRoom"`
	SubTotalSnack float64                 `json:"subTotalSnack"`
	Total         float64                 `json:"total"`
}

type RoomReservationRequest struct {
	ID          int       `json:"roomID"` // agar lebih eksplisit
	StartTime   time.Time `json:"startTime"`
	EndTime     time.Time `json:"endTime"`
	Participant int       `json:"participant"` // peserta per ruangan
	SnackID     int       `json:"snackID"`
	AddSnack    bool      `json:"addSnack"` // kalau ruangan ini pakai snack atau tidak
}

type ReservationRequestBody struct {
	UserID            int                      `json:"userID"`
	Name              string                   `json:"name"`
	PhoneNumber       string                   `json:"phoneNumber"`
	Company           string                   `json:"company"`
	Notes             string                   `json:"notes"`
	TotalParticipants int                      `json:"totalParticipants"` // total keseluruhan peserta
	AddSnack          bool                     `json:"addSnack"`          // apakah reservasi ini melibatkan snack
	Rooms             []RoomReservationRequest `json:"rooms"`
}

// Response struct history
type HistoryResponse struct {
	Message string               `json:"message"`
	Data    []ReservationHistory `json:"data"`
}

// Data struct h
type ReservationHistory struct {
	ID            int     `json:"id"`
	Name          string  `json:"name"`
	PhoneNumber   float64 `json:"phoneNumber"`
	Company       string  `json:"company"`
	SubTotalSnack float64 `json:"subTotalSnack"`
	SubTotalRoom  float64 `json:"subTotalRoom"`
	GrandTotal    float64 `json:"grandTotal"`
	Type          string  `json:"type"`
	Status        string  `json:"status"`
	CreatedAt     string  `json:"createdAt"`
}

// Struct Reservation History :
// Untuk respons utama
type ReservationHistoryResponse struct {
	Message   string                   `json:"message"`
	Data      []ReservationHistoryData `json:"data"`
	Page      int                      `json:"page"`
	PageSize  int                      `json:"pageSize"`
	TotalPage int                      `json:"totalPage"`
	TotalData int                      `json:"totalData"`
}

// Data utama per reservation
type ReservationHistoryData struct {
	ID            int                            `json:"id"`
	Name          string                         `json:"name"`
	PhoneNumber   string                         `json:"phoneNumber"`
	Company       string                         `json:"company"`
	SubTotalSnack float64                        `json:"subTotalSnack"`
	SubTotalRoom  float64                        `json:"subTotalRoom"`
	Total         float64                        `json:"total"`
	Status        string                         `json:"status"`
	CreatedAt     time.Time                      `json:"createdAt"`
	UpdatedAt     sql.NullTime                   `json:"updatedAt"`
	Rooms         []ReservationHistoryRoomDetail `json:"rooms"`
}

// Detail room di dalam reservation
type ReservationHistoryRoomDetail struct {
	ID         int     `json:"id"`
	Price      float64 `json:"price"`
	Name       string  `json:"name"`
	Type       string  `json:"type"`
	TotalRoom  float64 `json:"totalRoom"`
	TotalSnack float64 `json:"totalSnack"`
}

var BaseURL string = "http://localhost:8080"
var db *sql.DB
var JwtSecret []byte
var DefaultAvatarURL string = BaseURL + "/assets/default/img/default_profile.jpg"
var DefaultRoomURL string = BaseURL + "/assets/default/img/default_room.jpg"

// @title E-Meeting API
// @version 1.0
// @description This is a sample server for E-Meeting.
// @termsOfService http://swagger.io/terms/

// @securityDefinitions.apikey  BearerAuth
// @in                          header
// @name                        Authorization
// @description                 Type "Bearer" followed by a space and JWT token.

func main() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file")
		return
	}

	dbHost := os.Getenv("db_host")
	dbPort, _ := strconv.Atoi(os.Getenv("db_port"))
	dbUser := os.Getenv("db_user")
	dbPassword := os.Getenv("db_password")
	dbName := os.Getenv("db_name")

	db = connectDB(dbUser, dbPassword, dbName, dbHost, dbPort)

	e := echo.New()

	e.Validator = &CustomValdator{validator: validator.New()}

	// route for swagger
	e.GET("/swagger/*", echoSwagger.WrapHandler)

	//assets
	e.Static("/assets", "./assets")

	// route for login, register, password reset
	e.POST("/login", login)
	e.POST("/register", RegisterUser)
	e.POST("password/reset_request", PasswordReset)
	e.PUT("/password/reset/:id", PasswordResetId) //id ini token reset password yang dikirim via email

	// harus pake auth

	authGroup := e.Group("/")
	authGroup.Use(middlewareAuth)
	authGroup.POST("uploads", UploadImage)

	// route for rooms
	authGroup.POST("rooms", CreateRoom)
	authGroup.GET("rooms", GetRooms)
	authGroup.GET("rooms/:id", GetRoomByID)
	authGroup.PUT("rooms/:id", UpdateRoom)
	authGroup.DELETE("rooms/:id", DeleteRoom)
	authGroup.GET("snacks", GetSnacks)

	// route for reservations
	authGroup.GET("reservation/calculation", CalculateReservation)
	authGroup.POST("reservation", CreateReservation)
	authGroup.GET("reservation/history", GetReservationHistory)

	// route group users
	userGroup := e.Group("/users")
	userGroup.Use(middlewareAuth)

	// route users
	userGroup.GET("/:id", GetUserByID)
	userGroup.PUT("/:id", UpdateUserByID)

	e.Logger.Fatal(e.Start(":8080"))

}

func connectDB(username, password, dbname, host string, port int) *sql.DB {
	// connect to db
	connSt := "host=" + host + " port=" + strconv.Itoa(port) + " user=" + username + " password=" + password + " dbname=" + dbname + " sslmode=disable"
	db, err := sql.Open("postgres", connSt)
	if err != nil {
		log.Fatal(err)
	}
	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Connected to DB " + dbname + " successfully on port" + strconv.Itoa(port))
	return db
}

func login(c echo.Context) error {
	var loginData Login

	if err := c.Bind(&loginData); err != nil {
		//error code 400
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Login Failed"}) //"Invalid Input"
	}
	//cek username apakah email atau username
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
			// error 500 //ini kan error 401, jadi gimana?
			return c.JSON(http.StatusUnauthorized, echo.Map{"error": "Invalid Credentials"}) //"User Not Found"
		}
		// error 500
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Unknown Error"}) //"Database Error"
	}

	// hash the provided password and compare with stored hash
	err = bcrypt.CompareHashAndPassword([]byte(storedPasswordHash), []byte(loginData.Password))
	if err != nil {
		// error 500 //ini kan error 401, jadi gimana?
		return c.JSON(http.StatusUnauthorized, echo.Map{"error": "Invalid Credentials"}) //"Invalid Password"
	}

	// ambil role dari db
	var role string
	err = db.QueryRow(`SELECT role FROM users WHERE username=$1`, storedUsername).Scan(&role)
	if err != nil {
		// error 500
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Unknown Error"}) //"Database Error"
	}

	// generate JWT access token
	token, err := generateAccessToken(storedUsername, role)
	if err != nil {
		// error 500
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Unknown Error"}) //"Token Generation Failed"
	}

	// generate JWT refresh token
	refreshToken, err := generateRefreshToken(storedUsername, role)
	if err != nil {
		// error 500
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Unknown Error"}) //"Token Generation Failed"
	}

	// ambil id dari tabel users
	var user_id string
	err = db.QueryRow(`SELECT id FROM users WHERE username=$1`, storedUsername).Scan(&user_id)
	if err != nil {
		// error 500
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Unknown Error"}) //"Database Error"
	}

	// return token
	c.Response().Header().Set("Authorization", "Bearer "+token)
	c.Response().Header().Set("Refresh-Token", "Bearer "+refreshToken)
	c.Response().Header().Set("id", user_id)

	// apa yang dimasukan ke cookie?

	return c.JSON(http.StatusOK, echo.Map{"message": "Login successful", "accessToken": token, "refreshToken": refreshToken})
}

func isEmail(input string) bool {
	// simple check for email format
	for _, char := range input {
		if char == '@' {
			return true
		}
	}
	return false
}

func generateAccessToken(username string, role string) (string, error) {
	JwtSecret = []byte(os.Getenv("secret_key"))
	claims := &Claims{
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(500 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(JwtSecret)
}

func generateRefreshToken(username string, role string) (string, error) {
	JwtSecret = []byte(os.Getenv("secret_key"))
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

func generateResetToken(email string) (string, error) {
	JwtSecret = []byte(os.Getenv("secret_key"))
	claims := &Claims{
		Username: email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(30 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(JwtSecret)
}

// RegisterUser godoc
// @Summary Register a new user
// @Description Register a new user
// @Tags User
// @Accept json
// @Produce json
// @Param user body User true "User object to be registered"
// @Success 201 {object} User
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /register [post]
func RegisterUser(c echo.Context) error {
	var newUser User

	if err := c.Bind(&newUser); err != nil {
		//error code 400
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Bad Request"}) //"Invalid Input"
	}

	if err := c.Validate(&newUser); err != nil {
		//error code 400
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Bad Request"}) //"Validation Error"
	}

	// cek password apakah ada angka, huruf besar, huruf kecil, dan simbol
	if !isValidPassword(newUser.Password) {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Password must contain at least one uppercase letter, one lowercase letter, one number, and one special character"})
	}

	// hash password
	hashedPassword, err := hashPassword(newUser.Password)
	if err != nil {
		// error 500
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Internal Server Error"}) //"Password Hashing Failed"
	}

	//insert variable default, Enum status, role, lang
	status := "active"

	// insert to db
	sqlStatement := `INSERT INTO users (username, email, password_hash, name, status) VALUES ($1, $2, $3, $4, $5)`
	_, err = db.Exec(sqlStatement, newUser.Username, newUser.Email, hashedPassword, newUser.Name, status)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Internal Server Error"}) //"Database Error"
	}

	return c.JSON(http.StatusOK, echo.Map{"message": "User registered successfully"})
}

func isValidPassword(password string) bool {
	var hasUpper, hasLower, hasNumber, hasSpecial bool
	for _, char := range password {
		switch {
		case 'A' <= char && char <= 'Z':
			hasUpper = true
		case 'a' <= char && char <= 'z':
			hasLower = true
		case '0' <= char && char <= '9':
			hasNumber = true
		case (char >= 33 && char <= 47) || (char >= 58 && char <= 64) ||
			(char >= 91 && char <= 96) || (char >= 123 && char <= 126):
			hasSpecial = true
		}
	}
	return hasUpper && hasLower && hasNumber && hasSpecial
}

func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// PasswordResetId godoc
// @Summary Reset user password
// @Description Reset user password using a valid reset token
// @Tags User
// @Accept json
// @Produce json
// @Param id path string true "Reset Token"
// @Param password body PasswordConfirmReset true "New Password and Confirm Password"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /password/reset/{id} [put]
func PasswordResetId(c echo.Context) error {
	id := c.Param("id")
	var passReset PasswordConfirmReset

	//cek apakah id ini valid JWT
	token, err := jwt.Parse(id, func(token *jwt.Token) (interface{}, error) {
		return JwtSecret, nil
	})
	if err != nil {
		//error code 400
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Bad Request"}) //"Invalid Input"
	}
	if !token.Valid {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Bad Request"}) //"Invalid Token"
	}

	if err := c.Bind(&passReset); err != nil {
		//error code 400
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Bad Request"}) //"Invalid Input"
	}

	//validasi apakah new password dan confirm password sama
	if passReset.NewPassword != passReset.ConfirmPassword {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "New password and confirm password do not match"})
	}
	if err := c.Validate(&passReset); err != nil {
		//error code 400
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Bad Request"}) //"Validation Error"
	}

	// cek password apakah ada angka, huruf besar, huruf kecil, dan simbol
	if !isValidPassword(passReset.NewPassword) {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Password must contain at least one uppercase letter, one lowercase letter, one number, and one special character"})
	}

	// hash new password
	hashedPassword, err := hashPassword(passReset.NewPassword)
	if err != nil {
		// error 500
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Internal Server Error"}) //"Password Hashing Failed"
	}
	// update password di db berdasarkan id (token reset password)
	sqlStatement := `UPDATE users SET password_hash=$1 WHERE id=$2`
	_, err = db.Exec(sqlStatement, hashedPassword, id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Internal Server Error"}) //"Database Error"
	}

	return c.JSON(http.StatusOK, echo.Map{"message": "Password reset successfully"})
}

// PasswordReset godoc
// @Summary Request password reset
// @Description Request a password reset token to be sent to the user's email
// @Tags User
// @Accept json
// @Produce json
// @Param email body ResetRequest true "Email"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /password/reset [post]
func PasswordReset(c echo.Context) error {
	var resetReq ResetRequest

	if err := c.Bind(&resetReq); err != nil {
		//error code 400
		return c.JSON(http.StatusBadRequest, err) //"Invalid Input"
	}
	if err := c.Validate(&resetReq); err != nil {
		//error code 400
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Bad Request"}) //"Validation Error"
	}

	// cek apakah email ada di db
	var storedEmail string
	err := db.QueryRow(`SELECT email FROM users WHERE email=$1`, resetReq.Email).Scan(&storedEmail)
	if err != nil {
		if err == sql.ErrNoRows {
			// error 404
			return c.JSON(http.StatusNotFound, echo.Map{"error": "Email not found"}) //"Email Not Found"
		}
		// error 500
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Internel Server Error"}) //"Database Error"
	}

	// keluarkan token reset password (JWT)
	resetToken, err := generateResetToken(storedEmail)
	if err != nil {
		// error 500
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Internal Server Error"}) //"Token Generation Failed"
	}

	fmt.Println("Password reset requested for email:", resetReq.Email)

	return c.JSON(http.StatusOK, echo.Map{"message": "Update Password Success!", "token": resetToken})
}

// fungsi middleware untuk login dan verif jwt
func middlewareAuth(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		authHeader := c.Request().Header.Get("Authorization")
		if authHeader == "" {
			return c.JSON(http.StatusUnauthorized, echo.Map{"error": "Unauthorized"})
		}
		token, err := jwt.Parse(authHeader, func(token *jwt.Token) (interface{}, error) {
			return JwtSecret, nil
		})

		if err != nil || !token.Valid {
			return c.JSON(http.StatusUnauthorized, echo.Map{"error": "Invalid token"})
		}

		//ekstrak claims
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return c.JSON(http.StatusUnauthorized, echo.Map{"error": "Invalid token claims"})
		}

		fmt.Println("Authenticated user:", claims["username"])

		//lanjut ke handler
		return next(c)
	}
}

// GetUserByID godoc
// @Summary Get user by ID
// @Description Retrieve user details by user ID
// @Tags User
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security     BearerAuth
// @Router /users/{id} [get]
func GetUserByID(c echo.Context) error {
	id := c.Param("id")

	idInt, err := strconv.Atoi(id)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid ID"})
	}

	var user getUser
	sqlStatement := `SELECT id, username, email, name, avatar_url, lang, role, status, created_at, updated_at FROM users WHERE id=$1`
	err = db.QueryRow(sqlStatement, idInt).Scan(
		&user.Id, &user.Username, &user.Email, &user.Name,
		&user.Avatar_url, &user.Lang, &user.Role, &user.Status,
		&user.Created_at, &user.Updated_at,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return c.JSON(http.StatusNotFound, echo.Map{"error": "User not found"})
		}
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Database error", "detail": err.Error()})
	}

	//jika user.Avatar_url kosong, ganti ke default
	if user.Avatar_url == "" {
		user.Avatar_url = DefaultAvatarURL
	}

	return c.JSON(http.StatusOK, echo.Map{
		"data":    user,
		"message": "User retrieved successfully",
	})
}

// buat fungsi UpdateUserByID dengan request dan response sesuai updateUser struct
// UpdateUserByID godoc
// @Summary Update user by ID
// @Description Update user details by user ID
// @Tags User
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Param user body main.updateUser true "User object to be updated"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security     BearerAuth
// @Router /users/{id} [put]
func UpdateUserByID(c echo.Context) error {
	id := c.Param("id")
	idInt, err := strconv.Atoi(id)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid ID"})
	}

	var user updateUser
	if err := c.Bind(&user); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid request body"})
	}

	user.Updated_at = time.Now().Format(time.RFC3339)

	// --- Ambil data user saat ini ---
	var currentUser updateUser
	query := `SELECT username, email, avatar_url FROM users WHERE id=$1`
	err = db.QueryRow(query, idInt).Scan(&currentUser.Username, &currentUser.Email, &currentUser.Avatar_url)
	if err != nil {
		log.Println("Error fetching current user:", err)
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "User not found"})
	}

	// === Cek Username ===
	if user.Username != "" && user.Username != currentUser.Username {
		var exists bool
		err = db.QueryRow(`SELECT EXISTS(SELECT 1 FROM users WHERE username=$1 AND id<>$2)`, user.Username, idInt).Scan(&exists)
		if err != nil {
			log.Println("Error checking username:", err)
			return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Database check failed"})
		}
		if exists {
			log.Println("Username already taken, keeping old username.")
			user.Username = currentUser.Username
		}
	} else {
		user.Username = currentUser.Username
	}

	// === Cek Email ===
	if user.Email != "" && user.Email != currentUser.Email {
		var exists bool
		err = db.QueryRow(`SELECT EXISTS(SELECT 1 FROM users WHERE email=$1 AND id<>$2)`, user.Email, idInt).Scan(&exists)
		if err != nil {
			log.Println("Error checking email:", err)
			return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Database check failed"})
		}
		if exists {
			log.Println("Email already taken, keeping old email.")
			user.Email = currentUser.Email
		}
	} else {
		user.Email = currentUser.Email
	}

	// === Jika ada avatar baru ===
	if user.Avatar_url != "" {
		tempURL := user.Avatar_url
		fileName := filepath.Base(tempURL)

		os.MkdirAll("./assets/image", os.ModePerm)
		os.MkdirAll("./assets/image/users", os.ModePerm)

		tempPath := filepath.Join("./assets/temp", fileName)
		finalPath := filepath.Join("./assets/image/users", fileName)

		// Pindahkan file
		err = os.Rename(tempPath, finalPath)
		if err != nil {
			log.Println("Failed to move image:", err)
			return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to move image"})
		}

		// Buat URL final
		baseURL := c.Scheme() + "://" + c.Request().Host
		user.Avatar_url = baseURL + "/assets/image/users/" + fileName

		// Hapus avatar lama (jika bukan default)
		if currentUser.Avatar_url != "" && !strings.Contains(currentUser.Avatar_url, "default") {
			oldFile := filepath.Base(currentUser.Avatar_url)
			os.Remove("./assets/image/users/" + oldFile)
		}
	} else {
		// ambil nilai avatar lama pada database users
		// jika ada, maka gunakan nilai avatar lama
		// jika tidak ada, maka gunakan nilai default
		var avatar_url string
		err = db.QueryRow(`SELECT avatar_url FROM users WHERE id=$1`, idInt).Scan(&avatar_url)
		if err != nil {
			log.Println("Error fetching avatar_url:", err)
			return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Database error"})
		}
		if avatar_url != "" {
			user.Avatar_url = avatar_url
		}

	}

	// --- Update user ---
	sqlStatement := `
		UPDATE users 
		SET username=$1, email=$2, name=$3, avatar_url=$4, 
			lang=$5, role=$6, status=$7, updated_at=$8 
		WHERE id=$9
	`
	_, err = db.Exec(sqlStatement,
		user.Username, user.Email, user.Name, user.Avatar_url,
		user.Lang, user.Role, user.Status, user.Updated_at, idInt,
	)
	if err != nil {
		log.Println("Error updating user:", err)
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Database error"})
	}

	return c.JSON(http.StatusOK, echo.Map{
		"message": "User updated successfully",
		"data":    user,
	})
}

// fungsi memasukan gambar ke folder temp dan mengembalikan url gambarnya
// UploadImage godoc
// @Summary Save an image
// @Description Upload an image to temp folder and return its URL
// @Tags Image
// @Accept multipart/form-data
// @Produce json
// @Param image formData file true "Image file"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /save-image [post]
func UploadImage(c echo.Context) error {
	file, err := c.FormFile("image")
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Failed to upload image"})
	}

	// Validasi tipe file
	contentType := file.Header.Get("Content-Type")
	if !(strings.HasPrefix(contentType, "image/jpeg") || strings.HasPrefix(contentType, "image/png")) {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid file type"})
	}

	// Validasi ukuran max 1MB
	if file.Size > 1024*1024 {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "File size is too large"})
	}

	src, err := file.Open()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to open image file"})
	}
	defer src.Close()

	// Buat folder temp jika belum ada
	os.MkdirAll("./assets/temp", os.ModePerm)

	// Buat nama unik
	ext := filepath.Ext(file.Filename)
	filename := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
	tempPath := filepath.Join("./assets/temp", filename)

	// Simpan file
	dst, err := os.Create(tempPath)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to save image"})
	}
	defer dst.Close()
	io.Copy(dst, src)

	// Buat URL yang dikembalikan ke frontend
	baseURL := c.Scheme() + "://" + c.Request().Host
	imageURL := baseURL + "/assets/temp/" + filename

	return c.JSON(http.StatusOK, echo.Map{
		"message":  "Image uploaded successfully",
		"imageURL": imageURL,
	})
}

// (POST /rooms) - Tambah ruangan baru
// CreateRoom godoc
// @Summary Create a new room
// @Description Create a new room
// @Tags Room
// @Accept json
// @Produce json
// @Param room body RoomRequest true "Room details"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /rooms [post]
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
// UpdateRoom godoc
// @Summary Update a room by ID
// @Description Update a room by ID
// @Tags Room
// @Accept json
// @Produce json
// @Param id path string true "Room ID"
// @Param room body main.RoomRequest true "Room details"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /rooms/{id} [put]
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

// (GET /snacks) - List snack
// GetSnacks godoc
// @Summary Get all snacks
// @Description Get all snacks
// @Tags Snack
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Router /snacks [get]
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

// (GET /reservation/calculation)
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
func CalculateReservation(c echo.Context) error {
	roomID, _ := strconv.Atoi(c.QueryParam("room_id"))
	snackID, _ := strconv.Atoi(c.QueryParam("snack_id"))
	startTimeStr := c.QueryParam("startTime")
	endTimeStr := c.QueryParam("endTime")
	participant, _ := strconv.Atoi(c.QueryParam("participant"))
	//userID := c.QueryParam("user_id")
	name := c.QueryParam("name")
	phoneNumber := c.QueryParam("phoneNumber")
	company := c.QueryParam("company")

	//cek userID sama dengan user_id pada middleware

	// Validasi awal ---
	if roomID == 0 || startTimeStr == "" || endTimeStr == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"message": "missing required parameters"})
	}

	startTime, err := time.Parse(time.RFC3339, startTimeStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"message": "invalid startTime format (must be RFC3339)"})
	}
	endTime, err := time.Parse(time.RFC3339, endTimeStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"message": "invalid endTime format (must be RFC3339)"})
	}

	// Ambil data room
	var room Room
	err = db.QueryRow(`
		SELECT id, name, room_type, capacity, price_per_hour, picture_url, created_at, updated_at
		FROM rooms WHERE id = $1
	`, roomID).Scan(&room.ID, &room.Name, &room.RoomType, &room.Capacity, &room.PricePerHour, &room.PictureURL, &room.CreatedAt, &room.UpdatedAt)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"message": "room not found"})
	}

	// Ambil data snack
	var snack Snack
	err = db.QueryRow(`
		SELECT id, name, unit, price, category
		FROM snacks WHERE id = $1
	`, snackID).Scan(&snack.ID, &snack.Name, &snack.Unit, &snack.Price, &snack.Category)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"message": "snack not found"})
	}

	// Cek booking bentrok
	var existing int
	err = db.QueryRow(`
		SELECT COUNT(*) 
		FROM reservation_details 
		WHERE room_id = $1
		AND (
			(start_at, end_at) OVERLAPS ($2, $3)
		)
	`, roomID, startTime, endTime).Scan(&existing)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"message": "internal server error"})
	}
	if existing > 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"message": "booking bentrok"})
	}

	// Hitung total
	durationMinutes := int(endTime.Sub(startTime).Minutes())
	durationHours := float64(durationMinutes) / 60.0

	subTotalRoom := room.PricePerHour * durationHours
	subTotalSnack := snack.Price * float64(participant)
	total := subTotalRoom + subTotalSnack

	// Siapkan response struct
	roomDetail := RoomCalculationDetail{
		Name:          room.Name,
		PricePerHour:  room.PricePerHour,
		ImageURL:      room.PictureURL,
		Capacity:      room.Capacity,
		Type:          room.RoomType,
		SubTotalSnack: subTotalSnack,
		SubTotalRoom:  subTotalRoom,
		StartTime:     startTime,
		EndTime:       endTime,
		Duration:      durationMinutes,
		Participant:   participant,
		Snack: Snack{
			ID:       snack.ID,
			Name:     snack.Name,
			Unit:     snack.Unit,
			Price:    snack.Price,
			Category: snack.Category,
		},
	}

	response := CalculateReservationResponse{
		Message: "success",
		Data: CalculateReservationData{
			Rooms:         []RoomCalculationDetail{roomDetail},
			PersonalData:  PersonalData{Name: name, PhoneNumber: phoneNumber, Company: company},
			SubTotalRoom:  subTotalRoom,
			SubTotalSnack: subTotalSnack,
			Total:         total,
		},
	}

	return c.JSON(http.StatusOK, response)
}

// (POST /reservation)
// CreateReservation godoc
// @Summary Create a new reservation
// @Description Create a new reservation
// @Tags Reservation
// @Accept json
// @Produce json
// @Param request body ReservationRequestBody true "Reservation request body"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /reservation [post]
func CreateReservation(c echo.Context) error {
	var req ReservationRequestBody

	// Bind JSON
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "invalid request format"})
	}

	// Validasi dasar
	if req.UserID <= 0 || req.Name == "" || req.PhoneNumber == "" || req.Company == "" || len(req.Rooms) == 0 {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "invalid request format"})
	}

	for _, room := range req.Rooms {
		if room.StartTime.IsZero() || room.EndTime.IsZero() {
			return c.JSON(http.StatusBadRequest, echo.Map{"message": "invalid start or end time"})
		}
	}

	// Cek bentrok booking
	for _, room := range req.Rooms {
		var existing int
		err := db.QueryRow(`
			SELECT COUNT(*)
			FROM reservation_details
			WHERE room_id = $1
			AND (start_at, end_at) OVERLAPS ($2, $3)
		`, room.ID, room.StartTime, room.EndTime).Scan(&existing)
		if err != nil {
			log.Println("Error checking overlap:", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"message": "internal server error"})
		}
		if existing > 0 {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"message": fmt.Sprintf("Room %d has already been booked for that time range", room.ID),
			})
		}
	}

	// Mulai transaksi
	tx, err := db.Begin()
	if err != nil {
		log.Println("Error starting transaction:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"message": "internal server error"})
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		} else {
			tx.Commit()
		}
	}()

	// Insert ke tabel reservations
	var reservationID int
	err = tx.QueryRow(`
		INSERT INTO reservations (
			user_id, contact_name, contact_phone, contact_company,
			note, status_reservation, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, 'booked', NOW(), NOW())
		RETURNING id
	`, req.UserID, req.Name, req.PhoneNumber, req.Company, req.Notes).Scan(&reservationID)
	if err != nil {
		log.Println("Error inserting reservation:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"message": "internal server error"})
	}

	// Variabel untuk subtotal
	var subtotalSnack float64
	var subtotalRoom float64

	// Loop tiap room
	for _, room := range req.Rooms {
		var roomTable Room
		err = tx.QueryRow(`
			SELECT id, name, room_type, capacity, price_per_hour, picture_url, created_at, updated_at
			FROM rooms WHERE id = $1
		`, room.ID).Scan(
			&roomTable.ID,
			&roomTable.Name,
			&roomTable.RoomType,
			&roomTable.Capacity,
			&roomTable.PricePerHour,
			&roomTable.PictureURL,
			&roomTable.CreatedAt,
			&roomTable.UpdatedAt,
		)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"message": "internal server error"})
		}

		var snackTable Snack
		err = tx.QueryRow(`
			SELECT id, name, unit, price, category
			FROM snacks WHERE id = $1
		`, room.SnackID).Scan(
			&snackTable.ID,
			&snackTable.Name,
			&snackTable.Unit,
			&snackTable.Price,
			&snackTable.Category,
		)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"message": "internal server error"})
		}

		// Hitung durasi dan total harga
		durationMinute := int(room.EndTime.Sub(room.StartTime).Minutes())
		totalRoom := (float64(durationMinute) / 60.0) * roomTable.PricePerHour
		totalSnack := float64(room.Participant) * snackTable.Price

		// Tambahkan ke subtotal
		subtotalRoom += totalRoom
		subtotalSnack += totalSnack

		// Insert ke reservation_details
		_, err = tx.Exec(`
			INSERT INTO reservation_details (
				reservation_id,
				room_id, room_name, room_price,
				snack_id, snack_name, snack_price,
				duration_minute, total_participants,
				total_room, total_snack,
				start_at, end_at,
				created_at, updated_at
			)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,NOW(),NOW())
		`,
			reservationID,
			room.ID, roomTable.Name, roomTable.PricePerHour,
			room.SnackID, snackTable.Name, snackTable.Price,
			durationMinute, room.Participant,
			totalRoom, totalSnack,
			room.StartTime, room.EndTime,
		)
		if err != nil {
			log.Println("Error inserting reservation detail:", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"message": "internal server error"})
		}

		// Update subtotal dan total di tabel reservations
		total := subtotalRoom + subtotalSnack
		_, err = tx.Exec(`
			UPDATE reservations
			SET subtotal_room = $1,
				subtotal_snack = $2,
				duration_minute = $3,
				total = $4,
				total_participants = $5,
				add_snack = $6,
				updated_at = NOW()
			WHERE id = $7
		`, subtotalRoom, subtotalSnack, durationMinute, total, room.Participant, room.AddSnack, reservationID)
		if err != nil {
			log.Println("Error updating reservation totals:", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"message": "internal server error"})
		}
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
func GetReservationHistory(c echo.Context) error {
	startDate := c.QueryParam("startDate")
	endDate := c.QueryParam("endDate")
	roomType := c.QueryParam("type")
	status := c.QueryParam("status")

	// Validasi room type
	validTypes := map[string]bool{
		"small": true, "medium": true, "large": true,
	}
	if !validTypes[strings.ToLower(roomType)] {
		return c.JSON(http.StatusBadRequest, map[string]string{"message": "room type is not valid"})
	}

	// Pagination parameter
	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page <= 0 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(c.QueryParam("pageSize"))
	if pageSize <= 0 {
		pageSize = 10
	}
	offset := (page - 1) * pageSize

	// query filter
	query := `
	SELECT 
		r.id, r.contact_name, r.contact_phone, r.contact_company,
		COALESCE(SUM(rd.snack_price),0) AS sub_total_snack,
		COALESCE(SUM(rd.room_price),0) AS sub_total_room,
		COALESCE(SUM(rd.snack_price + rd.room_price),0) AS total,
		r.status_reservation, r.created_at, r.updated_at
	FROM reservations r
	JOIN reservation_details rd ON rd.reservation_id = r.id
	JOIN rooms rm ON rm.id = rd.room_id
	WHERE 1=1
	`

	args := []interface{}{}
	argIdx := 1

	if startDate != "" {
		query += fmt.Sprintf(" AND r.created_at >= $%d", argIdx)
		args = append(args, startDate)
		argIdx++
	}
	if endDate != "" {
		query += fmt.Sprintf(" AND r.created_at <= $%d", argIdx)
		args = append(args, endDate)
		argIdx++
	}
	if roomType != "" {
		query += fmt.Sprintf(" AND rm.room_type = $%d", argIdx)
		args = append(args, roomType)
		argIdx++
	}
	if status != "" {
		query += fmt.Sprintf(" AND r.status_reservation = $%d", argIdx)
		args = append(args, status)
		argIdx++
	}

	query += `
	GROUP BY r.id, r.contact_name, r.contact_phone, r.contact_company, r.status_reservation, r.created_at, r.updated_at
	ORDER BY r.created_at DESC
	LIMIT $%d OFFSET $%d
	`
	query = fmt.Sprintf(query, argIdx, argIdx+1)
	args = append(args, pageSize, offset)

	// Jalankan query utama
	rows, err := db.Query(query, args...)
	if err != nil {
		log.Println("Error fetching reservation history:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"message": "internal server error"})
	}
	defer rows.Close()

	var histories []ReservationHistoryData
	for rows.Next() {
		var h ReservationHistoryData
		err := rows.Scan(
			&h.ID, &h.Name, &h.PhoneNumber, &h.Company,
			&h.SubTotalSnack, &h.SubTotalRoom, &h.Total,
			&h.Status, &h.CreatedAt, &h.UpdatedAt,
		)
		if err != nil {
			log.Println("Error scanning reservation:", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"message": "internal server error"})
		}

		// Ambil data room per reservation
		roomRows, err := db.Query(`
			SELECT rm.id, rm.price_per_hour, rm.name, rm.room_type,
				COALESCE(rd.room_price,0), COALESCE(rd.snack_price,0)
			FROM reservation_details rd
			JOIN rooms rm ON rm.id = rd.room_id
			WHERE rd.reservation_id = $1
		`, h.ID)
		if err != nil {
			log.Println("Error fetching rooms:", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"message": "internal server error"})
		}

		for roomRows.Next() {
			var r ReservationHistoryRoomDetail
			err := roomRows.Scan(
				&r.ID, &r.Price, &r.Name, &r.Type,
				&r.TotalRoom, &r.TotalSnack,
			)
			if err != nil {
				log.Println("Error scanning room detail:", err)
				return c.JSON(http.StatusInternalServerError, map[string]string{"message": "internal server error"})
			}
			h.Rooms = append(h.Rooms, r)
		}
		roomRows.Close()

		histories = append(histories, h)
	}

	// --- Hitung total data ---
	var totalData int
	countQuery := `
		SELECT COUNT(DISTINCT r.id)
		FROM reservations r
		JOIN reservation_details rd ON rd.reservation_id = r.id
		JOIN rooms rm ON rm.id = rd.room_id
		WHERE 1=1
	`
	countArgs := []interface{}{}
	argCount := 1

	if startDate != "" {
		countQuery += fmt.Sprintf(" AND r.created_at >= $%d", argCount)
		countArgs = append(countArgs, startDate)
		argCount++
	}
	if endDate != "" {
		countQuery += fmt.Sprintf(" AND r.created_at <= $%d", argCount)
		countArgs = append(countArgs, endDate)
		argCount++
	}
	if roomType != "" {
		countQuery += fmt.Sprintf(" AND rm.room_type = $%d", argCount)
		countArgs = append(countArgs, roomType)
		argCount++
	}
	if status != "" {
		countQuery += fmt.Sprintf(" AND r.status_reservation = $%d", argCount)
		countArgs = append(countArgs, status)
		argCount++
	}

	err = db.QueryRow(countQuery, countArgs...).Scan(&totalData)
	if err != nil {
		log.Println("Error counting data:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"message": "internal server error"})
	}

	totalPage := int(math.Ceil(float64(totalData) / float64(pageSize)))

	// --- Jika tidak ada data ---
	if len(histories) == 0 {
		return c.JSON(http.StatusNotFound, map[string]string{"message": "url not found"})
	}

	// --- Response sukses ---
	return c.JSON(http.StatusOK, ReservationHistoryResponse{
		Message:   "Reservation history fetched successfully",
		Data:      histories,
		Page:      page,
		PageSize:  pageSize,
		TotalPage: totalPage,
		TotalData: totalData,
	})
}

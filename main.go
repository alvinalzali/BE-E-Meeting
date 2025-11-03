package main

import (
	"database/sql"
	"fmt"
	"io"
	"log"
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

type RoomRequest struct {
	Name         string  `json:"name"`
	Type         string  `json:"type"`
	Capacity     int     `json:"capacity"`
	PricePerHour float64 `json:"pricePerHour"`
	ImageURL     string  `json:"imageURL"`
}

type RoomSearchRequest struct {
	Filter     string `json:"filter,omitempty"`       // sama seperti name search
	RoomTypeID string `json:"room_type_id,omitempty"` // "small","medium","large"
	Capacity   int    `json:"capacity,omitempty"`
	Page       int    `json:"page,omitempty"`
	PageSize   int    `json:"pageSize,omitempty"`
}

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

type Snack struct {
	ID       int     `json:"id"`
	Name     string  `json:"name"`
	Unit     string  `json:"unit"`
	Price    float64 `json:"price"`
	Category string  `json:"category"`
}

type SnackInfo struct {
	ID       int     `json:"id"`
	Name     string  `json:"name"`
	Unit     string  `json:"unit"`
	Price    float64 `json:"price"`
	Category string  `json:"category"`
}

type RoomInfo struct {
	Name         string     `json:"name"`
	PricePerHour float64    `json:"pricePerHour"`
	ImageURL     string     `json:"imageURL"`
	Capacity     int        `json:"capacity"`
	Type         string     `json:"type"`
	TotalSnack   float64    `json:"totalSnack"`
	TotalRoom    float64    `json:"totalRoom"`
	StartTime    string     `json:"startTime"`
	EndTime      string     `json:"endTime"`
	Duration     int        `json:"duration"`
	Participant  int        `json:"participant"`
	Snack        *SnackInfo `json:"snack,omitempty"`
}

type PersonalData struct {
	Name        string `json:"name"`
	PhoneNumber string `json:"phoneNumber"`
	Company     string `json:"company"`
}

type UpdateReservationRequest struct {
	ReservationID int    `json:"reservation_id" validate:"required"`
	Status        string `json:"status" validate:"required,oneof=booked cancel paid"`
}

type SimpleMessageResponse struct {
	Message string `json:"message"`
}

// route GET /reservations/schedules
type Schedule struct {
	StartTime string `json:"startTime"`
	EndTime   string `json:"endTime"`
	Status    string `json:"status"`
}

type RoomScheduleInfo struct {
	ID          string     `json:"id"`
	RoomName    string     `json:"roomName"`
	CompanyName string     `json:"companyName"`
	Schedules   []Schedule `json:"schedules"`
}

type ScheduleResponse struct {
	Message   string             `json:"message"`
	Data      []RoomScheduleInfo `json:"data"`
	Page      int                `json:"page"`
	PageSize  int                `json:"pageSize"`
	TotalPage int                `json:"totalPage"`
	TotalData int                `json:"totalData"`
}

// route GET /dashboard
type DashboardRoom struct {
	ID                int     `json:"id"`
	Name              string  `json:"name"`
	Omzet             float64 `json:"omzet"`
	PercentageOfUsage float64 `json:"percentageOfUsage"`
}

type DashboardResponse struct {
	Message string `json:"message"`
	Data    struct {
		TotalRoom        int             `json:"totalRoom"`
		TotalVisitor     int             `json:"totalVisitor"`
		TotalReservation int             `json:"totalReservation"`
		TotalOmzet       float64         `json:"totalOmzet"`
		Rooms            []DashboardRoom `json:"rooms"`
	} `json:"data"`
}

// route GET /rooms/:id/reservation
type RoomSchedule struct {
	ID               int       `json:"id"`
	StartTime        time.Time `json:"startTime"`
	EndTime          time.Time `json:"endTime"`
	Status           string    `json:"status"`
	TotalParticipant int       `json:"totalParticipant"`
}

var BaseURL string = "http://localhost:8080"
var ImageURL string
var db *sql.DB
var JwtSecret []byte
var DefaultAvatar string = BaseURL + "/assets/default/img/default_profile.jpg"

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
	e.POST("/password/reset_request", PasswordReset)
	e.PUT("/password/reset/:id", PasswordResetId) //id ini token reset password yang dikirim via email
	e.POST("/uploads", UploadImage)

	// Rooms routes dengan middlewareAPIContract untuk setiap endpoint
	e.POST("/rooms", CreateRoom, middlewareAPIContract)
	e.GET("/rooms", GetRooms, middlewareAPIContract)
	e.GET("/rooms/:id", GetRoomByID, middlewareAPIContract)
	e.GET("/rooms/:id/reservation", GetRoomReservationSchedule, middlewareAPIContract)
	e.PUT("/rooms/:id", UpdateRoom, middlewareAPIContract)
	e.DELETE("/rooms/:id", DeleteRoom, middlewareAPIContract)

	// Snacks route
	e.GET("/snacks", GetSnacks, middlewareAPIContract)

	// Public routes tidak perlu middleware
	e.POST("/login", login)
	e.POST("/register", RegisterUser)
	e.POST("/password/reset_request", PasswordReset)
	e.PUT("/password/reset/:id", PasswordResetId)
	e.POST("/uploads", UploadImage)

	//routes reservation
	// e.GET("/reservation/calculation", CalculateReservation)
	e.POST("/reservation/status", UpdateReservationStatus)
	e.GET("/reservation/:id", GetReservationByID)
	e.GET("/reservations/schedules", GetReservationSchedules, middlewareAuth)

	// dashboard dan users group tetap menggunakan middlewareAuth
	e.GET("/dashboard", GetDashboard, middlewareAuth)

	userGroup := e.Group("/users")
	userGroup.Use(middlewareAuth)
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

// login godoc
// @Summary User login
// @Description Authenticate user and return JWT tokens
// @Tags User
// @Accept json
// @Produce json
// @Param login body Login true "Login credentials"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /login [post]
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

	// return token
	c.Response().Header().Set("Authorization", "Bearer "+token)
	c.Response().Header().Set("Refresh-Token", "Bearer "+refreshToken)

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

// fungsi middleware untuk validasi header dan JWT khusus API rooms/snacks
func middlewareAPIContract(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Validasi Accept header (jika disertakan harus JSON)
		accept := c.Request().Header.Get("Accept")
		if accept != "" && !strings.Contains(strings.ToLower(accept), "application/json") {
			return c.JSON(http.StatusBadRequest, echo.Map{"message": "Invalid Accept header, must be application/json"})
		}

		// Validasi Content-Type untuk method yang mengirim body
		if c.Request().Method == http.MethodPost || c.Request().Method == http.MethodPut || c.Request().Method == http.MethodPatch {
			ct := c.Request().Header.Get("Content-Type")
			if ct != "" && !strings.Contains(strings.ToLower(ct), "application/json") && !strings.HasPrefix(strings.ToLower(ct), "multipart/form-data") {
				return c.JSON(http.StatusBadRequest, echo.Map{"message": "Invalid Content-Type, must be application/json or multipart/form-data"})
			}
		}

		// Periksa Authorization Bearer token (wajib untuk semua route di group)
		authHeader := c.Request().Header.Get("Authorization")
		if authHeader == "" {
			return c.JSON(http.StatusUnauthorized, echo.Map{"message": "Missing authorization token"})
		}
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			return c.JSON(http.StatusUnauthorized, echo.Map{"message": "Invalid authorization format"})
		}
		tokenString := parts[1]

		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
			return JwtSecret, nil
		})
		if err != nil || !token.Valid {
			return c.JSON(http.StatusUnauthorized, echo.Map{"message": "Invalid or expired token"})
		}

		// set claims ke context untuk handler jika perlu
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)

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
		user.Avatar_url = DefaultAvatar
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

	//masukan update_at dengan waktu sekarang
	user.Updated_at = time.Now().Format(time.RFC3339)

	//jika user upload gambar baru, load dari variabel global imageURL
	if ImageURL != "" {
		user.Avatar_url = ImageURL
	}

	sqlStatement := `UPDATE users SET username=$1, email=$2, name=$3, avatar_url=$4, lang=$5, role=$6, status=$7, updated_at=$8 WHERE id=$9`
	_, err = db.Exec(sqlStatement, user.Username, user.Email, user.Name, user.Avatar_url, user.Lang, user.Role, user.Status, user.Updated_at, idInt)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Database error", "detail": err.Error()})
	}

	//hapus gambar di temp melalui variabel global ImageURL
	if ImageURL != "" {
		filePath := strings.TrimPrefix(ImageURL, "/")
		err = os.Remove(BaseURL + "/assets/temp/" + filePath)
		if err != nil {
			fmt.Println("Failed to delete temp image:", err)
		}
		// reset variabel global ImageURL
		ImageURL = ""
	}
	return c.JSON(http.StatusOK, echo.Map{
		"message": "User updated successfully",
		"data":    user,
	})
}

// fungsi memasukan gambar ke folder temp dan mengembalikan url gambarnya
// UploadImage godoc
// @Summary Save an image
// @Description Save an image
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

	// Buka file upload
	src, err := file.Open()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to open image file"})
	}
	defer src.Close()

	// Pastikan folder temp ada
	os.MkdirAll("./assets/temp", os.ModePerm)

	// Buat nama file baru berdasarkan timestamp
	ext := filepath.Ext(file.Filename)
	timestamp := time.Now().Unix()
	filename := fmt.Sprintf("%d%s", timestamp, ext)
	filePath := "./assets/temp/" + filename

	// Simpan ke folder
	dst, err := os.Create(filePath)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to save image"})
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to copy image"})
	}

	// Buat URL image (pastikan BaseURL kamu sudah didefinisikan)
	imageURL := BaseURL + "/assets/temp/" + filename

	return c.JSON(http.StatusOK, echo.Map{
		"message":  "Image uploaded successfully",
		"imageURL": imageURL,
	})
}

// (POST /rooms) - Tambah ruangan baru
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
func CreateRoom(c echo.Context) error {
	var req RoomRequest

	// Bind JSON body
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "invalid request format"})
	}

	// Basic validation
	if req.Name == "" || req.Type == "" || req.Capacity <= 0 || req.PricePerHour <= 0 {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "invalid room data"})
	}

	// Prepare imageURL variable (will be stored in DB)
	imageURL := strings.TrimSpace(req.ImageURL)
	finalFilename := ""
	createdInRooms := false

	// If frontend provided imageURL pointing to temp, move it to rooms
	if imageURL != "" && strings.Contains(imageURL, "/assets/temp/") {
		tempFilename := filepath.Base(imageURL)
		tempPath := filepath.Join(".", "assets", "temp", tempFilename)

		// ensure temp exists
		if _, err := os.Stat(tempPath); err == nil {
			// ensure rooms dir exists
			uploadDir := filepath.Join(".", "assets", "rooms")
			if err := os.MkdirAll(uploadDir, 0755); err != nil {
				return c.JSON(http.StatusInternalServerError, echo.Map{"message": "error creating upload directory"})
			}

			// create target filename
			finalFilename = fmt.Sprintf("%d%s", time.Now().UnixNano(), filepath.Ext(tempFilename))
			newPath := filepath.Join(uploadDir, finalFilename)

			// try rename, fallback to copy+remove
			if err := os.Rename(tempPath, newPath); err != nil {
				// fallback
				src, err := os.Open(tempPath)
				if err != nil {
					return c.JSON(http.StatusInternalServerError, echo.Map{"message": "error processing uploaded image"})
				}
				defer src.Close()

				dst, err := os.Create(newPath)
				if err != nil {
					return c.JSON(http.StatusInternalServerError, echo.Map{"message": "error processing uploaded image"})
				}
				defer dst.Close()

				if _, err = io.Copy(dst, src); err != nil {
					// cleanup partial file
					_ = os.Remove(newPath)
					return c.JSON(http.StatusInternalServerError, echo.Map{"message": "error saving image"})
				}
				_ = os.Remove(tempPath)
			}
			imageURL = fmt.Sprintf("%s/assets/rooms/%s", BaseURL, finalFilename)
			createdInRooms = true
		} else {
			// temp file not found -> ignore and allow file upload path
			imageURL = ""
		}
	}

	// If no temp-image moved, accept direct multipart upload
	if imageURL == "" {
		file, err := c.FormFile("image")
		if err != nil {
			return c.JSON(http.StatusBadRequest, echo.Map{"message": "image file is required"})
		}

		// Validate file size (1MB)
		if file.Size > 1<<20 {
			return c.JSON(http.StatusBadRequest, echo.Map{"message": "image file size must be less than 1MB"})
		}

		src, err := file.Open()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, echo.Map{"message": "error opening uploaded file"})
		}
		defer src.Close()

		// read header for mime detection
		buf := make([]byte, 512)
		n, _ := src.Read(buf)
		contentType := http.DetectContentType(buf[:n])
		if contentType != "image/jpeg" && contentType != "image/png" {
			return c.JSON(http.StatusBadRequest, echo.Map{"message": "invalid file type, only JPG/PNG allowed"})
		}
		// reset reader
		if _, err := src.Seek(0, 0); err != nil {
			return c.JSON(http.StatusInternalServerError, echo.Map{"message": "error processing file"})
		}

		// ensure rooms dir exists
		uploadDir := filepath.Join(".", "assets", "rooms")
		if err := os.MkdirAll(uploadDir, 0755); err != nil {
			return c.JSON(http.StatusInternalServerError, echo.Map{"message": "error creating upload directory"})
		}

		finalFilename = fmt.Sprintf("%d%s", time.Now().UnixNano(), filepath.Ext(file.Filename))
		dstPath := filepath.Join(uploadDir, finalFilename)

		dst, err := os.Create(dstPath)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, echo.Map{"message": "error creating destination file"})
		}
		defer dst.Close()

		if _, err = io.Copy(dst, src); err != nil {
			_ = os.Remove(dstPath)
			return c.JSON(http.StatusInternalServerError, echo.Map{"message": "error saving file"})
		}

		imageURL = fmt.Sprintf("%s/assets/rooms/%s", BaseURL, finalFilename)
		createdInRooms = true
	}

	// Insert into DB
	query := `
        INSERT INTO rooms (name, room_type, capacity, price_per_hour, picture_url, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
    `
	_, err := db.Exec(query, req.Name, req.Type, req.Capacity, req.PricePerHour, imageURL)
	if err != nil {
		// cleanup created file in rooms when DB insert fails
		if createdInRooms && finalFilename != "" {
			_ = os.Remove(filepath.Join(".", "assets", "rooms", finalFilename))
		}
		log.Println("CreateRoom DB insert error:", err)
		return c.JSON(http.StatusInternalServerError, echo.Map{"message": "error saving room data"})
	}

	return c.JSON(http.StatusCreated, echo.Map{"message": "room created successfully", "imageURL": imageURL})
}

// (GET /rooms) - List ruangan
// GetRooms godoc
// @Summary Get a list of rooms
// @Description Get a list of rooms, supports both query params and JSON body
// @Tags Room
// @Accept json
// @Produce json
// @Param filter query string false "Room name filter (via query)"
// @Param type query string false "Room type (via query)"
// @Param capacity query int false "Room capacity (via query)"
// @Param page query int false "Page number (via query)"
// @Param pageSize query int false "Page size (via query)"
// @Param searchBody body RoomSearchRequest false "Search criteria (via JSON body)"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /rooms [get]
func GetRooms(c echo.Context) error {
	// Initialize request struct
	var req RoomSearchRequest

	// Check Content-Type header
	ct := c.Request().Header.Get("Content-Type")

	// Handle JSON body if Content-Type is application/json
	if strings.HasPrefix(ct, "application/json") {
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, echo.Map{
				"message": "invalid request format",
				"error":   err.Error(),
			})
		}
	} else {
		// Handle query parameters
		if v := c.QueryParam("filter"); v != "" {
			req.Filter = v
		}
		if v := c.QueryParam("room_type_id"); v != "" {
			req.RoomTypeID = v
		}
		if v := c.QueryParam("capacity"); v != "" {
			if i, err := strconv.Atoi(v); err == nil {
				req.Capacity = i
			}
		}
		if v := c.QueryParam("page"); v != "" {
			if i, err := strconv.Atoi(v); err == nil {
				req.Page = i
			}
		}
		if v := c.QueryParam("pageSize"); v != "" {
			if i, err := strconv.Atoi(v); err == nil {
				req.PageSize = i
			}
		}
	}

	// Validate page and pageSize
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 10
	}
	offset := (req.Page - 1) * req.PageSize

	// Build query
	query := `
        SELECT id, name, room_type, capacity, price_per_hour, picture_url, 
               created_at, 
               COALESCE(updated_at, created_at) as updated_at
        FROM rooms
        WHERE 1=1
    `
	var args []interface{}
	argIndex := 1

	// Add filters
	if req.Filter != "" {
		query += fmt.Sprintf(" AND LOWER(name) LIKE LOWER($%d)", argIndex)
		args = append(args, "%"+req.Filter+"%")
		argIndex++
	}
	if req.RoomTypeID != "" {
		query += fmt.Sprintf(" AND room_type = $%d", argIndex)
		args = append(args, req.RoomTypeID)
		argIndex++
	}
	if req.Capacity > 0 {
		query += fmt.Sprintf(" AND capacity >= $%d", argIndex)
		args = append(args, req.Capacity)
		argIndex++
	}

	// Get total count
	countQuery := "SELECT COUNT(*) FROM (" + query + ") AS total"
	var totalData int
	if err := db.QueryRow(countQuery, args...).Scan(&totalData); err != nil {
		log.Println("Count query error:", err)
		return c.JSON(http.StatusInternalServerError, echo.Map{"message": "internal server error"})
	}

	// Add pagination
	query += fmt.Sprintf(" ORDER BY id ASC LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, req.PageSize, offset)

	// Execute query
	rows, err := db.Query(query, args...)
	if err != nil {
		log.Println("GetRooms query error:", err)
		return c.JSON(http.StatusInternalServerError, echo.Map{"message": "internal server error"})
	}
	defer rows.Close()

	// Process results using sql.NullTime
	var rooms []Room
	for rows.Next() {
		var r Room
		var updatedAt sql.NullTime

		if err := rows.Scan(
			&r.ID,
			&r.Name,
			&r.RoomType,
			&r.Capacity,
			&r.PricePerHour,
			&r.PictureURL,
			&r.CreatedAt,
			&updatedAt,
		); err != nil {
			log.Println("GetRooms scan error:", err)
			return c.JSON(http.StatusInternalServerError, echo.Map{"message": "internal server error"})
		}

		// If updated_at is null, use created_at
		if updatedAt.Valid {
			r.UpdatedAt = updatedAt.Time
		} else {
			r.UpdatedAt = r.CreatedAt
		}

		rooms = append(rooms, r)
	}

	totalPage := (totalData + req.PageSize - 1) / req.PageSize
	return c.JSON(http.StatusOK, echo.Map{
		"message":   "success",
		"data":      rooms,
		"page":      req.Page,
		"pageSize":  req.PageSize,
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
        SELECT id, name, room_type, capacity, price_per_hour, picture_url, created_at, 
               COALESCE(updated_at, created_at) as updated_at
        FROM rooms WHERE id = $1
    `
	var r Room
	var updatedAt sql.NullTime

	err = db.QueryRow(query, id).Scan(
		&r.ID,
		&r.Name,
		&r.RoomType,
		&r.Capacity,
		&r.PricePerHour,
		&r.PictureURL,
		&r.CreatedAt,
		&updatedAt,
	)

	if err == sql.ErrNoRows {
		return c.JSON(http.StatusNotFound, echo.Map{"message": "room not found"})
	} else if err != nil {
		log.Println("GetRoomByID DB error:", err)
		return c.JSON(http.StatusInternalServerError, echo.Map{"message": "internal server error"})
	}

	// If updated_at is null, use created_at
	if updatedAt.Valid {
		r.UpdatedAt = updatedAt.Time
	} else {
		r.UpdatedAt = r.CreatedAt
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
// GetReservationCalculation godoc
// @Summary Get reservation calculation
// @Description Get reservation calculation
// @Tags Reservation
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Router /reservation/calculation [get]
// func CalculateReservation(c echo.Context) error {
// 	//ambil query data dari parameter request URL
// 	roomID := c.QueryParam("room_id")
// 	snackID := c.QueryParam("snack_id")
// 	startTime := c.QueryParam("startTime")
// 	endTime := c.QueryParam("endTime")
// 	participant := c.QueryParam("participant")
// 	userID := c.QueryParam("user_id")
// 	name := c.QueryParam("name")
// 	phoneNumber := c.QueryParam("phoneNumber")
// 	company := c.QueryParam("company")

// 	//validasi parameter wajib
// 	if roomID == "" || startTime == "" || endTime == "" {
// 		return c.JSON(http.StatusBadRequest, echo.Map{"message": "missing required parameters"})
// 	}

// 	//unauthorized
// 	if userID == "" || name == "" || phoneNumber == "" || company == "" {
// 		return c.JSON(http.StatusUnauthorized, echo.Map{"message": "unauthorized"})
// 	}

// 	// --- Konversi angka dan waktu ---
// 	roomIDInt, err := strconv.Atoi(roomID)
// 	if err != nil {
// 		return c.JSON(http.StatusBadRequest, echo.Map{"message": "invalid room_id"})
// 	}

// 	snackIDInt := 0
// 	if snackID != "" {
// 		snackIDInt, err = strconv.Atoi(snackID)
// 		if err != nil {
// 			return c.JSON(http.StatusBadRequest, echo.Map{"message": "invalid snack_id"})
// 		}
// 	}

// 	participantInt, err := strconv.Atoi(participant)
// 	if err != nil {
// 		return c.JSON(http.StatusBadRequest, echo.Map{"message": "invalid participant"})
// 	}

// 	start, err := time.Parse(time.RFC3339, startTime)
// 	if err != nil {
// 		return c.JSON(http.StatusBadRequest, echo.Map{"message": "invalid startTime format (use RFC3339)"})
// 	}

// 	end, err := time.Parse(time.RFC3339, endTime)
// 	if err != nil {
// 		return c.JSON(http.StatusBadRequest, echo.Map{"message": "invalid endTime format (use RFC3339)"})
// 	}

// 	if !end.After(start) {
// 		return c.JSON(http.StatusBadRequest, echo.Map{"message": "endTime must be after startTime"})
// 	}

// 	durationMinutes := int(end.Sub(start).Minutes())

// 	// ambil

// }

// GetReservationByID godoc
// @Summary Detail reservation by ID
// @Description Get full reservation detail (master + reservation details) by reservation ID
// @Tags Reservation
// @Produce json
// @Param id path int true "Reservation ID"
// @Success 200 {object} ReservationByIDResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /reservation/{id} [get]
func GetReservationByID(c echo.Context) error {
	// auth check removed so endpoint is public

	idParam := c.Param("id")
	if idParam == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "invalid reservation id"})
	}
	id, err := strconv.Atoi(idParam)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "invalid reservation id"})
	}

	var contactName, contactPhone, contactCompany sql.NullString
	var subtotalSnack, subtotalRoom, total sql.NullFloat64
	var status sql.NullString

	err = db.QueryRow(`
        SELECT contact_name, contact_phone, contact_company, 
               COALESCE(subtotal_snack, 0) as subtotal_snack, 
               COALESCE(subtotal_room, 0) as subtotal_room, 
               COALESCE(total, 0) as total,
               COALESCE(status_reservation::text, '') as status_reservation
        FROM reservations
        WHERE id = $1
    `, id).Scan(&contactName, &contactPhone, &contactCompany, &subtotalSnack, &subtotalRoom, &total, &status)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.JSON(http.StatusNotFound, echo.Map{"message": "url not found"})
		}
		log.Println("GetReservationByID master query:", err)
		return c.JSON(http.StatusInternalServerError, echo.Map{"message": "internal server error"})
	}

	rows, err := db.Query(`
        SELECT 
            COALESCE(r.name, '') as room_name,
            COALESCE(r.price_per_hour, 0) as price_per_hour,
            COALESCE(r.picture_url, '') as image_url,
            COALESCE(r.capacity, 0) as capacity,
            COALESCE(r.room_type::text, 'small') as room_type,
            COALESCE(rd.total_snack, 0) as total_snack,
            COALESCE(rd.total_room, 0) as total_room,
            rd.start_at,
            rd.end_at,
            COALESCE(rd.duration_minute, 0) as duration,
            COALESCE(rd.total_participants, 0) as participant,
            s.id as snack_id,
			COALESCE(s.name, '') as snack_name,
            COALESCE(s.unit::text, '') as snack_unit,
            COALESCE(s.price, 0) as snack_price,
            COALESCE(s.category::text, '') as snack_category
        FROM reservation_details rd
        LEFT JOIN rooms r ON rd.room_id = r.id
        LEFT JOIN snacks s ON rd.snack_id = s.id
        WHERE rd.reservation_id = $1
    `, id)
	if err != nil {
		log.Println("GetReservationByID details query:", err)
		return c.JSON(http.StatusInternalServerError, echo.Map{"message": "internal server error"})
	}
	defer rows.Close()

	rooms := make([]RoomInfo, 0)
	for rows.Next() {
		var room RoomInfo
		var snack SnackInfo
		var startAt, endAt sql.NullTime

		err := rows.Scan(
			&room.Name, &room.PricePerHour, &room.ImageURL, &room.Capacity, &room.Type,
			&room.TotalSnack, &room.TotalRoom, &startAt, &endAt, &room.Duration, &room.Participant,
			&snack.ID, &snack.Name, &snack.Unit, &snack.Price, &snack.Category,
		)
		if err != nil {
			log.Println("Scan error:", err)
			return c.JSON(http.StatusInternalServerError, echo.Map{"message": "internal server error"})
		}

		if startAt.Valid {
			room.StartTime = startAt.Time.Format(time.RFC3339)
		}
		if endAt.Valid {
			room.EndTime = endAt.Time.Format(time.RFC3339)
		}

		if snack.ID > 0 {
			room.Snack = &snack
		}

		rooms = append(rooms, room)
	}

	log.Println("GetReservationByID rooms returned:", len(rooms))

	response := ReservationByIDResponse{
		Message: "success",
		Data: ReservationByIDData{
			Rooms: rooms,
			PersonalData: PersonalData{
				Name:        contactName.String,
				PhoneNumber: contactPhone.String,
				Company:     contactCompany.String,
			},
			SubTotalSnack: subtotalSnack.Float64,
			SubTotalRoom:  subtotalRoom.Float64,
			Total:         total.Float64,
			Status:        status.String,
		},
	}

	response.Data.Status = status.String

	return c.JSON(http.StatusOK, response)
}

// Add this response types
type ReservationByIDData struct {
	Rooms         []RoomInfo   `json:"rooms"`
	PersonalData  PersonalData `json:"personalData"`
	SubTotalSnack float64      `json:"subTotalSnack"`
	SubTotalRoom  float64      `json:"subTotalRoom"`
	Total         float64      `json:"total"`
	Status        string       `json:"status"`
}

type ReservationByIDResponse struct {
	Message string              `json:"message"`
	Data    ReservationByIDData `json:"data"`
}

// UpdateReservationStatus godoc
// @Summary Update reservation status
// @Description Update status of a reservation (booked/canceled/paid)
// @Tags Reservation
// @Accept json
// @Produce json
// @Param request body UpdateReservationStatusRequest true "Status update request"
// @Success 200 {object} map[string]string "message: update status success"
// @Failure 400 {object} map[string]string "message: bad request/reservation already canceled/paid"
// @Failure 401 {object} map[string]string "message: unauthorized"
// @Failure 404 {object} map[string]string "message: url not found"
// @Failure 500 {object} map[string]string "message: internal server error"
// @Router /reservation/status [post]
func UpdateReservationStatus(c echo.Context) error {
	var req UpdateReservationRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, SimpleMessageResponse{Message: "invalid request format"})
	}
	req.Status = strings.TrimSpace(req.Status)
	if req.Status == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "bad request"})
	}
	if req.Status != "booked" && req.Status != "cancel" && req.Status != "paid" {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "bad request"})
	}
	if req.ReservationID == 0 {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "bad request"})
	}

	var currentStatus sql.NullString
	err := db.QueryRow(`SELECT status_reservation FROM reservations WHERE id=$1`, req.ReservationID).Scan(&currentStatus)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.JSON(http.StatusNotFound, echo.Map{"message": "url not found"})
		}
		log.Println("UpdateReservationStatus select error:", err)
		return c.JSON(http.StatusInternalServerError, echo.Map{"message": "internal server error"})
	}

	// Aturan perubahan status yang diizinkan:
	// - booked -> paid/cancel
	// - paid -> cancel
	// - cancel -> tidak bisa diubah
	if currentStatus.Valid {
		switch currentStatus.String {
		case "booked":
			// dari booked bisa ke paid atau cancel
			if req.Status != "paid" && req.Status != "cancel" {
				return c.JSON(http.StatusBadRequest, echo.Map{
					"message": "from booked status can only change to paid or cancel",
				})
			}
		case "paid":
			// dari paid hanya bisa ke cancel
			if req.Status != "cancel" {
				return c.JSON(http.StatusBadRequest, echo.Map{
					"message": "from paid status can only change to cancel",
				})
			}
		case "cancel":
			// status cancel tidak bisa diubah
			return c.JSON(http.StatusBadRequest, echo.Map{
				"message": "canceled reservation cannot be changed",
			})
		}

		if currentStatus.String == req.Status {
			return c.JSON(http.StatusBadRequest, echo.Map{
				"message": "new status must be different from current status",
			})
		}
	}

	_, err = db.Exec(`UPDATE reservations SET status_reservation=$1::status_reservation WHERE id=$2`,
		req.Status, req.ReservationID)
	if err != nil {
		log.Println("UpdateReservationStatus update error:", err)
		return c.JSON(http.StatusInternalServerError, echo.Map{"message": "internal server error"})
	}

	return c.JSON(http.StatusOK, echo.Map{"message": "update status success"})
}

// Add this handler function
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
// @Success 200 {object} ScheduleResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /reservations/schedules [get]
func GetReservationSchedules(c echo.Context) error {
	// Parse date parameters
	startDate := c.QueryParam("startDate")
	endDate := c.QueryParam("endDate")

	if startDate == "" || endDate == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"message": "start date and end date are required",
		})
	}

	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"message": "invalid start date format, use YYYY-MM-DD",
		})
	}

	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"message": "invalid end date format, use YYYY-MM-DD",
		})
	}

	if start.After(end) {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"message": "start date must be before end date",
		})
	}

	// Parse pagination parameters
	page, _ := strconv.Atoi(c.QueryParam("page"))
	pageSize, _ := strconv.Atoi(c.QueryParam("pageSize"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}

	offset := (page - 1) * pageSize

	// Get total count
	var totalData int
	countQuery := `
        SELECT COUNT(DISTINCT rd.room_id)
        FROM reservation_details rd
        JOIN reservations r ON rd.reservation_id = r.id
        WHERE DATE(rd.start_at) BETWEEN $1 AND $2
    `
	err = db.QueryRow(countQuery, start, end).Scan(&totalData)
	if err != nil {
		log.Println("Count query error:", err)
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"message": "internal server error",
		})
	}

	// Get schedules
	query := `
        WITH RoomReservations AS (
            SELECT DISTINCT rd.room_id
            FROM reservation_details rd
            WHERE DATE(rd.start_at) BETWEEN $1 AND $2
            LIMIT $3 OFFSET $4
        )
        SELECT 
            r.id,
            r.name AS room_name,
            res.contact_company,
            rd.start_at,
            rd.end_at,
            CASE
                WHEN rd.end_at < NOW() THEN 'Done'
                WHEN rd.start_at <= NOW() AND rd.end_at >= NOW() THEN 'In Progress'
                ELSE 'Up Coming'
            END as status
        FROM RoomReservations rr
        JOIN rooms r ON rr.room_id = r.id
        LEFT JOIN reservation_details rd ON r.id = rd.room_id
        LEFT JOIN reservations res ON rd.reservation_id = res.id
        WHERE DATE(rd.start_at) BETWEEN $1 AND $2
        ORDER BY r.id, rd.start_at
    `

	rows, err := db.Query(query, start, end, pageSize, offset)
	if err != nil {
		log.Println("Schedule query error:", err)
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"message": "internal server error",
		})
	}
	defer rows.Close()

	scheduleMap := make(map[string]*RoomScheduleInfo)
	for rows.Next() {
		var (
			roomID, roomName   string
			companyName        sql.NullString
			startTime, endTime time.Time
			status             string
		)

		err := rows.Scan(&roomID, &roomName, &companyName, &startTime, &endTime, &status)
		if err != nil {
			log.Println("Row scan error:", err)
			return c.JSON(http.StatusInternalServerError, echo.Map{
				"message": "internal server error",
			})
		}

		if _, exists := scheduleMap[roomID]; !exists {
			scheduleMap[roomID] = &RoomScheduleInfo{
				ID:          roomID,
				RoomName:    roomName,
				CompanyName: companyName.String,
				Schedules:   make([]Schedule, 0),
			}
		}

		scheduleMap[roomID].Schedules = append(scheduleMap[roomID].Schedules, Schedule{
			StartTime: startTime.Format(time.RFC3339),
			EndTime:   endTime.Format(time.RFC3339),
			Status:    status,
		})
	}

	// Convert map to slice
	schedules := make([]RoomScheduleInfo, 0, len(scheduleMap))
	for _, schedule := range scheduleMap {
		schedules = append(schedules, *schedule)
	}

	totalPages := (totalData + pageSize - 1) / pageSize

	response := ScheduleResponse{
		Message:   "success",
		Data:      schedules,
		Page:      page,
		PageSize:  pageSize,
		TotalPage: totalPages,
		TotalData: totalData,
	}

	return c.JSON(http.StatusOK, response)
}

// Add this handler function
// GetDashboard godoc
// @Summary Get dashboard analytics
// @Description Get analytics data for paid transactions within date range
// @Tags Dashboard
// @Accept json
// @Produce json
// @Param startDate query string true "Start date (YYYY-MM-DD)"
// @Param endDate query string true "End date (YYYY-MM-DD)"
// @Success 200 {object} DashboardResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /dashboard [get]
func GetDashboard(c echo.Context) error {
	// Parse date parameters
	startDate := c.QueryParam("startDate")
	endDate := c.QueryParam("endDate")

	if startDate == "" || endDate == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"message": "start date and end date are required",
		})
	}

	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"message": "invalid start date format, use YYYY-MM-DD",
		})
	}

	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"message": "invalid end date format, use YYYY-MM-DD",
		})
	}

	if start.After(end) {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"message": "start date must be smaller than end date",
		})
	}

	// Get total rooms
	var totalRoom int
	err = db.QueryRow(`SELECT COUNT(*) FROM rooms`).Scan(&totalRoom)
	if err != nil {
		log.Println("Total rooms query error:", err)
		return c.JSON(http.StatusInternalServerError, echo.Map{"message": "internal server error"})
	}

	// Get total visitors and reservations for paid transactions
	var totalVisitor, totalReservation int
	var totalOmzet float64
	err = db.QueryRow(`
        SELECT 
            COALESCE(SUM(rd.total_participants), 0) as total_visitors,
            COUNT(DISTINCT r.id) as total_reservations,
            COALESCE(SUM(r.total), 0) as total_omzet
        FROM reservations r
        JOIN reservation_details rd ON r.id = rd.reservation_id
        WHERE r.status_reservation = 'paid'
        AND DATE(r.created_at) BETWEEN $1 AND $2
    `, start, end).Scan(&totalVisitor, &totalReservation, &totalOmzet)
	if err != nil {
		log.Println("Totals query error:", err)
		return c.JSON(http.StatusInternalServerError, echo.Map{"message": "internal server error"})
	}

	// Get room-specific stats
	rows, err := db.Query(`
        WITH RoomStats AS (
            SELECT 
                r.id,
                r.name,
                COALESCE(SUM(res.total), 0) as omzet,
                COUNT(DISTINCT res.id) as reservation_count
            FROM rooms r
            LEFT JOIN reservation_details rd ON r.id = rd.room_id
            LEFT JOIN reservations res ON rd.reservation_id = res.id
                AND res.status_reservation = 'paid'
                AND DATE(res.created_at) BETWEEN $1 AND $2
            GROUP BY r.id, r.name
        )
        SELECT 
            id,
            name,
            omzet,
            CASE 
                WHEN $3 = 0 THEN 0
                ELSE (reservation_count::float / $3::float) * 100
            END as percentage_of_usage
        FROM RoomStats
        ORDER BY omzet DESC
    `, start, end, totalReservation)
	if err != nil {
		log.Println("Room stats query error:", err)
		return c.JSON(http.StatusInternalServerError, echo.Map{"message": "internal server error"})
	}
	defer rows.Close()

	var rooms []DashboardRoom
	for rows.Next() {
		var room DashboardRoom
		err := rows.Scan(&room.ID, &room.Name, &room.Omzet, &room.PercentageOfUsage)
		if err != nil {
			log.Println("Room stats scan error:", err)
			return c.JSON(http.StatusInternalServerError, echo.Map{"message": "internal server error"})
		}
		rooms = append(rooms, room)
	}

	response := DashboardResponse{
		Message: "get dashboard data success",
	}
	response.Data.TotalRoom = totalRoom
	response.Data.TotalVisitor = totalVisitor
	response.Data.TotalReservation = totalReservation
	response.Data.TotalOmzet = totalOmzet
	response.Data.Rooms = rooms

	return c.JSON(http.StatusOK, response)
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
func GetRoomReservationSchedule(c echo.Context) error {
	// Get room ID from path parameter
	roomID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"message": "invalid room id",
		})
	}

	// Get date filter from query parameter
	dateStr := c.QueryParam("date")
	var dateFilter time.Time
	if dateStr != "" {
		dateFilter, err = time.Parse("2006-01-02", dateStr)
		if err != nil {
			return c.JSON(http.StatusBadRequest, echo.Map{
				"message": "invalid date format, use YYYY-MM-DD",
			})
		}
	} else {
		dateFilter = time.Now()
	}

	// Check if room exists
	var roomExists bool
	err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM rooms WHERE id = $1)", roomID).Scan(&roomExists)
	if err != nil {
		log.Println("Room check error:", err)
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"message": "internal server error",
		})
	}
	if !roomExists {
		return c.JSON(http.StatusNotFound, echo.Map{
			"message": "room not found",
		})
	}

	// Query reservations for the room
	query := `
        SELECT 
            rd.id,
            rd.start_at,
            rd.end_at,
            r.status_reservation,
            rd.total_participants
        FROM reservation_details rd
        JOIN reservations r ON rd.reservation_id = r.id
        WHERE rd.room_id = $1
        AND DATE(rd.start_at) = DATE($2)
        ORDER BY rd.start_at ASC
    `

	rows, err := db.Query(query, roomID, dateFilter)
	if err != nil {
		log.Println("Query error:", err)
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"message": "internal server error",
		})
	}
	defer rows.Close()

	schedules := []RoomSchedule{}
	for rows.Next() {
		var schedule RoomSchedule
		err := rows.Scan(
			&schedule.ID,
			&schedule.StartTime,
			&schedule.EndTime,
			&schedule.Status,
			&schedule.TotalParticipant,
		)
		if err != nil {
			log.Println("Row scan error:", err)
			return c.JSON(http.StatusInternalServerError, echo.Map{
				"message": "internal server error",
			})
		}
		schedules = append(schedules, schedule)
	}

	// Get room details
	var room Room
	var updatedAt sql.NullTime // <-- tambahkan deklarasi ini

	err = db.QueryRow(`
        SELECT id, name, room_type, capacity, price_per_hour, picture_url, created_at, updated_at 
        FROM rooms 
        WHERE id = $1
    `, roomID).Scan(
		&room.ID, &room.Name, &room.RoomType, &room.Capacity,
		&room.PricePerHour, &room.PictureURL, &room.CreatedAt, &updatedAt,
	)
	if err != nil {
		log.Println("Room details query error:", err)
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"message": "internal server error",
		})
	}
	if updatedAt.Valid {
		room.UpdatedAt = updatedAt.Time
	} else {
		room.UpdatedAt = room.CreatedAt
	}

	return c.JSON(http.StatusOK, echo.Map{
		"message": "success",
		"data": echo.Map{
			"room":      room,
			"schedules": schedules,
			"date":      dateFilter.Format("2006-01-02"),
		},
	})
}

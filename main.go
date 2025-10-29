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

type RoomReservationRequest struct {
	ID          int       `json:"id"`
	StartTime   time.Time `json:"startTime"`
	EndTime     time.Time `json:"endTime"`
	Participant int       `json:"participant"`
	SnackID     int       `json:"snackID"`
}

type ReservationRequestBody struct {
	UserID      int                      `json:"userID"`
	Name        string                   `json:"name"`
	PhoneNumber string                   `json:"phoneNumber"`
	Company     string                   `json:"company"`
	Notes       string                   `json:"notes"`
	Rooms       []RoomReservationRequest `json:"rooms"`
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

	// route for rooms
	e.POST("/rooms", CreateRoom)
	e.GET("/rooms", GetRooms)
	e.GET("/rooms/:id", GetRoomByID)
	e.PUT("/rooms/:id", UpdateRoom)
	e.DELETE("/rooms/:id", DeleteRoom)
	e.GET("/snacks", GetSnacks)

	// route for reservations
	e.GET("/reservation/calculation", CalculateReservation)
	e.POST("/reservation", CreateReservation)

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

	//

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
	//ambil query data dari parameter request URL
	roomID, _ := strconv.Atoi(c.QueryParam("room_id"))
	snackID, _ := strconv.Atoi(c.QueryParam("snack_id"))
	startTimeStr := c.QueryParam("startTime")
	endTimeStr := c.QueryParam("endTime")
	participant, _ := strconv.Atoi(c.QueryParam("participant"))
	userID := c.QueryParam("user_id")
	name := c.QueryParam("name")
	phoneNumber := c.QueryParam("phoneNumber")
	company := c.QueryParam("company")

	// --- Validasi awal ---
	if roomID == 0 || startTimeStr == "" || endTimeStr == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"message": "missing required parameters",
		})
	}

	startTime, err := time.Parse(time.RFC3339, startTimeStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"message": "invalid startTime format (must be RFC3339)",
		})
	}
	endTime, err := time.Parse(time.RFC3339, endTimeStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"message": "invalid endTime format (must be RFC3339)",
		})
	}

	// --- Ambil data room ---
	var (
		roomName     string
		roomType     string
		roomCapacity int
		pricePerHour float64
		roomImageURL sql.NullString
	)

	err = db.QueryRow(`
		SELECT name, room_type, capacity, price_per_hour, picture_url
		FROM rooms WHERE id = $1
	`, roomID).Scan(&roomName, &roomType, &roomCapacity, &pricePerHour, &roomImageURL)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"message": "room not found"})
	}

	// --- Ambil data snack ---
	var (
		snackName     string
		snackUnit     string
		snackPrice    float64
		snackCategory string
	)

	err = db.QueryRow(`
		SELECT name, unit, price, category
		FROM snacks WHERE id = $1
	`, snackID).Scan(&snackName, &snackUnit, &snackPrice, &snackCategory)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"message": "snack not found"})
	}

	// --- Cek booking bentrok ---
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
		return c.JSON(http.StatusBadRequest, map[string]string{
			"message": "booking bentrok",
		})
	}

	// hitung durasi dan total penginapan (minutes)
	durationMinutes := int(endTime.Sub(startTime).Minutes())
	durationHours := float64(durationMinutes) / 60.0

	subTotalRoom := pricePerHour * durationHours
	subTotalSnack := snackPrice * float64(participant)
	total := subTotalRoom + subTotalSnack

	// --- Simpan ke tabel reservations ---
	query := `
		INSERT INTO reservations (
			user_id, contact_name, contact_phone, contact_company,
			duration_minute, total_participants,
			subtotal_snack, subtotal_room, total,
			status_reservation, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, 'booked', NOW(), NOW())
		RETURNING id
		`

	var reservationID int
	err = db.QueryRow(
		query,
		userID, name, phoneNumber, company,
		durationMinutes, participant,
		subTotalSnack, subTotalRoom, total,
	).Scan(&reservationID)
	if err != nil {
		log.Println("Error inserting reservation:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"message": "failed to save reservation",
		})
	}

	// ---  response ---
	response := map[string]interface{}{
		"message": "success",
		"data": map[string]interface{}{
			"rooms": []map[string]interface{}{
				{
					"name":          roomName,
					"pricePerHour":  pricePerHour,
					"imageURL":      roomImageURL.String,
					"capacity":      roomCapacity,
					"type":          roomType,
					"subTotalSnack": subTotalSnack,
					"subTotalRoom":  subTotalRoom,
					"startTime":     startTime,
					"endTime":       endTime,
					"duration":      durationMinutes,
					"participant":   participant,
					"snack": map[string]interface{}{
						"id":       snackID,
						"name":     snackName,
						"unit":     snackUnit,
						"price":    snackPrice,
						"category": snackCategory,
					},
				},
			},
			"personalData": map[string]interface{}{
				"name":        name,
				"phoneNumber": phoneNumber,
				"company":     company,
			},
			"subTotalRoom":  subTotalRoom,
			"subTotalSnack": subTotalSnack,
			"total":         total,
		},
	}

	return c.JSON(http.StatusOK, response)
}

// (PUT /reservation)
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

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "invalid request format"})
	}

	// validasi
	if req.UserID <= 0 || req.Name == "" || req.PhoneNumber == "" || req.Company == "" || req.Notes == "" || len(req.Rooms) == 0 {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "invalid request format"})
	}

	// --- Cek booking bentrok ---
	for _, room := range req.Rooms {
		var existing int
		err := db.QueryRow(`
			SELECT COUNT(*)
			FROM reservation_details
			WHERE room_id = $1
			AND (
				(start_at, end_at) OVERLAPS ($2, $3)
			)
		`, room.ID, room.StartTime, room.EndTime).Scan(&existing)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"message": "internal server error 1"})
		}
		if existing > 0 {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"message": "Room has been booked",
			})
		}
	}

	// --- Simpan ke tabel reservations ---
	query := `
		INSERT INTO reservations (
			user_id, contact_name, contact_phone, contact_company,
			note, status_reservation, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, 'booked', NOW(), NOW())
		RETURNING id
		`

	var reservationID int
	err := db.QueryRow(
		query,
		req.UserID, req.Name, req.PhoneNumber, req.Company,
		req.Notes,
	).Scan(&reservationID)
	if err != nil {
		log.Println("Error inserting reservation:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"message": "Internal server error 2",
		})
	}

	// ambil nilai data dari tabel room berdasrakan Rooms.ID
	query = `
		Select id, name, room_type, capacity, price_per_hour, image_url, created_at, updated_at
		FROM rooms
		WHERE id = $1
		`

	var roomTable Room
	err = db.QueryRow(query, strconv.Itoa(req.Rooms[0].ID)).Scan(
		&roomTable.ID,
		&roomTable.Name,
		&roomTable.RoomType,
		&roomTable.Capacity,
		&roomTable.PricePerHour,
		&roomTable.PictureURL,
		&roomTable.CreatedAt,
		&roomTable.UpdatedAt,
	)

	// ambil nilai data dari tabel snack berdasrakan Rooms.SnackID
	query = `
		Select id, name, unit, price, category
		FROM snacks
		WHERE id = $1
		`

	var snackTable Snack
	err = db.QueryRow(query, strconv.Itoa(req.Rooms[0].SnackID)).Scan(

		&snackTable.ID,
		&snackTable.Name,
		&snackTable.Unit,
		&snackTable.Price,
		&snackTable.Category,
	)

	//debug query

	//simpan data ke ReservationDetail
	//--- Simpan ke tabel reservation_details
	query = `
		INSERT INTO reservation_details (
			reservation_id,
			room_id, room_name, room_price,
			snack_id, snack_name, snack_price,
			start_at, end_at, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())
		`

	for _, room := range req.Rooms {
		_, err := db.Exec(
			query,
			reservationID,
			room.ID, roomTable.Name, roomTable.PricePerHour,
			room.SnackID, snackTable.Name, snackTable.Price,
			room.StartTime, room.EndTime,
		)
		if err != nil {
			log.Println("Error inserting reservation details:", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"message": "Internal server error 3",
			})
		}
	}

	//masukan req ke variabel output untuk dimasukan ke return
	// return c.JSON(http.StatusOK, map[string]interface{}{
	// 	"message": "Reservation created successfully",
	// 	"req":     req,
	// })

	snackid := strconv.Itoa(req.Rooms[0].SnackID) //.req.Rooms[0].SnackID
	tableid := strconv.Itoa(req.Rooms[0].ID)      //req.Rooms[0].ID

	return c.JSON(http.StatusOK, map[string]string{
		"message":    "Reservation created successfully",
		"roomTable":  tableid,
		"snackTable": snackid,
	})

}

// func HistoryReservation(c echo.Context) error {
// 	startDateStr := c.QueryParam("startDate")
// 	endDateStr := c.QueryParam("endDate")
// 	roomType := c.QueryParam("roomType")
// 	status := c.QueryParam("status")

// 	page, _ := strconv.Atoi(c.QueryParam("page"))
// 	if page < 1 {
// 		page = 1
// 	}

// 	pageSize, _ := strconv.Atoi(c.QueryParam("pageSize"))
// 	if pageSize < 1 {
// 		pageSize = 10
// 	}

// 	offset := (page - 1) * pageSize

// 	// validasi room type
// 	validTypes := map[string]bool{
// 		"small":  true,
// 		"medium": true,
// 		"large":  true,
// 	}
// 	if _, ok := validTypes[roomType]; !ok {
// 		return c.JSON(http.StatusBadRequest, map[string]string{
// 			"message": "Invalid room type",
// 		})
// 	}

// 	// Query utama
// 	query := `
// 		SELECT
// 			reservations.id,
// 			reservations.contact_name,
// 			reservations.contact_phone,
// 			reservations.contact_company,
// 			reservations.subtotal_snack,
// 			reservations.subtotal_room,
// 			reservations.total,
// 			reservations.status_reservation,
// 			reservations.created_at,
// 			reservations.updated_at,
// 		FROM
// 		WHERE 1=1
// 	`
// 	args := []interface{}{}
// 	argIndex := 1

// 	if startDateStr != "" {
// 		query += fmt.Sprintf(" AND reservations.created_at >= $%d", argIndex)
// 		args = append(args, startDateStr)
// 		argIndex++
// 	}

// 	if endDateStr != "" {
// 		query += fmt.Sprintf(" AND reservations.created_at <= $%d", argIndex)
// 		args = append(args, endDateStr)
// 		argIndex++
// 	}

// 	if roomType != "" {
// 		query += fmt.Sprintf(" AND reservations.room_type = $%d", argIndex)
// 		args = append(args, roomType)
// 		argIndex++
// 	}

// 	if status != "" {
// 		query += fmt.Sprintf(" AND reservations.status_reservation = $%d", argIndex)
// 		args = append(args, status)
// 		argIndex++
// 	}

// 	query += " ORDER BY reservations.created_at DESC LIMIT $%d OFFSET $%d"
// 	args = append(args, pageSize, offset)
// 	query = fmt.Sprintf(query, argIndex, argIndex+1)

// 	rows, err := db.Query(query, args...)
// 	if err != nil {
// 		log.Println("DB query error:", err)
// 		return c.JSON(http.StatusInternalServerError, map[string]string{
// 			"message": "Internal server error",
// 		})
// 	}
// 	defer rows.Close()

// 	var histories []map[string]interface{}

// 	for rows.Next() {
// 		var (
// 			id                int
// 			contactName       string
// 			contactPhone      string
// 			contactCompany    string
// 			subtotalSnack     float64
// 			subtotalRoom      float64
// 			total             float64
// 			statusReservation string
// 			createdAt         time.Time
// 			updatedAt         time.Time
// 		)
// 		if err := rows.Scan(
// 			&id,
// 			&contactName,
// 			&contactPhone,
// 			&contactCompany,
// 			&subtotalSnack,
// 			&subtotalRoom,
// 			&total,
// 			&statusReservation,
// 			&createdAt,
// 			&updatedAt,
// 		); err != nil {
// 			return c.JSON(http.StatusInternalServerError, map[string]string{
// 				"message": "Internal server error",
// 			})
// 		}
// 	}

// 	// ambil detail room dari reservation_details
// 	roomQuery := `
// 		SELECT
// 			reservation_details.room_id,
// 			reservation_details.room_price,
// 			reservation_details.room_name,
// 			reservation_details.room_type,
// 			reservation_details.total_room,
// 			reservation_details.total_snack
// 		FROM
// 			reservation_details JOIN reservations ON reservation_details.reservation_id = reservations.id
// 		WHERE
// 			reservations.id = $1
// 	`
// 	roomRows, err := db.Query(roomQuery, id)
// 	if err != nil {
// 		log.Println("DB query error:", err)
// 		return c.JSON(http.StatusInternalServerError, map[string]string{
// 			"message": "Internal server error",
// 		})
// 	}
// 	defer roomRows.Close()

// 	var rooms []map[string]interface{}
// 	for roomRows.Next() {
// 		var (
// 			roomID     int
// 			roomPrice  float64
// 			roomName   string
// 			roomType   string
// 			totalRoom  int
// 			totalSnack int
// 		)
// 		if err := roomRows.Scan(
// 			&roomID,
// 			&roomPrice,
// 			&roomName,
// 			&roomType,
// 			&totalRoom,
// 			&totalSnack,
// 		); err != nil {
// 			return c.JSON(http.StatusInternalServerError, map[string]string{
// 				"message": "Internal server error",
// 			})
// 		}
// 		rooms = append(rooms, map[string]interface{}{
// 			"room_id":     roomID,
// 			"room_price":  roomPrice,
// 			"room_name":   roomName,
// 			"room_type":   roomType,
// 			"total_room":  totalRoom,
// 			"total_snack": totalSnack,
// 		})
// 	}

// 	histories = append(histories, map[string]interface{}{
// 		"id":                 id,
// 		"contact_name":       contactName,
// 		"contact_phone":      contactPhone,
// 		"contact_company":    contactCompany,
// 		"subtotal_snack":     subtotalSnack,
// 		"subtotal_room":      subtotalRoom,
// 		"total":              total,
// 		"status_reservation": statusReservation,
// 		"created_at":         createdAt,
// 		"updated_at":         updatedAt,
// 		"rooms":              rooms,
// 	})

// 	// hitung total data
// 	var totalData int
// 	if err := db.QueryRow("SELECT COUNT(*) FROM reservations").Scan(&totalData); err != nil {
// 		return c.JSON(http.StatusInternalServerError, map[string]string{
// 			"message": "Internal server error",
// 		})
// 	}

// 	return c.JSON(http.StatusOK, map[string]interface{}{
// 		"message":   "Success",
// 		"data":      histories,
// 		"page":      page,
// 		"pageSize":  pageSize,
// 		"totalData": totalData,
// 	})
// }

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

	"BE-E-Meeting/app/entities"
	"BE-E-Meeting/app/handler"
	_ "BE-E-Meeting/docs"

	"github.com/go-playground/validator/v10"
	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
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

type SimpleMessageResponse struct {
	Message string `json:"message"`
}

// var BaseURL string = "http://172.16.148.101:8082"
var BaseURL string = "http://localhost:8080"
var db *sql.DB
var JwtSecret []byte
var DefaultAvatarURL string = BaseURL + "/assets/default/default_profile.jpg"
var DefaultRoomURL string = BaseURL + "/assets/default/default_room.jpg"

// @title E-Meeting API
// @version 1.0
// @description This is a sample server for E-Meeting.
// @termsOfService http://swagger.io/terms/

// @securityDefinitions.apikey  BearerAuth
// @in                          header
// @name                        Authorization
// @description                 Type "Bearer" followed by a space and JWT token.

func main() {
	// Load .env file if it exists (optional for Docker environments)
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Warning: .env file not found, using environment variables from system")
	}

	dbHost := os.Getenv("db_host")
	dbPort, _ := strconv.Atoi(os.Getenv("db_port"))
	dbUser := os.Getenv("db_user")
	dbPassword := os.Getenv("db_password")
	dbName := os.Getenv("db_name")

	db = ConnectDB(dbUser, dbPassword, dbName, dbHost, dbPort)

	// Skip interactive migration prompt in Docker (use separate migrate service)
	skipMigration := os.Getenv("SKIP_MIGRATION")
	if skipMigration != "true" {
		//berikan inputan switch 1 untuk migrate up lalu kembali ke menu, 2 untuk migrate down, 3 untuk continue
		fmt.Println("Enter 1 for migrate up, 2 for migrate down, 3 for continue:")
		var input int
		fmt.Scanln(&input)
		switch input {
		case 1:
			migrateUp(db)
		case 2:
			migrateDown(db)
		}
	}

	JwtSecret = []byte(os.Getenv("jwt_secret"))

	e := echo.New()

	e.Validator = &CustomValdator{validator: validator.New()}

	// route for swagger
	e.GET("/swagger/*", echoSwagger.WrapHandler) //runnning

	//assets
	e.Static("/assets", "./assets")

	// route for login, register, password reset
	e.POST("/login", login)                                                            //running
	e.POST("/register", RegisterUser)                                                  //running
	e.POST("password/reset_request", PasswordReset)                                    //running
	e.PUT("/password/reset/:id", PasswordResetId, roleAuthMiddleware("admin", "user")) //id ini token reset password yang dikirim via email //runnning

	// harus pake auth
	e.POST("/uploads", UploadImage, roleAuthMiddleware("admin", "user")) //runnning

	// route for rooms
	e.POST("/rooms", CreateRoom, roleAuthMiddleware("admin"))                                        // running
	e.GET("/rooms", GetRooms, roleAuthMiddleware("admin", "user"))                                   // running
	e.GET("/rooms/:id", GetRoomByID, roleAuthMiddleware("admin", "user"))                            // running
	e.GET("/rooms/:id/reservation", GetRoomReservationSchedule, roleAuthMiddleware("admin", "user")) // running
	e.PUT("/rooms/:id", UpdateRoom, roleAuthMiddleware("admin"))
	e.DELETE("/rooms/:id", DeleteRoom, roleAuthMiddleware("admin"))

	// route for snacks
	e.GET("/snacks", GetSnacks, roleAuthMiddleware("admin", "user"))

	// route for reservations
	e.GET("/reservation/calculation", CalculateReservation, roleAuthMiddleware("admin", "user"))
	e.POST("/reservation", CreateReservation, roleAuthMiddleware("admin", "user"))
	e.GET("/reservation/history", GetReservationHistory, roleAuthMiddleware("user"))
	e.PUT("/reservation/status", UpdateReservationStatus, roleAuthMiddleware("admin", "user"))
	e.GET("/reservation/:id", GetReservationByID, roleAuthMiddleware("admin", "user"))
	e.GET("/reservations/schedules", GetReservationSchedules, roleAuthMiddleware("admin"))

	// dashboard dan users group tetap menggunakan middlewareAuth
	e.GET("/dashboard", GetDashboard, roleAuthMiddleware("admin"))

	// route users
	e.GET("/users/:id", GetUserByID, roleAuthMiddleware("admin", "user")) //runnning
	e.PUT("/users/:id", UpdateUserByID, roleAuthMiddleware("admin", "user"))

	e.Logger.Fatal(e.Start(":8080"))

}

func migrateUp(db *sql.DB) {
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		log.Fatal(err)
	}
	m, err := migrate.NewWithDatabaseInstance(
		"file://migrations",
		"postgres", driver)
	if err != nil {
		log.Fatal(err)
	}
	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		log.Fatal(err)
	}

	fmt.Println("Migrate up successfully")
}

func migrateDown(db *sql.DB) {
	//cek kalau database dirty

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		log.Fatal(err)
	}
	m, err := migrate.NewWithDatabaseInstance(
		"file://migrations",
		"postgres", driver)
	if err != nil {
		log.Fatal(err)
	}
	err = m.Down()
	if err != nil && err != migrate.ErrNoChange {
		log.Fatal(err)
	}

	fmt.Println("Migrate down successfully")
}

func ConnectDB(username, password, dbname, host string, port int) *sql.DB {
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
// @Summary Login user
// @Description Login user
// @Tags Auth
// @Accept json
// @Produce json
// @Param loginData body entities.Login true "Login data"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /login [post]
func login(c echo.Context) error {
	var loginData entities.Login

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
	claims := &entities.Claims{
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
	claims := &entities.Claims{
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
	claims := &entities.Claims{
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
// @Tags Auth
// @Accept json
// @Produce json
// @Param user body entities.User true "User object"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /register [post]
func RegisterUser(c echo.Context) error {
	var newUser entities.User

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
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Internal Server Error cek"}) //"Password Hashing Failed"
	}

	//insert variable default, Enum status, role, lang
	status := "active"

	//var default avatar
	avatar := DefaultAvatarURL

	// insert to db
	sqlStatement := `INSERT INTO users (username, email, password_hash, name, status, avatar_url) VALUES ($1, $2, $3, $4, $5, $6)`
	_, err = db.Exec(sqlStatement, newUser.Username, newUser.Email, hashedPassword, newUser.Name, status, avatar)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Internal Server Error database"}) //"Database Error"
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
// @Security ApiKeyAuth
// @Summary Reset user password
// @Description Reset user password using a valid reset token
// @Tags User
// @Accept json
// @Produce json
// @Param id path string true "Reset Token"
// @Param user body entities.PasswordConfirmReset true "User object"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /password/reset/{id} [put]
func PasswordResetId(c echo.Context) error {
	id := c.Param("id")
	var passReset entities.PasswordConfirmReset

	// validasi input id
	userID, err := strconv.Atoi(id)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Bad Request"})
	}

	// ambil jwt dari auth header
	userToken := c.Get("user").(*jwt.Token)
	claims := userToken.Claims.(jwt.MapClaims)

	// ambil username dari jwt
	usernameFromToken := claims["username"].(string)

	// ambil username dari db berdasarkan id
	var usernameFromDB string
	err = db.QueryRow("SELECT username FROM users WHERE id = $1", userID).Scan(&usernameFromDB)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Bad Request"})
	}

	// bandingkan jwt dan db
	if usernameFromToken != usernameFromDB {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Bad Request"})
	}

	req := c.Request()
	fmt.Println("HEADERS:", req.Header)

	// proses password
	if err := c.Bind(&passReset); err != nil {
		//error code 400
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "CBad Request"}) //"Invalid Input"
	}

	//validasi apakah new password dan confirm password sama
	if passReset.NewPassword != passReset.ConfirmPassword {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "New password and confirm password do not match"})
	}
	if err := c.Validate(&passReset); err != nil {
		//error code 400
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "DBad Request"}) //"Validation Error"
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
// @Param user body entities.ResetRequest true "User object"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /password/reset [post]
func PasswordReset(c echo.Context) error {
	var resetReq entities.ResetRequest

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
func roleAuthMiddleware(requiredRoles ...string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {

			// Ambil Authorization Header
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return c.JSON(http.StatusUnauthorized, echo.Map{"error": "Unauthorized"})
			}

			if !strings.HasPrefix(authHeader, "Bearer ") {
				return c.JSON(http.StatusUnauthorized, echo.Map{"error": "Invalid Authorization header"})
			}
			tokenString := strings.TrimPrefix(authHeader, "Bearer ")

			// parsing jwt
			token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
				return JwtSecret, nil
			})

			if err != nil || !token.Valid {
				return c.JSON(http.StatusUnauthorized, echo.Map{"error": "Invalid token"})
			}

			// ✅ SIMPAN TOKEN KE CONTEXT
			c.Set("user", token)

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				return c.JSON(http.StatusUnauthorized, echo.Map{"error": "Invalid token claims"})
			}

			fmt.Println("Authenticated user:", claims["username"])
			fmt.Println("Claims:", claims)
			fmt.Println("Claims:", claims["role"])

			// Ambil role dari claims
			var userRoles []string

			// Coba ekstrak sebagai []interface{} (untuk multiple roles)
			if rolesClaimSlice, ok := claims["role"].([]interface{}); ok {
				for _, roleInterface := range rolesClaimSlice {
					if role, ok := roleInterface.(string); ok {
						userRoles = append(userRoles, role)
					}
				}
			} else if roleClaimString, ok := claims["role"].(string); ok {
				// Coba ekstrak sebagai string (untuk single role)
				userRoles = append(userRoles, roleClaimString)
			} else {
				// Jika tidak dapat di-parse sebagai slice atau string, print dan kembalikan Unauthorized
				fmt.Println("Claims[role] tidak dalam format yang diharapkan:", claims["role"])
				// Cek apakah Anda perlu mengembalikan error di sini jika Anda yakin role HARUS ada
				return c.JSON(http.StatusUnauthorized, echo.Map{"error": "Role claim missing or invalid format"})
			}

			// Tambahkan pengecekan jika userRoles kosong setelah ekstraksi
			if len(userRoles) == 0 {
				return c.JSON(http.StatusUnauthorized, echo.Map{"error": "No valid roles found in token"})
			}

			// Cek lagi untuk debug
			fmt.Println("User Roles yang berhasil diekstrak:", userRoles)

			// cek role sesuai dengan role required
			for _, requiredRole := range requiredRoles {
				for _, userRole := range userRoles {
					if requiredRole == userRole {
						// lanjurt ke handler kalau cocok
						return next(c)
					}
				}
			}

			// kalau role ga cocok
			return c.JSON(http.StatusUnauthorized, echo.Map{"error": "Unauthorized"})
		}
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

	// ambil jwt dari auth header
	userToken := c.Get("user").(*jwt.Token)
	claims := userToken.Claims.(jwt.MapClaims)

	// ambil username dari jwt
	usernameFromToken := claims["username"].(string)

	// ambil username dari db berdasarkan id
	var usernameFromDB string
	err = db.QueryRow("SELECT username FROM users WHERE id = $1", idInt).Scan(&usernameFromDB)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Bad Request"})
	}

	// bandingkan jwt dan db
	if usernameFromToken != usernameFromDB {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Bad Request"})
	}

	var user entities.GetUser
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
// @Param user body entities.UpdateUser true "User object"
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

	// ambil jwt dari auth header
	userToken := c.Get("user").(*jwt.Token)
	claims := userToken.Claims.(jwt.MapClaims)

	// ambil username dari jwt
	usernameFromToken := claims["username"].(string)

	// ambil username dari db berdasarkan id
	var usernameFromDB string
	err = db.QueryRow("SELECT username FROM users WHERE id = $1", idInt).Scan(&usernameFromDB)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Bad Request"})
	}

	// bandingkan jwt dan db
	if usernameFromToken != usernameFromDB {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Bad Request"})
	}

	var user entities.UpdateUser
	if err := c.Bind(&user); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid request body"})
	}

	user.Updated_at = time.Now().Format(time.RFC3339)

	// --- Ambil data user saat ini ---
	var currentUser entities.UpdateUser
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
	// cek jika Avatar_url kosong atau default
	avatarResult, err := handler.HandleAvatarUpdate(c, idInt, currentUser.Avatar_url, user.Avatar_url)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Avatar update failed"})
	}

	user.Avatar_url = avatarResult

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
// @Security BearerAuth
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
// @Security BearerAuth
// @Router /rooms [post]
func CreateRoom(c echo.Context) error {
	var req entities.RoomRequest

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "invalid request format"})
	}

	if req.Name == "" || req.Type == "" || req.Capacity <= 0 || req.PricePerHour <= 0 {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "invalid room data"})
	}

	var roomImage string
	var err error

	// proses avatar room
	roomImage, err = handler.HandleRoomImageCreate(c, req.ImageURL, DefaultRoomURL)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"message": "failed processing room image",
		})
	}

	req.ImageURL = roomImage

	// Insert into DB
	query := `
        INSERT INTO rooms (name, room_type, capacity, price_per_hour, picture_url, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
    `
	_, err = db.Exec(query, req.Name, req.Type, req.Capacity, req.PricePerHour, req.ImageURL)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"message": "internal server error"})
	}

	return c.JSON(http.StatusCreated, echo.Map{
		"message":  "room created successfully",
		"imageURL": req.ImageURL,
	})
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
// @Security BearerAuth
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

	var rooms []entities.Room
	for rows.Next() {
		var r entities.Room
		var createdAt sql.NullTime
		var updatedAt sql.NullTime
		if err := rows.Scan(&r.ID, &r.Name, &r.RoomType, &r.Capacity, &r.PricePerHour, &r.PictureURL, &createdAt, &updatedAt); err != nil {
			log.Println("GetRooms scan error:", err)
			return c.JSON(http.StatusInternalServerError, echo.Map{"message": "internal server error"})
		}
		if createdAt.Valid {
			r.CreatedAt = createdAt.Time
		} else {
			r.CreatedAt = time.Time{}
		}
		if updatedAt.Valid {
			r.UpdatedAt = updatedAt.Time
		} else {
			r.UpdatedAt = time.Time{}
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
// @Security BearerAuth
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
	var r entities.Room
	var createdAt sql.NullTime
	var updatedAt sql.NullTime
	err = db.QueryRow(query, id).Scan(
		&r.ID, &r.Name, &r.RoomType, &r.Capacity, &r.PricePerHour,
		&r.PictureURL, &createdAt, &updatedAt,
	)
	if createdAt.Valid {
		r.CreatedAt = createdAt.Time
	} else {
		r.CreatedAt = time.Time{}
	}
	if updatedAt.Valid {
		r.UpdatedAt = updatedAt.Time
	} else {
		r.UpdatedAt = time.Time{}
	}

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
// @Param room body entities.RoomRequest true "Room details"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /rooms/{id} [put]
func UpdateRoom(c echo.Context) error {
	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "invalid room id"})
	}

	var req entities.RoomRequest
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
// @Security BearerAuth
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
// @Security BearerAuth
// @Router /snacks [get]
func GetSnacks(c echo.Context) error {
	rows, err := db.Query(`SELECT id, name, unit, price, category FROM snacks ORDER BY id ASC`)
	if err != nil {
		log.Println("DB query error:", err)
		return c.JSON(http.StatusInternalServerError, echo.Map{"message": "internal server error"})
	}
	defer rows.Close()

	var snacks []entities.Snack
	for rows.Next() {
		var s entities.Snack
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
// @Security BearerAuth
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
	var room entities.Room
	err = db.QueryRow(`
		SELECT id, name, room_type, capacity, price_per_hour, picture_url, created_at, updated_at
		FROM rooms WHERE id = $1
	`, roomID).Scan(&room.ID, &room.Name, &room.RoomType, &room.Capacity, &room.PricePerHour, &room.PictureURL, &room.CreatedAt, &room.UpdatedAt)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"message": "room not found"})
	}

	// Ambil data snack
	var snack entities.Snack
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
	roomDetail := entities.RoomCalculationDetail{
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
		Snack: entities.Snack{
			ID:       snack.ID,
			Name:     snack.Name,
			Unit:     snack.Unit,
			Price:    snack.Price,
			Category: snack.Category,
		},
	}

	response := entities.CalculateReservationResponse{
		Message: "success",
		Data: entities.CalculateReservationData{
			Rooms:         []entities.RoomCalculationDetail{roomDetail},
			PersonalData:  entities.PersonalData{Name: name, PhoneNumber: phoneNumber, Company: company},
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
// @Param request body entities.ReservationRequestBody true "Reservation request body"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /reservation [post]
func CreateReservation(c echo.Context) error {
	var req entities.ReservationRequestBody

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
		var roomTable entities.Room
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

		var snackTable entities.Snack
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
// @Security BearerAuth
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

	var histories []entities.ReservationHistoryData
	for rows.Next() {
		var h entities.ReservationHistoryData
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
			var r entities.ReservationHistoryRoomDetail
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
	return c.JSON(http.StatusOK, entities.ReservationHistoryResponse{
		Message:   "Reservation history fetched successfully",
		Data:      histories,
		Page:      page,
		PageSize:  pageSize,
		TotalPage: totalPage,
		TotalData: totalData,
	})
}

// GetReservationByID godoc
// @Summary Detail reservation by ID
// @Description Get full reservation detail (master + reservation details) by reservation ID
// @Tags Reservation
// @Produce json
// @Param id path int true "Reservation ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Security BearerAuth
// @Router /reservation/{id} [get]
func GetReservationByID(c echo.Context) error {
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
			COALESCE(subtotal_room, 0) as subtotal_room, 
			COALESCE(subtotal_snack, 0) as subtotal_snack, 
			COALESCE(total, 0) as total,
			COALESCE(status_reservation::text, '') as status_reservation
		FROM reservations
		WHERE id = $1
	`, id).Scan(&contactName, &contactPhone, &contactCompany, &subtotalRoom, &subtotalSnack, &total, &status)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.JSON(http.StatusNotFound, echo.Map{"message": "url not found"})
		}
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
		return c.JSON(http.StatusInternalServerError, echo.Map{"message": "internal server error"})
	}
	defer rows.Close()

	rooms := make([]entities.RoomInfo, 0)
	for rows.Next() {
		var room entities.RoomInfo
		var snack entities.Snack
		var startAt, endAt sql.NullTime
		err := rows.Scan(
			&room.Name, &room.PricePerHour, &room.ImageURL, &room.Capacity, &room.Type,
			&room.TotalSnack, &room.TotalRoom, &startAt, &endAt, &room.Duration, &room.Participant,
			&snack.ID, &snack.Name, &snack.Unit, &snack.Price, &snack.Category,
		)
		if err != nil {
			continue // skip this row, try next
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

	if len(rooms) == 0 {
		return c.JSON(http.StatusOK, entities.ReservationByIDResponse{
			Message: "success",
			Data: entities.ReservationByIDData{
				Rooms: rooms,
				PersonalData: entities.PersonalData{
					Name:        contactName.String,
					PhoneNumber: contactPhone.String,
					Company:     contactCompany.String,
				},
				SubTotalSnack: subtotalSnack.Float64,
				SubTotalRoom:  subtotalRoom.Float64,
				Total:         total.Float64,
				Status:        status.String,
			},
		})
	}

	return c.JSON(http.StatusOK, entities.ReservationByIDResponse{
		Message: "success",
		Data: entities.ReservationByIDData{
			Rooms: rooms,
			PersonalData: entities.PersonalData{
				Name:        contactName.String,
				PhoneNumber: contactPhone.String,
				Company:     contactCompany.String,
			},
			SubTotalSnack: subtotalSnack.Float64,
			SubTotalRoom:  subtotalRoom.Float64,
			Total:         total.Float64,
			Status:        status.String,
		},
	})
}

// UpdateReservationStatus godoc
// @Summary Update reservation status
// @Description Update status of a reservation (booked/canceled/paid)
// @Tags Reservation
// @Accept json
// @Produce json
// @Param reservation body entities.UpdateReservationRequest true "Reservation details"
// @Success 200 {object} map[string]string "message: update status success"
// @Failure 400 {object} map[string]string "message: bad request/reservation already canceled/paid"
// @Failure 401 {object} map[string]string "message: unauthorized"
// @Failure 404 {object} map[string]string "message: url not found"
// @Failure 500 {object} map[string]string "message: internal server error"
// @Security BearerAuth
// @Router /reservation/status [put]
func UpdateReservationStatus(c echo.Context) error {
	var req entities.UpdateReservationRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, SimpleMessageResponse{Message: "invalid request format"})
	}
	req.Status = strings.TrimSpace(req.Status)
	// allow passing reservation id via query param for convenience
	if req.ReservationID == 0 {
		if q := c.QueryParam("reservation_id"); q != "" {
			if id, err := strconv.Atoi(q); err == nil {
				req.ReservationID = id
			}
		}
		// also accept reservation id from path param /reservation/:id/status
		if req.ReservationID == 0 {
			if p := c.Param("id"); p != "" {
				if id, err := strconv.Atoi(p); err == nil {
					req.ReservationID = id
				}
			}
		}
	}
	// If still missing reservation id, try to infer from JWT token in Authorization header.
	// We parse the token locally here (handler-only) to avoid modifying team-owned middleware.
	if req.ReservationID == 0 {
		authHeader := c.Request().Header.Get("Authorization")
		if authHeader != "" {
			tokenStr := strings.TrimSpace(authHeader)
			if strings.HasPrefix(strings.ToLower(tokenStr), "bearer ") {
				tokenStr = strings.TrimSpace(tokenStr[7:])
			}

			// parse token with same secret used by generateAccessToken
			token, err := jwt.ParseWithClaims(tokenStr, &entities.Claims{}, func(t *jwt.Token) (interface{}, error) {
				return []byte(os.Getenv("secret_key")), nil
			})
			if err != nil {
				// token present but invalid
				return c.JSON(http.StatusUnauthorized, echo.Map{"message": "invalid token"})
			}
			if token != nil && token.Valid {
				if claims, ok := token.Claims.(*entities.Claims); ok {
					username := claims.Username
					if username != "" {
						// get user id from username
						var userID int
						if err := db.QueryRow(`SELECT id FROM users WHERE username=$1`, username).Scan(&userID); err == nil {
							// get latest reservation for this user
							var latestID int
							if err := db.QueryRow(`SELECT id FROM reservations WHERE user_id=$1 ORDER BY created_at DESC LIMIT 1`, userID).Scan(&latestID); err == nil {
								req.ReservationID = latestID
							}
						}
					}
				}
			}
		}
	}
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
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
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

	scheduleMap := make(map[string]*entities.RoomScheduleInfo)
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
			scheduleMap[roomID] = &entities.RoomScheduleInfo{
				ID:          roomID,
				RoomName:    roomName,
				CompanyName: companyName.String,
				Schedules:   make([]entities.Schedule, 0),
			}
		}

		scheduleMap[roomID].Schedules = append(scheduleMap[roomID].Schedules, entities.Schedule{
			StartTime: startTime.Format(time.RFC3339),
			EndTime:   endTime.Format(time.RFC3339),
			Status:    status,
		})
	}

	// Convert map to slice
	schedules := make([]entities.RoomScheduleInfo, 0, len(scheduleMap))
	for _, schedule := range scheduleMap {
		schedules = append(schedules, *schedule)
	}

	totalPages := (totalData + pageSize - 1) / pageSize

	response := entities.ScheduleResponse{
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
// @Success 200 {object} map[string]interface{}
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

	var rooms []entities.DashboardRoom
	for rows.Next() {
		var room entities.DashboardRoom
		err := rows.Scan(&room.ID, &room.Name, &room.Omzet, &room.PercentageOfUsage)
		if err != nil {
			log.Println("Room stats scan error:", err)
			return c.JSON(http.StatusInternalServerError, echo.Map{"message": "internal server error"})
		}
		rooms = append(rooms, room)
	}

	response := entities.DashboardResponse{
		Message: "get dashboard data success",
	}
	response.Data.TotalRoom = totalRoom
	response.Data.TotalVisitor = totalVisitor
	response.Data.TotalReservation = totalReservation
	response.Data.TotalOmzet = totalOmzet
	response.Data.Rooms = rooms

	return c.JSON(http.StatusOK, response)
}

// Add this handler function
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
// @Security BearerAuth
// @Router /rooms/{id}/reservation [get]
func GetRoomReservationSchedule(c echo.Context) error {

	// cek user

	// Get room ID from path parameter
	roomID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"message": "invalid room id",
		})
	}

	// Support both date and datetime range filters
	// Prefer start_datetime & end_datetime (RFC3339). If not provided, fall back to date=YYYY-MM-DD
	startDTStr := c.QueryParam("start_datetime")
	endDTStr := c.QueryParam("end_datetime")
	dateStr := c.QueryParam("date")

	var useRange bool
	var startDT, endDT time.Time
	if startDTStr != "" && endDTStr != "" {
		startDT, err = time.Parse(time.RFC3339, startDTStr)
		if err != nil {
			return c.JSON(http.StatusBadRequest, echo.Map{"message": "invalid start_datetime (must be RFC3339)"})
		}
		endDT, err = time.Parse(time.RFC3339, endDTStr)
		if err != nil {
			return c.JSON(http.StatusBadRequest, echo.Map{"message": "invalid end_datetime (must be RFC3339)"})
		}
		useRange = true
	}

	var dateFilter time.Time
	if !useRange {
		if dateStr != "" {
			dateFilter, err = time.Parse("2006-01-02", dateStr)
			if err != nil {
				return c.JSON(http.StatusBadRequest, echo.Map{"message": "invalid date format, use YYYY-MM-DD"})
			}
		} else {
			dateFilter = time.Now()
		}
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
	var rows *sql.Rows
	if useRange {
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
			AND (rd.start_at, rd.end_at) OVERLAPS ($2, $3)
			ORDER BY rd.start_at ASC
		`
		rows, err = db.Query(query, roomID, startDT, endDT)
	} else {
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
		rows, err = db.Query(query, roomID, dateFilter)
	}
	if err != nil {
		log.Println("Query error:", err)
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"message": "internal server error",
		})
	}
	defer rows.Close()

	schedules := []entities.RoomSchedule{}
	for rows.Next() {
		var schedule entities.RoomSchedule
		var startAt sql.NullTime
		var endAt sql.NullTime
		var status sql.NullString
		var participants sql.NullInt64

		err := rows.Scan(
			&schedule.ID,
			&startAt,
			&endAt,
			&status,
			&participants,
		)
		if err != nil {
			log.Println("Row scan error:", err)
			return c.JSON(http.StatusInternalServerError, echo.Map{"message": "internal server error"})
		}
		if startAt.Valid {
			schedule.StartTime = startAt.Time
		}
		if endAt.Valid {
			schedule.EndTime = endAt.Time
		}
		if status.Valid {
			schedule.Status = status.String
		}
		if participants.Valid {
			schedule.TotalParticipant = int(participants.Int64)
		}
		schedules = append(schedules, schedule)
	}

	// Get room details
	var room entities.Room
	var createdAt sql.NullTime
	var updatedAt sql.NullTime
	err = db.QueryRow(`
		SELECT id, name, room_type, capacity, price_per_hour, picture_url, created_at, updated_at 
		FROM rooms 
		WHERE id = $1
	`, roomID).Scan(
		&room.ID, &room.Name, &room.RoomType, &room.Capacity,
		&room.PricePerHour, &room.PictureURL, &createdAt, &updatedAt,
	)
	if err != nil {
		log.Println("Room details query error:", err)
		return c.JSON(http.StatusInternalServerError, echo.Map{"message": "internal server error"})
	}
	if createdAt.Valid {
		room.CreatedAt = createdAt.Time
	}
	if updatedAt.Valid {
		room.UpdatedAt = updatedAt.Time
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

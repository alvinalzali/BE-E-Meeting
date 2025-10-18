package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
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
	Password   string `json:"password" validate:"omitempty"`
	Name       string `json:"name" validate:"omitempty"`
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

var db *sql.DB
var JwtSecret []byte

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

	// route for login, register, password reset
	e.POST("/login", login)
	e.POST("/register", RegisterUser)
	e.POST("/password/reset_request", PasswordReset)
	e.PUT("/password/reset/:id", PasswordResetId) //id ini token reset password yang dikirim via email

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
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
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

	return c.JSON(http.StatusOK, echo.Map{
		"data":    user,
		"message": "User retrieved successfully",
	})
}

// UpdateUserByID godoc
// @Summary Update user by ID
// @Description Update user details by user ID require authentication from header
// @Tags User
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Param user body updateUser true "User object to be updated"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security     BearerAuth
// @Router /users/{id} [put]
func UpdateUserByID(c echo.Context) error {
	id := c.Param("id")
	var userUpdate updateUser

	if err := c.Bind(&userUpdate); err != nil {
		//error code 400
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Bad Request"}) //"Invalid Input"
	}
	if err := c.Validate(&userUpdate); err != nil {
		//error code 400
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Bad Request"}) //"Validation Error"
	}
	// cek password apakah ada angka, huruf besar, huruf kecil, dan simbol
	if userUpdate.Password != "" && !isValidPassword(userUpdate.Password) {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Password must contain at least one uppercase letter, one lowercase letter, one number, and one special character"})
	}

	// hash password jika diupdate
	var hashedPassword string
	var err error
	if userUpdate.Password != "" {
		hashedPassword, err = hashPassword(userUpdate.Password)
		if err != nil {
			// error 500
			return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Internal Server Error"}) //"Password Hashing Failed"
		}
	}

	// update db berdasarkan id dan field yang diupdate saja
	sqlStatement := `UPDATE users SET `
	params := []interface{}{}
	paramCount := 1
	if userUpdate.Password != "" {
		sqlStatement += `password_hash=$` + strconv.Itoa(paramCount) + `, `
		params = append(params, hashedPassword)
		paramCount++
	}
	if userUpdate.Name != "" {
		sqlStatement += `name=$` + strconv.Itoa(paramCount) + `, `
		params = append(params, userUpdate.Name)
		paramCount++
	}
	if userUpdate.Avatar_url != "" {
		sqlStatement += `avatar_url=$` + strconv.Itoa(paramCount) + `, `
		params = append(params, userUpdate.Avatar_url)
		paramCount++
	}
	if userUpdate.Lang != "" {
		sqlStatement += `lang=$` + strconv.Itoa(paramCount) + `, `
		params = append(params, userUpdate.Lang)
		paramCount++
	}
	if userUpdate.Role != "" {
		sqlStatement += `role=$` + strconv.Itoa(paramCount) + `, `
		params = append(params, userUpdate.Role)
		paramCount++
	}
	if userUpdate.Status != "" {
		sqlStatement += `status=$` + strconv.Itoa(paramCount) + `, `
		params = append(params, userUpdate.Status)
		paramCount++
	}
	if userUpdate.Email != "" {
		sqlStatement += `email=$` + strconv.Itoa(paramCount) + `, `
		params = append(params, userUpdate.Email)
		paramCount++
	}
	if userUpdate.Username != "" {
		sqlStatement += `username=$` + strconv.Itoa(paramCount) + `, `
		params = append(params, userUpdate.Username)
		paramCount++
	}
	// remove last comma and space
	if len(params) == 0 {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "No fields to update"})
	}
	sqlStatement = sqlStatement[:len(sqlStatement)-2]
	sqlStatement += ` WHERE id=$` + strconv.Itoa(paramCount)
	params = append(params, id)
	_, err = db.Exec(sqlStatement, params...)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Internal Server Error"}) //"Database Error"
	}
	return c.JSON(http.StatusOK, echo.Map{"message": "User updated successfully"})
}

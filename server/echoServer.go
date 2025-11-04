package server

import (
	"BE-E-MEETING/config"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	echoSwagger "github.com/swaggo/echo-swagger"
)

type echoServer struct {
	app *echo.Echo
	cfg *config.Config
}

type CustomValidator struct {
	validator *validator.Validate
}

func (cv *CustomValidator) Validate(i interface{}) error {
	return cv.validator.Struct(i)
}

func NewEchoServer(cfg *config.Config) Server {
	e := echo.New()
	e.Validator = &CustomValidator{validator: validator.New()}

	// Serve Swagger documentation
	e.GET("/swagger/*", echoSwagger.WrapHandler)

	// Serve static assets
	e.Static("/assets", "./assets")

	return &echoServer{
		app: e,
		cfg: cfg,
	}
}

func (s *echoServer) Start() error {
	return s.app.Start(":8080")
}

func (s *echoServer) GetEcho() *echo.Echo {
	return s.app
}

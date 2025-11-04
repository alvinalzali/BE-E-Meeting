package server

import "github.com/labstack/echo/v4"

type Server interface {
	Start() error
	GetEcho() *echo.Echo
}

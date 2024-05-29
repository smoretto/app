package service

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func Hello(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"hello": "world!",
	})
}

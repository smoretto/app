package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"

	"os"
	"os/signal"
	"time"

	"github.com/labstack/echo-contrib/echoprometheus"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

const application = "app"

var version = "unknown"

var appEnv = os.Getenv("APP_ENV")

func main() {
	opts := &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}

	var handler slog.Handler = slog.NewTextHandler(os.Stdout, opts)
	if appEnv == "production" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}
	logger := slog.New(handler)
	slog.SetDefault(logger)

	slog.Info("application started", "app", application, "version", version)

	app := echo.New()

	app.HideBanner = true
	app.HidePort = true

	app.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		Skipper: func(c echo.Context) bool {
			return c.Path() == "/health"
		},
		LogURI:       true,
		LogStatus:    true,
		LogMethod:    true,
		LogHost:      true,
		LogRemoteIP:  true,
		LogRequestID: true,
		LogLatency:   true,
		LogUserAgent: true,
		LogError:     true,
		HandleError:  true,

		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			if v.Error == nil {
				logger.LogAttrs(context.Background(), slog.LevelInfo, "request",
					slog.String("uri", v.URI),
					slog.Int("status", v.Status),
					slog.String("method", v.Method),
					slog.String("host", v.Host),
					slog.String("remote_ip", v.RemoteIP),
					slog.String("request_id", v.RequestID),
					slog.String("latency", v.Latency.String()),
					slog.String("user_agent", v.UserAgent),
				)
			} else {
				logger.LogAttrs(context.Background(), slog.LevelError, "request_error",
					slog.String("uri", v.URI),
					slog.Int("status", v.Status),
					slog.String("method", v.Method),
					slog.String("host", v.Host),
					slog.String("remote_ip", v.RemoteIP),
					slog.String("request_id", v.RequestID),
					slog.String("latency", v.Latency.String()),
					slog.String("user_agent", v.UserAgent),
					slog.String("err", v.Error.Error()),
				)
			}
			return nil
		},
	}))

	app.Use(middleware.Recover())
	app.Use(middleware.RequestID())
	app.Use(echoprometheus.NewMiddlewareWithConfig(echoprometheus.MiddlewareConfig{
		Subsystem:                 application,
		DoNotUseRequestPathFor404: true,
		Skipper: func(c echo.Context) bool {
			return c.Path() == "/health"
		},
	}))
	app.File("/", "assets/pages/index.html")
	app.GET("/health", healthHandler)
	app.GET("/", versionHandler)

	metrics := echo.New()
	metrics.HideBanner = true
	metrics.HidePort = true
	metrics.GET("/metrics", echoprometheus.NewHandler())

	go func() {
		if err := metrics.Start(":8081"); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()

	go func() {
		if err := app.Start(":8080"); err != nil && err != http.ErrServerClosed {
			log.Fatal("shutting down the server")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := app.Shutdown(ctx); err != nil {
		log.Fatal(err)
	}
	if err := metrics.Shutdown(ctx); err != nil {
		log.Fatal(err)
	}
}

func versionHandler(c echo.Context) error {
	return c.JSONBlob(http.StatusOK, []byte(fmt.Sprintf(`{"application": "%s", "version": "%s"}`, application, version)))
}

func healthHandler(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"alive": true,
	})
}

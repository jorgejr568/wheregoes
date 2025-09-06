package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/jorgejr568/wheregoes/internal/clients"
	"github.com/jorgejr568/wheregoes/internal/dto"
	"github.com/jorgejr568/wheregoes/internal/services"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func checkOrigin(origins []string) func(*http.Request) bool {
	return func(r *http.Request) bool {
		for _, origin := range origins {
			if origin == "*" || origin == r.Header.Get("Origin") {
				return true
			}
		}
		return false
	}
}

func Serve(ctx context.Context, port string) error {
	_, _, err := StartServerWithConfig(ctx, port, allowedOrigins)
	if err != nil {
		return err
	}

	// Wait for context to be cancelled (e.g., by signal)
	<-ctx.Done()
	return nil
}

// StartServerWithConfig starts a server and returns the listener so tests can get the actual port
func StartServerWithConfig(ctx context.Context, port string, origins []string) (net.Listener, *echo.Echo, error) {
	upgrader := websocket.Upgrader{
		CheckOrigin: checkOrigin(origins),
	}

	echoServer := echo.New()
	echoServer.HideBanner = true

	// Setup shutdown handler
	go func() {
		<-ctx.Done()
		log.Println("Shutting down server...")
		if err := echoServer.Shutdown(context.Background()); err != nil {
			log.Printf("Error shutting down server: %v", err)
		}
	}()

	service := services.NewTrackerService(clients.NewHttpFetcherClient())

	echoServer.Use(middleware.Logger())
	echoServer.Use(middleware.Recover())
	echoServer.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: origins,
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodOptions},
	}))

	echoServer.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	echoServer.POST("/tracks", func(c echo.Context) error {
		request := new(dto.TrackRequest)
		if err := c.Bind(request); err != nil {
			return err
		}

		response, err := service.Track(ctx, request.Url)
		if err != nil {
			if errors.Is(err, services.ErrCircularRedirection) {
				return c.JSON(http.StatusConflict, dto.NewTrackErrorResponse(err))
			}
			return err
		}

		return c.JSON(http.StatusOK, response)
	})

	echoServer.GET("/ws", func(c echo.Context) error {
		ws, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
		if err != nil {
			return err
		}
		defer func() {
			ws.Close()
		}()

	wsLoop:
		for {
			_, msg, err := ws.ReadMessage()
			if err != nil {
				if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
					c.Logger().Debug("Client closed connection")
					return nil
				}
				c.Logger().Error(err)
				return err
			}

			c.Logger().Info(fmt.Sprintf("Received message: %s", msg))

			request := new(dto.TrackRequest)
			if err = json.Unmarshal(msg, request); err != nil {
				c.Logger().Error(err)
				err = ws.WriteJSON(dto.NewTrackErrorResponse(err))
				if err != nil {
					c.Logger().Error("Error writing to websocket: ", err)
				}
				continue wsLoop
			}

			trackChannel := service.TrackChannel(ctx, request.Url)
			for response := range trackChannel {
				if response.Err != nil {
					err = ws.WriteJSON(dto.NewTrackErrorResponse(response.Err))
					if err != nil {
						c.Logger().Error("Error writing to websocket: ", err)
					}
					continue wsLoop
				}

				if response.Finished {
					err = ws.WriteJSON(dto.NewTrackFinishResponse())
					if err != nil {
						c.Logger().Error("Error writing to websocket: ", err)
					}

					c.Logger().Info("Finished tracking of", request.Url)
					continue wsLoop
				}

				checkpoint := response.Checkpoint
				err := ws.WriteJSON(dto.NewTrackCheckpointResponse(checkpoint))
				if err != nil {
					c.Logger().Error("Error writing to websocket: ", err)
				}
			}
		}
	})

	// Create listener to get actual port
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return nil, nil, err
	}

	// Start server in background
	go func() {
		log.Println("Starting server on port", port)
		if err := echoServer.Server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("Server error: %v", err)
		}
	}()

	return listener, echoServer, nil
}

var (
	allowedOrigins []string
)

func init() {
	if envAllowedOrigins := os.Getenv("ALLOWED_ORIGINS"); envAllowedOrigins != "" {
		allowedOrigins = strings.Split(envAllowedOrigins, ",")
		return
	}

	allowedOrigins = []string{"*"}
}

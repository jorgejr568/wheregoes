package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/jorgejr568/wheregoes/internal/clients"
	"github.com/jorgejr568/wheregoes/internal/dto"
	"github.com/jorgejr568/wheregoes/internal/services"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"log"
	"net/http"
	"os"
	"strings"
)

var (
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			for _, origin := range allowedOrigins {
				if origin == "*" || origin == r.Header.Get("Origin") {
					return true
				}
			}

			return false
		},
	}
)

func Serve(ctx context.Context) error {
	echoServer := echo.New()
	echoServer.HideBanner = true
	go func() {
		<-ctx.Done()

		log.Println("Shutting down server...")
		if err := echoServer.Shutdown(ctx); err != nil {
			log.Fatal(err)
		}
	}()

	service := services.NewTrackerService(clients.NewHttpFetcherClient())

	echoServer.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: allowedOrigins,
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
			for {
				select {
				case response := <-trackChannel:
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
		}

		return nil
	})

	err := echoServer.Start(":8080")
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
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

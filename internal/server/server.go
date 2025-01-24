package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/jorgejr568/wheregoes/internal/clients"
	"github.com/jorgejr568/wheregoes/internal/services"
	"github.com/labstack/echo/v4"
	"log"
	"net/http"
)

var (
	upgrader = websocket.Upgrader{}
)

type trackRequest struct {
	Url string `json:"url"`
}

type trackErrorResponse struct {
	Error string `json:"error"`
}

type trackFinishResponse struct {
	Finished bool `json:"finished"`
}

func newTrackErrorResponse(err error) trackErrorResponse {
	return trackErrorResponse{Error: err.Error()}
}

func newTrackFinishResponse() trackFinishResponse {
	return trackFinishResponse{Finished: true}
}

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

	echoServer.POST("/tracks", func(c echo.Context) error {
		request := new(trackRequest)
		if err := c.Bind(request); err != nil {
			return err
		}

		response, err := service.Track(ctx, request.Url)
		if err != nil {
			if errors.Is(err, services.ErrCircularRedirection) {
				return c.JSON(http.StatusConflict, newTrackErrorResponse(err))
			}

			return err
		}

		return c.JSON(http.StatusOK, response)
	})

	echoServer.GET("/tracksWs", func(c echo.Context) error {
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

			request := new(trackRequest)
			if err = json.Unmarshal(msg, request); err != nil {
				c.Logger().Error(err)
				continue wsLoop
			}

			trackChannel := service.TrackChannel(ctx, request.Url)
			for {
				select {
				case response := <-trackChannel:
					if response.Err != nil {
						err = ws.WriteJSON(newTrackErrorResponse(response.Err))
						if err != nil {
							c.Logger().Error("Error writing to websocket: ", err)
						}
						continue wsLoop
					}

					if response.Finished {
						err = ws.WriteJSON(newTrackFinishResponse())
						if err != nil {
							c.Logger().Error("Error writing to websocket: ", err)

						}

						c.Logger().Info("Finished tracking of", request.Url)
						continue wsLoop
					}

					checkpoint := response.Checkpoint
					err := ws.WriteJSON(checkpoint)
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

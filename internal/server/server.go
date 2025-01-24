package server

import (
	"context"
	"errors"
	"github.com/jorgejr568/wheregoes/internal/clients"
	"github.com/jorgejr568/wheregoes/internal/services"
	"github.com/labstack/echo/v4"
	"log"
	"net/http"
)

type trackRequest struct {
	Url string `json:"url"`
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
				return c.JSON(http.StatusConflict, map[string]string{"error": err.Error()})
			}

			return err
		}

		return c.JSON(http.StatusOK, response)
	})

	err := echoServer.Start(":8080")
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

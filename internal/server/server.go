package server

import (
	"context"
	"github.com/labstack/echo/v4"
	"log"
	"net/http"
	"wheregoes/internal/clients"
	"wheregoes/internal/services"
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
			return err
		}

		return c.JSON(http.StatusOK, response)
	})

	err := echoServer.Start(":8080")
	if err != nil && err != http.ErrServerClosed {
		return err
	}

	return nil
}

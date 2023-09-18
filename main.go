package main

import (
	"context"
	"os"
	"os/signal"
	"wheregoes/internal/server"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	signalCh := make(chan os.Signal)
	go func() {
		<-signalCh
		cancel()
	}()

	signal.Notify(signalCh, os.Interrupt)

	err := server.Serve(ctx)
	if err != nil {
		panic(err)
	}
}

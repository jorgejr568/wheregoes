package cmd

import (
	"context"
	"github.com/jorgejr568/wheregoes/internal/server"
	"github.com/spf13/cobra"
	"os"
	"os/signal"
)

func serve() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the server",
		Run: func(cmd *cobra.Command, args []string) {
			ctx, cancel := context.WithCancel(cmd.Context())
			signalCh := make(chan os.Signal, 1)
			go func() {
				<-signalCh
				cancel()
			}()

			signal.Notify(signalCh, os.Interrupt)

			port, _ := cmd.Flags().GetString("port")
			err := server.Serve(ctx, port)
			if err != nil {
				panic(err)
			}
		},
	}

	cmd.Flags().StringP("port", "p", "8080", "Port to listen on")
	return cmd
}

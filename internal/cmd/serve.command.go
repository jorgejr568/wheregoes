package cmd

import (
	"context"
	"github.com/spf13/cobra"
	"os"
	"os/signal"
	"wheregoes/internal/server"
)

func serve() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the server",
		Run: func(cmd *cobra.Command, args []string) {
			ctx, cancel := context.WithCancel(cmd.Context())
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
		},
	}

	cmd.Flags().StringP("port", "p", "8080", "Port to listen on")
	return cmd
}

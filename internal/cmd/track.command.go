package cmd

import (
	"fmt"
	"github.com/jorgejr568/wheregoes/internal/clients"
	"github.com/jorgejr568/wheregoes/internal/services"
	"github.com/labstack/gommon/color"
	"github.com/spf13/cobra"
	"log"
	"regexp"
)

func track() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "track [url]",
		Short: "Track a URL",
		Args:  cobra.ExactArgs(1),
		PreRun: func(cmd *cobra.Command, args []string) {
			urlRegex, err := regexp.Compile(`^https?://`)
			if err != nil {
				log.Fatal(err)
			}

			if !urlRegex.MatchString(args[0]) {
				log.Fatal("Invalid URL")
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			service := services.NewTrackerService(
				clients.NewHttpFetcherClient(),
			)

			trackerCh := service.TrackChannel(cmd.Context(), args[0])
			i := 0
			for {
				select {
				case response := <-trackerCh:
					if response.Err != nil {
						log.Fatal(response.Err)
					}

					if response.Finished {
						return
					}

					checkpoint := response.Checkpoint

					fmt.Print(
						color.Yellow(
							fmt.Sprintf("%d ....... %s (%d, %s)\n", i+1, checkpoint.Url, checkpoint.Status, checkpoint.Latency),
						),
					)
					i++
				}
			}
		},
	}

	cmd.Flags().Bool("json", false, "Print response in JSON format")

	return cmd
}

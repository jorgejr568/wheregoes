package cmd

import (
	"fmt"
	"github.com/labstack/gommon/color"
	"github.com/spf13/cobra"
	"log"
	"regexp"
	"wheregoes/internal/clients"
	"wheregoes/internal/services"
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

			response, err := service.Track(cmd.Context(), args[0])
			if err != nil {
				log.Fatal(err)
			}

			print(color.Yellow(fmt.Sprintf("URL: %s\n", response.Url)))
			print(
				color.Green(
					fmt.Sprintf("Final URL: %s\n\n\n\n", response.Checkpoints[len(response.Checkpoints)-1].Url),
				),
			)
			for i, checkpoint := range response.Checkpoints {
				print(
					color.Yellow(
						fmt.Sprintf("%d ....... %s (%d)\n", i+1, checkpoint.Url, checkpoint.Status),
					),
				)
			}

		},
	}

	return cmd
}

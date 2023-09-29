package cmd

import (
	"encoding/json"
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

			response, err := service.Track(cmd.Context(), args[0])
			if err != nil {
				log.Fatal(err)
			}

			if cmd.Flag("json").Value.String() == "true" {
				jsonResponse, err := json.Marshal(response.Checkpoints)
				if err != nil {
					log.Fatal(err)
				}

				fmt.Println(string(jsonResponse))
				return
			}

			print(color.Yellow(fmt.Sprintf("Initial URL: %s\n", args[0])))
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

	cmd.Flags().Bool("json", false, "Print response in JSON format")

	return cmd
}

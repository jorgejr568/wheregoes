package cmd

import (
	"fmt"
	"github.com/labstack/gommon/color"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var TrackCmd = track()
var ServeCmd = serve()

var DefaultCommand = TrackCmd

var RootCmd = &cobra.Command{
	Short: "Wheregoes is a CLI tool to track a URL",
	Long:  "Wheregoes is a CLI tool to track a URL",
	Run: func(cmd *cobra.Command, args []string) {
		if cmd.Flag("version").Value.String() == "true" {
			fmt.Printf(
				color.Green(
					fmt.Sprintf("Version: %s\n", "1.0.0"),
				),
			)
			return
		}

		DefaultCommand.Run(cmd, args)
		return
	},
	PreRun: func(cmd *cobra.Command, args []string) {
	},
	Args: DefaultCommand.Args,
}

func init() {
	RootCmd.AddCommand(TrackCmd)
	RootCmd.AddCommand(ServeCmd)

	RootCmd.Flags().BoolP("version", "v", false, "Print version number")
	DefaultCommand.Flags().VisitAll(func(flag *pflag.Flag) {
		RootCmd.Flags().AddFlag(flag)
	})
}

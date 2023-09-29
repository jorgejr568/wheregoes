package cmd

import (
	"fmt"
	"github.com/labstack/gommon/color"
	"github.com/spf13/cobra"
)

var RootCmd = &cobra.Command{
	Short: "Wheregoes is a CLI tool to track a URL",
	Long:  "Wheregoes is a CLI tool to track a URL",
}

func init() {
	RootCmd.AddCommand(track())
	RootCmd.AddCommand(serve())

	RootCmd.Flags().BoolP("version", "v", false, "Print version number")
	RootCmd.Run = func(cmd *cobra.Command, args []string) {
		print("args", args)
		if cmd.Flag("version").Value.String() == "true" {
			fmt.Printf(
				color.Green(
					fmt.Sprintf("Version: %s\n", "1.0.0"),
				),
			)
			return
		}

		hasArgs := len(args) > 0
		if hasArgs {
			print(color.Red(fmt.Sprintf("Unknown command: %s\n\n", args[0])))
			RootCmd.Commands()[0].Run(cmd, args)
		}

		err := cmd.Help()
		if err != nil {
			return
		}
	}
}

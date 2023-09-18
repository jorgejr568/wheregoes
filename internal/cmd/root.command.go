package cmd

import "github.com/spf13/cobra"

var RootCmd = new(cobra.Command)

func init() {
	RootCmd.AddCommand(serve())
	RootCmd.AddCommand(track())
}

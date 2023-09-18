package main

import "wheregoes/internal/cmd"

func main() {
	err := cmd.RootCmd.Execute()
	if err != nil {
		return
	}
}

package main

import "github.com/jorgejr568/wheregoes/internal/cmd"

func main() {
	err := cmd.RootCmd.Execute()
	if err != nil {
		return
	}
}

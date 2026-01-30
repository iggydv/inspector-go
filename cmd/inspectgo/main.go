package main

import (
	"os"

	"inspectgo/cmd/inspectgo/commands"
)

func main() {
	root := commands.NewRootCommand()
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

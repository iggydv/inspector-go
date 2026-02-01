package main

import (
	"inspectgo/cmd/commands"
	"os"
)

func main() {
	root := commands.NewRootCommand()
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

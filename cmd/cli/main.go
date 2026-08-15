package main

import (
	"os"

	"github.com/nevvesdev/distributed-lock-manager/cmd/cli/commands"
)

func main() {
	if err := commands.NewRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

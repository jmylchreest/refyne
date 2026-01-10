// Package main is the entry point for the refyne CLI.
package main

import (
	"os"

	"github.com/refyne/refyne/cmd/refyne/commands"
)

func main() {
	if err := commands.Execute(); err != nil {
		os.Exit(1)
	}
}

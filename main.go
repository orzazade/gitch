package main

import (
	"os"

	"github.com/orkhanrz/gitch/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

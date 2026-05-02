package main

import (
	"os"

	"github.com/r13v/plexus/internal/cli"
)

func main() {
	os.Exit(cli.Execute())
}

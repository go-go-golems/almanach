package main

import (
	"github.com/go-go-golems/almanach/internal/app"
	"github.com/spf13/cobra"
)

// Version is set at build time via -ldflags.
var Version = "dev"

func main() {
	app.Version = Version
	rootCmd, err := app.NewRootCommand(Version)
	cobra.CheckErr(err)
	cobra.CheckErr(rootCmd.Execute())
}

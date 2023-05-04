package scd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var version = "0.0.1"
var rootCmd = &cobra.Command{
	Use:     "scd",
	Version: version,
	Short:   "scd - a simple CLI for searching and downloading music from souncloud",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Hello")
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Whoops. There was an error while executing your CLI '%s'", err)
		os.Exit(1)
	}
}

package scd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var flagT bool
var flagA bool
var searchCmd = &cobra.Command{
	Use:   "search",
	Args:  cobra.ExactArgs(1),
	Short: "Seach for a song by title/artist",
	Run: func(cmd *cobra.Command, args []string) {
		searchString := args[0]
		if flagT && flagA {
			fmt.Println("Error: You can only use one of the flags -t or -a.")
			os.Exit(1)
		} else if flagT {
			fmt.Println("Flag -t is selected" + " Search: " + searchString)
		} else if flagA {
			fmt.Println("Flag -a is selected" + " Search: " + searchString)
		} else {
			fmt.Println("Error: Please select either flag -t or -a.")
			os.Exit(1)
		}
	},
}

func init() {
	searchCmd.Flags().BoolVarP(&flagT, "title", "t", false, "Search by song title")
	searchCmd.Flags().BoolVarP(&flagA, "artist", "a", false, "Search by song author")
	rootCmd.AddCommand(searchCmd)
}

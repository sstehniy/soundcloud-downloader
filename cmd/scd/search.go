package scd

import (
	"bufio"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/sstehniy/scd/pkg/scd"
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
			searchResults := scd.SearchSongsByTitle(searchString)
			if len(searchResults) == 0 {
				fmt.Println("Nothing found for your search: " + searchString)
				os.Exit(1)
			} else {
				fmt.Println("Search results for: " + searchString + ":")

				for index, song := range searchResults {
					if song.Available {
						fmt.Println("[" + fmt.Sprint(index+1) + "]" + " Title: " + song.Title + "; Artist: " + song.Author + "; Available: " + scd.Colorize("green", fmt.Sprint(song.Available)))
					} else {
						fmt.Println("[" + fmt.Sprint(index+1) + "]" + " Title: " + song.Title + "; Artist: " + song.Author + "; Available: " + scd.Colorize("red", fmt.Sprint(song.Available)))
					}
				}

				buf := bufio.NewReader(os.Stdin)
				var userChoice int
				for {
					fmt.Print("Please select a song to download: ")
					_, err := fmt.Fscan(buf, &userChoice)
					if err != nil {
						fmt.Println("Error: Please enter a valid number.")
					} else if userChoice > len(searchResults) || userChoice < 1 {
						fmt.Println("Error: Please enter a valid number.")
					} else if !searchResults[userChoice-1].Available {
						fmt.Println("Error: The song you selected is not available for download.")
					} else {
						break
					}
				}

				selected := searchResults[userChoice-1]

				fmt.Println("You select the song: " + selected.Title + " by " + selected.Author)
				fmt.Println("Track url: " + selected.Url)

				scd.DownloadTrackByUrl(&selected)
			}
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

package scd

import (
	"bufio"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/sstehniy/scd/pkg/scd"
)

var flagT bool
var flagP bool
var flagA bool
var searchCmd = &cobra.Command{
	Use:   "search",
	Args:  cobra.ExactArgs(1),
	Short: "Search for songs/playlists",
	Run: func(cmd *cobra.Command, args []string) {
		searchString := args[0]
		if flagT && flagP {
			fmt.Println("Error: You can only use one of the flags -t or -p.")
			os.Exit(1)
		} else if flagT {
			searchResults := scd.SearchSongsByTitle(searchString)
			if len(searchResults) == 0 {
				fmt.Println("Nothing found for your search query: " + searchString)
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
				fmt.Println(scd.Colorize("yellow", "Track url: "+selected.Url))

				scd.DownloadTrack(&selected, "")
				fmt.Println(scd.Colorize("green", "Download complete!"))
			}
		} else if flagP {

			searchResults := scd.SearchPlaylistsByTitle(searchString)
			if len(searchResults) == 0 {
				fmt.Println("Nothing found for your search query: " + searchString)
				os.Exit(1)
			} else {
				fmt.Println("Search results for: " + searchString + ":")

				for index, playlist := range searchResults {
					fmt.Println("[" + fmt.Sprint(index+1) + "]" + " Title: " + playlist.Title + "; Artist: " + playlist.Author + "; Track count: " + fmt.Sprint(playlist.TrackCount))
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
					} else {
						break
					}
				}

				selected := searchResults[userChoice-1]

				fmt.Println("You select the playlist: " + selected.Title + " by " + selected.Author)
				fmt.Println(scd.Colorize("yellow", "Playlist url: "+selected.Url))
				scd.DownloadPlaylist(&selected)
				fmt.Println(scd.Colorize("green", "Download complete!"))
			}
		} else if flagA {
			searchResults := scd.SearchAlbumsByTitle(searchString)
			if len(searchResults) == 0 {
				fmt.Println("Nothing found for your search query: " + searchString)
				os.Exit(1)
			} else {
				fmt.Println("Search results for: " + searchString + ":")

				for index, playlist := range searchResults {
					fmt.Println("[" + fmt.Sprint(index+1) + "]" + " Title: " + playlist.Title + "; Artist: " + playlist.Author + "; Track count: " + fmt.Sprint(playlist.TrackCount))
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
					} else {
						break
					}
				}

				selected := searchResults[userChoice-1]

				fmt.Println("You select the playlist: " + selected.Title + " by " + selected.Author)
				fmt.Println(scd.Colorize("yellow", "Playlist url: "+selected.Url))
				scd.DownloadAlbum(&selected)
				fmt.Println(scd.Colorize("green", "Download complete!"))
			}
		}
	},
}

func init() {
	searchCmd.Flags().BoolVarP(&flagT, "title", "t", false, "Search for songs")
	searchCmd.Flags().BoolVarP(&flagP, "playlist", "p", false, "Search for playlists")
	searchCmd.Flags().BoolVarP(&flagA, "album", "a", false, "Search for albums")
	rootCmd.AddCommand(searchCmd)
}

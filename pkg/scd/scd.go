package scd

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/schollz/progressbar/v3"
)

const (
	SoundCloudSongSearchURL     = "https://soundcloud.com/search/sounds?q="
	SoundCloudPlaylistSearchURL = "https://soundcloud.com/search/sets?q="
	SoundCloudBaseURL           = "https://soundcloud.com"
)

type SongData struct {
	Title     string
	Author    string
	Url       string
	Available bool
}

type PlaylistData struct {
	Title      string
	Author     string
	Url        string
	TrackCount int
}

type FetchResponse struct {
	data  []byte
	index int
}

func downloadChunk(url string, index int, arr *[]FetchResponse, bar *progressbar.ProgressBar) {
	resp, err := http.Get(url)
	if err != nil {
		log.Printf("failed to fetch the track: %v", err)
		return
	}
	defer resp.Body.Close()
	bar.Add(1)
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("failed to read bytes from the response: %v", err)
		return
	}
	(*arr)[index] = FetchResponse{
		data:  data,
		index: index,
	}

}

func downloadChunks(urls []string) *[]FetchResponse {
	bar := progressbar.Default(int64(len(urls)), "Track is being downloaded")
	bytesData := make([]FetchResponse, len(urls))

	var wg *sync.WaitGroup = &sync.WaitGroup{}
	for index, url := range urls {
		wg.Add(1)

		go func(url string, index int) {
			defer wg.Done()
			downloadChunk(url, index, &bytesData, bar)
		}(url, index)
	}

	wg.Wait()

	return &bytesData
}

func SearchSongsByTitle(searchString string) []SongData {
	browser := setupBrowser()

	defer browser.MustClose()

	page := loadPage(browser, SoundCloudSongSearchURL+strings.Trim(searchString, " "))
	defer page.MustClose()

	page.MustElement("#onetrust-accept-btn-handler").MustClick()
	page.MustWait(`()=>document.querySelector(".onetrust-pc-dark-filter").style.display == "none"`)

	if _, err := page.Timeout(500 * time.Microsecond).Element(`.sc-type-large.sc-text-h3.sc-text-light.sc-text-primary.searchList__emptyText`); err == nil {
		return []SongData{}
	}

	listItems := page.MustElementsByJS(`() => document.querySelectorAll(".searchList__item")`)
	return createSongDataFromSongSearchResults(listItems)
}

func SearchPlaylistsByTitle(searchString string) []PlaylistData {
	bar := progressbar.NewOptions(-1, progressbar.OptionSetDescription("Searching for playlists..."), progressbar.OptionSetItsString(""), progressbar.OptionSpinnerType(11), progressbar.OptionClearOnFinish(), progressbar.OptionSetElapsedTime(false))
	defer bar.Close()
	finishedChan := make(chan struct{})
	go func(bar *progressbar.ProgressBar, finished <-chan struct{}) {
		for {
			select {
			case <-finishedChan:
				bar.Finish()
				close(finishedChan)
				return
			default:
				{
					time.Sleep(time.Millisecond * 500)
					bar.Add(1)
				}
			}
		}
	}(bar, finishedChan)
	browser := setupBrowser()

	defer browser.MustClose()
	page := loadPage(browser, SoundCloudPlaylistSearchURL+strings.Trim(searchString, " "))
	defer page.MustClose()

	page.MustElement("#onetrust-accept-btn-handler").MustClick()
	page.MustWait(`()=>document.querySelector(".onetrust-pc-dark-filter").style.display == "none"`)
	if _, err := page.Timeout(200 * time.Microsecond).Element(`.sc-type-large.sc-text-h3.sc-text-light.sc-text-primary.searchList__emptyText`); err == nil {
		return []PlaylistData{}
	}

	listItems := page.MustElementsByJS(`() => document.querySelectorAll(".searchList__item")`)
	if len(listItems) > 5 {
		listItems = listItems[0:5]
	}
	data := createSongDataFromPlaylistSearchResults(listItems)
	finishedChan <- struct{}{}
	return data
}

func createSongDataFromSongSearchResults(listItems []*rod.Element) []SongData {
	output := []SongData{}
	for _, item := range listItems {
		_, err := item.Element(".sc-button-play.playButton.sc-button.sc-button-xlarge.sc-button-disabled")
		isDisabled := err != nil

		output = append(output, SongData{
			Title:     item.MustElement(".sc-link-primary.soundTitle__title.sc-link-dark.sc-text-h4").MustText(),
			Author:    item.MustElement(".soundTitle__usernameText").MustText(),
			Url:       SoundCloudBaseURL + *item.MustElement(".sc-link-primary.soundTitle__title.sc-link-dark.sc-text-h4").MustAttribute("href"),
			Available: isDisabled,
		})
	}
	return output
}

func createSongDataFromPlaylistSearchResults(listItems []*rod.Element) []PlaylistData {
	output := make([]PlaylistData, len(listItems))

	for index, item := range listItems {

		moreThenTen, err := item.Element(".compactTrackList__moreLink.sc-link-light.sc-link-primary.sc-border-light.sc-text-h4")
		if err == nil {
			moreThenTen.MustClick()
			item.MustWait(`() => document.querySelector(".compactTrackList__moreLink.sc-link-light.sc-link-primary.sc-border-light.sc-text-h4").textContent === "View fewer tracks"`)
		}

		count := len(item.MustElements(".compactTrackList__item"))

		output[index] = PlaylistData{
			Title:      item.MustElement(".sc-link-primary.soundTitle__title.sc-link-dark.sc-text-h4").MustText(),
			Author:     item.MustElement(".soundTitle__usernameText").MustText(),
			Url:        SoundCloudBaseURL + *item.MustElement(".sc-link-primary.soundTitle__title.sc-link-dark.sc-text-h4").MustAttribute("href"),
			TrackCount: count,
		}

	}
	return output
}

func DownloadTrack(songData *SongData) {
	browser := setupBrowser()

	defer browser.MustClose()

	hijackedUrl := []string{}
	wg := &sync.WaitGroup{}
	wg.Add(1)

	router := browser.HijackRequests()
	router.MustAdd("*", func(ctx *rod.Hijack) {
		ctx.LoadResponse(&http.Client{}, true)
		if strings.Contains(ctx.Request.URL().String(), "m3u8") {
			hijackedUrl = regexp.MustCompile(`(https?:\/\/[^\s]+)`).FindAllString(ctx.Response.Body(), -1)
			router.MustRemove("*")
			router.MustStop()
			wg.Done()
		}
	})
	go router.Run()

	page := loadPage(browser, songData.Url)

	defer page.MustClose()
	wg.Wait()
	chunks := *downloadChunks(hijackedUrl)
	rawBytes := []byte{}
	for _, resp := range chunks {
		rawBytes = append(rawBytes, resp.data...)
	}
	dir := "./new_dir"

	err := os.MkdirAll(dir, os.ModePerm) // create directory if it doesn't exist
	if err != nil {
		fmt.Println("Error creating directory:", err)
		return
	}
	filename := fmt.Sprintf("%s - %s.mp3", songData.Author, songData.Title)
	filepath := fmt.Sprintf("%s/%s", dir, filename)
	ioutil.WriteFile(filepath, rawBytes, 0644)
}

func DownloadPlaylist(playlistData *PlaylistData) {
	browser := setupBrowser()

	defer browser.MustClose()

	page := loadPage(browser, playlistData.Url)

	page.MustElement("#onetrust-accept-btn-handler").MustClick()
	page.MustWait(`()=>document.querySelector(".onetrust-pc-dark-filter").style.display == "none"`)
	currentTracksReveiled := 0
	for currentTracksReveiled < playlistData.TrackCount {
		dims := page.MustElement(".trackList__list.sc-clearfix.sc-list-nostyle").MustShape().Box()
		page.Mouse.MustScroll(0, dims.Y+dims.Height)

		page.MustWait(`() => !!Array.from(document.querySelectorAll(".trackList__item.sc-border-light-bottom.sc-px-2x")).slice(-1)[0].querySelector(".trackItem__content.sc-truncate")`)
		currentTracksReveiled = len(page.MustElementsByJS(`() => document.querySelectorAll(".trackList__item.sc-border-light-bottom.sc-px-2x")`))
		if currentTracksReveiled == playlistData.TrackCount {
			break
		}
	}
	elements := page.MustElementsByJS(`() => document.querySelectorAll(".trackList__item.sc-border-light-bottom.sc-px-2x")`)
	if len(elements) == 0 {
		log.Fatal("no matching elements found")
	}
	last := elements[len(elements)-1]
	shape := last.MustShape()
	if shape == nil {
		log.Fatal("element has no shape")
	}
	page.Mouse.MustScroll(0, shape.Box().Y)
	defer page.MustClose()

	filteredElements := []*rod.Element{}
	for _, element := range elements {
		_, err := element.Element(".compactTrackListItem__tierIndicator")
		if err != nil {
			filteredElements = append(filteredElements, element)
		}
	}
	songs := make([]SongData, len(filteredElements))
	var wg *sync.WaitGroup = &sync.WaitGroup{}

	var mutex sync.Mutex // declare a mutex

	for index, element := range elements {
		wg.Add(1)
		go func(element *rod.Element, index int, output *[]SongData, wg *sync.WaitGroup) {
			mutex.Lock()         // lock the shared variable before modification
			defer mutex.Unlock() // release the lock after modification
			song := SongData{
				Title:     element.MustElement(".trackItem__trackTitle.sc-link-dark.sc-link-primary.sc-font-light").MustText(),
				Url:       SoundCloudBaseURL + *element.MustElement(".trackItem__trackTitle.sc-link-dark.sc-link-primary.sc-font-light").MustAttribute("href"),
				Author:    element.MustElement(".trackItem__username").MustText(),
				Available: true,
			}
			(*output)[index] = song
			wg.Done()
		}(element, index, &songs, wg)
	}
	wg.Wait()

	fmt.Println(len(songs))
	songsChunks := createSongsChunks(&songs)
	// for _, song := range songs {
	// 	if song.Available {
	// 		DownloadTrack(&song)
	// 	}
	// }

	for _, chunk := range songsChunks {
		wg.Add(len(chunk))
		for _, song := range chunk {
			if song.Available {
				go func(song SongData) {
					defer wg.Done()
					DownloadTrack(&song)
				}(song)
			} else {
				wg.Done()
			}
		}
		wg.Wait()
	}

}

func createSongsChunks(songs *[]SongData) [][]SongData {
	chunks := [][]SongData{}
	for i := 0; i < len(*songs); i += 3 {
		end := i + 3
		if end > len(*songs) {
			end = len(*songs)
		}
		chunks = append(chunks, (*songs)[i:end])
	}
	return chunks
}

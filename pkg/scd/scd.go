package scd

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/schollz/progressbar/v3"
)

// CSS Selector Constants
const (
	ITEM_QUERY                   = ".searchList__item"
	TITLE_LINK_QUERY             = ".sc-link-primary.soundTitle__title"
	AUTHOR_USERNAME_QUERY        = ".soundTitle__usernameText"
	EMPTY_RESULTS_MESSAGE        = ".sc-type-large.sc-text-h3.sc-text-light.sc-text-primary.searchList__emptyText"
	PLAY_BUTTON_QUERY            = ".sc-button-play.playButton.sc-button.sc-button-xlarge.sc-button-disabled"
	MORE_LINK_QUERY              = ".compactTrackList__moreLink.sc-link-light.sc-link-primary.sc-border-light.sc-text-h4"
	COMPACT_TRACKLIST_ITEM_QUERY = ".compactTrackList__item"
	TRACK_LIST_ITEM_QUERY        = ".trackList__item.sc-border-light-bottom.sc-px-2x"
	TRACK_TITLE_QUERY            = ".trackItem__trackTitle.sc-link-dark.sc-link-primary.sc-font-light"
	TRACK_AUTHOR_QUERY           = ".trackItem__username.sc-link-light"
	TRACK_LIST_LIST_QUERY        = ".trackList__list.sc-clearfix.sc-list-nostyle"
	TIER_INDICATOR_QUERY         = ".compactTrackListItem__tierIndicator"
)

func downloadChunk(url string, index int, arr *[]FetchResponse) {
	resp, err := http.Get(url)
	if err != nil {
		log.Printf("failed to fetch the track: %v", err)
		return
	}
	defer resp.Body.Close()
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

	bytesData := make([]FetchResponse, len(urls))

	var wg *sync.WaitGroup = &sync.WaitGroup{}
	for index, url := range urls {
		wg.Add(1)

		go func(url string, index int) {

			downloadChunk(url, index, &bytesData)
			defer func() {
				wg.Done()

			}()
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

	err := acceptCookiesAndHandlePage(page)
	if err != nil {
		log.Println("failed to accept cookies and handle page", err)
	}

	if _, err := page.Timeout(500 * time.Microsecond).Element(EMPTY_RESULTS_MESSAGE); err == nil {
		return []SongData{}
	}

	listItems := page.MustElements(ITEM_QUERY)

	fmt.Println("Len results", len(listItems))
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

	err := acceptCookiesAndHandlePage(page)
	if err != nil {
		log.Println("failed to accept cookies and handle page", err)
	}
	if _, err := page.Timeout(200 * time.Microsecond).Element(EMPTY_RESULTS_MESSAGE); err == nil {
		return []PlaylistData{}
	}

	listItems := page.MustElements(ITEM_QUERY)
	if len(listItems) > 15 {
		listItems = listItems[0:15]
	}
	data := createSongDataFromPlaylistSearchResults(listItems)
	finishedChan <- struct{}{}
	return data
}

func SearchAlbumsByTitle(searchString string) []AlbumData {
	bar := progressbar.NewOptions(-1, progressbar.OptionSetDescription("Searching for albums..."), progressbar.OptionSetItsString(""), progressbar.OptionSpinnerType(11), progressbar.OptionClearOnFinish(), progressbar.OptionSetElapsedTime(false))
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
	page := loadPage(browser, SoundCloudAlbumSearchURL+strings.Trim(searchString, " "))
	defer page.MustClose()

	err := acceptCookiesAndHandlePage(page)
	if err != nil {
		log.Println("failed to accept cookies and handle page", err)
	}
	if _, err := page.Timeout(200 * time.Microsecond).Element(EMPTY_RESULTS_MESSAGE); err == nil {
		return []AlbumData{}
	}

	listItems := page.MustElements(ITEM_QUERY)

	if len(listItems) > 15 {
		listItems = listItems[0:15]
	}
	data := createSongDataFromAlbumSearchResults(listItems)
	finishedChan <- struct{}{}
	return data
}

func createSongDataFromSongSearchResults(listItems []*rod.Element) []SongData {
	output := []SongData{}
	for _, item := range listItems {
		_, err := item.Element(PLAY_BUTTON_QUERY)
		isDisabled := err != nil

		output = append(output, SongData{
			Title:     item.MustElement(TITLE_LINK_QUERY).MustText(),
			Author:    item.MustElement(AUTHOR_USERNAME_QUERY).MustText(),
			Url:       SoundCloudBaseURL + *item.MustElement(TITLE_LINK_QUERY).MustAttribute("href"),
			Available: isDisabled,
		})
	}
	return output
}

func createSongDataFromPlaylistSearchResults(listItems []*rod.Element) []PlaylistData {
	output := []PlaylistData{}

	for _, item := range listItems {
		moreThenTen, err := item.Element(MORE_LINK_QUERY)
		if err == nil {
			moreThenTen.MustClick()
			item.MustWait(`() => document.querySelector("` + MORE_LINK_QUERY + `").textContent === "View fewer tracks"`)
		}

		count := len(item.MustElements(COMPACT_TRACKLIST_ITEM_QUERY))
		if count == 0 {
			continue
		}
		output = append(output, PlaylistData{
			Title:      item.MustElement(TITLE_LINK_QUERY).MustText(),
			Author:     item.MustElement(AUTHOR_USERNAME_QUERY).MustText(),
			Url:        SoundCloudBaseURL + *item.MustElement(TITLE_LINK_QUERY).MustAttribute("href"),
			TrackCount: count,
		})

	}
	return output
}

func createSongDataFromAlbumSearchResults(listItems []*rod.Element) []AlbumData {
	output := []AlbumData{}

	for _, item := range listItems {

		moreThenTen, err := item.Element(MORE_LINK_QUERY)
		if err == nil {
			moreThenTen.MustClick()
			item.MustWait(`() => document.querySelector("` + MORE_LINK_QUERY + `").textContent === "View fewer tracks"`)
		}

		count := len(item.MustElements(COMPACT_TRACKLIST_ITEM_QUERY))
		if count == 0 {
			continue
		}
		output = append(output, AlbumData{
			Title:      item.MustElement(TITLE_LINK_QUERY).MustText(),
			Author:     item.MustElement(AUTHOR_USERNAME_QUERY).MustText(),
			Url:        SoundCloudBaseURL + *item.MustElement(TITLE_LINK_QUERY).MustAttribute("href"),
			TrackCount: count,
		})

	}
	return output
}

func DownloadTrack(songData *SongData, parentDir string) {
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
	currentUser, err := user.Current()
	if err != nil {
		panic(err)
	}
	var dir string
	if parentDir == "" {
		dir = filepath.Join(currentUser.HomeDir, "soundcloud-downloader")
	} else {
		dir = filepath.Join(currentUser.HomeDir, "soundcloud-downloader", parentDir)
	}

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, os.ModePerm)
		if err != nil {
			fmt.Println("Error creating directory:", err)
			return
		}
	} else if err != nil {
		fmt.Println("Error checking directory:", err)
		return
	}
	filename := fmt.Sprintf("%s - %s.mp3", songData.Author, songData.Title)
	filepath := fmt.Sprintf("%s/%s", dir, filename)
	ioutil.WriteFile(filepath, rawBytes, 0644)
}

func DownloadPlaylist(playlistData *PlaylistData) {
	prepareBar := progressbar.NewOptions(-1, progressbar.OptionSetDescription("Gathering tracks information"), progressbar.OptionSetItsString(""), progressbar.OptionSpinnerType(11), progressbar.OptionClearOnFinish(), progressbar.OptionSetElapsedTime(false), progressbar.OptionSetRenderBlankState(true))
	triggerCloseBar := make(chan struct{})

	go func() {
		for {
			select {
			case <-triggerCloseBar:
				return
			default:
				prepareBar.Add(1)
				time.Sleep(100 * time.Millisecond)
			}
		}
	}()

	browser := setupBrowser()

	defer browser.MustClose()

	page := loadPage(browser, playlistData.Url)

	err := acceptCookiesAndHandlePage(page)
	if err != nil {
		log.Println("failed to accept cookies and handle page", err)
	}

	currentTracksReveiled := 0
	for currentTracksReveiled < playlistData.TrackCount {

		dims := page.MustElement(TRACK_LIST_LIST_QUERY).MustShape().Box()
		page.Mouse.MustScroll(0, dims.Y+dims.Height)

		page.MustWait(`() => !!Array.from(document.querySelectorAll("` + TRACK_LIST_ITEM_QUERY + `")).slice(-1)[0].querySelector(".trackItem__content.sc-truncate")`)
		currentTracksReveiled = len(page.MustElements(TRACK_LIST_ITEM_QUERY))
		if currentTracksReveiled == playlistData.TrackCount {
			break
		}
	}
	elements := page.MustElements(TRACK_LIST_ITEM_QUERY)
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
	notAvailableElements := []int{}
	for index, element := range elements {
		_, err := element.Element(TIER_INDICATOR_QUERY)
		if err != nil {
			filteredElements = append(filteredElements, element)
		} else {
			notAvailableElements = append(notAvailableElements, index)
		}
	}
	songs := make([]SongData, len(filteredElements))
	var wg *sync.WaitGroup = &sync.WaitGroup{}

	mutex := &sync.Mutex{} // declare a mutex

	for index, element := range elements {
		wg.Add(1)
		go func(element *rod.Element, index int, output *[]SongData, wg *sync.WaitGroup) {
			mutex.Lock()         // lock the shared variable before modification
			defer mutex.Unlock() // release the lock after modification

			title, err := element.Element(TRACK_TITLE_QUERY)
			if err != nil {
				title = nil
			}
			url, err := element.Element(TRACK_TITLE_QUERY)
			if err != nil {

				log.Panicln(err, " url ", index)
			}
			author, err := element.Element(TRACK_AUTHOR_QUERY)
			if err != nil {
				author = nil
			}
			song := SongData{
				Title: func() string {
					if title == nil {
						return ""
					}
					return title.MustText()
				}(),
				Url: SoundCloudBaseURL + *url.MustAttribute("href"),
				Author: func() string {
					if author == nil {
						return ""
					}
					return author.MustText()
				}(),
				Available: true,
			}
			(*output)[index] = song
			wg.Done()
		}(element, index, &songs, wg)
	}
	wg.Wait()
	triggerCloseBar <- struct{}{}
	prepareBar.Close()
	songsChunks := createChunks(&songs, 3)

	loadingBar := progressbar.NewOptions(
		len(songs),
		progressbar.OptionFullWidth(),
		progressbar.OptionSetRenderBlankState(true),
		progressbar.OptionSetDescription("Downloading"),
		progressbar.OptionSetWriter(os.Stderr),
		progressbar.OptionSetItsString(""),
		progressbar.OptionSpinnerType(11),
		progressbar.OptionClearOnFinish(),
		progressbar.OptionShowCount(),
	)

	if len(notAvailableElements) > 0 {
		fmt.Println(Colorize("yellow", "Warning: some songs won't be downloaded as they are not available!"))
	}

	finished := []int{}
	for _, chunk := range songsChunks {
		wg.Add(len(chunk))
		for _, song := range chunk {
			if song.Available {
				go func(song SongData) {
					DownloadTrack(&song, fmt.Sprintf("%s - %s", playlistData.Title, playlistData.Author))
					finished = append(finished, 1)
					loadingBar.Add(1)
					wg.Done()
				}(song)
			} else {
				wg.Done()
			}
		}
		wg.Wait()
	}

	go func() {
		for {
			fmt.Printf("%d/%d \r", len(finished), len(songs))
			time.Sleep(1 * time.Second)
		}
	}()

}

func DownloadAlbum(albumData *AlbumData) {
	prepareBar := progressbar.NewOptions(-1, progressbar.OptionSetDescription("Gathering tracks information"), progressbar.OptionSetItsString(""), progressbar.OptionSpinnerType(11), progressbar.OptionClearOnFinish(), progressbar.OptionSetElapsedTime(false), progressbar.OptionSetRenderBlankState(true))
	triggerCloseBar := make(chan struct{})

	go func() {
		for {
			select {
			case <-triggerCloseBar:
				close(triggerCloseBar)
				return
			default:
				prepareBar.Add(1)
				time.Sleep(100 * time.Millisecond)
			}
		}
	}()
	browser := setupBrowser()

	defer browser.MustClose()

	page := loadPage(browser, albumData.Url)

	err := acceptCookiesAndHandlePage(page)
	if err != nil {
		log.Println("failed to accept cookies and handle page", err)
	}

	currentTracksReveiled := 0
	for currentTracksReveiled < albumData.TrackCount {

		dims := page.MustElement(TRACK_LIST_LIST_QUERY).MustShape().Box()
		page.Mouse.MustScroll(0, dims.Y+dims.Height)

		page.MustWait(`() => !!Array.from(document.querySelectorAll("` + TRACK_LIST_ITEM_QUERY + `")).slice(-1)[0].querySelector(".trackItem__content.sc-truncate")`)
		currentTracksReveiled = len(page.MustElements(TRACK_LIST_ITEM_QUERY))
		if currentTracksReveiled == albumData.TrackCount {
			break
		}
	}
	elements := page.MustElements(TRACK_LIST_ITEM_QUERY)
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
	notAvailableElements := []int{}
	for index, element := range elements {
		_, err := element.Element(TIER_INDICATOR_QUERY)
		if err != nil {
			filteredElements = append(filteredElements, element)
		} else {
			notAvailableElements = append(notAvailableElements, index)
		}
	}
	songs := make([]SongData, len(filteredElements))
	var wg *sync.WaitGroup = &sync.WaitGroup{}

	mutex := &sync.Mutex{} // declare a mutex

	for index, element := range elements {
		wg.Add(1)
		go func(element *rod.Element, index int, output *[]SongData, wg *sync.WaitGroup) {
			mutex.Lock()
			defer mutex.Unlock()

			title, err := element.Element(TRACK_TITLE_QUERY)
			if err != nil {
				title = nil
			}
			url, err := element.Element(TRACK_TITLE_QUERY)
			if err != nil {

				log.Panicln(err, " url ", index)
			}

			song := SongData{
				Title: func() string {
					if title == nil {
						return ""
					}
					return title.MustText()
				}(),
				Url:       SoundCloudBaseURL + *url.MustAttribute("href"),
				Author:    albumData.Author,
				Available: true,
			}
			(*output)[index] = song
			wg.Done()
		}(element, index, &songs, wg)
	}
	wg.Wait()
	triggerCloseBar <- struct{}{}
	prepareBar.Close()
	songsChunks := createChunks(&songs, 3)

	loadingBar := progressbar.NewOptions(
		len(songs),
		progressbar.OptionFullWidth(),
		progressbar.OptionSetRenderBlankState(true),
		progressbar.OptionSetDescription("Downloading"),
		progressbar.OptionSetWriter(os.Stderr),
		progressbar.OptionSetItsString(""),
		progressbar.OptionSpinnerType(11),
		progressbar.OptionClearOnFinish(),
		progressbar.OptionShowCount(),
	)

	if len(notAvailableElements) > 0 {
		fmt.Println(Colorize("yellow", "Warning: some songs won't be downloaded as they are not available!"))
	}

	finished := []int{}
	for _, chunk := range songsChunks {
		wg.Add(len(chunk))
		for _, song := range chunk {
			if song.Available {
				go func(song SongData) {
					DownloadTrack(&song, fmt.Sprintf("%s - %s", albumData.Title, albumData.Author))
					finished = append(finished, 1)
					loadingBar.Add(1)
					wg.Done()
				}(song)
			} else {
				wg.Done()
			}
		}
		wg.Wait()
	}

	go func() {
		for {
			fmt.Printf("%d/%d \r", len(finished), len(songs))
			time.Sleep(1 * time.Second)
		}
	}()
}

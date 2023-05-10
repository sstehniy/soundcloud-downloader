package scd

import (
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/schollz/progressbar/v3"
)

const (
	SoundCloudSearchURL = "https://soundcloud.com/search/sounds?q="
	SoundCloudBaseURL   = "https://soundcloud.com"
)

type SongData struct {
	Title     string
	Author    string
	Url       string
	Available bool
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

	var wg sync.WaitGroup
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
	browser := rod.New().MustConnect()
	defer browser.MustClose()

	page := browser.MustPage(SoundCloudSearchURL + strings.Trim(searchString, " "))
	page.MustWaitRequestIdle()()
	page.MustElement("#onetrust-accept-btn-handler").MustClick()
	page.MustWait(`()=>document.querySelector(".onetrust-pc-dark-filter").style.display == "none"`)

	if _, err := page.Timeout(500 * time.Microsecond).Element(`.sc-type-large.sc-text-h3.sc-text-light.sc-text-primary.searchList__emptyText`); err == nil {
		return []SongData{}
	}

	listItems := page.MustElementsByJS(`() => document.querySelectorAll(".searchList__item")`)
	return createSongDataFromList(listItems)
}

func createSongDataFromList(listItems []*rod.Element) []SongData {
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

func hijackRequests(browser *rod.Browser, hijackedUrl *[]string, wg *sync.WaitGroup) {
	router := browser.HijackRequests()
	router.MustAdd("*", func(ctx *rod.Hijack) {
		ctx.LoadResponse(&http.Client{}, true)
		if strings.Contains(ctx.Request.URL().String(), "m3u8") {
			*hijackedUrl = regexp.MustCompile(`(https?:\/\/[^\s]+)`).FindAllString(ctx.Response.Body(), -1)
			router.MustRemove("*")
			router.MustStop()
			wg.Done()
		}
	})
	go router.Run()
}

func setupPage(browser *rod.Browser, songData *SongData) (*rod.Page, error) {
	page := browser.MustPage(songData.Url)

	page.MustSetViewport(1366, 748, 1, false).MustWindowMaximize().MustWaitLoad()
	wait := page.MustWaitRequestIdle()
	wait()

	return page, nil
}

func DownloadTrackByUrl(songData *SongData) {
	browser := setupBrowser()

	defer browser.MustClose()

	hijackedUrl := []string{}
	wg := &sync.WaitGroup{}
	wg.Add(1)

	hijackRequests(browser, &hijackedUrl, wg)

	page, err := setupPage(browser, songData)
	if err != nil {
		return
	}
	defer page.MustClose()
	wg.Wait()
	chunks := *downloadChunks(hijackedUrl)
	rawBytes := []byte{}
	for _, resp := range chunks {
		rawBytes = append(rawBytes, resp.data...)
	}
	ioutil.WriteFile(songData.Title+".mp3", rawBytes, 0644)
}

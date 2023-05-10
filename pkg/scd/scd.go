package scd

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/schollz/progressbar/v3"
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

func downloadChunks(urls []string) *[]FetchResponse {
	bar := progressbar.Default(int64(len(urls)), "Track is being downloaded")
	bytesData := make([]FetchResponse, len(urls))

	var wg sync.WaitGroup
	for index, url := range urls {
		wg.Add(1)
		go func(url string, index int, arr *[]FetchResponse) {
			defer wg.Done()
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
			bar.Add(1)
		}(url, index, &bytesData)
	}

	wg.Wait()
	bar.Close()

	return &bytesData
}

// SearchSongsByTitle searches for songs by title
func SearchSongsByTitle(searchString string) []SongData {
	output := []SongData{}

	browser := rod.New().MustConnect()
	defer browser.MustClose()
	page := browser.MustPage("https://soundcloud.com/search/sounds?q=" + strings.Trim(searchString, " "))

	page.MustWaitRequestIdle()()

	page.MustElement("#onetrust-accept-btn-handler").MustClick()

	page.MustWait(`()=>document.querySelector(".onetrust-pc-dark-filter").style.display == "none"`)

	if _, err := page.Timeout(500 * time.Microsecond).Element(`.sc-type-large.sc-text-h3.sc-text-light.sc-text-primary.searchList__emptyText`); err == nil {
		return output
	}

	listItems := page.MustElementsByJS(`() => document.querySelectorAll(".searchList__item")`)

	for _, item := range listItems {

		_, err := item.Element(".sc-button-play.playButton.sc-button.sc-button-xlarge.sc-button-disabled")
		isDisabled := err != nil

		output = append(output, SongData{
			Title:     item.MustElement(".sc-link-primary.soundTitle__title.sc-link-dark.sc-text-h4").MustText(),
			Author:    item.MustElement(".soundTitle__usernameText").MustText(),
			Url:       "https://soundcloud.com" + *item.MustElement(".sc-link-primary.soundTitle__title.sc-link-dark.sc-text-h4").MustAttribute("href"),
			Available: isDisabled,
		})

	}
	return output
}

func DownloadTrackByUrl(songData *SongData) {
	ln := launcher.New().
		Set("no-sandbox", "true").
		Headless(true).
		Set("disable-notifications")

	ctl, err := ln.Launch()
	if err != nil {
		log.Println("cannot init launcher", err)
		return
	}

	browser := rod.New().
		ControlURL(ctl)

	err = browser.Connect()
	if err != nil {
		log.Println("cannot connect to browser", err)
		return
	}
	defer browser.MustClose()

	router := browser.HijackRequests()
	var hijackedUrl []string
	wg := &sync.WaitGroup{}
	wg.Add(1)

	router.MustAdd("*", func(ctx *rod.Hijack) {
		ctx.LoadResponse(
			&http.Client{}, true)
		if strings.Contains(ctx.Request.URL().String(), "m3u8") {
			fmt.Println("found m3u8")
			hijackedUrl = regexp.MustCompile(`(https?:\/\/[^\s]+)`).FindAllString(ctx.Response.Body(), -1)
			router.MustRemove("*")
			router.MustStop()
			wg.Done()
		}
	})
	go router.Run()

	var page *rod.Page
	err = rod.Try(func() {
		page = browser.
			MustPage(songData.Url).
			MustSetViewport(1366, 748, 1, false).
			MustWindowMaximize()
	})

	if err != nil {
		log.Println("cannot open page", err)
		return
	}
	defer page.MustClose()
	page.MustWaitLoad()
	wait := page.MustWaitRequestIdle()
	wait()

	page.MustScreenshot("screenshot.png")

	wg.Wait()
	chunks := *downloadChunks(hijackedUrl)
	rawBytes := []byte{}
	for _, resp := range chunks {
		rawBytes = append(rawBytes, resp.data...)
	}
	ioutil.WriteFile(songData.Title+".mp3", rawBytes, 0644)
}

package scd

import (
	"fmt"
	"log"
	"strings"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

func setupBrowser() *rod.Browser {
	ln := launcher.New().
		Set("no-sandbox", "true").
		Headless(true).
		Set("disable-notifications")

	ctl, err := ln.Launch()
	if err != nil {
		log.Println("cannot init launcher", err)
	}

	browser := rod.New().
		ControlURL(ctl)

	err = browser.Connect()
	if err != nil {
		log.Println("cannot connect to browser", err)
	}

	return browser
}

func setupRequestHijacker(browser *rod.Browser, sourceChannel chan string) {
	router := browser.HijackRequests()

	cancel := func() {
		router.Remove("*")
		router.MustStop()
	}
	requestHandler := func(ctx *rod.Hijack) {
		if strings.Contains(ctx.Request.URL().String(), "m3u8") {
			fmt.Println("found m3u8")
			sourceChannel <- ctx.Request.URL().String()
			close(sourceChannel)
			cancel()
		}

		ctx.ContinueRequest(&proto.FetchContinueRequest{})
	}

	router.MustAdd("*", requestHandler)
	go router.Run()

}

func loadPage(browser *rod.Browser, url string) *rod.Page {
	var page *rod.Page
	err := rod.Try(func() {
		page = browser.
			MustPage(url).
			MustSetViewport(1366, 748, 1, false).
			MustWindowMaximize()
	})

	if err != nil {
		log.Println("cannot open page", err)
	}

	page.MustWaitLoad()
	page.MustWaitRequestIdle()()

	page.HijackRequests()
	page.MustScreenshot("screenshot.png")
	return page
}

func clickPlayButton(page *rod.Page) {
	button := page.MustElement(".sc-button-play.playButton.sc-button.m-stretch")
	fmt.Println("button found")
	button.MustClick()
}

package scd

import (
	"github.com/go-rod/rod"
)

func acceptCookiesAndHandlePage(page *rod.Page) error {
	page.MustWaitElementsMoreThan("#onetrust-accept-btn-handler", 0)
	page.MustElement("#onetrust-accept-btn-handler").MustClick()
	page.MustWait(`()=>document.querySelector(".onetrust-pc-dark-filter").style.display == "none"`)
	return nil
}

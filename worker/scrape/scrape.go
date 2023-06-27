package scrape

import "net/http"

type ScrapeTask struct {
	ID       string   `json:"id"`
	Url      string   `json:"url"`
	Response string   `json:"response"`
	Error    []string `json:"error"`
}

func ScrapeHandler(task *ScrapeTask) error {
	resp, err := http.Get(task.Url)
	// check if content-type is application/pdf
	if err == nil && resp.Header.Get("Content-Type") == "application/pdf" {
		return ScrapeHandlerHttpClient(task, resp)
	} else { // otherwise use browser to simulate (in case blocked by waf)
		return ScrapeHandlerBrowser(task)
	}
}

func ScrapeInit() {
	ScrapeInitBrowser()
}

func ScrapeClose() {
	ScrapeCloseBrowser()
}

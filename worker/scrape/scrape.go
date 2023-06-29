package scrape

import (
	"crypto/tls"
	"github.com/SparkSecurity/wakizashi/worker/config"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type ScrapeTask struct {
	ID       string   `json:"id"`
	Url      string   `json:"url"`
	Response string   `json:"response"`
	Error    []string `json:"error"`
}

var ProxyHTTPClient *http.Client

func ScrapeHandler(task *ScrapeTask) error {
	resp, err := ProxyHTTPClient.Get(task.Url)
	// check if content-type is application/pdf
	if err == nil && !strings.Contains(resp.Header.Get("Content-Type"), "text/html") {
		return ScrapeHandlerHttpClient(task, resp)
	} else { // otherwise use browser to simulate (in case blocked by waf)
		return ScrapeHandlerBrowser(task)
	}
}

func ScrapeInit() {
	proxyUrl, err := url.Parse(config.Config.Proxy)
	if err != nil {
		panic(err)
	}
	ProxyHTTPClient = &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyUrl),
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
		Timeout: time.Duration(config.Config.HTTPTimeout) * time.Second,
	}
	ScrapeInitBrowser()
}

func ScrapeClose() {
	ScrapeCloseBrowser()
}

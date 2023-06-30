package scrape

import (
	"crypto/tls"
	"github.com/SparkSecurity/wakizashi/worker/config"
	"net/http"
	"net/url"
	"time"
)

type ScrapeTask struct {
	ID       string   `json:"id"`
	Url      string   `json:"url"`
	Response string   `json:"response"`
	Error    []string `json:"error"`
	Browser  bool     `json:"browser"`
}

var ProxyHTTPClient *http.Client

func ScrapeHandler(task *ScrapeTask) error {
	if task.Browser {
		return ScrapeHandlerBrowser(task)
	} else {
		return ScrapeHandlerHttpClient(task)
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

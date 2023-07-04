package scrape

import (
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/SparkSecurity/wakizashi/worker/config"
	"github.com/SparkSecurity/wakizashi/worker/storage"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var ProxyHTTPClient *http.Client

// ChromeHeaders Copy from Chrome DevTools Raw Headers, and remove Host & Accept-Encoding
const ChromeHeaders = `
Accept: text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7
Accept-Language: zh-CN,zh;q=0.9
Connection: keep-alive
Sec-Fetch-Dest: document
Sec-Fetch-Mode: navigate
Sec-Fetch-Site: none
Sec-Fetch-User: ?1
Upgrade-Insecure-Requests: 1
User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.0.0 Safari/537.36
sec-ch-ua: "Not.A/Brand";v="8", "Chromium";v="114", "Google Chrome";v="114"
sec-ch-ua-mobile: ?0
sec-ch-ua-platform: "Windows"
`

type kv struct {
	Key   string
	Value string
}

var headerList []kv

type Transport struct {
	ProxyTransport *http.Transport
}

func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	for _, header := range headerList {
		req.Header.Add(header.Key, header.Value)
	}
	return t.ProxyTransport.RoundTrip(req)
}

func InitHttpClient() {
	lines := strings.Split(ChromeHeaders, "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ": ", 2)
		headerList = append(headerList, kv{
			Key:   strings.TrimSpace(parts[0]),
			Value: strings.TrimSpace(parts[1]),
		})
	}
	proxyUrl, err := url.Parse(config.Config.Proxy)
	if err != nil {
		panic(err)
	}
	ProxyHTTPClient = &http.Client{
		Transport: &Transport{&http.Transport{
			Proxy: http.ProxyURL(proxyUrl),
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}},
		Timeout: time.Duration(config.Config.HTTPTimeout) * time.Second,
	}
}

func HandlerHttpClient(task *Task) error {
	resp, err := ProxyHTTPClient.Get(task.Url)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return errors.New(fmt.Sprintf("http status code is %d", resp.StatusCode))
	}
	fileId, err := storage.Storage.UploadFile(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		return err
	}
	task.Response = fileId
	return nil
}

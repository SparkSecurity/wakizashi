package scrape

import (
	"github.com/SparkSecurity/wakizashi/worker/storage"
)

func ScrapeHandlerHttpClient(task *ScrapeTask) error {
	resp, err := ProxyHTTPClient.Get(task.Url)
	if err != nil {
		return err
	}
	fileId, err := storage.Storage.UploadFile(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		return err
	}
	task.Response = fileId
	return nil
}

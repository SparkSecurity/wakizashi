package scrape

import (
	"github.com/SparkSecurity/wakizashi/worker/storage"
	"net/http"
)

func ScrapeHandlerHttpClient(task *ScrapeTask, resp *http.Response) error {
	fileId, err := storage.Storage.UploadFile(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		return err
	}
	task.Response = fileId
	return nil
}

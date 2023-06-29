package util

import (
	"archive/zip"
	"encoding/json"
	"github.com/SparkSecurity/wakizashi/manager/model"
	"github.com/SparkSecurity/wakizashi/manager/storage"
	"io"
)

// ZipFile takes a list of pages and zips them into a single file
// Returns the zip file as a byte array
type fileIndex struct {
	ID       string `json:"id"`
	Url      string `json:"url"`
	BodyHash string `json:"bodyHash"`
}

func ZipFile(pages []model.Page, stream io.Writer) error {
	var index []fileIndex
	zipWriter := zip.NewWriter(stream)
	defer func(zipWriter *zip.Writer) {
		_ = zipWriter.Close()
	}(zipWriter)

	// For each page, create a file with just the response body
	for _, page := range pages {
		stream, err := storage.Storage.DownloadFile(page.Response)
		if err != nil {
			return err
		}
		fileStream, err := zipWriter.Create("data/" + page.Response)
		if err != nil {
			return err
		}

		_, err = io.Copy(fileStream, stream)
		if err != nil {
			return err
		}
		err = stream.Close()
		if err != nil {
			return err
		}
		index = append(index, fileIndex{
			ID:       page.ID.Hex(),
			Url:      page.Url,
			BodyHash: page.Response,
		})
	}
	fileStream, err := zipWriter.Create("index.json")
	if err != nil {
		return err
	}
	err = json.NewEncoder(fileStream).Encode(index)
	if err != nil {
		return err
	}
	return nil
}

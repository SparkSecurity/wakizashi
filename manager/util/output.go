package util

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/SparkSecurity/wakizashi/manager/model"
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
	defer zipWriter.Close()

	s256 := sha256.New()
	// For each page, create a file with just the response body
	for _, page := range pages {
		s256.Reset()
		s256.Write([]byte(page.Response))
		hash := fmt.Sprintf("%x", s256.Sum(nil))
		// Stream response body into buffer -> io.stream -> zipWriter
		buffer := bytes.NewBufferString(page.Response)
		fileStream, err := zipWriter.Create("data/" + hash)
		if err != nil {
			return err
		}

		_, err = io.Copy(fileStream, buffer)
		if err != nil {
			return err
		}
		index = append(index, fileIndex{
			ID:       page.ID.Hex(),
			Url:      page.Url,
			BodyHash: hash,
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

package util

import (
	"archive/zip"
	"bytes"
	"github.com/SparkSecurity/wakizashi/manager/model"
	"io"
)

// ZipFile takes a list of pages and zips them into a single file
// Returns the zip file as a byte array
func ZipFile(pages []model.Page, stream io.Writer) error {

	zipWriter := zip.NewWriter(stream)
	defer zipWriter.Close()

	// For each page, create a file with just the response body
	for _, page := range pages {

		// Stream response body into buffer -> io.stream -> zipWriter
		buffer := bytes.NewBufferString(page.Response)     // hum then i guess store as jsons in a zip, with filename <id>.json and the content is json
		fileStream, err := zipWriter.Create(page.ID.Hex()) // hia but that will fuck the ability of normally open that zip
		if err != nil {
			return err
		}

		_, err = io.Copy(fileStream, buffer)
		if err != nil {
			return err
		}
	}

	return nil
}

package util

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"github.com/SparkSecurity/wakizashi/manager/model"
	"github.com/SparkSecurity/wakizashi/manager/storage"
	"io"
	"log"
	"sync"
)

// ZipFile takes a list of pages and zips them into a single file
// Returns the zip file as a byte array
type fileIndex struct {
	ID       string `json:"id"`
	Url      string `json:"url"`
	BodyHash string `json:"bodyHash"`
}

type file struct {
	File   io.ReadCloser
	FileID string
	Error  error
}

func downloadDataWorker(dataFileChan chan string, wg *sync.WaitGroup, fileChan chan *file) {
	defer wg.Done()
	for dataFile := range dataFileChan {
		stream, err := storage.Storage.DownloadFile(dataFile)
		if err != nil {
			fileChan <- &file{nil, dataFile, err}
			continue
		}
		fileChan <- &file{stream, dataFile, nil}
	}
}

func zipDataWorker(fileChan chan *file, zipWriter *zip.Writer, size int) chan error {
	errChan := make(chan error, size)
	go func() {
		defer close(errChan)
		for f := range fileChan {
			if f.Error != nil {
				log.Println("Failed to download file: " + f.FileID + " " + f.Error.Error())
				errChan <- f.Error
				continue
			}
			fileStream, err := zipWriter.Create("data/" + f.FileID)
			if err != nil {
				errChan <- err
				continue
			}
			_, err = io.Copy(fileStream, f.File)
			if err != nil {
				errChan <- err
				continue
			}
			err = f.File.Close()
			if err != nil {
				errChan <- err
				continue
			}
		}
	}()
	return errChan
}
func ZipFile(pages []model.Page, stream io.Writer, indexOnly bool) error {
	var index []fileIndex
	zipWriter := zip.NewWriter(stream)
	defer func(zipWriter *zip.Writer) {
		_ = zipWriter.Close()
	}(zipWriter)

	if !indexOnly {
		dataFileChan := make(chan string, len(pages))
		wg := new(sync.WaitGroup)
		fileChan := make(chan *file, 128)

		for i := 0; i < 32; i++ {
			wg.Add(1)
			go downloadDataWorker(dataFileChan, wg, fileChan)
		}

		// Create a goroutine to zip the files
		zipErrChan := zipDataWorker(fileChan, zipWriter, len(pages))

		// Create download tasks
		for _, page := range pages {
			dataFileChan <- page.Response
			index = append(index, fileIndex{
				ID:       page.ID.Hex(),
				Url:      page.Url,
				BodyHash: page.Response,
			})
		}
		close(dataFileChan)

		// Wait for download tasks
		wg.Wait()

		// Close the file response channel
		close(fileChan)

		// Wait for zip to finish
		globalErrorOccurred := false
		for {
			err, errorOccurred := <-zipErrChan
			if errorOccurred {
				globalErrorOccurred = true
				log.Println("Error occurred while zipping file: " + err.Error())
			} else {
				break
			}
		}
		if globalErrorOccurred {
			return errors.New("error occurred while zipping file")
		}
	} else {
		for _, page := range pages {
			index = append(index, fileIndex{
				ID:       page.ID.Hex(),
				Url:      page.Url,
				BodyHash: page.Response,
			})
		}
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

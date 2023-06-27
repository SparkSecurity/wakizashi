package storage

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
)

type LocalPathStorage struct {
	basePath string
	StorageInterface
}

func (s *LocalPathStorage) Init(basePath string) (err error) {
	s.basePath = basePath
	return
}

func (s *LocalPathStorage) UploadFile(stream io.Reader) (fileId string, err error) {
	s256 := sha256.New()
	// potential memory overuse
	rawBytes, err := io.ReadAll(stream)
	if err != nil {
		return "", err
	}
	s256.Write(rawBytes)
	fileId = fmt.Sprintf("%x", s256.Sum(nil))

	if _, err := os.Stat(s.basePath + "/" + fileId); err == nil { // file exists
		return fileId, nil
	}
	f, err := os.OpenFile(s.basePath+"/"+fileId, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return "", err
	}
	_, err = f.Write(rawBytes)
	return
}

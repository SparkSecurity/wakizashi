package storage

import (
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

func (s *LocalPathStorage) DownloadFile(fileId string) (stream io.Reader, err error) {
	f, err := os.OpenFile(s.basePath+"/"+fileId, os.O_RDONLY, 0)
	if err != nil {
		return nil, err
	}
	return f, nil
}

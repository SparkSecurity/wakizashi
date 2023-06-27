package storage

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"github.com/Cyberax/go-nfs-client/nfs4"
	"io"
)

type NFSStorage struct {
	client   *nfs4.NfsClient
	basePath string
	StorageInterface
}

func (s *NFSStorage) Init(server string, uid, gid uint32, machineName string, basePath string) (err error) {
	s.client, err = nfs4.NewNfsClient(context.Background(), server, nfs4.AuthParams{
		Uid:         uid,
		Gid:         gid,
		MachineName: machineName,
	})
	s.basePath = basePath
	return
}

func (s *NFSStorage) UploadFile(stream io.Reader) (fileId string, err error) {
	s256 := sha256.New()
	// potential memory overuse
	rawBytes, err := io.ReadAll(stream)
	if err != nil {
		return "", err
	}
	s256.Write(rawBytes)
	fileId = fmt.Sprintf("%x", s256.Sum(nil))
	if _, err := s.client.GetFileInfo(s.basePath + "/" + fileId); err == nil { // file exists
		return fileId, nil
	}
	_, err = s.client.WriteFile(s.basePath+"/"+fileId, true, 0, bytes.NewReader(rawBytes))
	return
}

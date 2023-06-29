package storage

import (
	"crypto/sha256"
	"fmt"
	"github.com/sile16/go-nfs-client/nfs"
	"github.com/sile16/go-nfs-client/nfs/rpc"
	"io"
	"log"
)

type NFSStorage struct {
	client *nfs.Target
	StorageInterface
}

func (s *NFSStorage) Init(server string, uid, gid uint32, machineName string, basePath string) (err error) {
	mount, err := nfs.DialMount(server, false)
	if err != nil {
		log.Fatalf("unable to dial MOUNT service: %v", err)
	}
	auth := rpc.NewAuthUnix(machineName, uid, gid)

	v, err := mount.Mount(basePath, auth.Auth(), false)
	if err != nil {
		log.Fatalf("unable to mount volume: %v", err)
	}
	s.client = v
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

	if _, _, err := s.client.Lookup(fileId); err == nil { // file exists
		return fileId, nil
	}
	f, err := s.client.OpenFile(fileId, 0644)
	if err != nil {
		return "", err
	}
	_, err = f.Write(rawBytes)
	if err != nil {
		return "", err
	}
	err = f.Close()
	return
}

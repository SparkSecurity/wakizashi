package storage

import (
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

func (s *NFSStorage) DownloadFile(fileId string) (stream io.Reader, err error) {
	f, err := s.client.OpenFile(fileId, 0644)
	return f, err
}

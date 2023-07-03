package storage

import (
	"crypto/sha256"
	"fmt"
	"github.com/sile16/go-nfs-client/nfs"
	"github.com/sile16/go-nfs-client/nfs/rpc"
	"io"
	"log"
	"strings"
	"sync"
)

type connectionInfo struct {
	server      string
	uid, gid    uint32
	machineName string
	basePath    string
}

type NFSStorage struct {
	connectionInfo connectionInfo
	client         *nfs.Target
	StorageInterface
}

var reconnectLock sync.Mutex

func (s *NFSStorage) Init(server string, uid, gid uint32, machineName string, basePath string) (err error) {
	s.connectionInfo = connectionInfo{
		server:      server,
		uid:         uid,
		gid:         gid,
		machineName: machineName,
		basePath:    basePath,
	}
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

func (s *NFSStorage) reInit() (err error) {
	mount, err := nfs.DialMount(s.connectionInfo.server, false)
	if err != nil {
		log.Fatalf("unable to dial MOUNT service: %v", err)
	}
	auth := rpc.NewAuthUnix(s.connectionInfo.machineName, s.connectionInfo.uid, s.connectionInfo.gid)

	v, err := mount.Mount(s.connectionInfo.basePath, auth.Auth(), false)
	if err != nil {
		log.Fatalf("unable to mount volume: %v", err)
	}
	s.client = v
	return
}

func (s *NFSStorage) EnsureConnected() {
	reconnectLock.Lock()
	defer reconnectLock.Unlock()
	if _, err := s.client.FSInfo(); err != nil {
		if strings.Contains(err.Error(), "rpc: client is shutting down") {
			log.Println("NFS client is shutting down, reconnecting...")
			err := s.reInit()
			if err != nil {
				panic(err)
			}
		} else {
			panic(err)
		}
	}
}

func (s *NFSStorage) UploadFile(stream io.Reader) (fileId string, err error) {
	s.EnsureConnected()
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

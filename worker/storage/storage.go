package storage

import (
	"github.com/SparkSecurity/wakizashi/worker/config"
	"io"
	"log"
	"net/url"
	"strconv"
)

type StorageInterface interface {
	UploadFile(stream io.Reader) (fileId string, err error)
}

var Storage StorageInterface

func CreateStorage() {
	uri, err := url.Parse(config.Config.StorageURI)
	if err != nil {
		panic(err)
	}
	switch uri.Scheme {
	case "nfs":
		if uri.Host == "" || uri.Path == "" || !uri.Query().Has("uid") || !uri.Query().Has("gid") || !uri.Query().Has("machineName") {
			panic("NFS storage scheme parsing failed! Example: nfs://127.0.0.1:2049/datadir?uid=1000&gid=1000&machineName=worker1")
		}
		uidInt, err := strconv.Atoi(uri.Query().Get("uid"))
		if err != nil {
			panic("NFS uid must be int")
		}
		gidInt, err := strconv.Atoi(uri.Query().Get("gid"))
		if err != nil {
			panic("NFS gid must be int")
		}
		storage := NFSStorage{}
		err = storage.Init(uri.Host, uint32(uidInt), uint32(gidInt), uri.Query().Get("machineName"), uri.Path)
		if err != nil {
			panic(err)
		}
		log.Println("NFS storage initialized")
		Storage = storage.StorageInterface
	default:
		panic("Unknown storage scheme! Supported storage scheme: nfs")
	}
}

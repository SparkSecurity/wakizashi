package config

import (
	"github.com/spf13/viper"
)

type PublisherConfig struct {
	ListenPort int

	MongoURI    string
	MongoDBName string

	MQURI      string
	StorageURI string
}

var Config PublisherConfig

func LoadConfig() {
	// User viper to read from .env/env vars
	viper.SetConfigName(".env")
	viper.SetConfigType("env")
	viper.AddConfigPath("./config")
	viper.AutomaticEnv()
	_ = viper.ReadInConfig()

	// Set app settings
	Config.ListenPort = viper.GetInt("LISTEN_PORT")
	Config.MongoURI = viper.GetString("MONGO_URI")
	Config.MongoDBName = viper.GetString("MONGO_DB_NAME")
	Config.MQURI = viper.GetString("MQ_URI")
	Config.StorageURI = viper.GetString("STORAGE_URI")
}

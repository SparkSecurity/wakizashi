package main

import "github.com/spf13/viper"

type PublisherConfig struct {
	MQURI string
	Proxy string
}

var Config PublisherConfig

func LoadConfig() {
	// User viper to read from .env/env vars
	viper.SetConfigName(".env")
	viper.SetConfigType("env")
	viper.AddConfigPath("./")
	viper.AutomaticEnv()
	_ = viper.ReadInConfig()

	// Set app settings
	Config.MQURI = viper.GetString("MQ_URI")
	Config.Proxy = viper.GetString("PROXY")
}

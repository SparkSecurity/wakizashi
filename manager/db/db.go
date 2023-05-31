package db

import (
	"context"
	"github.com/SparkSecurity/wakizashi/manager/config"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var MongoClient *mongo.Client
var DB *mongo.Database
var mongoCtx context.Context

func DBConnect() {
	var err error
	clientOptions := options.Client().ApplyURI(config.Config.MongoURI)
	mongoCtx = context.Background()
	MongoClient, err = mongo.Connect(mongoCtx, clientOptions)
	if err != nil {
		panic(err)
	}
	DB = MongoClient.Database(config.Config.MongoDBName)
}

func DBDisconnect() {
	_ = MongoClient.Disconnect(mongoCtx)
}

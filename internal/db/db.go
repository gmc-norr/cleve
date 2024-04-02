package db

import (
	"context"
	"fmt"
	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"time"
)

var Client mongo.Client
var RunCollection *mongo.Collection
var KeyCollection *mongo.Collection

func Init() {
	mongo_db := viper.GetString("database.name")
	mongo_host := viper.GetString("database.host")
	mongo_port := viper.GetInt("database.port")
	mongo_user := viper.GetString("database.user")
	mongo_password := viper.GetString("database.password")

	log.Printf("Connecting to database %s at %s:%d\n", mongo_db, mongo_host, mongo_port)
	mongoURI := fmt.Sprintf("%s:%d", mongo_host, mongo_port)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	opts := options.Client().SetHosts([]string{mongoURI}).SetAuth(options.Credential{
		Username: mongo_user,
		Password: mongo_password,
	})
	Client, err := mongo.Connect(ctx, opts)

	if err != nil {
		log.Fatalf("mongo.Connect() failed: %s\n", err)
	}

	if err := Client.Ping(ctx, nil); err != nil {
		log.Fatalf("error reaching database: %s\n", err)
	}

	RunCollection = Client.Database(viper.GetString("database.name")).Collection("runs")
	KeyCollection = Client.Database(viper.GetString("database.name")).Collection("keys")

	defer cancel()
}

func SetIndexes() error {
	name, err := SetRunIndex()
	if err != nil {
		return err
	}
	log.Printf("Set index %s on runs", name)
	return nil
}

func GetIndexes() (map[string][]map[string]string, error) {
	runIndex, err := GetRunIndex()
	if err != nil {
		return nil, err
	}

	var indexes = make(map[string][]map[string]string)
	indexes["runs"] = runIndex

	return indexes, nil
}

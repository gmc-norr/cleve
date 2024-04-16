package mongo

import (
	"context"
	"fmt"
	"github.com/gmc-norr/cleve"
	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"time"
)

// Some exports that will be useful in route handling and testing
var ErrNoDocuments = mongo.ErrNoDocuments
var IsDuplicateKeyError = mongo.IsDuplicateKeyError
var GenericDuplicateKeyError = mongo.WriteException{
	WriteErrors: mongo.WriteErrors{
		mongo.WriteError{
			Code: 11000,
		},
	},
}

type DB struct {
	Keys      cleve.APIKeyService
	Runs      cleve.RunService
	Platforms cleve.PlatformService
}

func Connect() (*DB, error) {
	mongo_db := viper.GetString("database.name")
	mongo_host := viper.GetString("database.host")
	mongo_port := viper.GetInt("database.port")
	mongo_user := viper.GetString("database.user")
	mongo_password := viper.GetString("database.password")

	log.Printf("Connecting to database %s at %s:%d\n", mongo_db, mongo_host, mongo_port)
	mongoURI := fmt.Sprintf("%s:%d", mongo_host, mongo_port)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	opts := options.Client().SetHosts([]string{mongoURI}).SetAuth(options.Credential{
		Username: mongo_user,
		Password: mongo_password,
	})
	client, err := mongo.Connect(ctx, opts)

	if err != nil {
		return nil, fmt.Errorf("mongo.Connect() failed: %s\n", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		log.Fatalf("error reaching database: %s\n", err)
	}

	runCollection := client.Database(viper.GetString("database.name")).Collection("runs")
	keyCollection := client.Database(viper.GetString("database.name")).Collection("keys")
	platformCollection := client.Database(viper.GetString("database.name")).Collection("platforms")

	return &DB{
		&APIKeyService{keyCollection},
		&RunService{runCollection},
		&PlatformService{platformCollection},
	}, nil
}

func (db *DB) SetIndexes() error {
	name, err := db.Runs.SetIndex()
	if err != nil {
		return err
	}
	log.Printf("Set index %s on runs", name)

	name, err = db.Platforms.SetIndex()
	if err != nil {
		return err
	}
	log.Printf("Set index %s on platforms", name)

	return nil
}

func (db *DB) GetIndexes() (map[string][]map[string]string, error) {
	runIndex, err := db.Runs.GetIndex()
	if err != nil {
		return nil, err
	}

	var indexes = make(map[string][]map[string]string)
	indexes["runs"] = runIndex

	return indexes, nil
}

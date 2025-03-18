package mongo

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Some exports that will be useful in route handling and testing
var (
	ErrNoDocuments           = mongo.ErrNoDocuments
	IsDuplicateKeyError      = mongo.IsDuplicateKeyError
	GenericDuplicateKeyError = mongo.WriteException{
		WriteErrors: mongo.WriteErrors{
			mongo.WriteError{
				Code: 11000,
			},
		},
	}
)

type DB struct {
	*mongo.Database
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

	return &DB{
		client.Database(mongo_db),
	}, nil
}

func (db DB) RunCollection() *mongo.Collection {
	return db.Collection("runs")
}

func (db DB) KeyCollection() *mongo.Collection {
	return db.Collection("keys")
}

func (db DB) RunQCCollection() *mongo.Collection {
	return db.Collection("run_qc")
}

func (db DB) SampleCollection() *mongo.Collection {
	return db.Collection("samples")
}

func (db DB) SampleSheetCollection() *mongo.Collection {
	return db.Collection("samplesheets")
}

func (db *DB) SetIndexes() error {
	name, err := db.SetRunIndex()
	if err != nil {
		return err
	}
	log.Printf("Set index %s on runs", name)

	name, err = db.SetRunQCIndex()
	if err != nil {
		return err
	}
	log.Printf("Set index %s on run qc", name)

	name, err = db.SetSampleSheetIndex()
	if err != nil {
		return err
	}
	log.Printf("Set index %s on samplesheets", name)

	return nil
}

func (db *DB) GetIndexes() (map[string][]map[string]string, error) {
	runIndex, err := db.RunIndex()
	if err != nil {
		return nil, err
	}

	runQcIndex, err := db.RunQCIndex()
	if err != nil {
		return nil, err
	}

	sampleSheetIndex, err := db.SampleSheetIndex()
	if err != nil {
		return nil, err
	}

	indexes := make(map[string][]map[string]string)
	indexes["runs"] = runIndex
	indexes["run_qc"] = runQcIndex
	indexes["samplesheets"] = sampleSheetIndex

	return indexes, nil
}

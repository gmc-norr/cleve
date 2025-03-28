package mongo

import (
	"context"
	"errors"
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
		return fmt.Errorf("failed to set index on runs, does the collection exist? %w", err)
	}
	log.Printf("Set index %s on runs", name)

	name, err = db.SetRunQCIndex()
	if err != nil {
		return fmt.Errorf("failed to set index on run qc, does the collection exist? %w", err)
	}
	log.Printf("Set index %s on run qc", name)

	name, err = db.SetSampleSheetIndex()
	if err != nil {
		return fmt.Errorf("failed to set index on samplesheets, does the collection exist? %w", err)
	}
	log.Printf("Set index %s on samplesheets", name)

	return nil
}

// Init creates all the mongodb collections needed by cleve. The function
// will exit at the first error it sees, if any. When a collection is
// created the indexes will also be set for this collection.
func (db *DB) Init(ctx context.Context) error {
	createCollection := func(name string) error {
		err := db.CreateCollection(ctx, name)
		if err != nil {
			if errors.As(err, &mongo.CommandError{}) {
				log.Printf("collection %s already exists", name)
				return nil
			}
			return err
		}
		log.Printf("created collection %s", name)
		return nil
	}
	if err := createCollection("runs"); err != nil {
		return err
	}
	if _, err := db.SetRunIndex(); err != nil {
		return err
	}
	if err := createCollection("keys"); err != nil {
		return err
	}
	if err := createCollection("run_qc"); err != nil {
		return err
	}
	if _, err := db.SetRunQCIndex(); err != nil {
		return err
	}
	if err := createCollection("samples"); err != nil {
		return err
	}
	if err := createCollection("samplesheets"); err != nil {
		return err
	}
	if _, err := db.SetSampleSheetIndex(); err != nil {
		return err
	}
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

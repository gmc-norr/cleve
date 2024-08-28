package mongo

import (
	"context"
	"fmt"
	"log"

	"github.com/gmc-norr/cleve"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (db DB) Platforms() ([]*cleve.Platform, error) {
	cursor, err := db.PlatformCollection().Find(context.TODO(), bson.D{})
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return []*cleve.Platform{}, nil
		}
		return nil, err
	}
	defer cursor.Close(context.TODO())

	var platforms []*cleve.Platform
	if err = cursor.All(context.TODO(), &platforms); err != nil {
		return platforms, err
	}
	return platforms, nil
}

func (db DB) Platform(name string) (*cleve.Platform, error) {
	var platform cleve.Platform
	err := db.PlatformCollection().FindOne(context.TODO(), bson.D{{Key: "name", Value: name}}).Decode(&platform)
	if err != nil {
		return nil, err
	}
	return &platform, nil
}

func (db DB) CreatePlatform(platform *cleve.Platform) error {
	_, err := db.PlatformCollection().InsertOne(context.TODO(), platform)
	return err
}

func (db DB) DeletePlatform(name string) error {
	res, err := db.PlatformCollection().DeleteOne(context.TODO(), bson.D{{Key: "name", Value: name}})
	if err == nil && res.DeletedCount == 0 {
		return fmt.Errorf(`no platform with name "%s" found`, name)
	}
	if err != nil {
		return err
	}
	return nil
}

func (db DB) SetPlatformIndex() (string, error) {
	indexModel := mongo.IndexModel{
		Keys: bson.D{
			{Key: "name", Value: 1},
		},
		Options: options.Index().SetUnique(true),
	}

	// TODO: do this as a transaction and roll back if anything fails
	res, err := db.PlatformCollection().Indexes().DropAll(context.TODO())
	if err != nil {
		return "", err
	}

	log.Printf("Dropped %d indexes\n", res.Lookup("nIndexesWas").Int32())

	name, err := db.PlatformCollection().Indexes().CreateOne(context.TODO(), indexModel)
	return name, err
}

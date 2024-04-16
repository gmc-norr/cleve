package mongo

import (
	"context"
	"fmt"
	"github.com/gmc-norr/cleve"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
)

type PlatformService struct{
	coll *mongo.Collection
}

func (s *PlatformService) All() ([]*cleve.Platform, error) {
	cursor, err := s.coll.Find(context.TODO(), bson.D{})
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

func (s *PlatformService) Get(name string) (*cleve.Platform, error) {
	var platform cleve.Platform
	err := s.coll.FindOne(context.TODO(), bson.D{{Key: "name", Value: name}}).Decode(&platform)
	if err != nil {
		return nil, err
	}
	return &platform, nil
}

func (s *PlatformService) Create(platform *cleve.Platform) error {
	_, err := s.coll.InsertOne(context.TODO(), platform)
	return err
}

func (s *PlatformService) Delete(name string) error {
	res, err := s.coll.DeleteOne(context.TODO(), bson.D{{Key: "name", Value: name}})
	if err == nil && res.DeletedCount == 0 {
		return fmt.Errorf(`no platform with name "%s" found`, name)
	}
	if err != nil {
		return err
	}
	return nil
}

func (s *PlatformService) SetIndex() (string, error) {
	indexModel := mongo.IndexModel{
		Keys: bson.D{
			{Key: "name", Value: 1},
		},
		Options: options.Index().SetUnique(true),
	}

	// TODO: do this as a transaction and roll back if anything fails
	res, err := s.coll.Indexes().DropAll(context.TODO())
	if err != nil {
		return "", err
	}

	log.Printf("Dropped %d indexes\n", res.Lookup("nIndexesWas").Int32())

	name, err := s.coll.Indexes().CreateOne(context.TODO(), indexModel)
	return name, err
}

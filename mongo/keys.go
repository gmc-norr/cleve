package mongo

import (
	"context"
	"fmt"
	"github.com/gmc-norr/cleve"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type APIKeyService struct {
	coll *mongo.Collection
}

func (s *APIKeyService) Create(k *cleve.APIKey) error {
	_, err := s.UserKey(k.User)
	if err == nil {
		return fmt.Errorf("key already exists for user %s", k.User)
	}
	if err != mongo.ErrNoDocuments {
		return err
	}
	if _, err := s.coll.InsertOne(context.TODO(), k); err != nil {
		return err
	}
	return nil
}

func (s *APIKeyService) Delete(k string) error {
	res, err := s.coll.DeleteOne(context.TODO(), bson.D{
		{Key: "key", Value: k},
	})
	if res.DeletedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return err
}

func (s *APIKeyService) Get(k string) (*cleve.APIKey, error) {
	var key cleve.APIKey
	err := s.coll.FindOne(context.TODO(), bson.D{{Key: "key", Value: k}}).Decode(&key)
	return &key, err
}

func (s *APIKeyService) All() ([]*cleve.APIKey, error) {
	var keys []*cleve.APIKey
	cursor, err := s.coll.Find(context.TODO(), bson.D{})
	if err != nil {
		return keys, err
	}
	if err = cursor.All(context.TODO(), &keys); err != nil {
		return keys, err
	}
	return keys, nil
}

func (s *APIKeyService) UserKey(user string) (*cleve.APIKey, error) {
	var key cleve.APIKey
	err := s.coll.FindOne(context.TODO(), bson.D{{Key: "user", Value: user}}).Decode(&key)
	return &key, err
}

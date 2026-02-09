package mongo

import (
	"context"
	"fmt"

	"github.com/gmc-norr/cleve"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func (db DB) CreateKey(k *cleve.APIKey) error {
	_, err := db.KeyFromId(k.Id)
	if err == nil {
		return fmt.Errorf("key already exists for user %s", k.User)
	}
	if err != mongo.ErrNoDocuments {
		return err
	}
	if _, err := db.KeyCollection().InsertOne(context.TODO(), k); err != nil {
		return err
	}
	return nil
}

func (db DB) DeleteKey(id []byte) error {
	res, err := db.KeyCollection().DeleteOne(context.TODO(), bson.D{
		{Key: "id", Value: id},
	})
	if res.DeletedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return err
}

func (db DB) Key(k string) (*cleve.APIKey, error) {
	var key cleve.APIKey
	err := db.KeyCollection().FindOne(context.TODO(), bson.D{{Key: "key", Value: k}}).Decode(&key)
	return &key, err
}

func (db DB) Keys() ([]*cleve.APIKey, error) {
	var keys []*cleve.APIKey
	cursor, err := db.KeyCollection().Find(context.TODO(), bson.D{})
	if err != nil {
		return keys, err
	}
	if err = cursor.All(context.TODO(), &keys); err != nil {
		return keys, err
	}
	return keys, nil
}

func (db DB) KeyFromId(id []byte) (*cleve.APIKey, error) {
	var key cleve.APIKey
	err := db.KeyCollection().FindOne(context.TODO(), bson.D{{Key: "id", Value: id}}).Decode(&key)
	return &key, err
}

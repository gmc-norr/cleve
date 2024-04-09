package mongo

import (
	"context"
	"fmt"
	"github.com/gmc-norr/cleve"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func AddKey(k *cleve.APIKey) error {
	_, err := GetUserKey(k.User)
	if err != mongo.ErrNoDocuments {
		return fmt.Errorf("key already exists for user %s", k.User)
	}
	if _, err := KeyCollection.InsertOne(context.TODO(), k); err != nil {
		return err
	}
	return nil
}

func GetKey(k string) (*cleve.APIKey, error) {
	var key cleve.APIKey
	err := KeyCollection.FindOne(context.TODO(), bson.D{{Key: "key", Value: k}}).Decode(&key)
	return &key, err
}

func GetUserKey(u string) (*cleve.APIKey, error) {
	var key cleve.APIKey
	err := KeyCollection.FindOne(context.TODO(), bson.D{{Key: "user", Value: u}}).Decode(&key)
	return &key, err
}

func GetKeys() ([]*cleve.APIKey, error) {
	var keys []*cleve.APIKey
	cursor, err := KeyCollection.Find(context.TODO(), bson.D{})
	if err != nil {
		return keys, err
	}
	if err = cursor.All(context.TODO(), &keys); err != nil {
		return keys, err
	}
	return keys, nil
}

func DeleteKey(key string) error {
	res, err := KeyCollection.DeleteOne(context.TODO(), bson.D{
		{Key: "key", Value: key},
	})
	if res.DeletedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return err
}

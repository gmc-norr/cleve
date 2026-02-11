package mongo

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/gmc-norr/cleve"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
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

func (db DB) KeyIndex() ([]map[string]string, error) {
	cursor, err := db.KeyCollection().Indexes().List(context.TODO())
	if err != nil {
		return []map[string]string{}, err
	}
	defer closeCursor(cursor, context.TODO())

	var indexes []map[string]string

	var result []bson.M
	if err = cursor.All(context.TODO(), &result); err != nil {
		return []map[string]string{}, err
	}

	for _, v := range result {
		i := map[string]string{}
		for k, val := range v {
			i[k] = fmt.Sprintf("%v", val)
		}
		indexes = append(indexes, i)
	}

	return indexes, nil
}

func (db DB) SetKeyIndex() (string, error) {
	indexModel := mongo.IndexModel{
		Keys: bson.D{
			{Key: "id", Value: 1},
		},
		Options: options.Index().SetUnique(true),
	}

	// TODO: do this as a transaction and roll back if anything fails
	res, err := db.KeyCollection().Indexes().DropAll(context.TODO())
	if err != nil {
		return "", err
	}

	slog.Info("dropped indexes", "collection", "keys", "count", res.Lookup("nIndexesWas").Int32())

	name, err := db.KeyCollection().Indexes().CreateOne(context.TODO(), indexModel)
	return name, err
}

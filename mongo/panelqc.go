package mongo

import (
	"context"
	"fmt"
	"log"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)



func (db DB) PanelQcIndex() ([]map[string]string, error) {
	cursor, err := db.PanelQcCollection().Indexes().List(context.TODO())
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


func (db DB) SetPanelQcIndex() (string, error) {
	indexModel := mongo.IndexModel{
		Keys: bson.D{
			{Key: "run_id", Value: 1},
		},
		Options: options.Index().SetUnique(true),
	}

	res, err := db.PanelQcCollection().Indexes().DropAll(context.TODO())
	if err != nil {
		return "", err
	}

	log.Printf("Dropped %d indexes\n", res.Lookup("nIndexesWas").Int32())

	name, err := db.PanelQcCollection().Indexes().CreateOne(context.TODO(), indexModel)
	return name, err
	
}

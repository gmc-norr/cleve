package mongo

import (
	"context"
	"fmt"

	"github.com/gmc-norr/cleve/interop"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type RunQcService struct {
	coll *mongo.Collection
}

func (s *RunQcService) Create(runId string, qc *interop.InteropSummary) error {
	_, err := s.coll.InsertOne(context.TODO(), qc)
	return err
}

func (s *RunQcService) Get(runId string) (*interop.InteropSummary, error) {
	var qc interop.InteropSummary
	err := s.coll.FindOne(context.TODO(), bson.D{{Key: "run_id", Value: runId}}).Decode(&qc)
	return &qc, err
}

func (s *RunQcService) GetIndex() ([]map[string]string, error) {
	cursor, err := s.coll.Indexes().List(context.TODO())
	defer cursor.Close(context.TODO())

	var indexes []map[string]string
	if err != nil {
		return []map[string]string{}, err
	}

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

func (s *RunQcService) SetIndex() (string, error) {
	indexModel := mongo.IndexModel{
		Keys: bson.D{
			{Key: "run_id", Value: 1},
		},
		Options: options.Index().SetUnique(true),
	}

	// TODO: do this as a transaction and roll back if anything fails
	_, err := s.coll.Indexes().DropAll(context.TODO())
	if err != nil {
		return "", err
	}

	name, err := s.coll.Indexes().CreateOne(context.TODO(), indexModel)
	return name, err
}

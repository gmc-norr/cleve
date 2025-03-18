package mongo

import (
	"context"

	"github.com/gmc-norr/cleve"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func (db DB) Platforms() (cleve.Platforms, error) {
	var pipeline mongo.Pipeline

	pipeline = append(pipeline, bson.D{
		{Key: "$set", Value: bson.M{
			"run_info.instrument_id": bson.D{
				{Key: "$ifNull", Value: bson.A{"$run_info.instrument_id", "$run_info.run.instrument"}},
			},
		}},
	})

	pipeline = append(pipeline, bson.D{
		{Key: "$group", Value: bson.M{
			"_id":           "$run_info.instrument_id",
			"instrument_id": bson.D{{Key: "$first", Value: "$run_info.instrument_id"}},
			"aliases":       bson.D{{Key: "$addToSet", Value: "$platform"}},
			"count": bson.D{
				{Key: "$count", Value: bson.D{}},
			},
		}},
	})

	cursor, err := db.RunCollection().Aggregate(context.TODO(), pipeline)
	platforms := cleve.Platforms{}
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return platforms, nil
		}
		return platforms, err
	}
	defer cursor.Close(context.TODO())

	err = cursor.All(context.TODO(), &platforms.Platforms)
	return platforms.Condense(), err
}

func (db DB) Platform(name string) (cleve.Platform, error) {
	platforms, err := db.Platforms()
	if err != nil {
		return cleve.Platform{}, err
	}
	p, ok := platforms.Get(name)
	if !ok {
		return cleve.Platform{}, mongo.ErrNoDocuments
	}
	return p, nil
}

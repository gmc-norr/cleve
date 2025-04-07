package mongo

import (
	"context"
	"fmt"

	"github.com/gmc-norr/cleve"
	"github.com/gmc-norr/cleve/interop"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (db DB) CreateRunQC(runId string, qc interop.InteropSummary) error {
	_, err := db.RunQCCollection().InsertOne(context.TODO(), qc)
	return err
}

func (db DB) DeleteRunQC(runId string) error {
	_, err := db.RunQCCollection().DeleteOne(context.TODO(), bson.D{
		{Key: "run_id", Value: runId},
	})
	return err
}

func (db DB) RunQCs(filter cleve.QcFilter) (cleve.QcResult, error) {
	var pipeline mongo.Pipeline
	var qc cleve.QcResult

	// Run ID filter
	if filter.RunIdQuery != "" {
		pipeline = append(pipeline, bson.D{
			{Key: "$match", Value: bson.D{
				{Key: "$expr", Value: bson.D{
					{Key: "$regexMatch", Value: bson.M{
						"input":   "$run_id",
						"regex":   filter.RunIdQuery,
						"options": "i",
					}},
				}},
			}},
		})
	}

	// Platform filter
	if filter.Platform != "" {
		platform, err := db.Platform(filter.Platform)
		if err != nil {
			return qc, err
		}
		platformNames := append(platform.Aliases, platform.Name)
		pipeline = append(pipeline, bson.D{
			{Key: "$match", Value: bson.D{
				{Key: "$expr", Value: bson.D{{Key: "$in", Value: bson.A{"$platform", platformNames}}}},
			}},
		})
	}

	qcFacet := mongo.Pipeline{}

	// Skip
	if filter.Page > 0 && filter.PageSize > 0 {
		qcFacet = append(qcFacet, bson.D{
			{Key: "$skip", Value: filter.PageSize * (filter.Page - 1)},
		})
	}

	// Limit
	if filter.PageSize > 0 {
		qcFacet = append(qcFacet, bson.D{
			{Key: "$limit", Value: filter.PageSize},
		})
	}

	// Sort
	qcFacet = append(qcFacet, bson.D{
		{Key: "$sort", Value: bson.D{
			{Key: "date", Value: -1},
		}},
	})

	// Facet to get both metadata and qc
	pipeline = append(pipeline, bson.D{
		{Key: "$facet", Value: bson.M{
			"metadata": bson.A{
				bson.D{{Key: "$count", Value: "total_count"}},
			},
			"interop": qcFacet,
		}},
	})

	pipeline = append(pipeline, bson.D{
		{
			Key: "$project",
			Value: bson.M{
				"metadata": bson.M{
					"$arrayElemAt": bson.A{"$metadata", 0},
				},
				"interop": 1,
			},
		},
	})

	pipeline = append(pipeline, bson.D{
		{
			Key: "$set",
			Value: bson.M{
				"metadata.count": bson.M{
					"$size": "$interop",
				},
				"metadata.page":      filter.Page,
				"metadata.page_size": filter.PageSize,
				"metadata.total_pages": bson.M{
					"$cond": bson.M{
						"if": bson.M{
							"$gt": bson.A{
								filter.PageSize,
								0,
							},
						},
						"then": bson.M{
							"$ceil": bson.M{
								"$divide": bson.A{
									"$metadata.total_count",
									filter.PageSize,
								},
							},
						},
						"else": 1,
					},
				},
			},
		},
	})

	cursor, err := db.RunQCCollection().Aggregate(context.TODO(), pipeline)
	if err != nil {
		return cleve.QcResult{}, err
	}
	defer cursor.Close(context.TODO())

	cursor.Next(context.TODO())
	err = cursor.Decode(&qc)
	if err != nil {
		return cleve.QcResult{}, err
	}
	if qc.TotalCount == 0 {
		qc.TotalPages = 1
	}
	if qc.Page > qc.TotalPages {
		return qc, PageOutOfBoundsError{
			page:       qc.Page,
			totalPages: qc.TotalPages,
		}
	}

	return qc, nil
}

func (db DB) RunQC(runId string) (interop.InteropSummary, error) {
	var qc interop.InteropSummary
	err := db.RunQCCollection().FindOne(context.TODO(), bson.D{{Key: "run_id", Value: runId}}).Decode(&qc)
	return qc, err
}

func (db DB) RunQCIndex() ([]map[string]string, error) {
	cursor, err := db.RunQCCollection().Indexes().List(context.TODO())
	if err != nil {
		return []map[string]string{}, err
	}
	defer cursor.Close(context.TODO())

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

func (db DB) SetRunQCIndex() (string, error) {
	indexModel := mongo.IndexModel{
		Keys: bson.D{
			{Key: "run_id", Value: 1},
		},
		Options: options.Index().SetUnique(true),
	}

	// TODO: do this as a transaction and roll back if anything fails
	_, err := db.RunQCCollection().Indexes().DropAll(context.TODO())
	if err != nil {
		return "", err
	}

	name, err := db.RunQCCollection().Indexes().CreateOne(context.TODO(), indexModel)
	return name, err
}

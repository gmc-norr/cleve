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

type RunQcService struct {
	coll *mongo.Collection
}

func (s *RunQcService) Create(runId string, qc *interop.InteropSummary) error {
	_, err := s.coll.InsertOne(context.TODO(), qc)
	return err
}

func (s *RunQcService) All(filter cleve.QcFilter) (cleve.QcResult, error) {
	var qcPipeline mongo.Pipeline
	var aggPipeline mongo.Pipeline
	var qc cleve.QcResult

	aggPipeline = append(aggPipeline, bson.D{
		{Key: "$lookup", Value: bson.M{
			"from": "runs",
			"localField": "run_id",
			"foreignField": "run_id",
			"as": "run",
		}},
	})

	// We expect only one run, so unwind the array
	qcPipeline = append(qcPipeline, bson.D{
		{Key: "$unwind", Value: "$run"},	
	})

	// Platform filter
	if filter.Platform != "" {
		platformFilter := bson.D{
			{Key: "$match", Value: bson.D{
				{Key: "run.platform", Value: filter.Platform},
			}},
		}
		aggPipeline = append(aggPipeline, platformFilter)
	}

	// Run ID filter
	if filter.RunID != "" {
		runIdFilter := bson.D{
			{Key: "$match", Value: bson.D{
				{Key: "$expr", Value: bson.D{
					{Key: "$regexMatch", Value: bson.M{
						"input":   "$run_id",
						"regex":   filter.RunID,
						"options": "i",
					}},
				}},
			}},
		}
		aggPipeline = append(aggPipeline, runIdFilter)
	}

	// Skip
	if filter.Page > 0 && filter.PageSize > 0 {
		qcPipeline = append(qcPipeline, bson.D{
			{Key: "$skip", Value: filter.PageSize * (filter.Page - 1)},
		})
	}

	// Limit
	if filter.PageSize > 0 {
		qcPipeline = append(qcPipeline, bson.D{
			{Key: "$limit", Value: filter.PageSize},
		})
	}

	// Sort
	qcPipeline = append(qcPipeline, bson.D{
		{Key: "$sort", Value: bson.D{
			{Key: "run.run_info.run.date", Value: -1},
		}},
	})

	// Facet to get both metadata and qc
	aggPipeline = append(aggPipeline, bson.D{
		{Key: "$facet", Value: bson.M{
			"metadata": bson.A{
				bson.D{{Key: "$count", Value: "total_count"}},
			},
			"qc": qcPipeline,
		}},
	})

	aggPipeline = append(aggPipeline, bson.D{
		{Key: "$unwind", Value: "$metadata"},
	})

	aggPipeline = append(aggPipeline, bson.D{
		{Key: "$set", Value: bson.D{
			{Key: "metadata.count", Value: bson.D{
				{Key: "$size", Value: "$qc"},
			}},
			{Key: "metadata.page", Value: filter.Page},
			{Key: "metadata.page_size", Value: filter.PageSize},
		}},
	})

	if filter.PageSize > 0 {
		aggPipeline = append(aggPipeline, bson.D{
			{Key: "$set", Value: bson.D{
				{Key: "metadata.total_pages", Value: bson.D{
					{Key: "$ceil", Value: bson.D{
						{Key: "$divide", Value: bson.A{"$metadata.total_count", filter.PageSize}}},
					},
				}},
			}},
		})
	} else {
		aggPipeline = append(aggPipeline, bson.D{
			{Key: "$set", Value: bson.D{
				{Key: "metadata.total_pages", Value: 1},
				{Key: "metadata.page_size", Value: "$metadata.total_count"},
			}},
		})
	}

	cursor, err := s.coll.Aggregate(context.TODO(), aggPipeline)
	if err != nil {
		return qc, err
	}

	if cursor.Next(context.TODO()) {
		var qc cleve.QcResult
		cursor.Current.Lookup("metadata").Unmarshal(&qc.RunMetadata)
		rawQc := cursor.Current.Lookup("qc")
		err := rawQc.Unmarshal(&qc.Qc)
		return qc, err
	}

	return qc, cursor.Err()
}

func (s *RunQcService) Get(runId string) (*interop.InteropSummary, error) {
	var qc interop.InteropSummary
	err := s.coll.FindOne(context.TODO(), bson.D{{Key: "run_id", Value: runId}}).Decode(&qc)
	return &qc, err
}

func (s *RunQcService) GetTotalQ30(runId string) (float64, error) {
	qc, err := s.Get(runId)
	if err != nil {
		return 0, err
	}
	return float64(qc.RunSummary["Total"].PercentQ30), nil
}

func (s *RunQcService) GetTotalErrorRate(runId string) (float64, error) {
	e, err := s.Get(runId)
	if err != nil {
		return 0, err
	}
	return float64(e.RunSummary["Total"].ErrorRate), nil
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

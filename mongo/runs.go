package mongo

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gmc-norr/cleve"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (db DB) Runs(filter cleve.RunFilter) (cleve.RunResult, error) {
	var pipeline mongo.Pipeline

	// Sort by sequencing date
	pipeline = append(pipeline, bson.D{
		{Key: "$sort", Value: bson.D{
			{Key: "run_info.run.date", Value: -1},
		}},
	})

	// Filter on run id
	if filter.RunID != "" {
		pipeline = append(pipeline, bson.D{
			{Key: "$match", Value: bson.D{
				{Key: "$expr", Value: bson.D{
					{Key: "$regexMatch", Value: bson.M{
						"input":   "$run_id",
						"regex":   filter.RunID,
						"options": "i",
					}},
				}},
			}},
		})
	}

	// Filter on platform
	if filter.Platform != "" {
		pipeline = append(pipeline, bson.D{
			{Key: "$match", Value: bson.D{
				{Key: "platform", Value: filter.Platform},
			}},
		})
	}

	// Sort state history chronologically
	pipeline = append(pipeline, bson.D{
		{Key: "$set", Value: bson.D{
			{Key: "state_history", Value: bson.D{
				{Key: "$sortArray", Value: bson.M{
					"input": "$state_history", "sortBy": bson.D{{Key: "time", Value: -1}},
				}},
			}},
		},
		},
	})

	// Filter on most recent state
	if filter.State != "" {
		pipeline = append(pipeline, bson.D{
			{Key: "$match", Value: bson.D{
				{Key: "$expr", Value: bson.D{
					{Key: "$eq", Value: bson.A{
						bson.D{{Key: "$arrayElemAt", Value: bson.A{"$state_history.state", 0}}},
						filter.State,
					}},
				}},
			}},
		})
	}

	// Add samplesheet information
	pipeline = append(pipeline, bson.D{
		{Key: "$lookup", Value: bson.M{
			"from":         "samplesheets",
			"localField":   "run_id",
			"foreignField": "run_id",
			"as":           "samplesheet",
			"pipeline": bson.A{
				bson.M{
					"$project": bson.M{
						"path":              1,
						"modification_time": 1,
					},
				},
			},
		}},
	})

	// Count number of analyses
	pipeline = append(pipeline, bson.D{
		{Key: "$set", Value: bson.D{
			{
				Key: "analysis_count",
				Value: bson.D{
					{Key: "$cond", Value: bson.M{
						"if": bson.D{
							{Key: "$isArray", Value: "$analysis"},
						}, "then": bson.D{
							{Key: "$size", Value: "$analysis"},
						}, "else": 0,
					}},
				},
			},
		}},
	})

	// Exclude run parameters and analysis
	if filter.Brief {
		pipeline = append(pipeline, bson.D{
			{Key: "$unset", Value: bson.A{"run_parameters", "analysis"}},
		})
	}

	pipeline = append(pipeline, bson.D{
		{Key: "$unwind", Value: bson.M{
			"path":                       "$samplesheet",
			"preserveNullAndEmptyArrays": true,
		}},
	})

	runFacet := mongo.Pipeline{}

	// Skip
	if filter.Page > 0 {
		runFacet = append(runFacet, bson.D{
			{Key: "$skip", Value: filter.PageSize * (filter.Page - 1)},
		})
	}

	// Limit
	if filter.PageSize > 0 {
		runFacet = append(runFacet, bson.D{
			{Key: "$limit", Value: filter.PageSize},
		})
	}

	// Facetting
	pipeline = append(pipeline, bson.D{
		{Key: "$facet", Value: bson.M{
			"metadata": bson.A{
				bson.D{{Key: "$count", Value: "total_count"}},
			},
			"runs": runFacet,
		}},
	})

	pipeline = append(pipeline, bson.D{
		{
			Key: "$project",
			Value: bson.M{
				"metadata": bson.M{
					"$arrayElemAt": bson.A{"$metadata", 0},
				},
				"runs": 1,
			},
		},
	})

	pipeline = append(pipeline, bson.D{
		{
			Key: "$set",
			Value: bson.M{
				"metadata.count": bson.M{
					"$size": "$runs",
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

	cursor, err := db.RunCollection().Aggregate(context.TODO(), pipeline)
	if err != nil {
		return cleve.RunResult{}, err
	}
	defer cursor.Close(context.TODO())

	for cursor.Next(context.TODO()) {
		var r cleve.RunResult
		err := cursor.Decode(&r)
		if err != nil {
			return cleve.RunResult{}, err
		}
		if r.PaginationMetadata.Page > r.PaginationMetadata.TotalPages {
			return r, fmt.Errorf(
				"page %d is out of range, there are only %d pages",
				r.PaginationMetadata.Page,
				r.PaginationMetadata.TotalPages,
			)
		}
		return r, nil
	}

	err = cursor.Err()
	return cleve.RunResult{}, err
}

func (db DB) Run(runId string, brief bool) (*cleve.Run, error) {
	var run *cleve.Run

	matchStage := bson.D{
		{Key: "$match", Value: bson.D{{Key: "run_id", Value: runId}}},
	}

	sortStage := bson.D{
		{Key: "$set", Value: bson.D{
			{Key: "state_history", Value: bson.D{
				{Key: "$sortArray", Value: bson.M{
					"input": "$state_history", "sortBy": bson.D{{Key: "time", Value: -1}},
				}},
			}},
		},
		},
	}

	// Get samplesheet information
	lookupStage := bson.D{
		{Key: "$lookup", Value: bson.M{
			"from":         "samplesheets",
			"localField":   "run_id",
			"foreignField": "run_id",
			"as":           "samplesheet",
			"pipeline": bson.A{
				bson.M{
					"$project": bson.M{
						"path":              1,
						"modification_time": 1,
					},
				},
			},
		}},
	}

	// Unwind array of samplesheets
	unwindStage := bson.D{
		{Key: "$unwind", Value: bson.M{
			"path":                       "$samplesheet",
			"preserveNullAndEmptyArrays": true,
		}},
	}

	// Count number of analyses
	setStage := bson.D{
		{Key: "$set", Value: bson.D{
			{
				Key: "analysis_count",
				Value: bson.D{
					{Key: "$cond", Value: bson.M{
						"if": bson.D{
							{Key: "$isArray", Value: "$analysis"},
						}, "then": bson.D{
							{Key: "$size", Value: "$analysis"},
						}, "else": 0,
					}},
				},
			},
		}},
	}

	unsetStage := bson.D{
		{Key: "$unset", Value: bson.A{
			"run_parameters",
			"analysis",
		}},
	}

	var aggPipeline mongo.Pipeline
	if brief {
		aggPipeline = mongo.Pipeline{matchStage, setStage, unsetStage, sortStage, lookupStage, unwindStage}
	} else {
		aggPipeline = mongo.Pipeline{matchStage, setStage, sortStage, lookupStage, unwindStage}
	}

	coll := db.RunCollection()
	cursor, err := coll.Aggregate(context.TODO(), aggPipeline)
	if err != nil {
		return run, err
	}

	ok := cursor.Next(context.TODO())
	if !ok {
		return run, mongo.ErrNoDocuments
	}

	if err = cursor.Decode(&run); err != nil {
		return run, err
	}
	err = cursor.Decode(&run)
	return run, err
}

func (db DB) CreateRun(r *cleve.Run) error {
	r.Created = time.Now()
	r.ID = primitive.NewObjectID()
	_, err := db.RunCollection().InsertOne(context.TODO(), r)
	return err
}

func (db DB) DeleteRun(runId string) error {
	res, err := db.RunCollection().DeleteOne(context.TODO(), bson.D{{Key: "run_id", Value: runId}})
	if err == nil && res.DeletedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return err
}

func (db DB) SetRunState(runId string, state cleve.RunState) error {
	runState := cleve.TimedRunState{State: state, Time: time.Now()}
	update := bson.D{{Key: "$push", Value: bson.D{{Key: "state_history", Value: runState}}}}
	result, err := db.RunCollection().UpdateOne(context.TODO(), bson.D{{Key: "run_id", Value: runId}}, update)
	if err == nil && result.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return err
}

func (db DB) SetRunPath(runId string, path string) error {
	dirStat, err := os.Stat(path)
	if err != nil {
		return err
	}

	if !dirStat.IsDir() {
		return fmt.Errorf("not a directory: %s", path)
	}

	update := bson.D{{Key: "$set", Value: bson.D{{Key: "path", Value: path}}}}
	result, err := db.RunCollection().UpdateOne(context.TODO(), bson.D{{Key: "run_id", Value: runId}}, update)
	if err == nil && result.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}

	return nil
}

func (db DB) GetRunStateHistory(runId string) ([]cleve.TimedRunState, error) {
	opts := options.FindOne().SetProjection(bson.D{{Key: "state_history", Value: 1}})
	res := db.RunCollection().FindOne(context.TODO(), bson.D{{Key: "run_id", Value: runId}}, opts)

	var stateHistory []cleve.TimedRunState
	err := res.Decode(&stateHistory)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return stateHistory, nil
		}
		return stateHistory, err
	}

	return stateHistory, nil
}

func (db DB) RunIndex() ([]map[string]string, error) {
	cursor, err := db.RunCollection().Indexes().List(context.TODO())
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

func (db DB) SetRunIndex() (string, error) {
	indexModel := mongo.IndexModel{
		Keys: bson.D{
			{Key: "run_id", Value: 1},
		},
		Options: options.Index().SetUnique(true),
	}

	// TODO: do this as a transaction and roll back if anything fails
	res, err := db.RunCollection().Indexes().DropAll(context.TODO())
	if err != nil {
		return "", err
	}

	log.Printf("Dropped %d indexes\n", res.Lookup("nIndexesWas").Int32())

	name, err := db.RunCollection().Indexes().CreateOne(context.TODO(), indexModel)
	return name, err
}

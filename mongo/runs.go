package mongo

import (
	"context"
	"fmt"
	"github.com/gmc-norr/cleve"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"time"
)

type RunService struct {
	coll *mongo.Collection
}

func (s *RunService) All(filter cleve.RunFilter) ([]*cleve.Run, error) {
	var runs []*cleve.Run

	var aggPipeline mongo.Pipeline

	// Filter on run id
	if filter.RunID != "" {
		aggPipeline = append(aggPipeline, bson.D{
			{Key: "$match", Value: bson.D{
				{Key: "$expr", Value: bson.D{
					{Key: "$regexMatch", Value: bson.M{
						"input": "$run_id",
						"regex": filter.RunID,
						"options": "i",
					}},
				}},
			}},
		})
	}

	// Filter on platform
	if filter.Platform != "" {
		aggPipeline = append(aggPipeline, bson.D{
			{Key: "$match", Value: bson.D{
				{Key: "platform", Value: filter.Platform},
			}},
		})
	}

	// Sort state history chronologically
	aggPipeline = append(aggPipeline, bson.D{
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
		aggPipeline = append(aggPipeline, bson.D{
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

	// Count number of analyses
	aggPipeline = append(aggPipeline, bson.D{
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

	// Sort by sequencing date
	aggPipeline = append(aggPipeline, bson.D{
		{Key: "$sort", Value: bson.D{
			{Key: "run_info.run.date", Value: -1},
		}},
	})

	// Exclude run parameters and analysis
	if filter.Brief {
		aggPipeline = append(aggPipeline, bson.D{
			{Key: "$unset", Value: bson.A{"run_parameters", "analysis"}},
		})
	}

	cursor, err := s.coll.Aggregate(context.TODO(), aggPipeline)
	if err != nil {
		return runs, err
	}
	defer cursor.Close(context.TODO())

	for cursor.Next(context.TODO()) {
		var r cleve.Run
		err := cursor.Decode(&r)
		if err != nil {
			return runs, err
		}
		runs = append(runs, &r)
	}

	if err := cursor.Err(); err != nil {
		return runs, err
	}

	if len(runs) == 0 {
		return []*cleve.Run{}, nil
	}

	return runs, nil
}

func (s *RunService) Get(runId string, brief bool) (*cleve.Run, error) {
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
		aggPipeline = mongo.Pipeline{matchStage, setStage, unsetStage, sortStage}
	} else {
		aggPipeline = mongo.Pipeline{matchStage, setStage, sortStage}
	}

	cursor, err := s.coll.Aggregate(context.TODO(), aggPipeline)
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

func (s *RunService) Create(r *cleve.Run) error {
	r.Created = time.Now()
	r.ID = primitive.NewObjectID()
	_, err := s.coll.InsertOne(context.TODO(), r)
	return err
}

func (s *RunService) Delete(runId string) error {
	res, err := s.coll.DeleteOne(context.TODO(), bson.D{{Key: "run_id", Value: runId}})
	if err == nil && res.DeletedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return err
}

func (s *RunService) SetState(runId string, state cleve.RunState) error {
	runState := cleve.TimedRunState{State: state, Time: time.Now()}
	update := bson.D{{Key: "$push", Value: bson.D{{Key: "state_history", Value: runState}}}}
	result, err := s.coll.UpdateOne(context.TODO(), bson.D{{Key: "run_id", Value: runId}}, update)
	if err == nil && result.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return err
}

func (s *RunService) GetStateHistory(runId string) ([]cleve.TimedRunState, error) {
	opts := options.FindOne().SetProjection(bson.D{{Key: "state_history", Value: 1}})
	res := s.coll.FindOne(context.TODO(), bson.D{{Key: "run_id", Value: runId}}, opts)

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

func (s *RunService) GetIndex() ([]map[string]string, error) {
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

func (s *RunService) SetIndex() (string, error) {
	indexModel := mongo.IndexModel{
		Keys: bson.D{
			{Key: "run_id", Value: 1},
		},
		Options: options.Index().SetUnique(true),
	}

	// TODO: do this as a transaction and roll back if anything fails
	res, err := s.coll.Indexes().DropAll(context.TODO())
	if err != nil {
		return "", err
	}

	log.Printf("Dropped %d indexes\n", res.Lookup("nIndexesWas").Int32())

	name, err := s.coll.Indexes().CreateOne(context.TODO(), indexModel)
	return name, err
}

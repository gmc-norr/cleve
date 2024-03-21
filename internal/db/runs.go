package db

import (
	"context"
	"errors"
	"fmt"
	"github.com/gmc-norr/cleve/internal/db/runstate"
	"github.com/gmc-norr/cleve/runparameters"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"time"
)

type Run struct {
	ID             primitive.ObjectID          `bson:"_id" json:"id"`
	RunID          string                      `bson:"run_id" json:"run_id"`
	ExperimentName string                      `bson:"experiment_name" json:"experiment_name"`
	Path           string                      `bson:"path" json:"path"`
	Platform       string                      `bson:"platform" json:"platform"`
	Created        time.Time                   `bson:"created" json:"created"`
	StateHistory   []runstate.TimedRunState    `bson:"state_history" json:"state_history"`
	RunParameters  runparameters.RunParameters `bson:"run_parameters" json:"run_parameters"`
}

func (*Run) New() (Run, error) {
	return Run{}, errors.New("not implemented")
}

func (r *Run) UnmarshalBSONValue(t bsontype.Type, data []byte) error {
	var rawData bson.Raw
	err := bson.Unmarshal(data, &rawData)
	if err != nil {
		return err
	}

	r.ID = rawData.Lookup("_id").ObjectID()
	r.RunID = rawData.Lookup("run_id").StringValue()
	r.ExperimentName = rawData.Lookup("experiment_name").StringValue()
	r.Path = rawData.Lookup("path").StringValue()
	r.Platform = rawData.Lookup("platform").StringValue()
	r.Created = rawData.Lookup("created").Time()

	var stateHistory []runstate.TimedRunState
	err = rawData.Lookup("state_history").Unmarshal(&stateHistory)
	if err != nil {
		return err
	}

	r.StateHistory = stateHistory

	rp := rawData.Lookup("run_parameters")

	switch r.Platform {
	case "NextSeq":
		var nextSeqRP runparameters.NextSeqParameters
		if err = rp.Unmarshal(&nextSeqRP); err != nil {
			return err
		}
		r.RunParameters = nextSeqRP
	case "NovaSeq":
		var novaSeqRP runparameters.NovaSeqParameters
		if err = rp.Unmarshal(&novaSeqRP); err != nil {
			return err
		}
		r.RunParameters = novaSeqRP
	default:
		r.RunParameters = nil
	}

	if err != nil {
		return err
	}
	return nil
}

func GetRuns() ([]*Run, error) {
	var runs []*Run
	opts := options.Find().SetSort(bson.D{{Key: "created", Value: -1}})
	cursor, err := RunCollection.Find(context.TODO(), bson.D{}, opts)
	defer cursor.Close(context.TODO())

	if err != nil {
		return runs, err
	}

	for cursor.Next(context.TODO()) {
		var r Run
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
		return []*Run{}, nil
	}

	return runs, nil
}

func GetRun(runId string) (*Run, error) {
	var run *Run

	res := RunCollection.FindOne(context.TODO(), bson.D{{Key: "run_id", Value: runId}})
	err := res.Decode(&run)
	if err != nil {
		return run, err
	}

	return run, err
}

func AddRun(r *Run) error {
	r.Created = time.Now()
	r.ID = primitive.NewObjectID()
	_, err := RunCollection.InsertOne(context.TODO(), r)
	return err
}

func DeleteRun(runId string) error {
	res, err := RunCollection.DeleteOne(context.TODO(), bson.D{{Key: "run_id", Value: runId}})
	if err == nil && res.DeletedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return err
}

func UpdateRunState(runId string, state runstate.RunState) error {
	runState := runstate.TimedRunState{State: state, Time: time.Now()}
	update := bson.D{{Key: "$push", Value: bson.D{{Key: "state_history", Value: runState}}}}
	result, err := RunCollection.UpdateOne(context.TODO(), bson.D{{Key: "run_id", Value: runId}}, update)
	if err == nil && result.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return err
}

func GetStateHistory(runId string) ([]runstate.TimedRunState, error) {
	opts := options.FindOne().SetProjection(bson.D{{Key: "state_history", Value: 1}})
	res := RunCollection.FindOne(context.TODO(), bson.D{{Key: "run_id", Value: runId}}, opts)

	var stateHistory []runstate.TimedRunState
	err := res.Decode(&stateHistory)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return stateHistory, nil
		}
		return stateHistory, err
	}

	return stateHistory, nil
}

func GetRunIndex() ([]map[string]string, error) {
	cursor, err := RunCollection.Indexes().List(context.TODO())
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

func SetRunIndex() (string, error) {
	indexModel := mongo.IndexModel{
		Keys: bson.D{
			{Key: "run_id", Value: 1},
		},
		Options: options.Index().SetUnique(true),
	}

	// TODO: do this as a transaction and roll back if anything fails
	res, err := RunCollection.Indexes().DropAll(context.TODO())
	if err != nil {
		return "", err
	}

	log.Printf("Dropped %d indexes\n", res.Lookup("nIndexesWas").Int32())

	name, err := RunCollection.Indexes().CreateOne(context.TODO(), indexModel)
	return name, err
}

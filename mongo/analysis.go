package mongo

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/gmc-norr/cleve"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (db DB) Analyses(filter cleve.AnalysisFilter) (cleve.AnalysisResult, error) {
	var (
		analyses cleve.AnalysisResult
		pipeline mongo.Pipeline
	)

	analyses.PaginationMetadata = cleve.PaginationMetadata{
		Page:     filter.Page,
		PageSize: filter.PageSize,
	}

	if filter.AnalysisId != "" {
		pipeline = append(pipeline, bson.D{
			{Key: "$match", Value: bson.D{
				{Key: "analysis_id", Value: filter.AnalysisId},
			}},
		})
	}

	if filter.RunId != "" {
		pipeline = append(pipeline, bson.D{
			{Key: "$match", Value: bson.D{
				{Key: "runs", Value: filter.RunId},
			}},
		})
	}

	if filter.Software != "" {
		pipeline = append(pipeline, bson.D{
			{Key: "$match", Value: bson.D{
				{Key: "software", Value: filter.Software},
			}},
		})
	}

	pipeline = append(pipeline, bson.D{
		{Key: "$set", Value: bson.D{
			{Key: "state_history", Value: bson.D{
				{Key: "$sortArray", Value: bson.M{
					"input": "$state_history", "sortBy": bson.D{{Key: "time", Value: -1}},
				}},
			}},
		}},
	})

	if filter.State.IsValid() {
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

	metaPipeline := append(pipeline, bson.D{
		{Key: "$count", Value: "total_count"},
	})

	cursor, err := db.AnalysesCollection().Aggregate(context.TODO(), metaPipeline)
	if err != nil {
		return analyses, err
	}
	defer closeCursor(cursor, context.TODO())

	if !cursor.Next(context.TODO()) {
		analyses.TotalCount = 0
	}
	if err := cursor.Decode(&analyses.PaginationMetadata); analyses.TotalCount > 0 && err != nil {
		return analyses, err
	}

	if analyses.PageSize > 0 {
		analyses.TotalPages = analyses.TotalCount / analyses.PageSize
		if analyses.TotalCount%analyses.PageSize > 0 {
			analyses.TotalPages += 1
		}
	} else {
		analyses.TotalPages = 1
	}

	if filter.Page > 0 {
		pipeline = append(pipeline, bson.D{
			{Key: "$skip", Value: filter.PageSize * (filter.Page - 1)},
		})
	}

	if filter.PageSize > 0 {
		pipeline = append(pipeline, bson.D{
			{Key: "$limit", Value: filter.PageSize},
		})
	}

	cursor, err = db.AnalysesCollection().Aggregate(context.TODO(), pipeline)
	if err != nil {
		return analyses, err
	}
	defer closeCursor(cursor, context.TODO())

	for cursor.Next(context.TODO()) {
		var analysis cleve.Analysis
		err = cursor.Decode(&analysis)
		if err != nil {
			return analyses, err
		}
		if analysis.InputFiles == nil {
			analysis.InputFiles = make([]cleve.AnalysisFileFilter, 0)
		}
		if analysis.OutputFiles == nil {
			analysis.OutputFiles = make([]cleve.AnalysisFile, 0)
		}
		analyses.Count += 1
		analyses.Analyses = append(analyses.Analyses, &analysis)
	}

	if analyses.TotalCount == 0 {
		analyses.TotalPages = 1
		analyses.Analyses = make([]*cleve.Analysis, 0)
	}
	if analyses.Page > analyses.TotalPages {
		return analyses, PageOutOfBoundsError{
			page:       analyses.Page,
			totalPages: analyses.TotalPages,
		}
	}

	return analyses, nil
}

// Analysis fetches a single analysis based on its ID. An optional run ID constraint can be given
// as the second argument in order to constrain the anlyses to a particular run. If more than one
// run ID is given, a non-nil error will be returned. If no documents are found given the
// analysis ID and any run ID constraint, a `mongo.ErrNoDocuments` error will be returned.
func (db DB) Analysis(analysisId string, runId ...string) (*cleve.Analysis, error) {
	if len(runId) > 1 {
		return nil, fmt.Errorf("only a single run ID can be given")
	}
	filter := cleve.NewAnalysisFilter()
	filter.AnalysisId = analysisId
	if len(runId) == 1 {
		filter.RunId = runId[0]
	}
	analyses, err := db.Analyses(filter)
	if err != nil {
		return nil, err
	}
	if analyses.Count == 0 {
		return nil, mongo.ErrNoDocuments
	}
	return analyses.Analyses[0], nil
}

func (db DB) CreateAnalysis(analysis *cleve.Analysis) error {
	type aux struct {
		Created         time.Time `bson:"created"`
		Updated         time.Time `bson:"updated"`
		*cleve.Analysis `bson:",inline"`
	}
	auxAnalysis := aux{
		Created:  time.Now(),
		Updated:  time.Now(),
		Analysis: analysis,
	}
	_, err := db.AnalysesCollection().InsertOne(
		context.TODO(),
		auxAnalysis,
	)
	return err
}

func (db DB) SetAnalysisState(analysisId string, parentId string, state cleve.State) error {
	filter := bson.D{{Key: "analysis_id", Value: analysisId}, {Key: "parent_id", Value: parentId}}
	update := bson.D{
		{Key: "$push", Value: bson.D{
			{Key: "state_history", Value: cleve.TimedRunState{
				State: state,
				Time:  time.Now(),
			}},
		}},
		{Key: "$set", Value: bson.D{
			{Key: "updated", Value: time.Now()},
		}},
	}
	res, err := db.AnalysesCollection().UpdateOne(context.TODO(), filter, update)
	if err == nil && res.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return err
}

func (db DB) SetAnalysisPath(analysisId string, parentId string, path string) error {
	filter := bson.D{{Key: "analysis_id", Value: analysisId}, {Key: "parent_id", Value: parentId}}
	update := bson.D{
		{Key: "$set", Value: bson.D{
			{Key: "path", Value: path},
			{Key: "updated", Value: time.Now()},
		}},
	}
	res, err := db.AnalysesCollection().UpdateOne(context.TODO(), filter, update)
	if err != nil && res.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return err
}

func (db DB) SetAnalysisFiles(analysisId string, parentId string, files []cleve.AnalysisFile) error {
	filter := bson.D{{Key: "analysis_id", Value: analysisId}, {Key: "parent_id", Value: parentId}}
	update := bson.D{
		{Key: "$set", Value: bson.D{
			{Key: "files", Value: files},
			{Key: "updated", Value: time.Now()},
		}},
	}
	res, err := db.AnalysesCollection().UpdateOne(context.TODO(), filter, update)
	if err != nil && res.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return err
}

func (db DB) AnalysesIndex() ([]map[string]string, error) {
	cursor, err := db.AnalysesCollection().Indexes().List(context.TODO())
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

func (db DB) SetAnalysesIndex() (string, error) {
	indexModel := mongo.IndexModel{
		Keys: bson.D{
			{Key: "analysis_id", Value: 1},
		},
		Options: options.Index().SetUnique(true),
	}

	// TODO: do this as a transaction and roll back if anything fails
	res, err := db.AnalysesCollection().Indexes().DropAll(context.TODO())
	if err != nil {
		return "", err
	}

	slog.Info("dropped indexes", "count", res.Lookup("nIndexesWas").Int32())

	name, err := db.AnalysesCollection().Indexes().CreateOne(context.TODO(), indexModel)
	return name, err
}

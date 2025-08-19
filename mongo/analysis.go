package mongo

import (
	"context"
	"log/slog"

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

	if filter.ParentId != "" {
		pipeline = append(pipeline, bson.D{
			{Key: "$match", Value: bson.D{
				{Key: "parent_id", Value: filter.ParentId},
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

	if filter.State != cleve.StateInvalid {
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

	if filter.Level != cleve.LevelInvalid {
		pipeline = append(pipeline, bson.D{
			{Key: "$match", Value: bson.D{
				{Key: "level", Value: filter.Level},
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

func (db DB) Analysis(analysisId string, parentId string) (*cleve.Analysis, error) {
	filter := cleve.NewAnalysisFilter()
	filter.AnalysisId = analysisId
	filter.ParentId = parentId
	analyses, err := db.Analyses(filter)
	if err != nil {
		return nil, err
	}
	if analyses.Count == 0 {
		return nil, mongo.ErrNoDocuments
	}
	return analyses.Analyses[0], nil
}

func (db DB) CreateAnalysis(runId string, analysis *cleve.Analysis) error {
	update := bson.D{{
		Key: "$push", Value: bson.D{
			{Key: "analysis", Value: analysis},
		},
	}}
	res, err := db.RunCollection().UpdateOne(
		context.TODO(),
		bson.D{{Key: "run_id", Value: runId}},
		update,
	)

	if err == nil && res.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}

	return nil
}

func (db DB) SetAnalysisState(runId string, analysisId string, state cleve.State) error {
	filter := bson.D{{Key: "run_id", Value: runId}, {Key: "analysis.analysis_id", Value: analysisId}}
	update := bson.D{{
		Key: "$set", Value: bson.D{
			{Key: "analysis.$.state", Value: state.String()},
		},
	}}

	res, err := db.RunCollection().UpdateOne(context.TODO(), filter, update)
	if err == nil && res.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}

	return err
}

func (db DB) SetAnalysisPath(runId string, analysisId string, path string) error {
	filter := bson.D{{Key: "run_id", Value: runId}, {Key: "analysis.analysis_id", Value: analysisId}}
	update := bson.D{{
		Key: "$set", Value: bson.D{
			{Key: "analysis.$.path", Value: path},
		},
	}}
	res, err := db.RunCollection().UpdateOne(context.TODO(), filter, update)
	if err != nil && res.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return err
}

// func (db DB) SetAnalysisSummary(runId string, analysisId string, summary *cleve.AnalysisSummary) error {
// 	filter := bson.D{{Key: "run_id", Value: runId}, {Key: "analysis.analysis_id", Value: analysisId}}
// 	update := bson.D{{
// 		Key: "$set", Value: bson.D{
// 			{Key: "analysis.$.summary", Value: summary},
// 		},
// 	}}
//
// 	res, err := db.RunCollection().UpdateOne(context.TODO(), filter, update)
// 	if err == nil && res.MatchedCount == 0 {
// 		return mongo.ErrNoDocuments
// 	}
//
// 	return err
// }

func (db DB) SetAnalysesIndex() (string, error) {
	indexModel := mongo.IndexModel{
		Keys: bson.D{
			{Key: "analysis_id", Value: 1},
			{Key: "parent_id", Value: 1},
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

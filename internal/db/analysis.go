package db

import (
	"context"
	"github.com/gmc-norr/cleve"
	"github.com/gmc-norr/cleve/analysis"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func GetAnalyses(runId string) ([]*analysis.Analysis, error) {
	run, err := GetRun(runId, false)
	if err != nil {
		return nil, err
	}

	analyses := run.Analysis
	if analyses == nil {
		return []*analysis.Analysis{}, nil
	}
	return analyses, nil
}

func GetAnalysis(runId string, analysisId string) (*analysis.Analysis, error) {
	analyses, err := GetAnalyses(runId)
	if err != nil {
		return nil, err
	}

	for _, analysis := range analyses {
		if analysis.AnalysisId == analysisId {
			return analysis, nil
		}
	}

	return nil, mongo.ErrNoDocuments
}

func AddAnalysis(runId string, analysis *analysis.Analysis) error {
	update := bson.D{{
		Key: "$push", Value: bson.D{
			{Key: "analysis", Value: analysis},
		},
	}}
	res, err := RunCollection.UpdateOne(
		context.TODO(),
		bson.D{{Key: "run_id", Value: runId}},
		update,
	)

	if err == nil && res.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}

	return nil
}

func UpdateAnalysisState(runId string, analysisId string, state cleve.RunState) error {
	filter := bson.D{{Key: "run_id", Value: runId}, {Key: "analysis.analysis_id", Value: analysisId}}
	update := bson.D{{
		Key: "$set", Value: bson.D{
			{Key: "analysis.$.state", Value: state.String()},
		},
	}}

	res, err := RunCollection.UpdateOne(context.TODO(), filter, update)
	if err == nil && res.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}

	return err
}

func UpdateAnalysisSummary(runId string, analysisId string, summary *analysis.AnalysisSummary) error {
	filter := bson.D{{Key: "run_id", Value: runId}, {Key: "analysis.analysis_id", Value: analysisId}}
	update := bson.D{{
		Key: "$set", Value: bson.D{
			{Key: "analysis.$.summary", Value: summary},
		},
	}}

	res, err := RunCollection.UpdateOne(context.TODO(), filter, update)
	if err == nil && res.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}

	return err
}

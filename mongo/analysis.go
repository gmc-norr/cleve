package mongo

import (
	"context"
	"github.com/gmc-norr/cleve"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func (s *RunService) GetAnalyses(runId string) ([]*cleve.Analysis, error) {
	run, err := s.Get(runId, false)
	if err != nil {
		return nil, err
	}

	analyses := run.Analysis
	if analyses == nil {
		return []*cleve.Analysis{}, nil
	}
	return analyses, nil
}

func (s *RunService) GetAnalysis(runId string, analysisId string) (*cleve.Analysis, error) {
	analyses, err := s.GetAnalyses(runId)
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

func (s *RunService) CreateAnalysis(runId string, analysis *cleve.Analysis) error {
	update := bson.D{{
		Key: "$push", Value: bson.D{
			{Key: "analysis", Value: analysis},
		},
	}}
	res, err := s.coll.UpdateOne(
		context.TODO(),
		bson.D{{Key: "run_id", Value: runId}},
		update,
	)

	if err == nil && res.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}

	return nil
}

func (s *RunService) SetAnalysisState(runId string, analysisId string, state cleve.RunState) error {
	filter := bson.D{{Key: "run_id", Value: runId}, {Key: "analysis.analysis_id", Value: analysisId}}
	update := bson.D{{
		Key: "$set", Value: bson.D{
			{Key: "analysis.$.state", Value: state.String()},
		},
	}}

	res, err := s.coll.UpdateOne(context.TODO(), filter, update)
	if err == nil && res.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}

	return err
}

func (s *RunService) SetAnalysisSummary(runId string, analysisId string, summary *cleve.AnalysisSummary) error {
	filter := bson.D{{Key: "run_id", Value: runId}, {Key: "analysis.analysis_id", Value: analysisId}}
	update := bson.D{{
		Key: "$set", Value: bson.D{
			{Key: "analysis.$.summary", Value: summary},
		},
	}}

	res, err := s.coll.UpdateOne(context.TODO(), filter, update)
	if err == nil && res.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}

	return err
}

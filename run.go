package cleve

import (
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"log"
	"time"
)

type RunService interface {
	All(bool, string, string) ([]*Run, error)
	Create(*Run) error
	Delete(string) error
	Get(string, bool) (*Run, error)
	GetAnalyses(string) ([]*Analysis, error)
	GetAnalysis(string, string) (*Analysis, error)
	CreateAnalysis(string, *Analysis) error
	SetAnalysisState(string, string, RunState) error
	SetAnalysisSummary(string, string, *AnalysisSummary) error
	GetStateHistory(string) ([]TimedRunState, error)
	SetState(string, RunState) error
	GetIndex() ([]map[string]string, error)
	SetIndex() (string, error)
}

type Run struct {
	ID             primitive.ObjectID `bson:"_id" json:"id"`
	RunID          string             `bson:"run_id" json:"run_id"`
	ExperimentName string             `bson:"experiment_name" json:"experiment_name"`
	Path           string             `bson:"path" json:"path"`
	Platform       string             `bson:"platform" json:"platform"`
	Created        time.Time          `bson:"created" json:"created"`
	StateHistory   []TimedRunState    `bson:"state_history" json:"state_history"`
	RunParameters  RunParameters      `bson:"run_parameters,omitempty" json:"run_parameters,omitempty"`
	Analysis       []*Analysis        `bson:"analysis,omitempty" json:"analysis,omitempty"`
	AnalysisCount  int32              `bson:"analysis_count" json:"analysis_count"`
}

func (r *Run) UnmarshalBSON(data []byte) error {
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
	ac, err := rawData.LookupErr("analysis_count")
	if err == nil {
		r.AnalysisCount = ac.Int32()
	}

	err = rawData.Lookup("state_history").Unmarshal(&r.StateHistory)
	if err != nil {
		return err
	}

	ra := rawData.Lookup("analysis")
	if len(ra.Value) > 0 {
		err = ra.Unmarshal(&r.Analysis)
		if err != nil {
			return err
		}
	}

	if r.Analysis == nil {
		r.Analysis = []*Analysis{}
	}

	rp := rawData.Lookup("run_parameters")

	if len(rp.Value) > 0 {
		switch r.Platform {
		case "NextSeq":
			var nextSeqRP NextSeqParameters
			if err = rp.Unmarshal(&nextSeqRP); err != nil {
				log.Println(err)
				return err
			}
			r.RunParameters = nextSeqRP
		case "NovaSeq":
			var novaSeqRP NovaSeqParameters
			if err = rp.Unmarshal(&novaSeqRP); err != nil {
				log.Println(err)
				return err
			}
			r.RunParameters = novaSeqRP
		default:
			r.RunParameters = nil
		}
	} else {
		r.RunParameters = nil
	}

	return nil
}

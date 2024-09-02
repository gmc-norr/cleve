package cleve

import (
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type RunResult struct {
	PaginationMetadata `bson:"metadata" json:"metadata"`
	Runs               []*Run `bson:"runs" json:"runs"`
}

type Run struct {
	ID             primitive.ObjectID `bson:"_id" json:"id"`
	RunID          string             `bson:"run_id" json:"run_id"`
	ExperimentName string             `bson:"experiment_name" json:"experiment_name"`
	Path           string             `bson:"path" json:"path"`
	Platform       string             `bson:"platform" json:"platform"`
	Created        time.Time          `bson:"created" json:"created"`
	StateHistory   []TimedRunState    `bson:"state_history" json:"state_history"`
	SampleSheet    *SampleSheetInfo   `bson:"samplesheet" json:"samplesheet"`
	RunParameters  RunParameters      `bson:"run_parameters,omitempty" json:"run_parameters,omitempty"`
	RunInfo        RunInfo            `bson:"run_info,omitempty" json:"run_info,omitempty"`
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

	ss := rawData.Lookup("samplesheet")
	if len(ss.Value) > 0 {
		err = ss.Unmarshal(&r.SampleSheet)
		if err != nil {
			return err
		}
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

	ri := rawData.Lookup("run_info")
	if len(ri.Value) > 0 {
		var runInfo RunInfo
		if err = ri.Unmarshal(&runInfo); err != nil {
			return err
		}
		r.RunInfo = runInfo
	}

	return nil
}

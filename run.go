package cleve

import (
	"fmt"
	"strings"
	"time"

	"github.com/gmc-norr/cleve/interop"
	"go.mongodb.org/mongo-driver/bson"
)

type RunResult struct {
	PaginationMetadata `bson:"metadata" json:"metadata"`
	Runs               []*Run `bson:"runs" json:"runs"`
}

// Run represents an Illumina sequencing run.
type Run struct {
	SchemaVersion    int                   `bson:"schema_version" json:"schema_version"`
	RunID            string                `bson:"run_id" json:"run_id"`
	ExperimentName   string                `bson:"experiment_name" json:"experiment_name"`
	Path             string                `bson:"path" json:"path"`
	Platform         string                `bson:"platform" json:"platform"`
	Created          time.Time             `bson:"created" json:"created"`
	StateHistory     []TimedRunState       `bson:"state_history" json:"state_history"`
	SampleSheet      *SampleSheetInfo      `bson:"samplesheet,omitempty" json:"samplesheet"`
	SampleSheetFiles []SampleSheetInfo     `bson:"samplesheets,omitempty" json:"samplesheets"`
	RunParameters    interop.RunParameters `bson:"run_parameters,omitzero" json:"run_parameters,omitzero"`
	RunInfo          interop.RunInfo       `bson:"run_info,omitzero" json:"run_info,omitzero"`
	Analysis         []*Analysis           `bson:"analysis,omitempty" json:"analysis,omitempty"`
	AnalysisCount    int                   `bson:"analysis_count" json:"analysis_count"`
}

// Unmarshals a BSON representation of a run.
// This supports schema version 1 and 2. If the schema verison is not defined in the
// document, it is assumed to be version 1. The goal is to eventually deprecate version 1.
func (r *Run) UnmarshalBSON(data []byte) error {
	var v struct {
		SchemaVersion int `bson:"schema_version"`
	}

	if err := bson.Unmarshal(data, &v); err != nil {
		return err
	}

	if v.SchemaVersion == 0 {
		r.SchemaVersion = 1
	} else {
		r.SchemaVersion = v.SchemaVersion
	}

	switch r.SchemaVersion {
	case 1:
		rpV1, err := unmarshalRunV1(data)
		if err != nil {
			return err
		}
		*r = rpV1
	case 2:
		type RunAlias Run
		if err := bson.Unmarshal(data, (*RunAlias)(r)); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported schema version: %d", r.SchemaVersion)
	}

	return nil
}

func unmarshalRunV1(data []byte) (r Run, err error) {
	type RunV1 struct {
		RunID            string            `bson:"run_id" json:"run_id"`
		ExperimentName   string            `bson:"experiment_name" json:"experiment_name"`
		Path             string            `bson:"path" json:"path"`
		Platform         string            `bson:"platform" json:"platform"`
		Created          time.Time         `bson:"created" json:"created"`
		StateHistory     []TimedRunState   `bson:"state_history" json:"state_history"`
		SampleSheet      *SampleSheetInfo  `bson:"samplesheet,omitempty" json:"samplesheet"`
		SampleSheetFiles []SampleSheetInfo `bson:"samplesheets,omitempty" json:"samplesheets"`
		RunInfo          struct {
			Version int `bson:"version"`
			Run     struct {
				RunID          string    `bson:"runid"`
				Date           time.Time `bson:"date"`
				Instrument     string    `bson:"instrument"`
				Flowcell       string    `bson:"flowcell"`
				FlowcellLayout struct {
					LaneCount      int `bson:"lane_count"`
					SurfaceCount   int `bson:"surface_count"`
					SwathCount     int `bson:"swath_count"`
					TileCount      int `bson:"tile_count"`
					SectionPerLane int `bson:"section_per_lane"`
				} `bson:"flowcell_layout"`
				Reads struct {
					Read []struct {
						Number    int    `bson:"number"`
						Cycles    int    `bson:"cycles"`
						IsIndex   string `bson:"is_indexed_read"`
						IsRevComp string `bson:"is_reverse_complemented"`
					} `bson:""`
				} `bson:"reads"`
			} `bson:"run"`
		} `bson:"run_info,omitzero" json:"run_info,omitzero"`
		Analysis      []*Analysis `bson:"analysis,omitempty" json:"analysis,omitempty"`
		AnalysisCount int         `bson:"analysis_count" json:"analysis_count"`
	}

	v1 := RunV1{}
	if err = bson.Unmarshal(data, &v1); err != nil {
		return r, err
	}

	platform := interop.IdentifyPlatform(v1.RunInfo.Run.Instrument)
	flowcell := interop.IdentifyFlowcell(v1.RunInfo.Run.Flowcell)

	r.RunID = v1.RunID
	r.ExperimentName = v1.ExperimentName
	r.Path = v1.Path
	r.Platform = platform
	r.Created = v1.Created
	r.StateHistory = v1.StateHistory
	r.SampleSheet = v1.SampleSheet
	r.SampleSheetFiles = v1.SampleSheetFiles
	r.Analysis = v1.Analysis
	r.AnalysisCount = v1.AnalysisCount
	r.RunInfo = interop.RunInfo{
		Version:      v1.RunInfo.Version,
		RunId:        v1.RunInfo.Run.RunID,
		Date:         (time.Time)(v1.RunInfo.Run.Date),
		Platform:     platform,
		FlowcellName: flowcell,
		InstrumentId: v1.RunInfo.Run.Instrument,
		FlowcellId:   v1.RunInfo.Run.Flowcell,
		Flowcell: interop.FlowcellInfo{
			Lanes:          v1.RunInfo.Run.FlowcellLayout.LaneCount,
			Surfaces:       v1.RunInfo.Run.FlowcellLayout.SurfaceCount,
			Swaths:         v1.RunInfo.Run.FlowcellLayout.SwathCount,
			Tiles:          v1.RunInfo.Run.FlowcellLayout.TileCount,
			SectionPerLane: v1.RunInfo.Run.FlowcellLayout.SectionPerLane,
		},
	}
	for _, read := range v1.RunInfo.Run.Reads.Read {
		isIndex := read.IsIndex == "Y"
		isRevComp := read.IsRevComp == "Y"
		readName := fmt.Sprintf("Read %d", read.Number)
		if isIndex {
			readName += " (I)"
		}
		r.RunInfo.Reads = append(r.RunInfo.Reads, interop.ReadInfo{
			Name:      readName,
			Number:    read.Number,
			Cycles:    read.Cycles,
			IsIndex:   isIndex,
			IsRevComp: isRevComp,
		})
	}

	// Conform old run parameters to current format
	if strings.HasPrefix(platform, "NovaSeq") {
		novaseqRunparameters, err := unmarshalNovaSeqV1RunParameters(data)
		if err != nil {
			return r, err
		}
		r.RunParameters = novaseqRunparameters
	} else if strings.HasPrefix(platform, "NextSeq") {
		nextseqRunparameters, err := unmarshalNextSeqV1RunParameters(data)
		if err != nil {
			return r, err
		}
		r.RunParameters = nextseqRunparameters
	} else {
		return r, fmt.Errorf("unknown run parameter format")
	}

	return r, nil
}

func unmarshalNovaSeqV1RunParameters(data []byte) (rp interop.RunParameters, err error) {
	type auxRunParams struct {
		ExperimentName string `bson:"experimentname"`
		Side           string `bson:"side"`
		Consumables    []struct {
			Type           string    `bson:"type"`
			Name           string    `bson:"name"`
			Version        string    `bson:"version"`
			SerialNumber   string    `bson:"serialnumber"`
			PartNumber     string    `bson:"partnumber"`
			LotNumber      string    `bson:"lotnumber"`
			ExpirationDate time.Time `bson:"expirationdate"`
			Mode           int       `bson:"mode"`
		} `bson:"consumableinfo"`
	}
	params := struct {
		auxRunParams `bson:"run_parameters"`
	}{}
	err = bson.Unmarshal(data, &params)
	if err != nil {
		return rp, err
	}
	rp = interop.RunParameters{
		ExperimentName: params.ExperimentName,
		Side:           params.Side,
	}
	for _, consumable := range params.Consumables {
		interopConsumable := interop.Consumable{
			Type:           consumable.Type,
			Name:           consumable.Name,
			Version:        consumable.Version,
			Mode:           consumable.Mode,
			SerialNumber:   consumable.SerialNumber,
			PartNumber:     consumable.PartNumber,
			LotNumber:      consumable.LotNumber,
			ExpirationDate: time.Time(consumable.ExpirationDate),
		}
		if strings.ToLower(consumable.Type) == "flowcell" {
			rp.Flowcell = interopConsumable
		} else {
			rp.Consumables = append(rp.Consumables, interopConsumable)
		}
	}

	return rp, nil
}

func unmarshalNextSeqV1RunParameters(data []byte) (rp interop.RunParameters, err error) {
	type auxRunParams struct {
		ExperimentName  string `bson:"experimentname"`
		FlowcellRfidTag struct {
			SerialNumber   string    `bson:"serialnumber"`
			PartNumber     string    `bson:"partnumber"`
			LotNumber      string    `bson:"lotnumber"`
			ExpirationDate time.Time `bson:"expirationdate"`
		} `bson:"flowcellrfidtag"`
		PR2BottleRfidTag struct {
			SerialNumber   string    `bson:"serialnumber"`
			PartNumber     string    `bson:"partnumber"`
			LotNumber      string    `bson:"lotnumber"`
			ExpirationDate time.Time `bson:"expirationdate"`
		} `bson:"pr2bottlerfidtag"`
		ReagentKitRfidTag struct {
			SerialNumber   string    `bson:"serialnumber"`
			PartNumber     string    `bson:"partnumber"`
			LotNumber      string    `bson:"lotnumber"`
			ExpirationDate time.Time `bson:"expirationdate"`
		} `bson:"reagentrfidtag"`
	}
	params := struct {
		auxRunParams `bson:"run_parameters"`
	}{}
	err = bson.Unmarshal(data, &params)
	if err != nil {
		return rp, err
	}
	rp = interop.RunParameters{
		ExperimentName: params.ExperimentName,
	}
	rp.Flowcell = interop.Consumable{
		Type:           "FlowCell",
		SerialNumber:   params.FlowcellRfidTag.SerialNumber,
		PartNumber:     params.FlowcellRfidTag.PartNumber,
		LotNumber:      params.FlowcellRfidTag.LotNumber,
		ExpirationDate: time.Time(params.FlowcellRfidTag.ExpirationDate),
	}
	rp.Consumables = []interop.Consumable{
		{
			Type:           "Buffer",
			SerialNumber:   params.PR2BottleRfidTag.SerialNumber,
			PartNumber:     params.PR2BottleRfidTag.PartNumber,
			LotNumber:      params.PR2BottleRfidTag.LotNumber,
			ExpirationDate: time.Time(params.PR2BottleRfidTag.ExpirationDate),
		},
		{
			Type:           "Reagent",
			SerialNumber:   params.ReagentKitRfidTag.SerialNumber,
			PartNumber:     params.ReagentKitRfidTag.PartNumber,
			LotNumber:      params.ReagentKitRfidTag.LotNumber,
			ExpirationDate: time.Time(params.ReagentKitRfidTag.ExpirationDate),
		},
	}
	return rp, nil
}

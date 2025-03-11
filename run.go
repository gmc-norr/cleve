package cleve

import (
	"fmt"
	"log"
	"strconv"
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
		type RunV1 struct {
			RunID            string            `bson:"run_id" json:"run_id"`
			ExperimentName   string            `bson:"experiment_name" json:"experiment_name"`
			Path             string            `bson:"path" json:"path"`
			Platform         string            `bson:"platform" json:"platform"`
			Created          time.Time         `bson:"created" json:"created"`
			StateHistory     []TimedRunState   `bson:"state_history" json:"state_history"`
			SampleSheet      *SampleSheetInfo  `bson:"samplesheet,omitempty" json:"samplesheet"`
			SampleSheetFiles []SampleSheetInfo `bson:"samplesheets,omitempty" json:"samplesheets"`
			RunInfo          RunInfo           `bson:"run_info,omitzero" json:"run_info,omitzero"`
			Analysis         []*Analysis       `bson:"analysis,omitempty" json:"analysis,omitempty"`
			AnalysisCount    int               `bson:"analysis_count" json:"analysis_count"`
		}

		v1 := RunV1{}
		if err := bson.Unmarshal(data, &v1); err != nil {
			return err
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
			isIndex := read.IsIndexedRead == "Y"
			isRevComp := read.IsReverseComplement == "Y"
			readName := fmt.Sprintf("Read %d", read.Number)
			if isIndex {
				readName += " (I)"
			}
			r.RunInfo.Reads = append(r.RunInfo.Reads, interop.ReadInfo{
				Name:      readName,
				Number:    read.Number,
				Cycles:    read.NumCycles,
				IsIndex:   isIndex,
				IsRevComp: isRevComp,
			})
		}

		// Conform old run parameters to current format
		if strings.HasPrefix(platform, "NovaSeq") {
			novaSeqParams := struct {
				NovaSeqParameters `bson:"run_parameters"`
			}{
				NovaSeqParameters{},
			}
			err := bson.Unmarshal(data, &novaSeqParams)
			if err != nil {
				return err
			}
			r.RunParameters = interop.RunParameters{
				ExperimentName: novaSeqParams.ExperimentName,
				Side:           novaSeqParams.Side,
			}
			for _, consumable := range novaSeqParams.ConsumableInfo {
				mode, err := strconv.Atoi(consumable.Mode)
				if err != nil {
					mode = 0
					log.Println("warning: unable to parse consumable mode")
				}
				interopConsumable := interop.Consumable{
					Type:           consumable.Type,
					Name:           consumable.Name,
					Version:        consumable.Version,
					Mode:           mode,
					SerialNumber:   consumable.SerialNumber,
					PartNumber:     consumable.PartNumber,
					LotNumber:      consumable.LotNumber,
					ExpirationDate: time.Time(consumable.ExpirationDate),
				}
				if strings.ToLower(consumable.Type) == "flowcell" {
					r.RunParameters.Flowcell = interopConsumable
				} else {
					r.RunParameters.Consumables = append(r.RunParameters.Consumables, interopConsumable)
				}
			}
		} else if strings.HasPrefix(platform, "NextSeq") {
			nextSeqParams := struct {
				NextSeqParameters `bson:"run_parameters"`
			}{
				NextSeqParameters{},
			}
			err := bson.Unmarshal(data, &nextSeqParams)
			if err != nil {
				return err
			}
			r.RunParameters = interop.RunParameters{
				ExperimentName: nextSeqParams.ExperimentName,
			}
			r.RunParameters.Flowcell = interop.Consumable{
				Type:           "FlowCell",
				SerialNumber:   nextSeqParams.FlowCellRfidTag.SerialNumber,
				PartNumber:     nextSeqParams.FlowCellRfidTag.PartNumber,
				LotNumber:      nextSeqParams.FlowCellRfidTag.LotNumber,
				ExpirationDate: time.Time(nextSeqParams.FlowCellRfidTag.ExpirationDate),
			}
			r.RunParameters.Consumables = []interop.Consumable{
				{
					Type:           "Buffer",
					SerialNumber:   nextSeqParams.PR2BottleRfidTag.SerialNumber,
					PartNumber:     nextSeqParams.PR2BottleRfidTag.PartNumber,
					LotNumber:      nextSeqParams.PR2BottleRfidTag.LotNumber,
					ExpirationDate: time.Time(nextSeqParams.PR2BottleRfidTag.ExpirationDate),
				},
				{
					Type:           "Reagent",
					SerialNumber:   nextSeqParams.ReagentKitRfidTag.SerialNumber,
					PartNumber:     nextSeqParams.ReagentKitRfidTag.PartNumber,
					LotNumber:      nextSeqParams.ReagentKitRfidTag.LotNumber,
					ExpirationDate: time.Time(nextSeqParams.ReagentKitRfidTag.ExpirationDate),
				},
			}
		} else {
			return fmt.Errorf("unknown run parameter format")
		}
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

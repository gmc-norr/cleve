package cleve

import (
	"time"
	"errors"
	"strings"
)

type PanelQcResult struct {
	PaginationMetadata                             `bson:"metadata" json:"metadata"`
	PanelQCs                                       []*PanelQc `bson:"runs" json:"runs"`
}

type PanelQc struct {
	RunId               string                      `bson:"run_id"`
	Created             time.Time                   `bson:"created"`
	GenePanel           []GenePanelQc               `bson:"genepanel"`
	
}

type GenePanelQc struct {
	GenePanelId               string                 `bson:"genepanel_id"`
	CompletnessThreshold      float64                `bson:"completness_threshold"`
	FeatureFraction           float64                `bson:"feature_fraction"`
	NumberSamples             int                    `bson:"number_of_samples"`
	Genes                     []GeneQC               `bson:"genes"`
}

type GeneQC struct {
	HGNC                  string                      `bson:"hgnc"`
	Symbol                string			          `bson:"symbol"`
	Mean_coverage         float64	   	              `bson:"mean_coverage"`
	Mean_completness      float64		              `bson:"mean_completness"`
	Mean_mapping_quality  float64					  `bson:"mean_mapping_quality"`
	Suboptional_coverage  bool					      `bson:"suboptional_coverage"`
}



func (p PanelQc) Validate() error {
	if p.RunId == "" {
		return errors.New("panelQC must have a Runid")
	}
	if len(p.GenePanel) == 0 {
		return errors.New("panel must contain at least one GenePanel")
	}
	for i, g := range p.GenePanel {
		if len(g.Genes) == 0 {
			return errors.New("Missing genes for at least one Gene Panel")
		}
		if strings.Contains(g.GenePanelId, ".") {
			p.GenePanel[i].GenePanelId = strings.ReplaceAll(p.GenePanel[i].GenePanelId, ".", "_")
		}
	}
	return nil
}
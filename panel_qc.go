package cleve

import (
	"time"
	"errors"
)

type PanelQcResult struct {
	PaginationMetadata                             `bson:"metadata" json:"metadata"`
	PanelQCs                                       []*PanelQc `bson:"runs" json:"runs"`
}

type PanelQc struct {
	RunId               string                      `bson:"run_id"`
	Created             time.Time                   `bson:"created"`
	GenePanel           []GenePanelQc               `bson:"genepanel,omitempty"`
	
}

//  Add PanelName
type GenePanelQc struct {
	GenePanelId               string                 `bson:"genepanel_id"`
	CompletnessThreshold      float64                `bson:"completness_threshold"`
	FeatureFraction           float64                `bson:"feature_fraction"`
	NumberSamples             int                    `bson:"number_of_samples"`
	Genes                     []GeneQc               `bson:"genes,omitempty"`
}

type GeneQc struct {
	HGNC                  string                      `bson:"hgnc"`
	Symbol                string			          `bson:"symbol"`
	Gene_type             string			          `bson:"gene_type"`
	Segment_duplication   string		              `bson:"segment_duplication"`
	Mean_coverage         float64	   	              `bson:"mean_coverage"`
	Mean_completness      float64		              `bson:"mean_completness"`
	Pseudogene            string			          `bson:"pseudogene"`
	Incomplete_exons      []IncompleteExons			  `bson:"incomplete_exons,omitempty"`
}


type IncompleteExons struct {
	Exon_number                  string                      `bson:"exon_number"`
	Completness                  float64                     `bson:"completness"`
	Transcript                   string                      `bson:"transcript"`
	Mane_select                  string                      `bson:"mane_select"`
}


func (p PanelQc) Validate() error {
	if p.RunId == "" {
		return errors.New("panelQC must have a Runid")
	}
	if len(p.GenePanel) == 0 {
		return errors.New("panel must contain at least one GenePanel")
	}
	for _, g := range p.GenePanel {
		if len(g.Genes) == 0 {
			return errors.New("Missing genes for at least one Gene Panel")
		}
	}
	return nil
}
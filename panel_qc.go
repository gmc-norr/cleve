package cleve

import (
	"time"
	"errors"
)

type PanelQcResult struct {
	PaginationMetadata                             `bson:"metadata" json:"metadata"`
	PanelQCs                                       []*PanelQc `bson:"runs" json:"runs"`
}

type GeneQcResult struct {
	GeneQcs                                        []*GeneQCs`bson:"geneqc" json:"geneqc"`
}

type ExonQcResult struct {
	ExonQcs                                        []*ExonQCs`bson:"exonqc" json:"exonqc"`
}

type PanelQc struct {
	RunId               string                      `bson:"run_id"`
	Created             time.Time                   `bson:"created"`
	GenePanel           []GenePanelQc               `bson:"genepanel"`
	
}

type GeneQCs struct {
	Genes                     []GeneQcrunid          `bson:"genes"`
	
}

type ExonQCs struct {
	Exons                     []IncompleteExonsRunid `bson:"exons"`
	
}

type GenePanelQc struct {
	GenePanelId               string                 `bson:"genepanel_id"`
	CompletnessThreshold      float64                `bson:"completness_threshold"`
	FeatureFraction           float64                `bson:"feature_fraction"`
	NumberSamples             int                    `bson:"number_of_samples"`
	Genes                     []GeneQc               `bson:"genes"`
}

type GeneQc struct {
	HGNC                  string                      `bson:"hgnc"`
	Symbol                string			          `bson:"symbol"`
	Gene_type             string			          `bson:"gene_type"`
	Segment_duplication   bool   		              `bson:"segment_duplication"`
	Mean_coverage         float64	   	              `bson:"mean_coverage"`
	Mean_completness      float64		              `bson:"mean_completness"`
	Mean_mapping_quality  float64					  `bson:"mean_mapping_quality"`
	Suboptional_coverage  bool					      `bson:"suboptional_coverage"`
	Pseudogene            string			          `bson:"pseudogene"`
	Incomplete_exons      []IncompleteExons			  `bson:"incomplete_exons,omitempty"`
}

type GeneQcrunid struct {
	RunId                   string                    `bson:"run_id"`
	HGNC                    string                    `bson:"hgnc"`
	Mean_coverage           float64	   	              `bson:"mean_coverage"`
	Mean_completness        float64		              `bson:"mean_completness"`
	Mean_mapping_quality    float64                   `bson:"mean_mapping_quality"`
}

type IncompleteExons struct {
	Exon_number                  float64              `bson:"exon_number"`
	Genomic_position_exon        string               `bson:"genomic_position_exon"`
	Mean_mapping_quality         float64              `bson:"mean_mapping_quality_exon"`
	Suboptional_coverage         bool                 `bson:"suboptional_coverage_exon"`
	Mean_Completness             float64              `bson:"mean_completness_exon"`
	Mean_Coverage                float64              `bson:"mean_coverage_exon"`
	Transcript                   string               `bson:"transcript"`
	Mane_select                  bool                 `bson:"mane_select"`
	Segment_duplication          bool                 `bson:"segment_duplication_exon"`
}

type IncompleteExonsRunid struct {
	RunId                        string               `bson:"run_id"`
	HGNC                         string               `bson:"hgnc"`
	Exon_number                  float64              `bson:"exon_number"`
	Mean_mapping_quality         float64              `bson:"mean_mapping_quality_exon"`
	Mean_Completness             float64              `bson:"mean_completness_exon"`
	Mean_Coverage                float64              `bson:"mean_coverage_exon"`
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
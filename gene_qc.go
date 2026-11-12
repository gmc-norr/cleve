package cleve

import (
	"errors"
)

type AllGenesQcResult struct {
	PaginationMetadata                                `bson:"metadata" json:"metadata"`
	AllGenes                                          []*GeneQc`bson:"geneqc" json:"geneqc"`
}

type ExonQcResult struct {
	ExonQcs                                        []*ExonQCs`bson:"exonqc" json:"exonqc"`
}


type ExonQCs struct {
	Exons                     []*IncompleteExonsRunid `bson:"exons"`
	
}

type GenesQc struct {
	RunId               string                      `bson:"run_id"`
	Date                int                         `bson:"date"`
	Genes               []GeneQc                    `bson:"genes"`
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
	Pathogenic_variants   string                      `bson:"pathogenic_variants"`
	Incomplete_exons      []IncompleteExons			  `bson:"incomplete_exons"`
}

type IncompleteExons struct {
	Exon_number                  float64              `bson:"exon_number"`
	Genomic_position_exon        string               `bson:"genomic_position_exon"`
	Mean_mapping_quality         float64              `bson:"mean_mapping_quality_exon"`
	Suboptional_coverage         bool                 `bson:"suboptional_coverage_exon"`
	Mean_Completness             float64              `bson:"mean_completness_exon"`
	Mean_Coverage                float64              `bson:"mean_coverage_exon"`
	Transcript                   string               `bson:"transcript"`
	Segment_duplication          bool                 `bson:"segment_duplication_exon"`
	Select_type                  string               `bson:"select_type"`
}

type IncompleteExonsRunid struct {
	RunId                        string               `bson:"run_id"`
	HGNC                         string               `bson:"hgnc"`
	Symbol                       string               `bson:"symbol"`
	Exon_number                  float64              `bson:"exon_number"`
	Mean_mapping_quality         float64              `bson:"mean_mapping_quality_exon"`
	Mean_Completness             float64              `bson:"mean_completness_exon"`
	Mean_Coverage                float64              `bson:"mean_coverage_exon"`
}

func (p GenesQc) Validate() error {
	if p.RunId == "" {
		return errors.New("panelQC must have a Runid")
	}
	if len(p.Genes) == 0 {
		return errors.New("Must contain at least one gene")
	}
	return nil
}

type GeneQCFile struct {
	Path     string       `bson:"path" json:"path"`
}

type GeneQCFiles []GeneQCFile
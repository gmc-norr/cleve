package runparameters

import (
	"encoding/xml"
	"log"
)

type NovaSeqParameters struct {
	Side                   string `xml:"Side"`
	Application            string `xml:"Application"`
	SystemSuiteVersion     string `xml:"SystemSuiteVersion"`
	OutputFolder           string `xml:"OutputFolder"`
	CloudUploadMode        string `xml:"CloudUploadMode"`
	RunSetupMode           string `xml:"RunSetupMode"`
	SecondaryAnalysisMode  string `xml:"SecondaryAnalysisMode"`
	InstrumentType         string `xml:"InstrumentType"`
	InstrumentSerialNumber string `xml:"InstrumentSerialNumber"`
	RunId                  string `xml:"RunId"`
	RunCounter             int    `xml:"RunCounter"`
	RecipeName             string `xml:"RecipeName"`
	RecipeVersion          string `xml:"RecipeVersion"`
	ExperimentName         string `xml:"ExperimentName"`
	FlowCellName           string `xml:"FlowCellName"`
	FlowCellType           string `xml:"FlowCellType"`
	ConsumableInfo         []struct {
		SerialNumber   string     `xml:"SerialNumber"`
		LotNumber      string     `xml:"LotNumber"`
		PartNumber     string     `xml:"PartNumber"`
		ExpirationDate CustomTime `xml:"ExpirationDate"`
		Type           string     `xml:"Type"`
		Mode           string     `xml:"Mode"`
		Version        string     `xml:"Version"`
		Name           string     `xml:"Name"`
	} `xml:"ConsumableInfo>ConsumableInfo"`
	PlannedReads struct {
		Read []struct {
			ReadName string `xml:"ReadName,attr"`
			Cycles   int    `xml:"Cycles,attr"`
		} `xml:"Read"`
	} `xml:"PlannedReads"`
	SecondaryAnalysisInfo struct {
		SecondaryAnalysisPlatformVersion string   `xml:"SecondaryAnalysisPlatformVersion"`
		SecondaryAnalysisWorkflow        []string `xml:"SecondaryAnalysisWorkflow>string"`
	} `xml:"SecondaryAnalysisInfo>SecondaryAnalysisInfo"`
	DisableBclCopy string `xml:"DisableBclCopy"`
}

func (p NovaSeqParameters) IsValid() bool {
	if p.Side == "" {
		return false
	}
	return true
}

func (p NovaSeqParameters) GetExperimentName() string {
	return p.ExperimentName
}

func (p NovaSeqParameters) GetRunID() string {
	return p.RunId
}

func (p NovaSeqParameters) Platform() string {
	return "NovaSeq"
}

func ParseNovaSeqRunParameters(d []byte) NovaSeqParameters {
	var params NovaSeqParameters
	if err := xml.Unmarshal(d, &params); err != nil {
		log.Fatal(err)
	}

	return params
}

package cleve

import (
	"encoding/xml"
	"log"
)

type NextSeqParameters struct {
	RunParametersVersion string `xml:"RunParametersVersion"`
	Setup                struct {
		SupportMultipleSurfacesInUI string `xml:"SupportMultipleSurfacesInUI"`
		ApplicationVersion          string `xml:"ApplicationVersion"`
		ApplicationName             string `xml:"ApplicationName"`
		NumTilesPerSwath            int    `xml:"NumTilesPerSwath"`
		NumSwaths                   int    `xml:"NumSwaths"`
		NumLanes                    int    `xml:"NumLanes"`
		Read1                       int    `xml:"Read1"`
		Read2                       int    `xml:"Read2"`
		Index1Read                  int    `xml:"Index1Read"`
		Index2Read                  int    `xml:"Index2Read"`
		SectionPerLane              int    `xml:"SectionPerLane"`
		LanePerSection              int    `xml:"LanePerSection"`
	} `xml:"Setup"`
	RunID                  string `xml:"RunID"`
	CopyServiceRunId       string `xml:"CopyServiceRunId"`
	InstrumentID           string `xml:"InstrumentID"`
	RunNumber              int    `xml:"RunNumber"`
	RTAVersion             string `xml:"RTAVersion"`
	SystemSuiteVersion     string `xml:"SystemSuiteVersion"`
	LocalRunManagerVersion string `xml:"LocalRunManagerVersion"`
	RecipeVersion          string `xml:"RecipeVersion"`
	FirmwareVersion        string `xml:"FirmwareVersion"`
	FlowCellRfidTag        struct {
		SerialNumber   string     `xml:"SerialNumber"`
		PartNumber     string     `xml:"PartNumber"`
		LotNumber      string     `xml:"LotNumber"`
		ExpirationDate CustomTime `xml:"ExpirationDate"`
	} `xml:"FlowCellRfidTag"`
	PR2BottleRfidTag struct {
		SerialNumber   string     `xml:"SerialNumber"`
		PartNumber     string     `xml:"PartNumber"`
		LotNumber      string     `xml:"LotNumber"`
		ExpirationDate CustomTime `xml:"ExpirationDate"`
	} `xml:"PR2BottleRfidTag"`
	ReagentKitRfidTag struct {
		SerialNumber   string     `xml:"SerialNumber"`
		PartNumber     string     `xml:"PartNumber"`
		LotNumber      string     `xml:"LotNumber"`
		ExpirationDate CustomTime `xml:"ExpirationDate"`
	} `xml:"ReagentKitRfidTag"`
	FlowCellSerial                        string `xml:"FlowCellSerial"`
	PR2BottleSerial                       string `xml:"PR2BottleSerial"`
	ReagentKitSerial                      string `xml:"ReagentKitSerial"`
	ReagentKitSerialWasEnteredInBaseSpace string `xml:"ReagentKitSerialWasEnteredInBaseSpace"`
	ExperimentName                        string `xml:"ExperimentName"`
	LibraryID                             string `xml:"LibraryID"`
	StateDescription                      string `xml:"StateDescription"`
	Chemistry                             string `xml:"Chemistry"`
	ChemistryVersion                      string `xml:"ChemistryVersion"`
	SelectedTiles                         struct {
		Tile []string `xml:"Tile"`
	} `xml:"SelectedTiles"`
	RunFolder                        string `xml:"RunFolder"`
	RTALogsFolder                    string `xml:"RTALogsFolder"`
	PreRunFolderRoot                 string `xml:"PreRunFolderRoot"`
	PreRunFolder                     string `xml:"PreRunFolder"`
	OutputFolder                     string `xml:"OutputFolder"`
	RecipeFolder                     string `xml:"RecipeFolder"`
	SimulationFolder                 string `xml:"SimulationFolder"`
	RunStartDate                     string `xml:"RunStartDate"`
	BaseSpaceUserName                string `xml:"BaseSpaceUserName"`
	LocalRunManagerUserName          string `xml:"LocalRunManagerUserName"`
	FocusMethod                      string `xml:"FocusMethod"`
	SurfaceToScan                    string `xml:"SurfaceToScan"`
	SaveFocusImages                  string `xml:"SaveFocusImages"`
	SaveScanImages                   string `xml:"SaveScanImages"`
	SelectiveSave                    string `xml:"SelectiveSave"`
	IsPairedEnd                      string `xml:"IsPairedEnd"`
	AnalysisWorkflowType             string `xml:"AnalysisWorkflowType"`
	CustomReadOnePrimer              string `xml:"CustomReadOnePrimer"`
	CustomReadTwoPrimer              string `xml:"CustomReadTwoPrimer"`
	CustomIndexOnePrimer             string `xml:"CustomIndexOnePrimer"`
	CustomIndexTwoPrimer             string `xml:"CustomIndexTwoPrimer"`
	UsesCustomReadOnePrimer          string `xml:"UsesCustomReadOnePrimer"`
	UsesCustomReadTwoPrimer          string `xml:"UsesCustomReadTwoPrimer"`
	UsesCustomIndexPrimer            string `xml:"UsesCustomIndexPrimer"`
	UsesCustomIndexTwoPrimer         string `xml:"UsesCustomIndexTwoPrimer"`
	BaseSpaceRunId                   string `xml:"BaseSpaceRunId"`
	LocalRunManagerRunId             string `xml:"LocalRunManagerRunId"`
	RunSetupType                     string `xml:"RunSetupType"`
	RunMode                          string `xml:"RunMode"`
	ComputerName                     string `xml:"ComputerName"`
	SequencingStarted                string `xml:"SequencingStarted"`
	PlannedRead1Cycles               string `xml:"PlannedRead1Cycles"`
	PlannedRead2Cycles               string `xml:"PlannedRead2Cycles"`
	PlannedIndex1ReadCycles          string `xml:"PlannedIndex1ReadCycles"`
	PlannedIndex2ReadCycles          string `xml:"PlannedIndex2ReadCycles"`
	IsRehyb                          string `xml:"IsRehyb"`
	PurgeConsumables                 string `xml:"PurgeConsumables"`
	MaxCyclesSupportedByReagentKit   string `xml:"MaxCyclesSupportedByReagentKit"`
	ExtraCyclesSupportedByReagentKit string `xml:"ExtraCyclesSupportedByReagentKit"`
	ModuleName                       string `xml:"ModuleName"`
	ModuleVersion                    string `xml:"ModuleVersion"`
	IncludedFile                     string `xml:"IncludedFile"`
}

func (p NextSeqParameters) IsValid() bool {
	return p.InstrumentID != ""
}

func (p NextSeqParameters) GetExperimentName() string {
	return p.ExperimentName
}

func (p NextSeqParameters) GetRunID() string {
	return p.RunID
}

func (p NextSeqParameters) Platform() string {
	return "NextSeq"
}

func (p NextSeqParameters) Flowcell() string {
	return p.Chemistry
}

func ParseNextSeqRunParameters(d []byte) NextSeqParameters {
	var params NextSeqParameters
	if err := xml.Unmarshal(d, &params); err != nil {
		log.Fatal(err)
	}

	return params
}

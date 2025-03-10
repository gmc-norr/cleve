package interop

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"regexp"
	"time"
)

// Current sequencers that we use and support
var instrumentIds = map[*regexp.Regexp]string{
	regexp.MustCompile(`^LH\d{5}$`): "NovaSeq X Plus",
	regexp.MustCompile(`^NB\d{6}$`): "NextSeq 5x0",
	regexp.MustCompile(`^M\d{5}$`):  "MiSeq",
}

// Current flowcells that we use and support
var flowcellIds = map[*regexp.Regexp]string{
	regexp.MustCompile(`D[A-Z0-9]{4}$`):              "Nano",     // MiSeq
	regexp.MustCompile(`G[A-Z0-9]{4}$`):              "Micro",    // MiSeq
	regexp.MustCompile(`A[A-Z0-9]{4}$`):              "Standard", // MiSeq
	regexp.MustCompile(`B[A-Z0-9]{4}$`):              "Standard", // MiSeq
	regexp.MustCompile(`C[A-Z0-9]{4}$`):              "Standard", // MiSeq
	regexp.MustCompile(`J[A-Z0-9]{4}$`):              "Standard", // MiSeq
	regexp.MustCompile(`K[A-Z0-9]{4}$`):              "Standard", // MiSeq
	regexp.MustCompile(`L[A-Z0-9]{4}$`):              "Standard", // MiSeq
	regexp.MustCompile(`^[A-Z0-9]{5}AF[A-Z0-9]{2}$`): "Mid",      // NextSeq 500/550
	regexp.MustCompile(`^[A-Z0-9]{5}AG[A-Z0-9]{2}$`): "High",     // NextSeq 500/550
	regexp.MustCompile(`^[A-Z0-9]{5}BG[A-Z0-9]{2}$`): "High",     // NextSeq 500/550
	regexp.MustCompile(`^H[A-Z0-9]{4}BGXX`):          "High",     // NextSeq
	regexp.MustCompile(`^H[A-Z0-9]{4}BGXY`):          "High",     // NextSeq
	regexp.MustCompile(`^[A-Z0-9]{6}LT1$`):           "1.5B",     // NovaSeq X Plus
	regexp.MustCompile(`^[A-Z0-9]{6}LT3$`):           "10B",      // NovaSeq X Plus
	regexp.MustCompile(`^[A-Z0-9]{6}LT4$`):           "25B",      // NovaSeq X Plus
}

type ReadInfo struct {
	Number    int         `xml:"Number,attr" bson:"number" json:"number"`
	Cycles    int         `xml:"NumCycles,attr" bson:"cycles" json:"cycles"`
	IsIndex   interopBool `xml:"IsIndexedRead,attr" bson:"is_index" json:"is_index"`
	IsRevComp interopBool `xml:"IsReverseComplemented,attr" bson:"is_revcomp" json:"is_revcomp"`
}

type flowcellInfo struct {
	Lanes          int `xml:"LaneCount,attr" bson:"lanes" json:"lanes"`
	Swaths         int `xml:"SwathCount,attr" bson:"swaths" json:"swaths"`
	Tiles          int `xml:"TileCount,attr" bson:"tiles" json:"tiles"`
	Surfaces       int `xml:"SurfaceCount,attr" bson:"surfaces" json:"surfaces"`
	SectionPerLane int `xml:"SectionPerLane,attr,omitempty" bson:"section_per_lane" json:"section_per_lane"`
}

// RunInfo is the representation of an Illumina RunInfo.xml file.
type RunInfo struct {
	Version      int          `bson:"version" json:"version"`
	RunId        string       `bson:"run_id" json:"run_id"`
	Date         time.Time    `bson:"date" json:"date"`
	Platform     string       `bson:"platform" json:"platform"`
	FlowcellName string       `bson:"flowcell_name" json:"flowcell_name"`
	InstrumentId string       `bson:"instrument_id" json:"instrument_id"`
	FlowcellId   string       `bson:"flowcell_id" json:"flowcell_id"`
	Reads        []ReadInfo   `bson:"reads" json:"reads"`
	Flowcell     flowcellInfo `bson:"flowcell" json:"flowcell"`
}

func ParseRunInfo(r io.Reader) (ri RunInfo, err error) {
	var payload struct {
		XMLName xml.Name `xml:"RunInfo"`
		Version int      `xml:"Version,attr"`
		Run     struct {
			Id           string       `xml:",attr"`
			Date         interopTime  `xml:"Date"`
			InstrumentId string       `xml:"Instrument"`
			FlowcellId   string       `xml:"Flowcell"`
			Reads        []ReadInfo   `xml:"Reads>Read"`
			Flowcell     flowcellInfo `xml:"FlowcellLayout"`
		} `xml:"Run"`
	}

	decoder := xml.NewDecoder(r)
	err = decoder.Decode(&payload)
	if err != nil {
		return ri, err
	}

	ri = RunInfo{
		Version:      payload.Version,
		RunId:        payload.Run.Id,
		Date:         payload.Run.Date.Time,
		InstrumentId: payload.Run.InstrumentId,
		FlowcellId:   payload.Run.FlowcellId,
		Reads:        payload.Run.Reads,
		Flowcell:     payload.Run.Flowcell,
	}

	ri.Platform = identifyPlatform(ri.InstrumentId)
	ri.FlowcellName = identifyFlowcell(ri.FlowcellId)

	switch ri.Version {
	case 2, 4, 6:
		return ri, nil
	default:
		return ri, fmt.Errorf("unsupported run info version: %d", ri.Version)
	}
}

// ReadRunInfo reads an Illumina RunInfo.xml file.
func ReadRunInfo(filename string) (ri RunInfo, err error) {
	f, err := os.Open(filename)
	if err != nil {
		return ri, err
	}
	defer f.Close()
	return ParseRunInfo(f)
}

// TileCount returns the number of tiles represented on the flow cell.
func (i RunInfo) TileCount() int {
	switch i.Version {
	case 2, 6:
		return i.Flowcell.Lanes * i.Flowcell.Surfaces * i.Flowcell.Swaths * i.Flowcell.Tiles
	case 4:
		return i.Flowcell.Lanes * i.Flowcell.Surfaces * i.Flowcell.Swaths * i.Flowcell.SectionPerLane * i.Flowcell.Tiles
	default:
		return 0
	}
}

// Get the platform name from the sequencer ID.
func identifyPlatform(iid string) string {
	for re, platform := range instrumentIds {
		if re.Match([]byte(iid)) {
			return platform
		}
	}
	return "unknown"
}

// Get the flowcell name from the flowcell ID.
func identifyFlowcell(fcid string) string {
	for re, flowcell := range flowcellIds {
		if re.Match([]byte(fcid)) {
			return flowcell
		}
	}
	return "unknown"
}

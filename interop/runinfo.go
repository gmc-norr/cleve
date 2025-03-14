package interop

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"regexp"
	"time"
)

type idMatcher struct {
	idPattern *regexp.Regexp
	name      string
}

func (m idMatcher) match(id string) bool {
	return m.idPattern.Match([]byte(id))
}

var readyMarkers = map[string]string{
	"NovaSeq X Plus": "CopyComplete.txt",
	"NextSeq 5x0":    "CopyComplete.txt",
	"MiSeq":          "CopyComplete.txt",
}

// Current sequencers that we use and support
var instrumentIds = []idMatcher{
	{idPattern: regexp.MustCompile(`^LH\d{5}$`), name: "NovaSeq X Plus"},
	{idPattern: regexp.MustCompile(`^NB\d{6}$`), name: "NextSeq 5x0"},
	{idPattern: regexp.MustCompile(`^M\d{5}$`), name: "MiSeq"},
}

// Current flowcells that we use and support
var flowcellIds = []idMatcher{
	{idPattern: regexp.MustCompile(`^[A-Z0-9]{5}AF[A-Z0-9]{2}$`), name: "Mid"},  // NextSeq 500/550
	{idPattern: regexp.MustCompile(`^[A-Z0-9]{5}AG[A-Z0-9]{2}$`), name: "High"}, // NextSeq 500/550
	{idPattern: regexp.MustCompile(`^[A-Z0-9]{5}BG[A-Z0-9]{2}$`), name: "High"}, // NextSeq 500/550
	{idPattern: regexp.MustCompile(`^H[A-Z0-9]{4}BGXX`), name: "High"},          // NextSeq
	{idPattern: regexp.MustCompile(`^H[A-Z0-9]{4}BGXY`), name: "High"},          // NextSeq
	{idPattern: regexp.MustCompile(`^[A-Z0-9]{6}LT1$`), name: "1.5B"},           // NovaSeq X Plus
	{idPattern: regexp.MustCompile(`^[A-Z0-9]{6}LT3$`), name: "10B"},            // NovaSeq X Plus
	{idPattern: regexp.MustCompile(`^[A-Z0-9]{6}LT4$`), name: "25B"},            // NovaSeq X Plus
	{idPattern: regexp.MustCompile(`D[A-Z0-9]{4}$`), name: "Nano"},              // MiSeq
	{idPattern: regexp.MustCompile(`G[A-Z0-9]{4}$`), name: "Micro"},             // MiSeq
	{idPattern: regexp.MustCompile(`A[A-Z0-9]{4}$`), name: "Standard"},          // MiSeq
	{idPattern: regexp.MustCompile(`B[A-Z0-9]{4}$`), name: "Standard"},          // MiSeq
	{idPattern: regexp.MustCompile(`C[A-Z0-9]{4}$`), name: "Standard"},          // MiSeq
	{idPattern: regexp.MustCompile(`J[A-Z0-9]{4}$`), name: "Standard"},          // MiSeq
	{idPattern: regexp.MustCompile(`K[A-Z0-9]{4}$`), name: "Standard"},          // MiSeq
	{idPattern: regexp.MustCompile(`L[A-Z0-9]{4}$`), name: "Standard"},          // MiSeq
}

type ReadInfo struct {
	Name      string `bson:"name,omitzero" json:"name,omitzero"`
	Number    int    `xml:"Number,attr" bson:"number,omitzero" json:"number,omitzero"`
	Cycles    int    `xml:"NumCycles,attr" bson:"cycles" json:"cycles"`
	IsIndex   bool   `xml:"IsIndexedRead,attr" bson:"is_index" json:"is_index"`
	IsRevComp bool   `xml:"IsReverseComplemented,attr" bson:"is_revcomp" json:"is_revcomp"`
}

type rawReadInfo struct {
	Number    int         `xml:"Number,attr" bson:"number" json:"number"`
	Cycles    int         `xml:"NumCycles,attr" bson:"cycles" json:"cycles"`
	IsIndex   interopBool `xml:"IsIndexedRead,attr" bson:"is_index" json:"is_index"`
	IsRevComp interopBool `xml:"IsReverseComplemented,attr" bson:"is_revcomp" json:"is_revcomp"`
}

type FlowcellInfo struct {
	Lanes          int `xml:"LaneCount,attr" bson:"lanes" json:"lanes"`
	Swaths         int `xml:"SwathCount,attr" bson:"swaths" json:"swaths"`
	Tiles          int `xml:"TileCount,attr" bson:"tiles" json:"tiles"`
	Surfaces       int `xml:"SurfaceCount,attr" bson:"surfaces" json:"surfaces"`
	SectionPerLane int `xml:"SectionPerLane,attr,omitempty" bson:"section_per_lane,omitzero" json:"section_per_lane,omitzero"`
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
	Flowcell     FlowcellInfo `bson:"flowcell" json:"flowcell"`
}

func ParseRunInfo(r io.Reader) (ri RunInfo, err error) {
	var payload struct {
		XMLName xml.Name `xml:"RunInfo"`
		Version int      `xml:"Version,attr"`
		Run     struct {
			Id           string        `xml:",attr"`
			Date         interopTime   `xml:"Date"`
			InstrumentId string        `xml:"Instrument"`
			FlowcellId   string        `xml:"Flowcell"`
			Reads        []rawReadInfo `xml:"Reads>Read"`
			Flowcell     FlowcellInfo  `xml:"FlowcellLayout"`
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
		Flowcell:     payload.Run.Flowcell,
	}

	for _, read := range payload.Run.Reads {
		readName := fmt.Sprintf("Read %d", read.Number)
		if read.IsIndex {
			readName += " (I)"
		}
		ri.Reads = append(ri.Reads, ReadInfo{
			Name:      readName,
			Number:    read.Number,
			Cycles:    read.Cycles,
			IsIndex:   bool(read.IsIndex),
			IsRevComp: bool(read.IsRevComp),
		})
	}

	ri.Platform = IdentifyPlatform(ri.InstrumentId)
	ri.FlowcellName = IdentifyFlowcell(ri.FlowcellId)

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
func IdentifyPlatform(iid string) string {
	for _, pm := range instrumentIds {
		if pm.match(iid) {
			return pm.name
		}
	}
	return "unknown"
}

// Get the flowcell name from the flowcell ID.
func IdentifyFlowcell(fcid string) string {
	for _, pm := range flowcellIds {
		if pm.match(fcid) {
			return pm.name
		}
	}
	return "unknown"
}

func PlatformReadyMarker(platform string) string {
	marker := readyMarkers[platform]
	return marker
}

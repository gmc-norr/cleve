package interop

import (
	"encoding/xml"
	"fmt"
	"os"
)

type ReadInfo struct {
	Number    int         `xml:"Number,attr"`
	Cycles    int         `xml:"NumCycles,attr"`
	IsIndex   interopBool `xml:"IsIndexedRead,attr"`
	IsRevComp interopBool `xml:"IsReverseComplemented,attr"`
}

// RunInfo is the representation of an Illumina RunInfo.xml file.
type RunInfo struct {
	Version    int
	RunId      string
	Date       interopTime
	Instrument string
	Reads      []ReadInfo
	Flowcell   struct {
		Lanes          int
		Swaths         int
		Tiles          int
		Surfaces       int
		SectionPerLane int
	}
}

// ReadRunInfo reads an Illumina RunInfo.xml file.
func ReadRunInfo(filename string) (ri RunInfo, err error) {
	var payload struct {
		XMLName xml.Name `xml:"RunInfo"`
		Version int      `xml:"Version,attr"`
		Run     struct {
			Id         string      `xml:",attr"`
			Date       interopTime `xml:"Date"`
			Instrument string      `xml:"Instrument"`
			Reads      []ReadInfo  `xml:"Reads>Read"`
			Flowcell   struct {
				Lanes          int `xml:"LaneCount,attr"`
				Swaths         int `xml:"SwathCount,attr"`
				Tiles          int `xml:"TileCount,attr"`
				Surfaces       int `xml:"SurfaceCount,attr"`
				SectionPerLane int `xml:"SectionPerLane,attr,omitempty"`
			} `xml:"FlowcellLayout"`
		} `xml:"Run"`
	}

	f, err := os.Open(filename)
	if err != nil {
		return ri, err
	}
	defer f.Close()
	decoder := xml.NewDecoder(f)
	err = decoder.Decode(&payload)
	if err != nil {
		return ri, err
	}

	ri = RunInfo{
		Version:    payload.Version,
		RunId:      payload.Run.Id,
		Date:       payload.Run.Date,
		Instrument: payload.Run.Instrument,
		Reads:      payload.Run.Reads,
		Flowcell: struct {
			Lanes          int
			Swaths         int
			Tiles          int
			Surfaces       int
			SectionPerLane int
		}(payload.Run.Flowcell),
	}

	switch payload.Version {
	case 2, 4, 6:
		return ri, nil
	default:
		return ri, fmt.Errorf("unsupported run info version: %d", ri.Version)
	}
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

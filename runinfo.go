package cleve

import (
	"bytes"
	"encoding/xml"
)

type RunInfo struct {
	Version int `xml:"Version,attr" bson:"version" json:"version"`
	Run     struct {
		RunID      string     `xml:"Id,attr" bson:"run_id" json:"run_id"`
		Number     int        `xml:"Number,attr" bson:"number" json:"number"`
		Flowcell   string     `xml:"Flowcell" bson:"flowcell" json:"flowcell"`
		Instrument string     `xml:"Instrument" bson:"instrument" json:"instrument"`
		Date       CustomTime `xml:"Date" bson:"date" json:"date"`
		Reads      struct {
			Read []struct {
				Number              int    `xml:"Number,attr" bson:"number" json:"number"`
				NumCycles           int    `xml:"NumCycles,attr" bson:"num_cycles" json:"num_cycles"`
				IsIndexedRead       string `xml:"IsIndexedRead,attr" bson:"is_indexed_read" json:"is_indexed_read"`
				IsReverseComplement string `xml:"IsReverseComplement,attr" bson:"is_reverse_complement" json:"is_reverse_complement"`
			} `xml:"Read" bson:"read" json:"read"`
		} `xml:"Reads" bson:"reads" json:"reads"`
		FlowcellLayout struct {
			LaneCount      int `xml:"LaneCount,attr" bson:"lane_count" json:"lane_count"`
			SurfaceCount   int `xml:"SurfaceCount,attr" bson:"surface_count" json:"surface_count"`
			SwathCount     int `xml:"SwathCount,attr" bson:"swath_count" json:"swath_count"`
			TileCount      int `xml:"TileCount,attr" bson:"tile_count" json:"tile_count"`
			SectionPerLane int `xml:"SectionPerLane,attr,omitempty" bson:"section_per_lane,omitempty" json:"section_per_lane,omitempty"`
			LanePerSection int `xml:"LanePerSection,attr,omitempty" bson:"lane_per_section,omitempty" json:"lane_per_section,omitempty"`
			TileSet        struct {
				TileNamingConvention string   `xml:"TileNamingConvention,attr" bson:"tile_naming_convention" json:"tile_naming_convention"`
				Tiles                []string `xml:"Tiles>Tile" bson:"tiles" json:"tiles"`
			} `xml:"TileSet" bson:"tileset" json:"tileset"`
		} `xml:"FlowcellLayout" bson:"flowcell_layout" json:"flowcell_layout"`
		ImageDimensions struct {
			Width  int `xml:"Width,attr" bson:"width" json:"width"`
			Height int `xml:"Height,attr" bson:"height" json:"height"`
		} `xml:"ImageDimensions" bson:"image_dimensions" json:"image_dimensions"`
		ImageChannels []string `xml:"ImageChannels>Name" bson:"image_channels" json:"image_channels"`
	} `xml:"Run" bson:"run" json:"run"`
}

func ParseRunInfo(data []byte) (RunInfo, error) {
	var runInfo RunInfo
	b := bytes.NewBuffer(data)
	decoder := xml.NewDecoder(b)
	if err := decoder.Decode(&runInfo); err != nil {
		return runInfo, err
	}
	return runInfo, nil
}

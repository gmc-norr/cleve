package interop

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
)

type interopTime struct {
	time.Time
}

func (t *interopTime) UnmarshalXML(d *xml.Decoder, s xml.StartElement) error {
	var rawTime string
	err := d.DecodeElement(&rawTime, &s)
	if err != nil {
		return err
	}

	layouts := []string{
		"2006-01-02T15:04:05-07:00",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05",
		"2006-01-02",
		"20060102",
		"060102",
	}

	var pt time.Time
	for _, l := range layouts {
		pt, err = time.Parse(l, rawTime)
		if err != nil {
			continue
		}
		t.Time = pt
		return nil
	}

	return fmt.Errorf("failed to parse time in %s: %q", s.Name.Local, rawTime)
}

type interopBool bool

func (b *interopBool) UnmarshalXMLAttr(attr xml.Attr) error {
	switch attr.Value {
	case "N":
		*b = false
	case "Y":
		*b = true
	default:
		return fmt.Errorf("invalid bool value %s", attr.Value)
	}
	return nil
}

type read struct {
	Name   string `bson:"name" json:"name"`
	Cycles int    `bson:"cycles" json:"cycles"`
}

type Consumable struct {
	Type           string    `bson:"type" json:"type"`
	Name           string    `bson:"name,omitzero" json:"name,omitzero"`
	Version        string    `bson:"version,omitzero" json:"version,omitzero"`
	Mode           int       `bson:"mode,omitzero" json:"mode,omitzero"`
	SerialNumber   string    `bson:"serial_number" json:"serial_number"`
	PartNumber     string    `bson:"part_number" json:"part_number"`
	LotNumber      string    `bson:"lot_number" json:"lot_number"`
	ExpirationDate time.Time `bson:"expiration_date" json:"expiration_date"`
}

type RunParameters struct {
	ExperimentName string       `bson:"experiment_name" json:"experiment_name"`
	Side           string       `bson:"side,omitzero" json:"side,omitzero"`
	Reads          []read       `bson:"reads" json:"reads"`
	Flowcell       Consumable   `bson:"flowcell" json:"flowcell"`
	Consumables    []Consumable `bson:"consumables" json:"consumables"`
}

type runParametersNovaSeq struct {
	XMLName        xml.Name `xml:"RunParameters"`
	ExperimentName string   `xml:"ExperimentName"`
	Side           string   `xml:"Side"`
	Reads          []struct {
		Name   string `xml:"ReadName,attr"`
		Cycles int    `xml:"Cycles,attr"`
	} `xml:"PlannedReads>Read"`
	Consumables []struct {
		Type           string      `xml:"Type"`
		Name           string      `xml:"Name"`
		SerialNumber   string      `xml:"SerialNumber"`
		PartNumber     string      `xml:"PartNumber"`
		LotNumber      string      `xml:"LotNumber"`
		ExpirationDate interopTime `xml:"ExpirationDate"`
		Mode           int         `xml:"Mode"`
		Version        string      `xml:"Version"`
	} `xml:"ConsumableInfo>ConsumableInfo"`
}

type runParametersNextSeq struct {
	XMLName        xml.Name `xml:"RunParameters"`
	ExperimentName string   `xml:"ExperimentName"`
	Setup          struct {
		Read1  int `xml:"Read1"`
		Read2  int `xml:"Read2"`
		Index1 int `xml:"Index1Read"`
		Index2 int `xml:"Index2Read"`
	} `xml:"Setup"`
	Chemistry string `xml:"Chemistry"`
	Flowcell  struct {
		SerialNumber   string      `xml:"SerialNumber"`
		PartNumber     string      `xml:"PartNumber"`
		LotNumber      string      `xml:"LotNumber"`
		ExpirationDate interopTime `xml:"ExpirationDate"`
	} `xml:"FlowCellRfidTag"`
	Buffer struct {
		SerialNumber   string      `xml:"SerialNumber"`
		PartNumber     string      `xml:"PartNumber"`
		LotNumber      string      `xml:"LotNumber"`
		ExpirationDate interopTime `xml:"ExpirationDate"`
	} `xml:"PR2BottleRfidTag"`
	ReagentKit struct {
		SerialNumber   string      `xml:"SerialNumber"`
		PartNumber     string      `xml:"PartNumber"`
		LotNumber      string      `xml:"LotNumber"`
		ExpirationDate interopTime `xml:"ExpirationDate"`
	} `xml:"ReagentKitRfidTag"`
}

type runParametersMiSeq struct {
	XMLName        xml.Name `xml:"RunParameters"`
	ExperimentName string   `xml:"ExperimentName"`
	Reads          []struct {
		Number  int         `xml:"Number,attr"`
		Cycles  int         `xml:"NumCycles,attr"`
		IsIndex interopBool `xml:"IsIndexedRead,attr"`
	} `xml:"Reads>RunInfoRead"`
	Flowcell struct {
		SerialNumber   string      `xml:"SerialNumber"`
		PartNumber     string      `xml:"PartNumber"`
		LotNumber      string      `xml:"LotNumber"`
		ExpirationDate interopTime `xml:"ExpirationDate"`
	} `xml:"FlowcellRFIDTag"`
	Buffer struct {
		SerialNumber   string      `xml:"SerialNumber"`
		PartNumber     string      `xml:"PartNumber"`
		LotNumber      string      `xml:"LotNumber"`
		ExpirationDate interopTime `xml:"ExpirationDate"`
	} `xml:"PR2BottleRFIDTag"`
	ReagentKit struct {
		SerialNumber   string      `xml:"SerialNumber"`
		PartNumber     string      `xml:"PartNumber"`
		LotNumber      string      `xml:"LotNumber"`
		ExpirationDate interopTime `xml:"ExpirationDate"`
	} `xml:"ReagentKitRFIDTag"`
}

// TODO: this should be used for the version in the final runparameters struct.
// This means that I have to make sure to parse the NovaSeq runparameters into
// this format too. I don't have a version for this.
type runParametersVersion struct {
	Platform string
	Major    int
	Minor    int
	Patch    int
}

func (v *runParametersVersion) UnmarshalXML(d *xml.Decoder, s xml.StartElement) error {
	if s.Name.Local != "RunParametersVersion" {
		return fmt.Errorf("invalid tag for RunParametersVersion")
	}
	rawRunParameterVersion := struct {
		Name  xml.Name `xml:"RunParametersVersion"`
		Value string   `xml:",chardata"`
	}{}
	err := d.DecodeElement(&rawRunParameterVersion, &s)
	if err != nil {
		return err
	}
	parts := strings.Split(rawRunParameterVersion.Value, "_")
	if len(parts) < 2 {
		return fmt.Errorf("wrong number of components in RunParamtersVersion string")
	}
	v.Platform = parts[0]

	for i := 1; i < len(parts); i++ {
		var err error
		switch i {
		case 1:
			v.Major, err = strconv.Atoi(parts[i])
		case 2:
			v.Minor, err = strconv.Atoi(parts[i])
		case 3:
			v.Minor, err = strconv.Atoi(parts[i])
		}
		if err != nil {
			return err
		}
	}
	return err
}

func parseVersion(r io.Reader) (string, error) {
	decoder := xml.NewDecoder(r)
	version := "unknown"

	instrumentType := struct {
		Name  xml.Name `xml:"InstrumentType"`
		Value string   `xml:",chardata"`
	}{}

	for {
		tok, err := decoder.Token()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return version, err
		}
		switch se := tok.(type) {
		case xml.StartElement:
			if se.Name.Local == "RunParametersVersion" {
				var rpVersion runParametersVersion
				err := decoder.DecodeElement(&rpVersion, &se)
				if err != nil {
					return version, err
				}
				return rpVersion.Platform, nil
			}

			if se.Name.Local == "InstrumentType" {
				err := decoder.DecodeElement(&instrumentType, &se)
				if err != nil {
					return version, err
				}
				if instrumentType.Value == "NovaSeqXPlus" {
					version = "NovaSeqXPlus"
					return version, nil
				}
			}
		}
	}

	return version, nil
}

func ParseRunParameters(r io.Reader) (RunParameters, error) {
	rp := RunParameters{}
	version, err := parseVersion(r)
	if err != nil {
		return rp, err
	}

	if f, ok := r.(*os.File); ok {
		if _, err := f.Seek(0, io.SeekStart); err != nil {
			return rp, err
		}
	} else {
		return rp, fmt.Errorf("failed to rewind file")
	}

	decoder := xml.NewDecoder(r)

	switch version {
	case "NovaSeqXPlus":
		var novaseq runParametersNovaSeq
		err = decoder.Decode(&novaseq)
		if err != nil {
			return rp, err
		}
		rp.Side = novaseq.Side
		rp.ExperimentName = novaseq.ExperimentName
		for _, r := range novaseq.Reads {
			rp.Reads = append(rp.Reads, read{
				Name:   r.Name,
				Cycles: r.Cycles,
			})
		}

		rp.Consumables = make([]Consumable, 0, len(novaseq.Consumables))
		for _, c := range novaseq.Consumables {
			if c.Type == "FlowCell" {
				rp.Flowcell = Consumable{
					Type:           c.Type,
					Name:           c.Name,
					Version:        c.Version,
					Mode:           c.Mode,
					SerialNumber:   c.SerialNumber,
					LotNumber:      c.LotNumber,
					PartNumber:     c.PartNumber,
					ExpirationDate: c.ExpirationDate.Time,
				}
				continue
			}
			rp.Consumables = append(rp.Consumables, Consumable{
				Type:           c.Type,
				Name:           c.Name,
				Version:        c.Version,
				Mode:           c.Mode,
				SerialNumber:   c.SerialNumber,
				LotNumber:      c.LotNumber,
				PartNumber:     c.PartNumber,
				ExpirationDate: c.ExpirationDate.Time,
			})
		}
	case "NextSeq":
		var nextseq runParametersNextSeq
		err = decoder.Decode(&nextseq)
		if err != nil {
			return rp, err
		}
		rp.ExperimentName = nextseq.ExperimentName
		rp.Reads = []read{
			{
				Name:   "Read1",
				Cycles: nextseq.Setup.Read1,
			},
			{
				Name:   "Index1",
				Cycles: nextseq.Setup.Index1,
			},
			{
				Name:   "Index2",
				Cycles: nextseq.Setup.Index2,
			},
			{
				Name:   "Read2",
				Cycles: nextseq.Setup.Read2,
			},
		}
		rp.Flowcell = Consumable{
			Type:           "FlowCell",
			Name:           nextseq.Chemistry,
			SerialNumber:   nextseq.Flowcell.SerialNumber,
			PartNumber:     nextseq.Flowcell.PartNumber,
			LotNumber:      nextseq.Flowcell.LotNumber,
			ExpirationDate: nextseq.Flowcell.ExpirationDate.Time,
		}

		rp.Consumables = make([]Consumable, 2)
		rp.Consumables[0] = Consumable{
			Type:           "Buffer",
			SerialNumber:   nextseq.Buffer.SerialNumber,
			PartNumber:     nextseq.Buffer.PartNumber,
			LotNumber:      nextseq.Buffer.LotNumber,
			ExpirationDate: nextseq.Buffer.ExpirationDate.Time,
		}

		rp.Consumables[1] = Consumable{
			Type:           "Reagent",
			SerialNumber:   nextseq.ReagentKit.SerialNumber,
			PartNumber:     nextseq.ReagentKit.PartNumber,
			LotNumber:      nextseq.ReagentKit.LotNumber,
			ExpirationDate: nextseq.ReagentKit.ExpirationDate.Time,
		}
	case "MiSeq":
		var miseq runParametersMiSeq
		err = decoder.Decode(&miseq)
		if err != nil {
			return rp, err
		}
		rp.ExperimentName = miseq.ExperimentName
		for _, r := range miseq.Reads {
			readName := fmt.Sprintf("Read %d", r.Number)
			if r.IsIndex {
				readName = fmt.Sprintf("%s (I)", readName)
			}
			rp.Reads = append(rp.Reads, read{
				Name:   readName,
				Cycles: r.Cycles,
			})
		}
		rp.Flowcell = Consumable{
			Type:           "FlowCell",
			SerialNumber:   miseq.Flowcell.SerialNumber,
			PartNumber:     miseq.Flowcell.PartNumber,
			LotNumber:      miseq.Flowcell.LotNumber,
			ExpirationDate: miseq.Flowcell.ExpirationDate.Time,
		}
		rp.Consumables = make([]Consumable, 2)
		rp.Consumables[0] = Consumable{
			Type:           "Buffer",
			SerialNumber:   miseq.Buffer.SerialNumber,
			PartNumber:     miseq.Buffer.PartNumber,
			LotNumber:      miseq.Buffer.LotNumber,
			ExpirationDate: miseq.Buffer.ExpirationDate.Time,
		}

		rp.Consumables[1] = Consumable{
			Type:           "Reagent",
			SerialNumber:   miseq.ReagentKit.SerialNumber,
			PartNumber:     miseq.ReagentKit.PartNumber,
			LotNumber:      miseq.ReagentKit.LotNumber,
			ExpirationDate: miseq.ReagentKit.ExpirationDate.Time,
		}
	default:
		return rp, fmt.Errorf("unsupported platform: %s", version)
	}

	return rp, nil
}

func ReadRunParameters(filename string) (RunParameters, error) {
	rp := RunParameters{}
	f, err := os.Open(filename)
	if err != nil {
		return rp, err
	}
	defer f.Close()
	return ParseRunParameters(f)
}

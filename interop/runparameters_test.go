package interop

import (
	"bytes"
	"encoding/xml"
	"errors"
	"os"
	"testing"
	"time"
)

func testConsumable(t *testing.T, c consumable, ec consumable) {
	if c.Type != ec.Type {
		t.Errorf("expected type %s, found %s", ec.Type, c.Type)
	}
	if c.Name != ec.Name {
		t.Errorf("expected name %s, found %s", ec.Name, c.Name)
	}
	if c.Version != ec.Version {
		t.Errorf("expected version %s, found %s", ec.Version, c.Version)
	}
	if c.SerialNumber != ec.SerialNumber {
		t.Errorf("expected serial number %s, found %s", ec.SerialNumber, c.SerialNumber)
	}
	if c.LotNumber != ec.LotNumber {
		t.Errorf("expected lot number %s, found %s", ec.LotNumber, c.LotNumber)
	}
	if c.PartNumber != ec.PartNumber {
		t.Errorf("expected part number %s, found %s", ec.PartNumber, c.PartNumber)
	}
	if c.Mode != ec.Mode {
		t.Errorf("expected mode %d, found %d", ec.Mode, c.Mode)
	}
	if !c.ExpirationDate.Equal(ec.ExpirationDate) {
		t.Errorf("expected expiration date %s, found %s", ec.ExpirationDate, c.ExpirationDate)
	}
}

func TestReadRunParameters(t *testing.T) {
	CET := time.FixedZone("CET", 1*60*60)
	CEST := time.FixedZone("CEST", 2*60*60)

	testcases := []struct {
		name           string
		path           string
		side           string
		experimentName string
		reads          []read
		flowcell       consumable
		consumables    []consumable
	}{
		{
			name:           "nextseq",
			experimentName: "250210_Archer_VP_Fusion nypan",
			path:           "./test/250210_NB551119_0457_AHL3Y2AFX7/RunParameters.xml",
			reads: []read{
				{
					Name:   "Read 1",
					Cycles: 151,
				},
				{
					Name:   "Read 2 (I)",
					Cycles: 8,
				},
				{
					Name:   "Read 3 (I)",
					Cycles: 8,
				},
				{
					Name:   "Read 3",
					Cycles: 151,
				},
			},
			flowcell: consumable{
				Type:         "FlowCell",
				Name:         "NextSeq Mid",
				SerialNumber: "HL3Y2AFX7",
				PartNumber:   "20022409",
				LotNumber:    "20893519",
				// 2026-10-07T00:00:00
				ExpirationDate: time.Date(2026, 10, 7, 0, 0, 0, 0, time.UTC),
			},
			consumables: []consumable{
				{
					Type:         "Buffer",
					SerialNumber: "NS6436487-BUFFR",
					PartNumber:   "15057941",
					LotNumber:    "100023199",
					// 2026-01-16T00:00:00
					ExpirationDate: time.Date(2026, 1, 16, 0, 0, 0, 0, time.UTC),
				},
				{
					Type:         "Reagent",
					SerialNumber: "NS6674745-REAGT",
					PartNumber:   "15057939",
					LotNumber:    "20897244",
					// 2026-04-02T00:00:00
					ExpirationDate: time.Date(2026, 4, 2, 0, 0, 0, 0, time.UTC),
				},
			},
		},
		{
			name:           "miseq",
			path:           "./test/250207_M00568_0665_000000000-LMWPP/RunParameters.xml",
			experimentName: "LymphoTrack_AL25.317",
			reads: []read{
				{
					Name:   "Read 1",
					Cycles: 301,
				},
				{
					Name:   "Read 2 (I)",
					Cycles: 6,
				},
				{
					Name:   "Read 3",
					Cycles: 301,
				},
			},
			flowcell: consumable{
				Type:         "FlowCell",
				SerialNumber: "000000000-LMWPP",
				PartNumber:   "15028382",
				LotNumber:    "20861318",
				// 2025-05-19T00:00:00
				ExpirationDate: time.Date(2025, 5, 19, 0, 0, 0, 0, time.UTC),
			},
			consumables: []consumable{
				{
					Type:         "Buffer",
					SerialNumber: "MS1233468-00PR2",
					PartNumber:   "15041807",
					LotNumber:    "20859374",
					// 2025-05-27T00:00:00
					ExpirationDate: time.Date(2025, 5, 27, 0, 0, 0, 0, time.UTC),
				}, {
					Type:         "Reagent",
					SerialNumber: "MS3330879-600V3",
					PartNumber:   "15043962",
					LotNumber:    "20859623",
					// 2025-04-25T00:00:00
					ExpirationDate: time.Date(2025, 4, 25, 0, 0, 0, 0, time.UTC),
				},
			},
		},
		{
			name:           "miseq old",
			path:           "./test/160122_M00568_0146_000000000-ALYCY/runParameters.xml",
			experimentName: "160111 HaloPlex Run2",
			reads: []read{
				{
					Name:   "Read 1",
					Cycles: 151,
				},
				{
					Name:   "Read 2 (I)",
					Cycles: 6,
				},
				{
					Name:   "Read 3",
					Cycles: 151,
				},
			},
			flowcell: consumable{
				Type:         "FlowCell",
				SerialNumber: "000000000-ALYCY",
				PartNumber:   "15028382",
				// 2016-12-09T00:00:00
				ExpirationDate: time.Date(2016, 12, 9, 0, 0, 0, 0, time.UTC),
			},
			consumables: []consumable{
				{
					Type:         "Buffer",
					SerialNumber: "MS2633925-00PR2",
					PartNumber:   "15041807",
					// 2016-11-19T00:00:00
					ExpirationDate: time.Date(2016, 11, 19, 0, 0, 0, 0, time.UTC),
				}, {
					Type:         "Reagent",
					SerialNumber: "MS3916328-600V3",
					PartNumber:   "15043962",
					// 2016-10-08T00:00:00
					ExpirationDate: time.Date(2016, 10, 8, 0, 0, 0, 0, time.UTC),
				},
			},
		},
		{
			name:           "novaseq",
			path:           "./test/20250123_LH00352_0033_A225H35LT1/RunParameters.xml",
			experimentName: "250123_PoN_16samples",
			side:           "A",
			reads: []read{
				{
					Name:   "Read 1",
					Cycles: 151,
				},
				{
					Name:   "Read 2 (I)",
					Cycles: 8,
				},
				{
					Name:   "Read 3 (I)",
					Cycles: 8,
				},
				{
					Name:   "Read 3",
					Cycles: 151,
				},
			},
			flowcell: consumable{
				Type:         "FlowCell",
				Name:         "1.5B",
				Version:      "1.0",
				SerialNumber: "225H35LT1",
				LotNumber:    "20896273",
				PartNumber:   "20083896",
				// 2026-03-26T00:00:00+01:00
				ExpirationDate: time.Date(2026, 3, 26, 0, 0, 0, 0, CET),
				Mode:           1,
			},
			consumables: []consumable{
				{
					Type:         "Reagent",
					Name:         "1.5B 300c",
					Version:      "1.5",
					SerialNumber: "LC4200443-15B3",
					LotNumber:    "20885266",
					PartNumber:   "20066617",
					// 2026-03-12T00:00:00+01:00
					ExpirationDate: time.Date(2026, 3, 12, 0, 0, 0, 0, CET),
					Mode:           1,
				},
				{
					Type:         "SampleTube",
					Name:         "2 Lane",
					Version:      "1.5",
					SerialNumber: "LC1010967-LS2",
					LotNumber:    "1000023606",
					PartNumber:   "20072272",
					// 2026-10-03T00:00:00+02:00
					ExpirationDate: time.Date(2026, 10, 3, 0, 0, 0, 0, CEST),
					Mode:           6,
				},
				{
					Type:         "Lyo",
					Name:         "Low",
					Version:      "1.5",
					SerialNumber: "LC2132269-LI4",
					LotNumber:    "18225119",
					PartNumber:   "20090569",
					// 2026-04-09T00:00:00+02:00
					ExpirationDate: time.Date(2026, 4, 9, 0, 0, 0, 0, CEST),
					Mode:           9,
				},
				{
					Type:         "Buffer",
					Name:         "Universal",
					Version:      "1.0",
					SerialNumber: "LC2409300873-1",
					LotNumber:    "24092601",
					PartNumber:   "20089853",
					// 2026-03-26T00:00:00+01:00
					ExpirationDate: time.Date(2026, 3, 26, 0, 0, 0, 0, CET),
					Mode:           3,
				},
			},
		},
	}

	for _, c := range testcases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := os.Stat(c.path); errors.Is(err, os.ErrNotExist) {
				t.Skip("test data not found, skipping")
			}
			rp, err := ReadRunParameters(c.path)
			if err != nil {
				t.Fatal(err)
			}

			if rp.ExperimentName != c.experimentName {
				t.Errorf("expected experiment name %q, found %q", c.experimentName, rp.ExperimentName)
			}

			if rp.Side != c.side {
				t.Errorf("expected side %q, found %q", c.side, rp.Side)
			}

			if len(rp.Reads) != len(c.reads) {
				t.Errorf("expected %d reads, found %d", len(c.reads), len(rp.Reads))
			}

			testConsumable(t, rp.Flowcell, c.flowcell)

			if len(rp.Consumables) != len(c.consumables) {
				t.Errorf("expected %d consumables, found %d", len(c.consumables), len(rp.Consumables))
			}

			for i, cons := range rp.Consumables {
				testConsumable(t, cons, c.consumables[i])
			}
		})
	}
}

func TestRunParametersVersionUnmarshal(t *testing.T) {
	testcases := []struct {
		name        string
		xml         []byte
		platform    string
		major       int
		minor       int
		patch       int
		shouldError bool
	}{
		{
			name:     "valid nextseq",
			xml:      []byte("<RunParametersVersion>NextSeq_4_0_0</RunParametersVersion>"),
			platform: "NextSeq",
			major:    4,
			minor:    0,
			patch:    0,
		},
		{
			name:     "valid miseq",
			xml:      []byte("<RunParametersVersion>MiSeq_1_3</RunParametersVersion>"),
			platform: "MiSeq",
			major:    1,
			minor:    3,
			patch:    0,
		},
		{
			name:        "weird novaseq",
			xml:         []byte("<InstrumentType>NovaSeqXPlus</InstrumentType>"),
			shouldError: true,
		},
	}

	for _, c := range testcases {
		t.Run(c.name, func(t *testing.T) {
			r := bytes.NewReader(c.xml)
			decoder := xml.NewDecoder(r)
			rpv := runParametersVersion{}
			tok, _ := decoder.Token()
			se, _ := tok.(xml.StartElement)
			err := decoder.DecodeElement(&rpv, &se)

			if err != nil && !c.shouldError {
				t.Fatal(err)
			} else if err == nil && c.shouldError {
				t.Fatal("expected an error, got nil")
			} else if err != nil {
				return
			}

			if rpv.Platform != c.platform {
				t.Fatalf("expected platform %s, got %s", c.platform, rpv.Platform)
			}
			if rpv.Major != c.major {
				t.Fatalf("expected major version %d, got %d", c.major, rpv.Major)
			}
			if rpv.Minor != c.minor {
				t.Fatalf("expected minor version %d, got %d", c.minor, rpv.Minor)
			}
			if rpv.Patch != c.patch {
				t.Fatalf("expected major version %d, got %d", c.patch, rpv.Patch)
			}
		})
	}
}

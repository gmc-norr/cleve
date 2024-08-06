package cleve

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"math"
	"testing"
)

func TestParseHeader(t *testing.T) {
	cases := []struct {
		Name         string
		HeaderString string
		Header       []string
	}{
		{
			"regular csv",
			"column1,column2,column3",
			[]string{"column1", "column2", "column3"},
		},
		{
			"header with spaces",
			"column 1,column 2,column 3",
			[]string{"column 1", "column 2", "column 3"},
		},
		{
			"header with comments",
			"# this is a comment\ncolumn 1,column 2,column 3",
			[]string{"column 1", "column 2", "column 3"},
		},
		{
			"header with components",
			"column1<c1;c2;c3>",
			[]string{"column1 c1", "column1 c2", "column1 c3"},
		},
		{
			"header with comments",
			"# Version: v1.3.1\n# Run Folder: test_run\ncolumn1<c1;c2;c3>",
			[]string{"column1 c1", "column1 c2", "column1 c3"},
		},
		{
			"header with components and regular columns",
			"column1,column2<c1;c2;c3>,column3,column4<first;second>",
			[]string{"column1", "column2 c1", "column2 c2", "column2 c3", "column3", "column4 first", "column4 second"},
		},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			r := csv.NewReader(bytes.NewReader([]byte(c.HeaderString)))
			parser := ImagingTableParser{reader: r}
			header, err := parser.header()
			if err != nil {
				t.Fatal(err)
			}
			if len(header) != len(c.Header) {
				t.Fatalf("expected %d items in header, got %d", len(c.Header), len(header))
			}
			for i := range header {
				if header[i] != c.Header[i] {
					t.Errorf("expected header item %d to be %s, got %s", i, c.Header[i], header[i])
				}
			}
		})
	}
}

func TestParseHeaderRecords(t *testing.T) {
	cases := []struct {
		Name        string
		HeaderIndex map[string]int
		Record      *ImagingRecord
		String      string
	}{
		{
			"regular csv",
			map[string]int{
				"Lane":          0,
				"Tile":          1,
				"Cycle":         2,
				"Read":          3,
				"% Occupied":    4,
				"% Pass Filter": 5,
			},
			&ImagingRecord{
				Lane:            1,
				Tile:            1234,
				Cycle:           53,
				Read:            2,
				PercentOccupied: 99.8,
				PercentPF:       85.3,
			},
			"1,1234,53,2,99.8,85.3",
		},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			r := csv.NewReader(bytes.NewReader([]byte(c.String)))
			parser := ImagingTableParser{reader: r, headerIndex: c.HeaderIndex}
			record, err := parser.record()
			if err != nil {
				t.Fatal(err)
			}
			if record.Lane != c.Record.Lane {
				t.Errorf("expected lane to be %d, got %d", c.Record.Lane, record.Lane)
			}
			if record.Tile != c.Record.Tile {
				t.Errorf("expected tile to be %d, got %d", c.Record.Tile, record.Tile)
			}
			if record.Cycle != c.Record.Cycle {
				t.Errorf("expected cycle to be %d, got %d", c.Record.Cycle, record.Cycle)
			}
			if record.Read != c.Record.Read {
				t.Errorf("expected read to be %d, got %d", c.Record.Read, record.Read)
			}
			if record.PercentOccupied != c.Record.PercentOccupied {
				t.Errorf("expected percent occupied to be %f, got %f", c.Record.PercentOccupied, record.PercentOccupied)
			}
			if record.PercentPF != c.Record.PercentPF {
				t.Errorf("expected percent PF to be %f, got %f", c.Record.PercentPF, record.PercentPF)
			}
		})
	}
}

func TestParseMissingRecords(t *testing.T) {
	cases := []struct {
		Name     string
		Text     string
		Expected []ImagingRecord
	}{
		{
			"missing % occupied",
			`Lane,Tile,Cycle,Read,% Pass Filter
1,1234,53,2,85.3
2,1234,53,2,78.8
1,1234,53,2,81.0`,
			[]ImagingRecord{
				{
					Lane:            1,
					Tile:            1234,
					Cycle:           53,
					Read:            2,
					PercentOccupied: 0.0,
					PercentPF:       85.3,
				},
				{
					Lane:            2,
					Tile:            1234,
					Cycle:           53,
					Read:            2,
					PercentOccupied: 0.0,
					PercentPF:       78.8,
				},
				{
					Lane:            1,
					Tile:            1234,
					Cycle:           53,
					Read:            2,
					PercentOccupied: 0.0,
					PercentPF:       81.0,
				},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			table, err := ParseImagingTable(bufio.NewReader(bytes.NewReader([]byte(c.Text))))
			if err != nil {
				t.Fatal(err)
			}
			if len(table.Records) != len(c.Expected) {
				t.Fatalf("expected %d records, got %d", len(c.Expected), len(table.Records))
			}
			for i := range table.Records {
				if table.Records[i] != c.Expected[i] {
					t.Errorf("expected record %d to be %+v, got %+v", i, c.Expected[i], table.Records[i])
				}
			}
		})
	}
}

func TestLaneTileSummary(t *testing.T) {
	cases := []struct {
		Name            string
		Table           *ImagingTable
		PercentOccupied map[int]map[int]float64
		PercentPF       map[int]map[int]float64
	}{
		{
			"summary test 1",
			&ImagingTable{
				Records: []ImagingRecord{
					{
						Lane:            1,
						Tile:            1234,
						Cycle:           53,
						Read:            2,
						PercentOccupied: 99.8,
						PercentPF:       85.3,
					},
					{
						Lane:            1,
						Tile:            1234,
						Cycle:           53,
						Read:            2,
						PercentOccupied: 99.8,
						PercentPF:       85.3,
					},
				},
			},
			map[int]map[int]float64{
				1: {
					1234: 99.8,
				},
			},
			map[int]map[int]float64{
				1: {
					1234: 85.3,
				},
			},
		},
		{
			"summary test 2",
			&ImagingTable{
				Records: []ImagingRecord{
					{
						Lane:            1,
						Tile:            1234,
						Cycle:           53,
						Read:            2,
						PercentOccupied: 99.8,
						PercentPF:       85.3,
					},
					{
						Lane:            1,
						Tile:            1234,
						Cycle:           53,
						Read:            2,
						PercentOccupied: 98.2,
						PercentPF:       78.5,
					},
					{
						Lane:            1,
						Tile:            1235,
						Cycle:           53,
						Read:            2,
						PercentOccupied: 99.8,
						PercentPF:       85.3,
					},
					{
						Lane:            1,
						Tile:            1235,
						Cycle:           53,
						Read:            2,
						PercentOccupied: 98.1,
						PercentPF:       81.7,
					},
					{
						Lane:            2,
						Tile:            1234,
						Cycle:           53,
						Read:            2,
						PercentOccupied: 99.8,
						PercentPF:       85.3,
					},
					{
						Lane:            2,
						Tile:            1234,
						Cycle:           53,
						Read:            2,
						PercentOccupied: 98.2,
						PercentPF:       78.5,
					},
					{
						Lane:            2,
						Tile:            1235,
						Cycle:           53,
						Read:            2,
						PercentOccupied: 99.8,
						PercentPF:       85.3,
					},
					{
						Lane:            2,
						Tile:            1235,
						Cycle:           53,
						Read:            2,
						PercentOccupied: 98.1,
						PercentPF:       81.7,
					},
				},
			},
			map[int]map[int]float64{
				1: {
					1234: 99.0,
					1235: 98.95,
				},
				2: {
					1234: 99.0,
					1235: 98.95,
				},
			},
			map[int]map[int]float64{
				1: {
					1234: 81.9,
					1235: 83.5,
				},
				2: {
					1234: 81.9,
					1235: 83.5,
				},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			summary := c.Table.LaneTileSummary()
			for _, tileStats := range summary {
				observed := tileStats.PercentOccupied
				expected := c.PercentOccupied[tileStats.Lane][tileStats.Tile]
				if math.Abs(observed-expected) > 1e-6 {
					t.Errorf("expected percent occupied to be %f, got %f", expected, observed)
				}

				observed = tileStats.PercentPF
				expected = c.PercentPF[tileStats.Lane][tileStats.Tile]
				if math.Abs(observed-expected) > 1e-6 {
					t.Errorf("expected percent PF to be %f, got %f", expected, observed)
				}
			}
		})
	}
}

func TestGenerateImagingTable(t *testing.T) {
	cases := []struct {
		Name string
		Path string
	}{
		{
			"novaseq",
			"test_data/novaseq_full",
		},
		{
			"nextseq",
			"test_data/nextseq1_full",
		},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			_, err := GenerateImagingTable(c.Name, c.Path)
			if err != nil {
				t.Fatalf("%s when generating imaging table", err.Error())
			}
		})
	}
}

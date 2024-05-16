package cleve

import (
	"encoding/xml"
	"io"
	"os"
	"testing"
	"time"
)

func TestRunparameters(t *testing.T) {
	cases := []struct {
		name     string
		filename string
		flowcell string
	}{
		{
			"novaseq",
			"test_data/novaseq_full/RunParameters.xml",
			"1.5B",
		},
		{
			"nextseq",
			"test_data/nextseq1_full/RunParameters.xml",
			"NextSeq Mid",
		},
	}

	for _, c := range cases {
		f, err := os.Open(c.filename)
		if err != nil {
			t.Fatal(err)
		}
		b, err := io.ReadAll(f)
		if err != nil {
			t.Fatal(err)
		}
		rp, err := ParseRunParameters([]byte(b))
		if err != nil {
			t.Fatal(err)
		}

		if rp.Flowcell() != c.flowcell {
			t.Errorf("expected flowcell %q, got %q", c.flowcell, rp.Flowcell())
		}
	}
}

func TestCustomTime(t *testing.T) {
	cases := []struct {
		input  string
		year   int
		month  int
		day    int
		hour   int
		minute int
		second int
	}{
		{
			"<Doc><Time>240313</Time></Doc>",
			2024,
			3,
			13,
			0,
			0,
			0,
		},
		{
			"<Doc><Time>2018-12-24</Time></Doc>",
			2018,
			12,
			24,
			0,
			0,
			0,
		},
		{
			"<Doc><Time>2024-04-03T19:06:20Z</Time></Doc>",
			2024,
			4,
			3,
			19,
			6,
			20,
		},
		{
			"<Doc><Time>2024-09-12T00:00:00+02:00</Time></Doc>",
			2024,
			9,
			12,
			0,
			0,
			0,
		},
	}

	for _, c := range cases {
		var res struct {
			Time CustomTime `xml:"Time"`
		}
		b := []byte(c.input)
		if err := xml.Unmarshal(b, &res); err != nil {
			t.Fatal(err)
		}
		t.Log(res.Time.String())
		if (time.Time)(res.Time).Year() != c.year {
			t.Errorf("expected year %d, got %d", c.year, (time.Time)(res.Time).Year())
		}
		if (time.Time)(res.Time).Month() != time.Month(c.month) {
			t.Errorf("expected month %d, got %d", c.month, (time.Time)(res.Time).Month())
		}
		if (time.Time)(res.Time).Day() != c.day {
			t.Errorf("expected day %d, got %d", c.day, (time.Time)(res.Time).Day())
		}
		if (time.Time)(res.Time).Hour() != c.hour {
			t.Errorf("expected day %d, got %d", c.hour, (time.Time)(res.Time).Hour())
		}
		if (time.Time)(res.Time).Minute() != c.minute {
			t.Errorf("expected day %d, got %d", c.minute, (time.Time)(res.Time).Minute())
		}
		if (time.Time)(res.Time).Second() != c.second {
			t.Errorf("expected day %d, got %d", c.second, (time.Time)(res.Time).Second())
		}
	}
}

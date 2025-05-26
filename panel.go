package cleve

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"slices"
	"strconv"
	"strings"
	"time"
)

type Time struct {
	time.Time
}

type GenePanel struct {
	GenePanelVersion `bson:",inline" json:",inline"`
	Id               string    `json:"id"`
	Name             string    `json:"name"`
	Description      string    `json:"description"`
	Categories       []string  `json:"categories"`
	Genes            []Gene    `json:"genes,omitzero"`
	Archived         bool      `json:"archived"`
	ArchivedAt       time.Time `bson:",omitzero" json:"archived_at,omitzero"`
}

type GenePanelVersion struct {
	Version Version   `bson:"-" json:"version"`
	Date    time.Time `json:"date"`
}

type Gene struct {
	HGNC    int
	Symbol  string
	Aliases []string
}

func NewGenePanel(name string, description string) GenePanel {
	n := time.Now()
	return GenePanel{
		GenePanelVersion: GenePanelVersion{
			Version: NewMinorVersion(1, 0),
			Date:    n,
		},
		Id:          name,
		Name:        name,
		Description: description,
		Genes:       make([]Gene, 0),
		ArchivedAt:  n,
	}
}

func (p *GenePanel) AddCategory(category string) {
	cat := strings.ToLower(strings.TrimSpace(category))
	if !slices.Contains(p.Categories, cat) {
		p.Categories = append(p.Categories, cat)
	}
}

func (p GenePanel) Validate() error {
	if p.Id == "" {
		return errors.New("panel must have an id")
	}
	if p.Name == "" {
		return errors.New("panel must have a name")
	}
	if p.Version.IsZero() {
		return errors.New("panel must have a version")
	}
	if len(p.Genes) == 0 {
		return errors.New("panel must contain at least one gene")
	}
	return nil
}

func parseKeyValue(s string) (string, string, error) {
	elems := strings.SplitN(s, "=", 2)
	if len(elems) != 2 {
		return "", "", errors.New("invalid metadata line")
	}
	key, _ := strings.CutPrefix(elems[0], "##")
	value := elems[1]
	return key, value, nil
}

func genePanelFromText(r io.Reader, delim rune) (GenePanel, error) {
	var p GenePanel

	csvReader := csv.NewReader(r)
	csvReader.Comma = delim
	csvReader.FieldsPerRecord = -1

	var (
		parsedHeader bool
		header       []string
		nRecords     int
	)

	records, err := csvReader.ReadAll()
	if err != nil {
		return p, err
	}

	for line, rec := range records {
		if strings.HasPrefix(rec[0], "##") {
			key, value, err := parseKeyValue(strings.Join(rec, fmt.Sprintf("%c", delim)))
			if err != nil {
				return p, err
			}
			switch key {
			case "display_name", "name":
				p.Name = value
			case "panel_id", "id":
				p.Id = value
			case "version":
				p.Version, err = ParseVersion(value)
				if err != nil {
					return p, fmt.Errorf("error parsing version on line %d: %w", line, err)
				}
				if p.Version.HasPatch() {
					return p, fmt.Errorf("version must only have major and minor numbers")
				}
			case "date":
				p.Date, err = time.Parse("2006-01-02", value)
				if err != nil {
					return p, fmt.Errorf("error parsing date on line %d: %w", line, err)
				}
			case "description":
				p.Description = value
			case "categories":
				for cat := range strings.SplitSeq(value, ",") {
					p.AddCategory(cat)
				}
			default:
				slog.Warn("unknown metadata field", "key", key)
			}
		} else if strings.HasPrefix(rec[0], "#") {
			if parsedHeader {
				return p, errors.New("multiple header lines found")
			}
			header = rec
			header[0], _ = strings.CutPrefix(header[0], "#")
			parsedHeader = true
		} else {
			if !parsedHeader {
				header = []string{"hgnc_id", "hgnc_symbol", "disease_associated_transcripts", "mosaicism", "reduced_penetrance"}
				parsedHeader = true
			}

			if nRecords == 0 {
				nRecords = len(rec)
			} else if len(rec) != nRecords {
				return p, fmt.Errorf("record on line %d: wrong number of fields", line+1)
			}

			g := Gene{}

			for i, val := range rec {
				switch header[i] {
				case "hgnc", "hgnc_id":
					hgnc, err := strconv.Atoi(val)
					if err != nil {
						return p, fmt.Errorf("failed to parse HGNC ID: %w", err)
					}
					g.HGNC = hgnc
				case "hgnc_symbol", "symbol":
					g.Symbol = val
				}
			}

			if g.HGNC == 0 {
				return p, fmt.Errorf("missing HGNC ID for record on line %d", line+1)
			}

			p.Genes = append(p.Genes, g)
		}
	}

	return p, err
}

func GenePanelFromCsv(r io.Reader) (GenePanel, error) {
	return genePanelFromText(r, ';')
}

func GenePanelFromTsv(r io.Reader) (GenePanel, error) {
	return genePanelFromText(r, '\t')
}

func GenePanelFromYaml(r io.Reader) (GenePanel, error) {
	var p GenePanel
	return p, nil
}

func (p *GenePanel) Add(gene Gene) {
	p.Genes = append(p.Genes, gene)
}

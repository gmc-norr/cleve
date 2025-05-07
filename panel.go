package cleve

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
)

type GenePanel struct {
	Id          string
	Name        string
	Version     string
	Description string
	Genes       []Gene
}

type Gene struct {
	HGNC    string
	Symbol  string
	Aliases []string
}

func NewGenePanel(name string, description string) GenePanel {
	return GenePanel{
		Id:          name,
		Name:        name,
		Description: description,
		Version:     "1.0",
		Genes:       make([]Gene, 0),
	}
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
				p.Version = value
			case "description":
				p.Description = value
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
					g.HGNC = val
				case "hgnc_symbol", "symbol":
					g.Symbol = val
				}
			}

			if g.HGNC == "" {
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

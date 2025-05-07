package panel

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gmc-norr/cleve"
	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add [flags] panel-definition",
	Short: "Add a gene panel to the database",
	Long: `Panels can be added from a couple of different file formats: csv, tsv.
CSV and TSV should be semicolon-separated and tab-separated, respectively. Each
row represents a gene and should contain the following columns:

- hgnc: HGNC ID of the gene (aliases: hgnc_id)
- symbol: The symbol to use for the gene (aliases: hgnc_symbol)
- disease_associated_transcripts: Comma-separated list of manually curated transcripts
- genetic_disease_models: Comma-separated list of manually curated inheritance patterns that are followed by a gene
- mosaicism: Whether the gene is known to be associated with mosaicism
- reduced_penetrance: Whether the gene is known to have reduced penetrance

The file should contain a header line that should begin with '#'. If the header is missing
the order of the columns must match the order given above. Metadata can also be defined
in the header as key-value pairs: '##key=value'. Supported keys are:

- id: Internal ID of the panel (aliases: panel_id)
- name: The name of the panel (aliases: display_name)
- version: Panel version
- description: Free-text description of the panel
`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		fileType, _ := cmd.Flags().GetString("filetype")
		f, err := os.Open(args[0])
		cobra.CheckErr(err)
		defer f.Close()

		var (
			parseError error
			p          cleve.GenePanel
		)
		switch fileType {
		case "tsv":
			p, parseError = cleve.GenePanelFromTsv(f)
		case "csv":
			p, parseError = cleve.GenePanelFromCsv(f)
		default:
			cobra.CheckErr("invalid filetype, should be one of tsv, csv")
		}
		cobra.CheckErr(parseError)

		if p.Name == "" {
			fname := filepath.Base(f.Name())
			stem := fname[:len(fname)-len(filepath.Ext(fname))]
			p.Name = filepath.Base(stem)
		}
		if p.Id == "" {
			p.Id = strings.ToLower(p.Name)
		}
		if p.Version == "" {
			p.Version = "1.0"
		}

		fmt.Printf("%+v\n", p)
	},
}

func init() {
	addCmd.Flags().StringP("name", "n", "", "name of the new panel, defaults to the name of the definition file")
	addCmd.Flags().StringP("id", "i", "", "ID for the new panel, defaults to a slug of the name of the definition file")
	addCmd.Flags().StringP("filetype", "f", "tsv", "filetype of the panel definition file")
}

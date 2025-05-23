package panel

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gmc-norr/cleve"
	"github.com/gmc-norr/cleve/mongo"
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
- date: Creation date of the panel (ISO)
- description: Free-text description of the panel
- categories: Comma-separated list of categories that the panel belongs to

If any of the metadata are given as parameters on the command line, these take precedenc
over what is in the file.
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

		if name, _ := cmd.Flags().GetString("name"); name != "" {
			p.Name = name
		} else if p.Name == "" {
			fname := filepath.Base(f.Name())
			stem := fname[:len(fname)-len(filepath.Ext(fname))]
			p.Name = filepath.Base(stem)
		}
		if id, _ := cmd.Flags().GetString("id"); id != "" {
			p.Id = id
		} else if p.Id == "" {
			p.Id = strings.ToLower(p.Name)
		}
		if version, _ := cmd.Flags().GetString("version"); version != "" {
			p.Version, err = cleve.ParseVersion(version)
			cobra.CheckErr(err)
			if p.Version.HasPatch() {
				cobra.CheckErr("version numbers may only contain major and minor numbers")
			}
		} else if p.Version.IsZero() {
			p.Version = cleve.NewMinorVersion(1, 0)
		}
		if description, _ := cmd.Flags().GetString("description"); description != "" {
			p.Description = description
		}
		if date, _ := cmd.Flags().GetString("date"); date != "" {
			p.Date, err = time.Parse("2006-01-02", date)
			cobra.CheckErr(err)
		} else if p.Date.IsZero() {
			p.Date = time.Now()
		}
		if categories, _ := cmd.Flags().GetStringSlice("categories"); len(categories) != 0 {
			for _, cat := range categories {
				p.AddCategory(cat)
			}
		}
		err = p.Validate()
		cobra.CheckErr(err)

		db, err := mongo.Connect()
		cobra.CheckErr(err)

		err = db.CreatePanel(p)
		if mongo.IsDuplicateKeyError(err) {
			cobra.CheckErr("a panel with this id and version already exists")
		}
		cobra.CheckErr(err)
	},
}

func init() {
	addCmd.Flags().StringP("id", "i", "", "ID for the new panel, defaults to a slug of the name of the definition file")
	addCmd.Flags().StringP("name", "n", "", "name of the new panel, defaults to the name of the definition file")
	addCmd.Flags().StringP("version", "v", "", "version for the new panel")
	addCmd.Flags().StringP("description", "d", "", "free-text description of the new panel")
	addCmd.Flags().StringSlice("categories", make([]string, 0), "comma-separated list of categories that the panel should belong to")
	addCmd.Flags().String("date", "", "creation date of the panel")
	addCmd.Flags().StringP("filetype", "f", "tsv", "filetype of the panel definition file")
}

package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/gmc-norr/cleve"
	"github.com/gmc-norr/cleve/mongo"
	"github.com/spf13/cobra"
)

func NewPanelCmd(cfg *CliConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "panel",
		Short: "Manage in-silico gene panels",
	}

	cmd.AddCommand(NewPanelAddCmd(cfg))
	cmd.AddCommand(NewPanelArchiveCmd(cfg))
	cmd.AddCommand(NewPanelListCmd(cfg))
	cmd.AddCommand(NewPanelDeleteCmd(cfg))

	return cmd
}

func NewPanelAddCmd(cfg *CliConfig) *cobra.Command {
	var (
		id          string
		name        string
		version     string
		description string
		date        string
		categories  []string
		fileType    string
	)
	cmd := &cobra.Command{
		Use:   "add",
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
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("panel definition required")
			}
			if len(args) != 1 {
				return fmt.Errorf("too many arguments")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := cfg.DB()
			if err != nil {
				return err
			}
			f, err := os.Open(args[0])
			cobra.CheckErr(err)
			defer func() {
				if err := f.Close(); err != nil {
					cfg.Logger.Error("failed to close file", "error", err)
				}
			}()

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

			if name != "" {
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
			if version != "" {
				p.Version, err = cleve.ParseVersion(version)
				cobra.CheckErr(err)
				if p.Version.HasPatch() {
					cobra.CheckErr("version numbers may only contain major and minor numbers")
				}
			} else if p.Version.IsZero() {
				p.Version = cleve.NewMinorVersion(1, 0)
			}
			if description != "" {
				p.Description = description
			}
			if date != "" {
				p.Date, err = time.Parse("2006-01-02", date)
				cobra.CheckErr(err)
			} else if p.Date.IsZero() {
				p.Date = time.Now()
			}
			if len(categories) != 0 {
				for _, cat := range categories {
					p.AddCategory(cat)
				}
			}
			err = p.Validate()
			cobra.CheckErr(err)

			err = db.CreatePanel(p)
			if mongo.IsDuplicateKeyError(err) {
				cobra.CheckErr("a panel with this id and version already exists")
			}
			return err
		},
	}

	cmd.Flags().StringVarP(&id, "id", "i", "", "ID for the new panel, defaults to a slug of the name of the definition file")
	cmd.Flags().StringVarP(&name, "name", "n", "", "Name of the new panel, defaults to the name of the definition file")
	cmd.Flags().StringVarP(&version, "version", "v", "", "Version for the new panel")
	cmd.Flags().StringVarP(&description, "description", "d", "", "Free-text description of the new panel")
	cmd.Flags().StringSliceVar(&categories, "categories", make([]string, 0), "Comma-separated list of categories that the panel should belong to")
	cmd.Flags().StringVar(&date, "date", "", "Creation date of the panel")
	cmd.Flags().StringVarP(&fileType, "filetype", "f", "tsv", "Filetype of the panel definition file")

	return cmd
}

func NewPanelArchiveCmd(cfg *CliConfig) *cobra.Command {
	var unarchive bool
	cmd := &cobra.Command{
		Use:   "archive panel",
		Short: "Archive or unarchive a gene panel",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("panel id required")
			}
			if len(args) > 1 {
				return fmt.Errorf("too many arguments")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := cfg.DB()
			if err != nil {
				return err
			}
			cfg.Logger.Info("modifying panel", "id", args[0], "archive", !unarchive)
			if unarchive {
				err = db.UnarchivePanel(args[0])
			} else {
				err = db.ArchivePanel(args[0])
			}
			if err != nil {
				if errors.Is(err, mongo.ErrNoDocuments) {
					_ = fmt.Sprintf("panel with ID %q not found", args[0])
				}
				return err
			}
			return nil
		},
	}

	cmd.Flags().BoolVarP(&unarchive, "unarchive", "u", false, "Unarchive a panel")

	return cmd
}

func NewPanelListCmd(cfg *CliConfig) *cobra.Command {
	var (
		showAll bool
		gene    string
	)
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List panels",
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := cfg.DB()
			if err != nil {
				return err
			}

			filter := cleve.NewPanelFilter()
			filter.Archived = showAll
			filter.Gene = gene

			panels, err := db.Panels(filter)
			if err != nil {
				return err
			}

			w := tabwriter.NewWriter(os.Stdout, 5, 4, 2, ' ', 0)
			_, _ = fmt.Fprintln(w, "id\tname\tversion\tarchived")
			_, _ = fmt.Fprintln(w, "--\t----\t-------\t--------")
			for _, p := range panels {
				_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%t\n", p.Id, p.Name, p.Version.String(), p.Archived)
			}
			_ = w.Flush()
			return nil
		},
	}

	cmd.Flags().BoolVarP(&showAll, "all", "a", false, "Show also archived panels")
	cmd.Flags().StringVarP(&gene, "gene", "g", "", "List panels containing a certain gene (symbol, case insensitive)")

	return cmd
}

func NewPanelDeleteCmd(cfg *CliConfig) *cobra.Command {
	var (
		deleteID      string
		deleteVersion string
	)
	cmd := &cobra.Command{
		Use:   "delete id [version]",
		Short: "Delete panel(s)",
		Long:  "Delete on or more panels. If version is omitted, all versions with the specified id will be deleted.",
		PreRun: func(cmd *cobra.Command, args []string) {
			r := bufio.NewReader(os.Stdin)
			deleteAll := deleteVersion == ""
			for {
				if deleteAll {
					fmt.Printf("this will delete all versions of %s, continue? [Y/n] ", deleteID)
				} else {
					fmt.Printf("this will delete version %s of %s, continue? [Y/n] ", deleteVersion, deleteID)
				}
				s, _ := r.ReadString('\n')
				s = strings.TrimSpace(s)
				s = strings.ToLower(s)
				if s == "n" {
					fmt.Fprintln(os.Stderr, "canceled")
					os.Exit(0)
					return
				} else if s == "y" {
					break
				} else {
					fmt.Fprintln(os.Stderr, "answer with Y or N")
					continue
				}
			}
		},
		Args: func(cmd *cobra.Command, args []string) error {
			if err := cobra.MinimumNArgs(1)(cmd, args); err != nil {
				return err
			}
			if err := cobra.MaximumNArgs(2)(cmd, args); err != nil {
				return err
			}
			deleteID = args[0]
			if len(args) > 1 {
				deleteVersion = args[1]
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := cfg.DB()
			if err != nil {
				return err
			}

			n, err := db.DeletePanel(deleteID, deleteVersion)
			cobra.CheckErr(err)
			if n == 0 && deleteVersion == "" {
				return fmt.Errorf("no entries found for %s", deleteID)
			} else if n == 0 {
				return fmt.Errorf("version %s not found for %s", deleteVersion, deleteID)
			} else if n == 1 {
				fmt.Printf("deleted %d entry of %s\n", n, deleteID)
			} else {
				fmt.Printf("deleted %d entries of %s\n", n, deleteID)
			}
			return nil
		},
	}

	return cmd
}

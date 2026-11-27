package main

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

func NewPlatformCmd(cfg *CliConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "platform",
		Short: "Manage sequencing platforms",
	}

	cmd.AddCommand(NewPlatformListCmd(cfg))

	return cmd
}

func NewPlatformListCmd(cfg *CliConfig) *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List sequencing platforms",
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := cfg.DB()
			if err != nil {
				return nil
			}

			platforms, err := db.Platforms()
			if err != nil {
				return err
			}

			if jsonOutput {
				if err := printJSON(platforms); err != nil {
					return err
				}
			} else {
				w := tabwriter.NewWriter(os.Stdout, 2, 4, 2, ' ', 0)
				_, _ = fmt.Fprint(w, "name\tinstrument ids\trun count\tready marker\taliases\n")
				_, _ = fmt.Fprint(w, "----\t--------------\t---------\t------------\t-------\n")
				for _, p := range platforms.Platforms {
					_, _ = fmt.Fprintf(w, "%s\t%v\t%d\t%s\t%v\n", p.Name, p.InstrumentIds, p.RunCount, p.ReadyMarker, p.Aliases)
				}
				_ = w.Flush()
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output json")

	return cmd
}

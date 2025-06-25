package panel

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/gmc-norr/cleve"
	"github.com/gmc-norr/cleve/mongo"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list [flags]",
	Short: "List panels",
	Run: func(cmd *cobra.Command, args []string) {
		db, err := mongo.Connect()
		cobra.CheckErr(err)

		showAll, _ := cmd.Flags().GetBool("all")
		gene, _ := cmd.Flags().GetString("gene")

		filter := cleve.NewPanelFilter()
		filter.Archived = showAll
		filter.Gene = gene

		panels, err := db.Panels(filter)
		cobra.CheckErr(err)

		w := tabwriter.NewWriter(os.Stdout, 5, 4, 2, ' ', 0)
		_, _ = fmt.Fprintln(w, "id\tname\tversion\tarchived")
		_, _ = fmt.Fprintln(w, "--\t----\t-------\t--------")
		for _, p := range panels {
			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%t\n", p.Id, p.Name, p.Version.String(), p.Archived)
		}
		_ = w.Flush()
	},
}

func init() {
	listCmd.Flags().BoolP("all", "a", false, "show also archived panels")
	listCmd.Flags().StringP("gene", "g", "", "list panels containing a certain gene (symbol, case insensitive)")
}

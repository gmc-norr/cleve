package platform

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"text/tabwriter"

	"github.com/gmc-norr/cleve/mongo"
	"github.com/spf13/cobra"
)

var (
	jsonOutput bool
	listCmd    = &cobra.Command{
		Use:   "list [flags]",
		Short: "List platforms in the database",
		Run: func(cmd *cobra.Command, args []string) {
			db, err := mongo.Connect()
			if err != nil {
				log.Fatal(err)
			}

			platforms, err := db.Platforms()
			if err != nil {
				log.Fatalf("error: %s", err)
			}

			if jsonOutput {
				jsonString, err := json.Marshal(&platforms)
				if err != nil {
					log.Fatalf("error: %s", err)
				}
				fmt.Println(string(jsonString))
			} else {
				w := tabwriter.NewWriter(os.Stdout, 2, 4, 2, ' ', 0)
				_, _ = fmt.Fprint(w, "name\tinstrument ids\trun count\tready marker\taliases\n")
				_, _ = fmt.Fprint(w, "----\t--------------\t---------\t------------\t-------\n")
				for _, p := range platforms.Platforms {
					_, _ = fmt.Fprintf(w, "%s\t%v\t%d\t%s\t%v\n", p.Name, p.InstrumentIds, p.RunCount, p.ReadyMarker, p.Aliases)
				}
				_ = w.Flush()
			}
		},
	}
)

func init() {
	listCmd.Flags().BoolVar(&jsonOutput, "json", false, "Output json")
}

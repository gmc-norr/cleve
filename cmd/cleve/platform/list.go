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
				fmt.Printf(string(jsonString))
			} else {
				w := tabwriter.NewWriter(os.Stdout, 2, 4, 2, ' ', 0)
				fmt.Fprint(w, "name\tserial tag\tserial prefix\tready marker\n")
				fmt.Fprint(w, "----\t----------\t-------------\t------------\n")
				for _, p := range platforms {
					fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", p.Name, p.SerialTag, p.SerialPrefix, p.ReadyMarker)
				}
				w.Flush()
			}
		},
	}
)

func init() {
	listCmd.Flags().BoolVar(&jsonOutput, "json", false, "Output json")
}

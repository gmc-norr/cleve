package panelqc

import (
	"fmt"
	"log"
	"strings"

	"github.com/gmc-norr/cleve"
	"github.com/gmc-norr/cleve/mongo"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list [flags]",
	Short: "List panel QC data",
	Run: func(cmd *cobra.Command, args []string) {

		db, err := mongo.Connect()
		if err != nil {
			log.Fatalf("error: %s", err)
		}
		// TODO Add so you can filter by runID -> see list.go for run
		filter := cleve.NewPanelQcFilter()

		panelqc, err := db.PanelQCs(filter)
		
		if err != nil {
				log.Fatal(err)
			}
		
		printTable(panelqc.PanelQCs)
		
	},
}

func printTable(panelqcs []*cleve.PanelQc) {
	for _, panelqc := range panelqcs {
		fmt.Printf("|-%s-|-%s-|\n", strings.Repeat("-", 10), strings.Repeat("-", 40))
		fmt.Printf("| %-10s | %-40s |\n", "Run ID", "Created At")
		fmt.Printf("|-%s-|-%s-|\n", strings.Repeat("-", 10), strings.Repeat("-", 40))
		fmt.Printf("| %s | %s |\n", panelqc.RunId, panelqc.Created.String())
		for _, genepanel := range panelqc.GenePanel {
			fmt.Printf("|-%s-|-%s-|\n", strings.Repeat("-", 10), strings.Repeat("-", 20))
			fmt.Printf("| %-10s | %-20s | %-20s | %-20s | \n", "GenePanelID", "Completness", "Feature Fraction", "Number of Samples")
			fmt.Printf("|-%s-|-%s-|\n", strings.Repeat("-", 32), strings.Repeat("-", 20))
			fmt.Printf("| %s | %f | %f | %v |\n", genepanel.GenePanelId, genepanel.CompletnessThreshold, genepanel.FeatureFraction, genepanel.NumberSamples)
		}
		fmt.Printf("|-%s-|-%s-|\n", strings.Repeat("-", 32), strings.Repeat("-", 20))
	}

}
package panelqc

import (
	"fmt"
	"log/slog"
	"strings"
	"os"
	"strconv"

	"github.com/gmc-norr/cleve"
	"github.com/gmc-norr/cleve/mongo"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list [flags] run_id",
	Short: "List panel QC data for a specific Run ID",
	Args: func(cmd *cobra.Command, args []string) error {
		
		if len(args) != 1 {
			return fmt.Errorf("Run ID is required")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {

		db, err := mongo.Connect()
		if err != nil {
			slog.Error("failed to connect to database", "error", err)
			os.Exit(1)
		}

		// Check if RunID exists
		panelqc, err := db.PanelQc(args[0])
		if err != nil {
			slog.Error("failed to extract data of RunId", args[0], "error", err)
			os.Exit(1)
		}

		printTable(panelqc)
	},
}

func truncateString(str string, maxLen int) string {
	if len(str) > maxLen {
		return str[0:(maxLen-3)] + "..."
	}
	return fmt.Sprintf("%-*s", maxLen, str)
}


func printTable(panelqc *cleve.PanelQc) {
	fmt.Printf("|-%s-|-%s-|\n", strings.Repeat("-", 35), strings.Repeat("-", 8))
	fmt.Printf("| %-35s | %-8s |\n", "Run ID", "Date")
	fmt.Printf("|%s |%s|\n", strings.Repeat("-", 36), strings.Repeat("-", 10))
	fmt.Printf("| %s | %s |\n", truncateString(panelqc.RunId, 35), truncateString(strconv.FormatInt(int64(panelqc.Date), 10), 8))
	fmt.Printf("|%s|\n", strings.Repeat("-", 48))
	for _, genepanel := range panelqc.GenePanel {
		fmt.Printf("|-%s-|-%s-|-%s-|-%s-|\n", strings.Repeat("-", 25), strings.Repeat("-", 15), strings.Repeat("-", 16), strings.Repeat("-", 20))
		fmt.Printf("| %-25s | %-15s | %-16s | %-20s | \n", "GenePanelID", "Completness", "Feature Fraction", "Number of Samples")
		fmt.Printf("|-%s-|-%s-|-%s-|-%s-|\n", strings.Repeat("-", 25), strings.Repeat("-", 15), strings.Repeat("-", 16), strings.Repeat("-", 20))
		fmt.Printf("| %s | %s | %s | %s |\n", truncateString(genepanel.GenePanelId, 25), truncateString(strconv.FormatFloat(genepanel.CompletnessThreshold, 'f', -1, 64), 15), truncateString(strconv.FormatFloat(genepanel.FeatureFraction, 'f', -1, 64), 16), truncateString(strconv.FormatInt(int64(genepanel.NumberSamples), 10), 20))
	}
	fmt.Printf("|-%s-|-%s-|-%s-|-%s-|\n", strings.Repeat("-", 25), strings.Repeat("-", 15), strings.Repeat("-", 16), strings.Repeat("-", 20))
}


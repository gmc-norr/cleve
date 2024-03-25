package run

import (
	"encoding/json"
	"fmt"
	"github.com/gmc-norr/cleve/internal/db"
	"github.com/spf13/cobra"
	"log"
	"os"
	"strings"
)

var csvOutput, jsonOutput, brief bool
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List sequencing runs",
	PreRun: func(cmd *cobra.Command, args []string) {
		if csvOutput && jsonOutput {
			log.Fatal("Cannot specify both --csv and --json")
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		db.Init()
		runs, err := db.GetRuns(brief)
		if err != nil {
			log.Fatal(err)
		}
		if csvOutput {
			printCSV(runs)
		} else if jsonOutput {
			printJSON(runs)
		} else {
			printTable(runs)
		}
	},
}

func init() {
	listCmd.Flags().BoolVar(&csvOutput, "csv", false, "Output in CSV format")
	listCmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in CSV format")
	listCmd.Flags().BoolVar(&brief, "brief", false, "Brief output")
}

func truncateString(str string, maxLen int) string {
	if len(str) > maxLen {
		return str[0:(maxLen-3)] + "..."
	}
	return fmt.Sprintf("%-*s", maxLen, str)
}

func printTable(runs []*db.Run) {
	fmt.Printf("|-%s-|-%s-|-%s-|-%s-|\n", strings.Repeat("-", 32), strings.Repeat("-", 20), strings.Repeat("-", 12), strings.Repeat("-", 33))
	fmt.Printf("| %-32s | %-20s | %-12s | %-33s |\n", "Run ID", "Run Name", "Platform", "Created At")
	fmt.Printf("|-%s-|-%s-|-%s-|-%s-|\n", strings.Repeat("-", 32), strings.Repeat("-", 20), strings.Repeat("-", 12), strings.Repeat("-", 33))
	for _, run := range runs {
		fmt.Printf("| %s | %s | %s | %s |\n", truncateString(run.RunID, 32), truncateString(run.ExperimentName, 20), truncateString(run.Platform, 12), truncateString(run.Created.String(), 33))
	}
	fmt.Printf("|-%s-|-%s-|-%s-|-%s-|\n", strings.Repeat("-", 32), strings.Repeat("-", 20), strings.Repeat("-", 12), strings.Repeat("-", 33))
}

func printCSV(runs []*db.Run) {
	fmt.Printf("Run ID,Run Name,Created At\n")
	for _, run := range runs {
		fmt.Printf("%s,%s,%s\n", run.RunID, run.ExperimentName, run.Created.String())
	}
}

func printJSON(runs []*db.Run) {
	json.NewEncoder(os.Stdout).Encode(runs)
}

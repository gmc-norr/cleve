package run

import (
	"fmt"
	"github.com/gmc-norr/cleve/analysis"
	"github.com/gmc-norr/cleve/internal/db"
	"github.com/gmc-norr/cleve/internal/db/runstate"
	"github.com/gmc-norr/cleve/runparameters"
	"github.com/spf13/cobra"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

var runPath string

func parseRunParameters(runParametersFile string) (runparameters.RunParameters, error) {
	var runParams runparameters.RunParameters
	runParamFile, err := os.Open(runParametersFile)
	if err != nil {
		return runParams, err
	}
	defer runParamFile.Close()
	runParamData, err := io.ReadAll(runParamFile)
	if err != nil {
		return runParams, err
	}

	return runparameters.ParseRunParameters(runParamData)
}

var addCmd = &cobra.Command{
	Use:   "add [flags] run_directory",
	Short: "Add a sequencing run",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return fmt.Errorf("run directory is required")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		db.Init()

		runDir, err := filepath.Abs(args[0])
		if err != nil {
			log.Fatal(err)
		}
		runParametersFile := filepath.Join(runDir, "RunParameters.xml")

		runParams, err := parseRunParameters(runParametersFile)
		if err != nil {
			log.Fatal(err)
		}

		var state runstate.RunState
		if err = state.Set("new"); err != nil {
			log.Fatal(err)
		}

		run := db.Run{
			RunID:          runParams.GetRunID(),
			ExperimentName: runParams.GetExperimentName(),
			Path:           runDir,
			Platform:       runParams.Platform(),
			RunParameters:  runParams,
			StateHistory:   []runstate.TimedRunState{{State: state, Time: time.Now()}},
			Analysis:       []*analysis.Analysis{},
		}

		if err = db.AddRun(&run); err != nil {
			log.Fatal(err)
		}

		log.Printf("Added run %s as %s", run.RunID, run.ID.Hex())
	},
}

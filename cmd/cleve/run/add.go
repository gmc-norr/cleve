package run

import (
	"fmt"
	"github.com/gmc-norr/cleve"
	"github.com/gmc-norr/cleve/mongo"
	"github.com/spf13/cobra"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

var runPath string

func parseRunParameters(runParametersFile string) (cleve.RunParameters, error) {
	var runParams cleve.RunParameters
	runParamFile, err := os.Open(runParametersFile)
	if err != nil {
		return runParams, err
	}
	defer runParamFile.Close()
	runParamData, err := io.ReadAll(runParamFile)
	if err != nil {
		return runParams, err
	}

	return cleve.ParseRunParameters(runParamData)
}

func parseRunInfo(runInfoFilename string) (cleve.RunInfo, error) {
	var runInfo cleve.RunInfo
	runInfoFile, err := os.Open(runInfoFilename)
	if err != nil {
		return runInfo, err
	}
	defer runInfoFile.Close()
	runInfoData, err := io.ReadAll(runInfoFile)
	if err != nil {
		return runInfo, err
	}

	return cleve.ParseRunInfo(runInfoData)
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
		db, err := mongo.Connect()
		if err != nil {
			log.Fatal(err)
		}

		runDir, err := filepath.Abs(args[0])
		if err != nil {
			log.Fatal(err)
		}
		runParametersFile := filepath.Join(runDir, "RunParameters.xml")
		runInfoFile := filepath.Join(runDir, "RunInfo.xml")

		runParams, err := parseRunParameters(runParametersFile)
		if err != nil {
			log.Fatal(err)
		}

		runInfo, err := parseRunInfo(runInfoFile)
		if err != nil {
			log.Fatal(err)
		}

		var state cleve.RunState
		if err = state.Set("new"); err != nil {
			log.Fatal(err)
		}

		run := cleve.Run{
			RunID:          runParams.GetRunID(),
			ExperimentName: runParams.GetExperimentName(),
			Path:           runDir,
			Platform:       runParams.Platform(),
			RunParameters:  runParams,
			RunInfo:        runInfo,
			StateHistory:   []cleve.TimedRunState{{State: state, Time: time.Now()}},
			Analysis:       []*cleve.Analysis{},
		}

		if err = db.Runs.Create(&run); err != nil {
			log.Fatal(err)
		}

		log.Printf("Added run %s as %s", run.RunID, run.ID.Hex())
	},
}

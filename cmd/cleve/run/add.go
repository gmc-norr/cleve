package run

import (
	"fmt"
	"log"
	"path/filepath"
	"time"

	"github.com/gmc-norr/cleve"
	"github.com/gmc-norr/cleve/interop"
	"github.com/gmc-norr/cleve/mongo"
	"github.com/spf13/cobra"
)

var (
	runState cleve.RunState
	addCmd   = &cobra.Command{
		Use:   "add [flags] run_directory",
		Short: "Add a sequencing run",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("run directory is required")
			}
			state, _ := cmd.Flags().GetString("state")
			return runState.Set(state)
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

			interopData, err := interop.InteropFromDir(runDir)
			if err != nil {
				log.Fatal(err)
			}

			sampleSheetFile, err := cleve.MostRecentSamplesheet(runDir)
			if err != nil {
				if err.Error() == "no samplesheet found" {
					log.Printf("no samplesheet found for run")
				} else {
					log.Fatal(err)
				}
			}

			if sampleSheetFile != "" {
				log.Printf("most recent samplesheet: %s", sampleSheetFile)
				samplesheet, err := cleve.ReadSampleSheet(sampleSheetFile)
				if err != nil {
					log.Fatal(err)
				}
				_, err = db.CreateSampleSheet(samplesheet, mongo.SampleSheetWithRunId(interopData.RunInfo.RunId))
				if err != nil {
					log.Fatal(err)
				}
			}

			run := cleve.Run{
				RunID:          interopData.RunInfo.RunId,
				ExperimentName: interopData.RunParameters.ExperimentName,
				Path:           runDir,
				Platform:       interopData.RunInfo.Platform,
				RunParameters:  interopData.RunParameters,
				RunInfo:        interopData.RunInfo,
				StateHistory:   []cleve.TimedRunState{{State: runState, Time: time.Now()}},
				Analysis:       []*cleve.Analysis{},
			}

			if err = db.CreateRun(&run); err != nil {
				log.Fatal(err)
			}

			if runState == cleve.Ready {
				log.Printf("adding qc for run %s", run.RunID)
				if err := db.CreateRunQC(run.RunID, interopData.Summarise()); err != nil {
					log.Fatal(err)
				}
			}

			log.Printf("Successfully added run %s", run.RunID)
		},
	}
)

func init() {
	addCmd.Flags().StringP("state", "s", "pending", "state of the run")
}

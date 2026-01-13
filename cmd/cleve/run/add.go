package run

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/gmc-norr/cleve"
	"github.com/gmc-norr/cleve/interop"
	"github.com/gmc-norr/cleve/mongo"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

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
			slog.Error("failed to connect to database", "error", err)
			os.Exit(1)
		}

		runDir, err := filepath.Abs(args[0])
		if err != nil {
			slog.Error("failed to get absolute path to run directory", "error", err)
			os.Exit(1)
		}

		interopData, err := interop.InteropFromDir(runDir)
		if err != nil {
			slog.Error("failed to read interop data", "error", err)
			os.Exit(1)
		}

		sampleSheetFile, err := cleve.MostRecentSamplesheet(runDir)
		if err != nil {
			// TODO: better handling of this error
			if err.Error() == "no samplesheet found" {
				slog.Warn("no samplesheet found for run")
			} else {
				slog.Error("failed to get most recent samplesheet", "error", err)
				os.Exit(1)
			}
		}

		if sampleSheetFile != "" {
			slog.Debug("most recent samplesheet", "path", sampleSheetFile)
			samplesheet, err := cleve.ReadSampleSheet(sampleSheetFile)
			if err != nil {
				slog.Error("failed to read samplesheet", "error", err)
				os.Exit(1)
			}
			_, err = db.CreateSampleSheet(samplesheet, mongo.SampleSheetWithRunId(interopData.RunInfo.RunId))
			if err != nil {
				slog.Error("failed to save samplesheet", "error", err)
				os.Exit(1)
			}
		}

		var webhook *cleve.Webhook
		if viperWebhook, ok := viper.Get("webhook").(*cleve.Webhook); ok {
			webhook = viperWebhook
		}

		run := cleve.Run{
			RunID:          interopData.RunInfo.RunId,
			ExperimentName: interopData.RunParameters.ExperimentName,
			Path:           runDir,
			Platform:       interopData.RunInfo.Platform,
			RunParameters:  interopData.RunParameters,
			RunInfo:        interopData.RunInfo,
		}
		currentState := run.State(false)
		run.StateHistory.Add(currentState)
		slog.Info("current run state", "state", currentState)

		if err = db.CreateRun(&run); err != nil {
			slog.Error("failed to save run", "error", err)
			os.Exit(1)
		}

		if run.StateHistory.LastState() == cleve.StateReady {
			slog.Info("adding qc data for run")
			if err := db.CreateRunQC(run.RunID, interopData.Summarise()); err != nil {
				slog.Error("failed to save qc data", "error", err)
				os.Exit(1)
			}
		}

		if err := webhook.Send(cleve.NewRunMessage(&run, "new run added", cleve.MessageStateUpdate)); err != nil {
			slog.Error("failed to send webhook message", "error", err)
		}

		slog.Info("successfully added", "run", run.RunID)
	},
}

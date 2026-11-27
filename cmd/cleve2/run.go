package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gmc-norr/cleve"
	"github.com/gmc-norr/cleve/cmd/cleve2/internal/cli"
	"github.com/gmc-norr/cleve/interop"
	"github.com/gmc-norr/cleve/mongo"
	"github.com/spf13/cobra"
)

func NewRunCmd(cfg *CliConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Manage sequencing runs",
	}

	cmd.AddCommand(NewRunListCmd(cfg))
	cmd.AddCommand(NewRunAddCmd(cfg))
	cmd.AddCommand(NewRunUpdateCmd(cfg))
	cmd.AddCommand(NewRunDeleteCmd(cfg))

	return cmd
}

func NewRunListCmd(cfg *CliConfig) *cobra.Command {
	var (
		csvOutput, jsonOutput bool
		platform, state       string
	)
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List sequencing runs",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if csvOutput && jsonOutput {
				return fmt.Errorf("cannot specify both --csv and --json, choose one")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := cfg.DB()
			if err != nil {
				return err
			}

			filter := cleve.RunFilter{
				Platform: platform,
				State:    state,
			}

			runs, err := db.Runs(filter)
			if err != nil {
				return err
			}

			if csvOutput {
				printRunCSV(runs.Runs)
			} else if jsonOutput {
				return printRunJSON(runs.Runs)
			} else {
				printRunTable(runs.Runs)
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&csvOutput, "csv", false, "Output CSV")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output JSON")
	cmd.Flags().StringVarP(&platform, "platform", "p", "", "Filter by platform")
	cmd.Flags().StringVarP(&state, "state", "s", "", "Filter by state")

	return cmd
}

func NewRunAddCmd(cfg *CliConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add run_directory",
		Short: "Add a sequencing run",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("run directory is required")
			}
			if len(args) > 1 {
				return fmt.Errorf("too many arguments")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := cfg.DB()
			if err != nil {
				return err
			}
			webhookClient, err := cfg.WebhookClient()
			if err != nil {
				return err
			}

			runDir, err := filepath.Abs(args[0])
			if err != nil {
				return fmt.Errorf("failed to get absolute path to run directory: %w", err)
			}

			interopData, err := interop.InteropFromDir(runDir)
			if err != nil {
				return fmt.Errorf("failed to read interop data: %w", err)
			}

			sampleSheetFile, err := cleve.MostRecentSamplesheet(runDir)
			if err != nil {
				// TODO: better handling of this error
				if err.Error() == "no samplesheet found" {
					cfg.Logger.Warn("no samplesheet found for run")
				} else {
					return fmt.Errorf("failed to get most recent samplesheet: %w", err)
				}
			}

			if sampleSheetFile != "" {
				cfg.Logger.Debug("most recent samplesheet", "path", sampleSheetFile)
				samplesheet, err := cleve.ReadSampleSheet(sampleSheetFile)
				if err != nil {
					cfg.Logger.Error("failed to read samplesheet", "error", err)
					os.Exit(1)
				}
				_, err = db.CreateSampleSheet(samplesheet, mongo.SampleSheetWithRunId(interopData.RunInfo.RunId))
				if err != nil {
					return fmt.Errorf("failed to save samplesheet: %w", err)
				}
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
			cfg.Logger.Info("current run state", "state", currentState)

			if err := db.CreateRun(&run); err != nil {
				return fmt.Errorf("failed to save run: %w", err)
			}

			if run.StateHistory.LastState() == cleve.StateReady {
				cfg.Logger.Info("adding qc data for run")
				if err := db.CreateRunQC(run.RunID, interopData.Summarise()); err != nil {
					return fmt.Errorf("failed to save qc data: %w", err)
				}
			}

			_ = cli.SendWebhookMessage(cmd.Context(), webhookClient, cleve.NewRunMessage(&run, "new run added", cleve.MessageStateUpdate))

			cfg.Logger.Info("successfully added", "run", run.RunID)
			return nil
		},
	}

	return cmd
}

func NewRunDeleteCmd(cfg *CliConfig) *cobra.Command {
	return &cobra.Command{
		Use: "delete run_id",
		Short: "Delete a sequencing run",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("run id is required")
			}
			if len(args) != 1 {
				return fmt.Errorf("too many arguments")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := cfg.DB()
			if err != nil {
				return err
			}
			if err := db.DeleteRun(args[0]); err != nil {
				return err
			}
			if err := db.DeleteRunQC(args[0]); err != nil {
				if err != mongo.ErrNoDocuments {
					return err
				}
			}
			if err := db.DeleteSampleSheet(args[0]); err != nil {
				if err != mongo.ErrNoDocuments {
					return err
				}
			}
			cfg.Logger.Info("deleted run", "run_id", args[0])
			return nil
		},
	}
}

func NewRunUpdateCmd(cfg *CliConfig) *cobra.Command {
	var state cleve.State
	var path string

	cmd := &cobra.Command{
		Use:   "update run_id",
		Short: "Update a sequencing run",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("run id is required")
			}
			if len(args) != 1 {
				return fmt.Errorf("too many arguments")
			}
			return nil
		},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if path == "" {
				return nil
			}
			if !filepath.IsAbs(path) {
				return fmt.Errorf("path must be absolute")
			}
			if stat, err := os.Stat(path); err != nil {
				return err
			} else {
				if !stat.IsDir() {
					return fmt.Errorf("path must be a directory")
				}
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := cfg.DB()
			if err != nil {
				return err
			}
			webhookClient, err := cfg.WebhookClient()
			if err != nil {
				return err
			}

			didSomething := false

			var run *cleve.Run
			run, err = db.Run(args[0])
			if err != nil {
				return fmt.Errorf("failed to fetch run information: %w", err)
			}

			newPath, _ := cmd.Flags().GetString("path")
			if newPath != "" {
				cfg.Logger.Info("updating run path", "run", args[0], "path", newPath)
				ri, err := interop.ReadRunInfo(filepath.Join(newPath, "RunInfo.xml"))
				if err != nil {
					return fmt.Errorf("failed to read run info: %w", err)
				}
				if run.RunID != ri.RunId {
					return fmt.Errorf("mismatching run ids, %q vs %q", run.RunID, ri.RunId)
				}
				run.Path = newPath
				didSomething = true
			}

			updateQc, _ := cmd.Flags().GetBool("update-qc")
			updateMetadata, _ := cmd.Flags().GetBool("update-metadata")

			// TODO: clean up when removing deprecated flags
			reloadQc, _ := cmd.Flags().GetBool("reload-qc")
			reloadMetadata, _ := cmd.Flags().GetBool("reload-metadata")
			updateQc = updateQc || reloadQc
			updateMetadata = updateMetadata || reloadMetadata

			stateUpdated := false

			// Update the run state. If the state was supplied on the command line, then use this state.
			// If not, then detect the state and set it accordingly, but only if the last state of the run
			// is not one of the ones that should be ignored.
			lastState := run.StateHistory.LastState()
			if state != 0 && lastState != state {
				stateUpdated = true
				cfg.Logger.Info("updating run state", "run", run.RunID, "old_state", lastState, "new_state", state)
				run.StateHistory.Add(state)
				didSomething = true
			} else {
				currentState := run.State(newPath != "")
				cfg.Logger.Debug("detected run state", "state", currentState)
				if lastState != currentState {
					cfg.Logger.Info("updating run state", "run", run.RunID, "old_state", lastState, "new_state", currentState)
					run.StateHistory.Add(currentState)
					didSomething = true
				}
			}

			if updateQc && run.StateHistory.LastState() == cleve.StateReady {
				cfg.Logger.Info("updating run qc data", "run", args[0])
				qc, err := interop.InteropFromDir(run.Path)
				if err != nil {
					cfg.Logger.Error("failed to read qc data", "path", run.Path, "error", err)
					os.Exit(1)
				}
				if err := db.UpdateRunQC(qc.Summarise()); err != nil {
					cfg.Logger.Error("failed to update qc data", "run", run.RunID, "error", err)
					os.Exit(1)
				}
				didSomething = true
			} else if updateQc {
				cfg.Logger.Warn("run is not ready, qc data will not be updated")
			}

			if updateMetadata && !run.StateHistory.LastState().IsMoved() {
				cfg.Logger.Info("updating run metadata", "run", args[0])
				runInfo, err := interop.ReadRunInfo(filepath.Join(run.Path, "RunInfo.xml"))
				if err != nil {
					cfg.Logger.Error("failed to read run info", "run", args[0], "error", err)
					os.Exit(1)
				}
				runParameters, err := interop.ReadRunParameters(filepath.Join(run.Path, "RunParameters.xml"))
				if err != nil {
					cfg.Logger.Error("failed to read run parameters", "run", args[0], "error", err)
					os.Exit(1)
				}
				run.RunInfo = runInfo
				run.RunParameters = runParameters
				didSomething = true
			}

			if didSomething {
				if err := db.UpdateRun(run); err != nil {
					cfg.Logger.Error("failed to update run", "run", run.RunID, "error", err)
					os.Exit(1)
				}
				if stateUpdated {
					_ = cli.SendWebhookMessage(cmd.Context(), webhookClient, cleve.NewRunMessage(run, "run state updated", cleve.MessageStateUpdate))
				}
			} else {
				cfg.Logger.Info("no changes made", "run", args[0])
			}
			return nil
		},
	}

	cmd.Flags().Var(&state, "state", "Run state")
	cmd.Flags().StringVarP(&path, "path", "p", "", "Absolute path to the run")
	cmd.Flags().Bool("update-qc", false, "Update QC data for the run")
	cmd.Flags().Bool("reload-qc", false, "Reload QC data for the run")
	cmd.Flags().Bool("update-metadata", false, "Update metadata for the run")
	cmd.Flags().Bool("reload-metadata", false, "Reload metadata for the run")

	_ = cmd.Flags().MarkDeprecated("reload-qc", "use --update-qc instead")
	_ = cmd.Flags().MarkDeprecated("reload-metadata", "use --update-metadata instead")

	return cmd
}

func truncateString(str string, maxLen int) string {
	if len(str) > maxLen {
		return str[0:(maxLen-3)] + "..."
	}
	return fmt.Sprintf("%-*s", maxLen, str)
}

func printRunTable(runs []*cleve.Run) {
	fmt.Printf(
		"|-%s-|-%s-|-%s-|-%s-|\n",
		strings.Repeat("-", 32),
		strings.Repeat("-", 20),
		strings.Repeat("-", 12),
		strings.Repeat("-", 33),
	)
	fmt.Printf(
		"| %-32s | %-20s | %-12s | %-33s |\n",
		"Run ID",
		"Run Name",
		"Platform",
		"Created At",
	)
	fmt.Printf(
		"|-%s-|-%s-|-%s-|-%s-|\n",
		strings.Repeat("-", 32),
		strings.Repeat("-", 20),
		strings.Repeat("-", 12),
		strings.Repeat("-", 33),
	)
	for _, run := range runs {
		fmt.Printf(
			"| %s | %s | %s | %s |\n",
			truncateString(run.RunID, 32),
			truncateString(run.ExperimentName, 20),
			truncateString(run.Platform, 12),
			truncateString(run.Created.String(), 33),
		)
	}
	fmt.Printf(
		"|-%s-|-%s-|-%s-|-%s-|\n",
		strings.Repeat("-", 32),
		strings.Repeat("-", 20),
		strings.Repeat("-", 12),
		strings.Repeat("-", 33),
	)
}

func printRunCSV(runs []*cleve.Run) {
	fmt.Printf("Run ID,Run Name,Created At\n")
	for _, run := range runs {
		fmt.Printf("%s,%s,%s\n", run.RunID, run.ExperimentName, run.Created.String())
	}
}

func printRunJSON(runs []*cleve.Run) error {
	return json.NewEncoder(os.Stdout).Encode(runs)
}

func printJSON(items any) error {
	return json.NewEncoder(os.Stdout).Encode(items)
}

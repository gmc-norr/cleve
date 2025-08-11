package run

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/gmc-norr/cleve"
	"github.com/gmc-norr/cleve/interop"
	"github.com/gmc-norr/cleve/mongo"
	"github.com/spf13/cobra"
)

var (
	stateArg    string
	stateUpdate cleve.RunState
	updateCmd   = &cobra.Command{
		Use:   "update [flags] run_id",
		Short: "Update a sequencing run",
		PreRun: func(cmd *cobra.Command, args []string) {
			path, _ := cmd.Flags().GetString("path")
			if path == "" {
				return
			}
			if !filepath.IsAbs(path) {
				cobra.CheckErr("path needs to be absolute")
			}
			if info, err := os.Stat(path); err != nil {
				cobra.CheckErr(err)
			} else if !info.IsDir() {
				cobra.CheckErr("path needs to be a directory")
			}
		},
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("run id is required")
			}
			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			db, err := mongo.Connect()
			if err != nil {
				slog.Error("failed to connect to database", "error", err)
				os.Exit(1)
			}
			didSomething := false

			var run *cleve.Run
			run, err = db.Run(args[0], false)
			if err != nil {
				slog.Error("failed to fetch run information", "run", args[0], "error", err)
			}

			newPath, _ := cmd.Flags().GetString("path")
			if newPath != "" {
				slog.Info("updating run path", "run", args[0], "path", newPath)
				ri, err := interop.ReadRunInfo(filepath.Join(newPath, "RunInfo.xml"))
				if err != nil {
					slog.Error("failed to read run info, is it a valid run directory?", "path", newPath, "error", err)
					os.Exit(1)
				}
				if run.RunID != ri.RunId {
					slog.Error("mismatching run ids", "db_runid", run.RunID, "disk_runid", ri.RunId)
					os.Exit(1)
				}
				if err := db.SetRunPath(args[0], newPath); err != nil {
					slog.Error("failed to update run path", "path", newPath, "error", err)
					os.Exit(1)
				}
				didSomething = true
			}

			reloadQc, _ := cmd.Flags().GetBool("reload-qc")
			reloadMetadata, _ := cmd.Flags().GetBool("reload-metadata")

			// Update the run state. If the state was supplied on the command line, then use this state.
			// If not, then detect the state and set it accordingly, but only if the last state of the run
			// is not one of the ones that should be ignored.
			lastState := run.StateHistory.LastState().State
			if stateArg != "" && lastState != stateUpdate {
				slog.Info("updating run state", "run", run.RunID, "old_state", lastState, "new_state", stateUpdate)
				err := db.SetRunState(run.RunID, stateUpdate)
				if err != nil {
					slog.Error("failed to set run state", "run", run.RunID, "new_state", stateUpdate, "error", err)
				}
				didSomething = true
			} else {
				currentState := run.State(newPath != "")
				slog.Debug("detected run state", "state", currentState)
				if lastState != currentState {
					slog.Info("updating run state", "run", run.RunID, "old_state", lastState, "new_state", currentState)
					if err := db.SetRunState(run.RunID, currentState); err != nil {
						slog.Error("failed to set run state", "run", run.RunID, "new_state", currentState, "error", err)
						os.Exit(1)
					}
					didSomething = true
				}
			}

			if reloadQc {
				slog.Info("updating run qc data", "run", args[0])
				qc, err := interop.InteropFromDir(run.Path)
				if err != nil {
					slog.Error("failed to read qc data", "path", run.Path, "error", err)
					os.Exit(1)
				}
				if err := db.UpdateRunQC(qc.Summarise()); err != nil {
					slog.Error("failed to update qc data", "run", run.RunID, "error", err)
					os.Exit(1)
				}
				didSomething = true
			}

			if reloadMetadata {
				slog.Info("updating run metadata", "run", args[0])
				runInfo, err := interop.ReadRunInfo(filepath.Join(run.Path, "RunInfo.xml"))
				if err != nil {
					slog.Error("failed to read run info", "run", args[0], "error", err)
					os.Exit(1)
				}
				runParameters, err := interop.ReadRunParameters(filepath.Join(run.Path, "RunParameters.xml"))
				if err != nil {
					slog.Error("failed to read run parameters", "run", args[0], "error", err)
					os.Exit(1)
				}
				run.RunInfo = runInfo
				run.RunParameters = runParameters
				if err := db.UpdateRun(run); err != nil {
					slog.Error("failed to update run", "run", run.RunID, "error", err)
					os.Exit(1)
				}
				didSomething = true
			}

			if !didSomething {
				slog.Info("no changes made", "run", args[0])
			}
		},
	}
)

func init() {
	allowedStates := make([]string, 0, len(cleve.ValidRunStates))
	for k := range cleve.ValidRunStates {
		allowedStates = append(allowedStates, k)
	}
	stateString := strings.Join(allowedStates, ", ")
	updateCmd.Flags().StringVar(&stateArg, "state", "", "Run state (one of "+stateString+")")
	updateCmd.Flags().StringP("path", "p", "", "Absolute path to the run")
	updateCmd.Flags().Bool("reload-qc", false, "Reload QC data for run")
	updateCmd.Flags().Bool("reload-metadata", false, "Reload metadata for run")

	cobra.OnInitialize(func() {
		if stateArg != "" {
			if err := stateUpdate.Set(stateArg); err != nil {
				slog.Error("failed to set state", "error", err)
				os.Exit(1)
			}
		}
	})
}

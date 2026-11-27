package main

import (
	"fmt"

	"github.com/gmc-norr/cleve"
	"github.com/gmc-norr/cleve/mongo"
	"github.com/spf13/cobra"
)

func NewSamplesheetCmd(cfg *CliConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "samplesheet",
		Short: "Interact with sample sheets",
	}

	cmd.AddCommand(NewSamplesheetAddCmd(cfg))

	return cmd
}

func NewSamplesheetAddCmd(cfg *CliConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add run_id samplesheet_path",
		Short: "Add a sample sheet and associate it with a run",
		Args: func(cmd *cobra.Command, args []string) error {
			if err := cobra.ExactArgs(2)(cmd, args); err != nil {
				return err
			}
			if args[0] == "" || args[1] == "" {
				return fmt.Errorf("arguments cannot be empty")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := cfg.DB()
			if err != nil {
				return err
			}

			runID := args[0]
			sheetPath := args[1]

			sampleSheet, err := cleve.ReadSampleSheet(sheetPath)
			if err != nil {
				return err
			}

			_, err = db.Run(runID)
			if err != nil {
				if err == mongo.ErrNoDocuments {
					return fmt.Errorf("error: run with id %q not found", runID)
				}
				return err
			}

			res, err := db.CreateSampleSheet(sampleSheet, mongo.SampleSheetWithRunId(runID))
			if err != nil {
				return err
			}

			switch {
			case res.MatchedCount == 0 && res.UpsertedCount == 1:
				cfg.Logger.Info("added sample sheet", "run_id", runID)
			case res.MatchedCount == 1 && res.ModifiedCount == 0:
				cfg.Logger.Info("a newer samplesheet already exists", "run_id", runID)
			case res.MatchedCount == 1 && res.ModifiedCount == 1:
				cfg.Logger.Info("updated samplesheet", "run_id", runID)
			default:
				cfg.Logger.Info("updated samplesheet", "result", res)
			}
			return nil
		},
	}

	return cmd
}

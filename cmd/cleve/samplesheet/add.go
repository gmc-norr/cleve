package samplesheet

import (
	"fmt"
	"log"

	"github.com/gmc-norr/cleve"
	"github.com/gmc-norr/cleve/mongo"
	"github.com/spf13/cobra"
)

var (
	runID     string
	sheetPath string
	addCmd    = &cobra.Command{
		Use:   "add [flags] run_id samplesheet_path",
		Short: "Add a SampleSheet and associate it with a run",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) > 2 {
				return fmt.Errorf("too many arguments")
			}
			if len(args) < 2 {
				return fmt.Errorf("too few arguments")
			}
			if args[0] == "" || args[1] == "" {
				return fmt.Errorf("arguments cannot be empty")
			}
			return nil
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			runID = args[0]
			sheetPath = args[1]
		},
		Run: func(cmd *cobra.Command, args []string) {
			db, err := mongo.Connect()
			if err != nil {
				log.Fatal(err)
			}

			sampleSheet, err := cleve.ReadSampleSheet(sheetPath)
			if err != nil {
				log.Fatal(err)
			}

			_, err = db.Run(runID, true)
			if err != nil {
				if err == mongo.ErrNoDocuments {
					log.Fatalf("error: run with id %q not found", runID)
				}
				log.Fatal(err)
			}

			res, err := db.CreateSampleSheet(sampleSheet, mongo.SampleSheetWithRunId(runID))
			if err != nil {
				log.Fatal(err)
			}

			switch {
			case res.MatchedCount == 0 && res.UpsertedCount == 1:
				log.Printf("added sample sheet for run %q", runID)
			case res.MatchedCount == 1 && res.ModifiedCount == 0:
				log.Printf("a newer samplesheet already exists for run %q", runID)
			case res.MatchedCount == 1 && res.ModifiedCount == 1:
				log.Printf("updated samplesheet for run %q", runID)
			default:
				log.Printf("%+v", res)
			}
		},
	}
)

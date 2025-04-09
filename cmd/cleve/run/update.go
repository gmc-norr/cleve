package run

import (
	"fmt"
	"log"
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
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("run id is required")
			}
			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			db, err := mongo.Connect()
			if err != nil {
				log.Fatal(err)
			}
			didSomething := false

			if stateArg != "" {
				log.Printf("Updating state of run %s to '%s'", args[0], stateUpdate.String())
				err := db.SetRunState(args[0], stateUpdate)
				if err != nil {
					log.Fatalf("error: %s", err)
				}
				didSomething = true
			}

			newPath, _ := cmd.Flags().GetString("path")
			if newPath != "" {
				log.Printf("Updating path of run %s to %s", args[0], newPath)
				if err := db.SetRunPath(args[0], newPath); err != nil {
					log.Fatalf("error: %s", err)
				}
				didSomething = true
			}

			reloadQc, _ := cmd.Flags().GetBool("reload-qc")
			if reloadQc {
				log.Printf("Updating QC data for run %s", args[0])
				run, err := db.Run(args[0], false)
				if err != nil {
					log.Fatalf("error: %s", err)
				}
				qc, err := interop.InteropFromDir(run.Path)
				if err != nil {
					log.Fatalf("error: %s", err)
				}
				if err := db.UpdateRunQC(qc.Summarise()); err != nil {
					log.Fatalf("error: %s", err)
				}
				didSomething = true
			}

			if !didSomething {
				log.Printf("No changes made to run %s", args[0])
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
	updateCmd.Flags().StringP("path", "p", "", "Path to the run")
	updateCmd.Flags().Bool("reload-qc", false, "Reload QC data for run")

	cobra.OnInitialize(func() {
		if stateArg != "" {
			if err := stateUpdate.Set(stateArg); err != nil {
				log.Fatalf("error: %s", err)
			}
		}
	})
}

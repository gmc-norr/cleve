package run

import (
	"fmt"
	"log"
	"strings"

	"github.com/gmc-norr/cleve"
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

	cobra.OnInitialize(func() {
		if stateArg != "" {
			if err := stateUpdate.Set(stateArg); err != nil {
				log.Fatalf("error: %s", err)
			}
		}
	})
}

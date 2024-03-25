package run

import (
	"fmt"
	"github.com/gmc-norr/cleve/internal/db"
	"github.com/gmc-norr/cleve/internal/db/runstate"
	"github.com/spf13/cobra"
	"log"
	"strings"
)

var (
	stateArg  string
	state     runstate.RunState
	updateCmd = &cobra.Command{
		Use:   "update [flags] run_id",
		Short: "Update a sequencing run",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("run id is required")
			}
			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			db.Init()
			didSomething := false

			if stateArg != "" {
				log.Printf("Updating state of run %s to '%s'", args[0], state.String())
				err := db.UpdateRunState(args[0], state)
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
	allowedStates := make([]string, 0, len(runstate.ValidRunStates))
	for k := range runstate.ValidRunStates {
		allowedStates = append(allowedStates, k)
	}
	stateString := strings.Join(allowedStates, ", ")
	updateCmd.Flags().StringVar(&stateArg, "state", "", "Run state (one of "+stateString+")")

	cobra.OnInitialize(func() {
		if stateArg != "" {
			if err := state.Set(stateArg); err != nil {
				log.Fatalf("error: %s", err)
			}
		}
	})
}

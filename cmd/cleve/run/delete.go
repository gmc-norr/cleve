package run

import (
	"fmt"
	"github.com/gmc-norr/cleve/mongo"
	"github.com/spf13/cobra"
	"log"
)

var deleteCmd = &cobra.Command{
	Use:   "delete [flags] run_id",
	Short: "Delete a sequencing run",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return fmt.Errorf("run id is required")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		mongo.Init()
		err := mongo.DeleteRun(args[0])
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("Deleted run %s", args[0])
	},
}

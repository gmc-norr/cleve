package panelqc

import (
	"fmt"
	"log"

	"github.com/gmc-norr/cleve/mongo"
	"github.com/spf13/cobra"
)


var deleteCmd = &cobra.Command{
	Use:   "delete [flags] run_id",
	Short: "Delete the PanelQc for a sequencing run",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return fmt.Errorf("run id is required")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {

		db, err := mongo.Connect()
		cobra.CheckErr(err)

		if err := db.DeletePanelQc(args[0]); err != nil {
			log.Fatal(err)
		}

		log.Printf("Deleted run %s", args[0])
		
	},
}
package platform

import (
	"fmt"
	"github.com/gmc-norr/cleve/mongo"
	"github.com/spf13/cobra"
	"log"
)

var (
	deleteName string
	deleteCmd  = &cobra.Command{
		Use:   "delete [flags] name",
		Short: "Delete a platform from the database",
		PreRun: func(cmd *cobra.Command, args []string) {
			deleteName = args[0]
		},
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) > 1 {
				return fmt.Errorf("error: too many arguments")
			}
			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			db, err := mongo.Connect()
			if err != nil {
				log.Fatalf("error: %s", err)
			}

			if err = db.Platforms.Delete(deleteName); err != nil {
				log.Fatalf("error: %s", err)
			}

			log.Printf(`successfully deleted platform "%s"`, deleteName)
		},
	}
)

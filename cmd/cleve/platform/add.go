package platform

import (
	"log"

	"github.com/gmc-norr/cleve"
	"github.com/gmc-norr/cleve/mongo"
	"github.com/spf13/cobra"
)

var (
	name        string
	readyMarker string
	addCmd      = &cobra.Command{
		Use:   "add [flags] name serialtag serialprefix",
		Short: "Add a platform to the database",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				log.Fatalf("name is required")
			}
			if args[0] == "" {
				log.Fatalf("arguments cannot be empty")
			}
			return nil
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			name = args[0]
		},
		Run: func(cmd *cobra.Command, args []string) {
			db, err := mongo.Connect()
			if err != nil {
				log.Fatalf("error: %s", err)
			}

			platform := cleve.Platform{
				Name:        name,
				ReadyMarker: readyMarker,
			}

			if err := db.CreatePlatform(&platform); err != nil {
				if mongo.IsDuplicateKeyError(err) {
					log.Fatalf("error: platform %s already exists", name)
				}
				log.Fatalf("error: %s", err)
			}

			log.Printf("added platform %s to the database", name)
		},
	}
)

func init() {
	addCmd.Flags().StringVarP(
		&readyMarker,
		"ready-marker",
		"r",
		"CopyComplete.txt",
		"Name of file that indicates that a run is ready",
	)
}

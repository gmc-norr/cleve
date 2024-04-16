package platform

import (
	"log"

	"github.com/gmc-norr/cleve"
	"github.com/gmc-norr/cleve/mongo"
	"github.com/spf13/cobra"
)

var (
	name         string
	serialTag    string
	serialPrefix string
	readyMarker  string
	addCmd       = &cobra.Command{
		Use:   "add [flags] name serialtag serialprefix",
		Short: "Add a platform to the database",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 3 {
				log.Fatalf("name, serialtag and serialprefix are required")
			}
			if args[0] == "" || args[1] == "" || args[2] == "" {
				log.Fatalf("arguments cannot be empty")
			}
			return nil
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			name = args[0]
			serialTag = args[1]
			serialPrefix = args[2]
		},
		Run: func(cmd *cobra.Command, args []string) {
			db, err := mongo.Connect()
			if err != nil {
				log.Fatalf("error: %s", err)
			}

			platform := cleve.Platform{
				Name:         name,
				SerialTag:    serialTag,
				SerialPrefix: serialPrefix,
				ReadyMarker:  readyMarker,
			}

			if err := db.Platforms.Create(&platform); err != nil {
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

package db

import (
	"context"
	"log"

	"github.com/gmc-norr/cleve/mongo"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialise database collections",
	Run: func(cmd *cobra.Command, args []string) {
		db, err := mongo.Connect()
		if err != nil {
			log.Fatal(err)
		}

		log.Println("creating database collections")

		if err := db.Init(context.TODO()); err != nil {
			log.Fatal(err)
		}
	},
}

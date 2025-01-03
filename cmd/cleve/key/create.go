package key

import (
	"fmt"
	"log"

	"github.com/gmc-norr/cleve"
	"github.com/gmc-norr/cleve/mongo"
	"github.com/spf13/cobra"
)

var createCmd = &cobra.Command{
	Use:   "create [flags] user",
	Short: "Create API key for a user/service",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return fmt.Errorf("user is required")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		db, err := mongo.Connect()
		if err != nil {
			log.Fatal(err)
		}

		key := cleve.NewAPIKey(args[0])
		if err := db.CreateKey(key); err != nil {
			log.Fatalf("error: %s", err)
		}

		fmt.Printf("Created API key for %s: %s\n", key.User, key.Key)
	},
}

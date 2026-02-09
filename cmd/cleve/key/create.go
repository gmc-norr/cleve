package key

import (
	"fmt"
	"log"
	"log/slog"

	"github.com/gmc-norr/cleve"
	"github.com/gmc-norr/cleve/mongo"
	"github.com/spf13/cobra"
)

var createCmd = &cobra.Command{
	Use:   "create USER [flags]",
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

		plainKey := cleve.NewPlainKey()
		key, err := cleve.NewAPIKey(plainKey, args[0])
		if err != nil {
			log.Fatalf("error: %s", err)
		}
		if err := db.CreateKey(key); err != nil {
			log.Fatalf("error: %s", err)
		}

		slog.Warn("API key only shown this time, make sure to save it")
		fmt.Printf("Created API key for %s: %s\n", key.User, plainKey)
	},
}

package key

import (
	"encoding/base64"
	"fmt"
	"log"

	"github.com/gmc-norr/cleve/mongo"
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete ID [flags]",
	Short: "Delete API key",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return fmt.Errorf("id is required")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		db, err := mongo.Connect()
		if err != nil {
			log.Fatal(err)
		}
		id, err := base64.URLEncoding.DecodeString(args[0])
		if err != nil {
			log.Fatalf("error: failed to decode ID: %s", err)
		}
		if err := db.DeleteKey(id); err != nil {
			if err == mongo.ErrNoDocuments {
				log.Fatal("error: key not found")
			}
			log.Fatalf("error: %s", err)
		}
		fmt.Printf("Deleted API key: %s\n", args[0])
	},
}

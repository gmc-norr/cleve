package key

import (
	"encoding/base64"
	"fmt"
	"log/slog"
	"os"

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
			slog.Error("failed to connect to database", "error", err)
			os.Exit(1)
		}
		id, err := base64.URLEncoding.DecodeString(args[0])
		if err != nil {
			slog.Error("failed to decode ID", "error", err)
			os.Exit(1)
		}
		if err := db.DeleteKey(id); err != nil {
			if err == mongo.ErrNoDocuments {
				slog.Error("key not found")
				os.Exit(1)
			}
			slog.Error("failed to delete key", "error", err)
			os.Exit(1)
		}
		fmt.Printf("Deleted API key: %s\n", args[0])
	},
}

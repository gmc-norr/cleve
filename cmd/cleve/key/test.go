package key

import (
	"encoding/base64"
	"log/slog"
	"os"

	"github.com/gmc-norr/cleve"
	"github.com/gmc-norr/cleve/mongo"
	"github.com/spf13/cobra"
)

var testCmd = &cobra.Command{
	Use:  "test KEY",
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		db, err := mongo.Connect()
		if err != nil {
			slog.Error("failed to connect to database", "error", err.Error())
			os.Exit(1)
		}
		b, err := base64.URLEncoding.DecodeString(args[0])
		if err != nil {
			slog.Error("failed to decode key", "error", err.Error())
			os.Exit(1)
		}
		plainKey := cleve.PlainKey(b)
		apiKey, err := db.KeyFromId(plainKey.Id())
		if err != nil {
			slog.Error("failed to get key from database", "error", err)
			os.Exit(1)
		}
		if err := apiKey.Compare(plainKey); err != nil {
			slog.Error("invalid API key")
			os.Exit(1)
		}
		slog.Info("matching API key found")
	},
}

package key

import (
	"encoding/base64"
	"fmt"
	"log"

	"github.com/gmc-norr/cleve/mongo"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List API keys",
	Run: func(cmd *cobra.Command, args []string) {
		db, err := mongo.Connect()
		if err != nil {
			log.Fatal(err)
		}
		keys, err := db.Keys()
		if err != nil {
			log.Fatal(err)
		}
		for _, key := range keys {
			fmt.Printf("%s: %s %s\n", key.User, key.Created.Format("2006-01-02T15:04"), base64.URLEncoding.EncodeToString(key.Id))
		}
	},
}

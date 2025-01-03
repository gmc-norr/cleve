package key

import (
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
			fmt.Printf("%s: %s\n", key.User, key.Key)
		}
	},
}

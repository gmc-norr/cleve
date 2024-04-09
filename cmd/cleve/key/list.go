package key

import (
	"fmt"
	"github.com/gmc-norr/cleve/mongo"
	"github.com/spf13/cobra"
	"log"
)

var (
	listCmd = &cobra.Command{
		Use:   "list",
		Short: "List API keys",
		Run: func(cmd *cobra.Command, args []string) {
			mongo.Init()
			keys, err := mongo.GetKeys()
			if err != nil {
				log.Fatal(err)
			}
			for _, key := range keys {
				fmt.Printf("%s: %s\n", key.User, key.Key)
			}
		},
	}
)

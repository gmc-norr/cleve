package db

import (
	"fmt"
	"github.com/gmc-norr/cleve/mongo"
	"github.com/spf13/cobra"
	"log"
)

var update bool

var indexCmd = &cobra.Command{
	Use:   "index",
	Short: "List and set database indexes",
	Run: func(cmd *cobra.Command, args []string) {
		mongo.Init()

		if update {
			err := mongo.SetIndexes()
			if err != nil {
				log.Fatal(err)
			}
		} else {
			indexes, err := mongo.GetIndexes()
			if err != nil {
				log.Fatal(err)
			}
			for k, v := range indexes {
				fmt.Printf("Collection %s:\n", k)
				for _, v2 := range v {
					for k3, v3 := range v2 {
						fmt.Printf("  %s: %v\n", k3, v3)
					}
					fmt.Println()
				}
				fmt.Println()
			}
		}
	},
}

func init() {
	indexCmd.Flags().BoolVar(&update, "update", false, "update indexes")
}

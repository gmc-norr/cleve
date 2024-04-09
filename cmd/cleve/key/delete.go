package key

import (
	"fmt"
	"github.com/gmc-norr/cleve/mongo"
	"github.com/spf13/cobra"
	"log"
)

var (
	deleteCmd = &cobra.Command{
		Use:   "delete",
		Short: "Delete API key",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("key is required")
			}
			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			mongo.Init()
			if err := mongo.DeleteKey(args[0]); err != nil {
				if err == mongo.ErrNoDocuments {
					log.Fatal("error: key not found")
				}
				log.Fatalf("error: %s", err)
			}
			fmt.Printf("Deleted API key: %s\n", args[0])
		},
	}
)

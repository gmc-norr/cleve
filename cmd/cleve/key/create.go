package key

import (
	"fmt"
	"github.com/gmc-norr/cleve"
	"github.com/gmc-norr/cleve/internal/db"
	"github.com/spf13/cobra"
	"log"
)

var (
	createCmd = &cobra.Command{
		Use:   "create [flags] user",
		Short: "Create API key for a user/service",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("user is required")
			}
			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			db.Init()

			key := cleve.NewAPIKey(args[0])
			if err := db.AddKey(key); err != nil {
				log.Fatalf("error: %s", err)
			}

			fmt.Printf("Created API key for %s: %s\n", key.User, key.Key)
		},
	}
)

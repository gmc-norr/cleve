package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewDBCommand(cfg *CliConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "db",
		Short: "Database management",
	}

	cmd.AddCommand(NewDBInitCmd(cfg))
	cmd.AddCommand(NewDBIndexCmd(cfg))

	return cmd
}

func NewDBInitCmd(cfg *CliConfig) *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialise database collections",
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := cfg.DB()
			if err != nil {
				return err
			}

			cfg.Logger.Info("creating database collections")

			return db.Init(cmd.Context())
		},
	}
}

func NewDBIndexCmd(cfg *CliConfig) *cobra.Command {
	var update bool
	cmd := &cobra.Command{
		Use:   "index",
		Short: "List and set database indexes",
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := cfg.DB()
			if err != nil {
				return err
			}

			if update {
				if err := db.SetIndexes(); err != nil {
					return err
				}
			} else {
				indexes, err := db.GetIndexes()
				if err != nil {
					return err
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
			return nil
		},
	}

	cmd.Flags().BoolVar(&update, "update", false, "Update indexes")

	return cmd
}

package main

import (
	"encoding/base64"
	"fmt"

	"github.com/gmc-norr/cleve"
	"github.com/gmc-norr/cleve/mongo"
	"github.com/spf13/cobra"
)

func NewKeyCmd(cfg *CliConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "key",
		Short: "API key management",
	}

	cmd.AddCommand(NewKeyListCmd(cfg))
	cmd.AddCommand(NewKeyCreateCmd(cfg))
	cmd.AddCommand(NewKeyDeleteCmd(cfg))
	cmd.AddCommand(NewKeyTestCmd(cfg))

	return cmd
}

func NewKeyListCmd(cfg *CliConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List API keys",
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := cfg.DB()
			if err != nil {
				return err
			}
			keys, err := db.Keys()
			if err != nil {
				return err
			}
			for _, key := range keys {
				fmt.Printf("%s: %s %s\n", key.User, key.Created.Local().Format("2006-01-02T15:04"), base64.URLEncoding.EncodeToString(key.Id))
			}
			return nil
		},
	}
	return cmd
}

func NewKeyCreateCmd(cfg *CliConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create user",
		Short: "Create an API key for a user/service",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("user is required")
			}
			if len(args) > 1 {
				return fmt.Errorf("too many arguments")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := cfg.DB()
			if err != nil {
				return err
			}
			plainKey := cleve.NewPlainKey()
			key, err := cleve.NewAPIKey(plainKey, args[0])
			if err != nil {
				return fmt.Errorf("failed to create API key: %w", err)
			}
			if err := db.CreateKey(key); err != nil {
				return fmt.Errorf("failed to save API key: %w", err)
			}

			cfg.Logger.Warn("API key only shown this time, make sure to save it")
			fmt.Printf("Created API key for %s: %s\n", key.User, plainKey)
			return nil
		},
	}
	return cmd
}

func NewKeyDeleteCmd(cfg *CliConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete id",
		Short: "Delete an API key",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("id is required")
			}
			if len(args) != 1 {
				return fmt.Errorf("too many arguments")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := cfg.DB()
			if err != nil {
				return err
			}
			id, err := base64.URLEncoding.DecodeString(args[0])
			if err != nil {
				return fmt.Errorf("failed to decode ID: %w", err)
			}
			if err := db.DeleteKey(id); err != nil {
				if err == mongo.ErrNoDocuments {
					return fmt.Errorf("key not found")
				}
				return fmt.Errorf("failed to delete key: %w", err)
			}
			fmt.Printf("Deleted API key: %s\n", args[0])
			return nil
		},
	}
	return cmd
}

func NewKeyTestCmd(cfg *CliConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "test key",
		Short: "Test an API key",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("key is required")
			}
			if len(args) != 1 {
				return fmt.Errorf("too many arguments")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := cfg.DB()
			if err != nil {
				return fmt.Errorf("failed to connect to database: %w", err)
			}
			b, err := base64.URLEncoding.DecodeString(args[0])
			if err != nil {
				return fmt.Errorf("failed to decode key: %w", err)
			}
			plainKey := cleve.PlainKey(b)
			apiKey, err := db.KeyFromId(plainKey.Id())
			if err != nil {
				return fmt.Errorf("failed to get key from database: %w", err)
			}
			if err := apiKey.Compare(plainKey); err != nil {
				return fmt.Errorf("invalid API key")
			}
			cfg.Logger.Info("matching API key found")
			return nil
		},
	}
	return cmd
}

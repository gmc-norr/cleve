package main

import (
	"context"
	"fmt"
	"os"

	"github.com/gmc-norr/cleve"
	"github.com/spf13/cobra"
)

func NewRootCmd(cfg *CliConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "cleve [FLAGS] COMMAND",
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       cleve.GetVersion(),
		PersistentPreRunE: func(cmd *cobra.Command, args []string) (err error) {
			configPath, _ := cmd.Flags().GetString("config")
			if err := cfg.initViper(configPath); err != nil {
				return err
			}

			_ = cfg.Viper.BindPFlag("loglevel", cmd.Flag("loglevel"))
			_ = cfg.Viper.BindPFlag("webhook_url", cmd.Flag("webhook-url"))
			_ = cfg.Viper.BindPFlag("webhook_api_key", cmd.Flag("webhook-api-key"))

			logLevel := cfg.Viper.GetString("loglevel")
			if err = cfg.initLogger(logLevel); err != nil {
				return err
			}

			return nil
		},
	}

	cmd.PersistentFlags().StringP("loglevel", "l", "WARN", "Log verbosity")
	cmd.PersistentFlags().StringP("config", "c", "", "Config file")
	cmd.PersistentFlags().String("webhook-url", "", "URL to send webhook messages to")
	cmd.PersistentFlags().String("webhook-api-key", "", `API key for the webhook service (<header-key>=<api-key>)`)

	cmd.AddCommand(NewRunCmd(cfg))
	cmd.AddCommand(NewPlatformCmd(cfg))
	cmd.AddCommand(NewDBCommand(cfg))
	cmd.AddCommand(NewKeyCmd(cfg))
	cmd.AddCommand(NewPanelCmd(cfg))
	cmd.AddCommand(NewSamplesheetCmd(cfg))
	cmd.AddCommand(NewServeCmd(cfg))
	cmd.AddCommand(NewUserCmd(cfg))

	return cmd
}

func Cleve() {
	cfg := &CliConfig{}
	err := NewRootCmd(cfg).ExecuteContext(context.Background())
	if err != nil {
		if cfg.Logger != nil {
			cfg.Logger.Error(err.Error())
		} else {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
		}
		os.Exit(1)
	}
}

func main() {
	Cleve()
}

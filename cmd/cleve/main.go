package main

import (
	"log"
	"log/slog"
	"os"

	"github.com/gmc-norr/cleve"
	"github.com/gmc-norr/cleve/cmd/cleve/db"
	"github.com/gmc-norr/cleve/cmd/cleve/key"
	"github.com/gmc-norr/cleve/cmd/cleve/panel"
	"github.com/gmc-norr/cleve/cmd/cleve/platform"
	"github.com/gmc-norr/cleve/cmd/cleve/run"
	"github.com/gmc-norr/cleve/cmd/cleve/samplesheet"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	configFile string
	rootCmd    = &cobra.Command{
		Use:     "cleve",
		Short:   "Interact with the sequencing database",
		Version: cleve.GetVersion(),
	}
)

func init() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(logger)

	cobra.OnInitialize(initConfig)

	rootCmd.SetVersionTemplate(`{{with .Name}}{{printf "%s " .}}{{end}}{{printf "%s\n" .Version}}`)
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "config file")
	rootCmd.PersistentFlags().String("webhook-url", viper.GetString("webhook_url"), "URL to send webhook messages to")
	rootCmd.PersistentFlags().String("webhook-api-key", viper.GetString("webhook-api-key"), "API key for the webhook service (\"<header-key>=<api-key>\")")

	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(run.RunCmd)
	rootCmd.AddCommand(db.DbCmd)
	rootCmd.AddCommand(key.KeyCmd)
	rootCmd.AddCommand(panel.PanelCmd)
	rootCmd.AddCommand(platform.PlatformCmd)
	rootCmd.AddCommand(samplesheet.SampleSheetCmd)
}

func initConfig() {
	if configFile != "" {
		viper.SetConfigFile(configFile)
	} else {
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
		viper.AddConfigPath(".")
		if _, ok := os.LookupEnv("XDG_CONFIG_HOME"); ok {
			viper.AddConfigPath("$XDG_CONFIG_HOME/cleve")
		} else {
			viper.AddConfigPath("$HOME/.config/cleve")
		}
		viper.AddConfigPath("/etc/cleve")
	}

	err := viper.ReadInConfig()
	if err != nil {
		log.Fatalf("error: %s", err)
	}

	slog.Info("config", "path", viper.ConfigFileUsed())

	// Basic validation
	dbConfig := viper.GetStringMap("database")
	if dbConfig == nil {
		log.Fatal("missing database config")
	}
	if dbConfig["host"] == nil {
		log.Fatal("missing database host")
	}
	if dbConfig["port"] == nil {
		log.Fatal("missing database port")
	}
	if dbConfig["user"] == nil {
		log.Fatal("missing database user")
	}
	if dbConfig["password"] == nil {
		log.Fatal("missing database password")
	}
	if dbConfig["name"] == nil {
		log.Fatal("missing database name")
	}

	_ = viper.BindPFlag("webhook_url", rootCmd.PersistentFlags().Lookup("webhook-url"))
	_ = viper.BindPFlag("webhook_api_key", rootCmd.PersistentFlags().Lookup("webhook-api-key"))

	cobra.CheckErr(viper.BindEnv("webhook_url", "CLEVE_WEBHOOK_URL"))
	cobra.CheckErr(viper.BindEnv("webhook_api_key", "CLEVE_WEBHOOK_API_KEY"))

	webhookApiKey, err := cleve.WebhookApiKeyFromString(viper.GetString("webhook_api_key"))
	cobra.CheckErr(err)
	webhookUrl := viper.GetString("webhook_url")
	if webhookUrl != "" {
		webhook := cleve.NewAuthWebhook(webhookUrl, webhookApiKey)
		if err := webhook.Check(); err != nil {
			slog.Error("failed to set up webhook", "url", webhookUrl, "error", err)
			os.Exit(1)
		}
		slog.Info("set up webhook", "url", webhookUrl)
		viper.Set("webhook", webhook)
	} else {
		slog.Info("no webhook url given, not setting up webhook")
	}
}

func main() {
	_ = rootCmd.Execute()
}

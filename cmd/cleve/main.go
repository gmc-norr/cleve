package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/gmc-norr/cleve"
	"github.com/gmc-norr/cleve/cmd/cleve/db"
	"github.com/gmc-norr/cleve/cmd/cleve/key"
	"github.com/gmc-norr/cleve/cmd/cleve/panel"
	"github.com/gmc-norr/cleve/cmd/cleve/platform"
	"github.com/gmc-norr/cleve/cmd/cleve/run"
	"github.com/gmc-norr/cleve/cmd/cleve/samplesheet"
	"github.com/maehler/webhook"
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

	viper.SetDefault("loglevel", "WARN")

	viper.MustBindEnv("webhook_url", "CLEVE_WEBHOOK_URL")
	viper.MustBindEnv("webhook_api_key", "CLEVE_WEBHOOK_API_KEY")
	viper.MustBindEnv("loglevel", "CLEVE_LOGLEVEL")

	if err := logger(); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
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

	webhookApiKey, err := cleve.WebhookApiKeyFromString(viper.GetString("webhook_api_key"))
	cobra.CheckErr(err)
	webhookUrl := viper.GetString("webhook_url")
	if webhookUrl != "" {
		headers := make(http.Header)
		if webhookApiKey.Value != "" {
			headers.Add(webhookApiKey.Key, webhookApiKey.Value)
		}
		client := webhook.NewClient(webhookUrl, webhook.ClientOpts.WithHeaders(headers))
		slog.Info("set up webhook", "url", webhookUrl)
		viper.Set("webhook", client)
	} else {
		slog.Info("no webhook url given, not setting up webhook")
	}
}

func logger() error {
	var logLevel slog.Level
	switch strings.ToLower(viper.GetString("loglevel")) {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		return fmt.Errorf("invalid log level: %s", viper.GetString("loglevel"))
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel}))
	slog.SetDefault(logger)
	return nil
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.SetContext(context.Background())

	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(run.RunCmd)
	rootCmd.AddCommand(db.DbCmd)
	rootCmd.AddCommand(key.KeyCmd)
	rootCmd.AddCommand(panel.PanelCmd)
	rootCmd.AddCommand(platform.PlatformCmd)
	rootCmd.AddCommand(samplesheet.SampleSheetCmd)

	rootCmd.SetVersionTemplate(`{{with .Name}}{{printf "%s " .}}{{end}}{{printf "%s\n" .Version}}`)
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "config file")
	rootCmd.PersistentFlags().String("webhook-url", viper.GetString("webhook_url"), "URL to send webhook messages to")
	rootCmd.PersistentFlags().String("webhook-api-key", viper.GetString("webhook_api_key"), "API key for the webhook service (\"<header-key>=<api-key>\")")
	rootCmd.PersistentFlags().StringP("loglevel", "l", "WARN", "Logging verbosity (case-insensitive: DEBUG, INFO, WARN, ERROR)")

	_ = viper.BindPFlag("webhook_url", rootCmd.PersistentFlags().Lookup("webhook-url"))
	_ = viper.BindPFlag("webhook_api_key", rootCmd.PersistentFlags().Lookup("webhook-api-key"))
	_ = viper.BindPFlag("loglevel", rootCmd.PersistentFlags().Lookup("loglevel"))
}

func main() {
	_ = rootCmd.Execute()
}

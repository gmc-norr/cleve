package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/gmc-norr/cleve"
	"github.com/gmc-norr/cleve/mongo"
	"github.com/maehler/webhook"
	"github.com/spf13/viper"
)

type CliConfig struct {
	Viper    *viper.Viper
	Logger   *slog.Logger
	LogLevel slog.Level

	// Database
	dbOnce sync.Once
	db     *mongo.DB
	dbErr  error

	// Webhook client
	webhookOnce   sync.Once
	webhookClient *webhook.Client
	webhookErr    error
}

func (cfg *CliConfig) WebhookClient() (*webhook.Client, error) {
	cfg.webhookOnce.Do(func() {
		url := cfg.Viper.GetString("webhook_url")
		if url == "" {
			cfg.Logger.Info("no webhook url given, not setting up webhooks")
			return
		}
		webhookApiKey, err := cleve.WebhookApiKeyFromString(cfg.Viper.GetString("webhook_api_key"))
		if err != nil {
			cfg.webhookErr = err
			return
		}
		headers := make(http.Header)
		if webhookApiKey.Value != "" {
			headers.Add(webhookApiKey.Key, webhookApiKey.Value)
		}
		cfg.webhookClient = webhook.NewClient(url, webhook.ClientOpts.WithHeaders(headers))
		cfg.Logger.Info("set up webhook", "url", url)
	})
	return cfg.webhookClient, cfg.webhookErr
}

func (cfg *CliConfig) DB() (*mongo.DB, error) {
	cfg.dbOnce.Do(func() {
		cfg.db, cfg.dbErr = mongo.Connect()
	})
	return cfg.db, cfg.dbErr
}

func (cfg *CliConfig) Close() {
}

func (cfg *CliConfig) initLogger(level string) error {
	var logLevel slog.Level
	switch strings.ToLower(level) {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		return fmt.Errorf("invalid log level: %s", level)
	}

	cfg.LogLevel = logLevel
	cfg.Logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel}))
	slog.SetDefault(cfg.Logger)
	return nil
}

func (cfg *CliConfig) initViper(configFile string) error {
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

	setViperDefaults()
	setViperEnvVars()

	err := viper.ReadInConfig()
	if err != nil {
		return err
	}

	// Basic validation
	dbConfig := viper.GetStringMap("database")
	if dbConfig == nil {
		return fmt.Errorf("missing database config")
	}
	if dbConfig["host"] == nil {
		return fmt.Errorf("missing database host")
	}
	if dbConfig["port"] == nil {
		return fmt.Errorf("missing database port")
	}
	if dbConfig["user"] == nil {
		return fmt.Errorf("missing database user")
	}
	if dbConfig["password"] == nil {
		return fmt.Errorf("missing database password")
	}
	if dbConfig["name"] == nil {
		return fmt.Errorf("missing database name")
	}

	cfg.Viper = viper.GetViper()

	return nil
}

func setViperDefaults() {
	viper.SetDefault("loglevel", "WARN")
	viper.SetDefault("auth.provider", "none")
	viper.SetDefault("run_poll_interval", 30)
	viper.SetDefault("analysis_poll_interval", 30)
}

func setViperEnvVars() {
	viper.MustBindEnv("url", "CLEVE_URL")
	viper.MustBindEnv("webhook_url", "CLEVE_WEBHOOK_URL")
	viper.MustBindEnv("webhook_api_key", "CLEVE_WEBHOOK_API_KEY")
	viper.MustBindEnv("loglevel", "CLEVE_LOGLEVEL")
	viper.MustBindEnv("auth.provider", "CLEVE_AUTHPROVIDER")
	viper.MustBindEnv("auth.realm_url", "CLEVE_KEYCLOAK_REALM_URL")
	viper.MustBindEnv("auth.internal_realm_url", "CLEVE_KEYCLOAK_INTERNAL_REALM_URL")
}

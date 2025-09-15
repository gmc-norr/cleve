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
}

func main() {
	_ = rootCmd.Execute()
}

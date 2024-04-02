package cmd

import (
	"github.com/gmc-norr/cleve/cmd/db"
	"github.com/gmc-norr/cleve/cmd/key"
	"github.com/gmc-norr/cleve/cmd/run"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"log"
)

var configFile string
var rootCmd = &cobra.Command{
	Use:     "cleve",
	Short:   "Interact with the sequencing database",
	Version: "0.1.0",
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "config file")

	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(run.RunCmd)
	rootCmd.AddCommand(db.DbCmd)
	rootCmd.AddCommand(key.KeyCmd)
}

func Execute() error {
	return rootCmd.Execute()
}

func initConfig() {
	if configFile != "" {
		viper.SetConfigFile(configFile)
	} else {
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
		viper.AddConfigPath("/etc/cleve")
		viper.AddConfigPath("$HOME/.config/cleve")
		viper.AddConfigPath(".")
	}

	err := viper.ReadInConfig()
	if err != nil {
		log.Fatalf("error: %s", err)
	}

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

	log.Printf("Using config file: %s", viper.ConfigFileUsed())
}

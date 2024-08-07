package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gmc-norr/cleve/gin"
	"github.com/gmc-norr/cleve/mongo"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	debug    bool
	host     string
	port     int
	logfile  string
	serveCmd = &cobra.Command{
		Use:   "serve",
		Short: "Serve the cleve api",
		Run: func(cmd *cobra.Command, args []string) {
			db, err := mongo.Connect()
			if err != nil {
				log.Fatal(err.Error())
			}
			host := viper.GetString("host")
			port := viper.GetInt("port")
			addr := fmt.Sprintf("%s:%d", host, port)
			router := gin.NewRouter(db, debug)
			log.Printf("Serving on %s", addr)
			log.Fatal(http.ListenAndServe(addr, router))
		},
	}
)

func init() {
	serveCmd.Flags().BoolVar(&debug, "debug", false, "serve in debug mode")
	serveCmd.Flags().StringVar(&host, "host", "localhost", "host")
	serveCmd.Flags().IntVarP(&port, "port", "p", 8080, "port")
	serveCmd.Flags().StringVar(&logfile, "logfile", "", "file to write logs in")
	viper.BindPFlag("host", serveCmd.Flags().Lookup("host"))
	viper.BindPFlag("port", serveCmd.Flags().Lookup("port"))
	viper.BindPFlag("logfile", serveCmd.Flags().Lookup("logfile"))
}

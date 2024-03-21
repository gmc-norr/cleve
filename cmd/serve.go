package cmd

import (
	"fmt"
	"github.com/gmc-norr/cleve/internal/db"
	"github.com/gmc-norr/cleve/internal/routes"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"log"
	"net/http"
)

var (
	host     string
	port     int
	serveCmd = &cobra.Command{
		Use:   "serve",
		Short: "Serve the cleve api",
		Run: func(cmd *cobra.Command, args []string) {
			db.Init()
			host := viper.GetString("host")
			port := viper.GetInt("port")
			addr := fmt.Sprintf("%s:%d", host, port)
			router := routes.NewRouter()
			log.Printf("Serving on %s", addr)
			log.Fatal(http.ListenAndServe(addr, router))
		},
	}
)

func init() {
	serveCmd.Flags().StringVar(&host, "host", "localhost", "host")
	serveCmd.Flags().IntVarP(&port, "port", "p", 8080, "port")
	viper.BindPFlag("host", serveCmd.Flags().Lookup("host"))
	viper.BindPFlag("port", serveCmd.Flags().Lookup("port"))
}

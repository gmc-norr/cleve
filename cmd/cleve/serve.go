package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gmc-norr/cleve/gin"
	"github.com/gmc-norr/cleve/mongo"
	"github.com/gmc-norr/cleve/watcher"
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
				slog.Error("failed to connect to database", "error", err)
				os.Exit(1)
			}
			host := viper.GetString("host")
			port := viper.GetInt("port")
			addr := fmt.Sprintf("%s:%d", host, port)

			loglevel := slog.LevelWarn
			if debug {
				loglevel = slog.LevelDebug
			}
			logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: loglevel}))
			slog.SetDefault(logger)

			runWatcher := watcher.NewRunWatcher(10*time.Second, db, logger)
			defer runWatcher.Stop()
			runWatcher.Start()

			interrupt := make(chan os.Signal, 1)
			signal.Notify(interrupt, syscall.SIGINT, syscall.SIGTERM)
			go func() {
				s := <-interrupt
				slog.Error("signal received, shutting down", "signal", s)
				runWatcher.Stop()
				os.Exit(1)
			}()

			router := gin.NewRouter(db, debug)
			logger.Info("serving cleve", "address", addr)
			err = http.ListenAndServe(addr, router)
			if err != nil {
				slog.Error("server crashed", "error", err)
				os.Exit(1)
			}
		},
	}
)

func init() {
	serveCmd.Flags().BoolVar(&debug, "debug", false, "serve in debug mode")
	serveCmd.Flags().StringVar(&host, "host", "localhost", "host")
	serveCmd.Flags().IntVarP(&port, "port", "p", 8080, "port")
	serveCmd.Flags().StringVar(&logfile, "logfile", "", "file to write logs in")
	_ = viper.BindPFlag("host", serveCmd.Flags().Lookup("host"))
	_ = viper.BindPFlag("port", serveCmd.Flags().Lookup("port"))
	_ = viper.BindPFlag("logfile", serveCmd.Flags().Lookup("logfile"))
}

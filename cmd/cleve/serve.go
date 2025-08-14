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

			pollInterval := viper.GetInt("run_poll_interval")
			if pollInterval < 1 {
				slog.Error("poll interval must be a positive, non-zero integer")
				os.Exit(1)
			}
			runWatcher := watcher.NewRunWatcher(time.Duration(pollInterval)*time.Second, db, logger)
			defer runWatcher.Stop()
			runStateEvents := runWatcher.Start()

			interrupt := make(chan os.Signal, 1)
			signal.Notify(interrupt, syscall.SIGINT, syscall.SIGTERM)
			go func() {
				s := <-interrupt
				slog.Error("signal received, shutting down", "signal", s)
				runWatcher.Stop()
				os.Exit(1)
			}()

			go func() {
				for events := range runStateEvents {
					for _, e := range events {
						slog.Debug("run state event", "event", e)
						if e.Changed {
							slog.Info("updating run state", "run", e.Id, "path", e.Path, "state", e.State)
							if err := db.SetRunState(e.Id, e.State); err != nil {
								slog.Error("failed to update run state", "run", e.Id, "error", err)
							}
						}
					}
				}
				slog.Info("stop handling run watcher events")
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
	defaultPollInterval := 30
	serveCmd.Flags().BoolVar(&debug, "debug", false, "serve in debug mode")
	serveCmd.Flags().StringVar(&host, "host", "localhost", "host")
	serveCmd.Flags().IntVarP(&port, "port", "p", 8080, "port")
	serveCmd.Flags().StringVar(&logfile, "logfile", "", "file to write logs in")
	serveCmd.Flags().Int("poll-interval", defaultPollInterval, "how often, in seconds, that state changes to runs should be checked")
	_ = viper.BindPFlag("host", serveCmd.Flags().Lookup("host"))
	_ = viper.BindPFlag("port", serveCmd.Flags().Lookup("port"))
	_ = viper.BindPFlag("logfile", serveCmd.Flags().Lookup("logfile"))
	_ = viper.BindPFlag("run_poll_interval", serveCmd.Flags().Lookup("poll-interval"))
	viper.SetDefault("run_poll_interval", defaultPollInterval)
}

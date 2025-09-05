package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gmc-norr/cleve"
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

			runPollInterval := viper.GetInt("run_poll_interval")
			if runPollInterval < 1 {
				slog.Error("poll interval must be a positive, non-zero integer")
				os.Exit(1)
			}
			runWatcher := watcher.NewRunWatcher(time.Duration(runPollInterval)*time.Second, db, logger)
			defer runWatcher.Stop()
			runStateEvents := runWatcher.Start()

			go func() {
				for events := range runStateEvents {
					for _, e := range events {
						slog.Debug("run state event", "event", e)
						if e.StateChanged {
							slog.Info("updating run state", "run", e.Id, "path", e.Path, "state", e.State)
							if err := db.SetRunState(e.Id, e.State); err != nil {
								slog.Error("failed to update run state", "run", e.Id, "error", err)
							}
						}
					}
				}
				slog.Info("stop handling run watcher events")
			}()

			analysisPollInterval := viper.GetInt("analysis_poll_interval")
			if analysisPollInterval < 1 {
				slog.Error("poll interval must be a positive, non-zero integer")
				os.Exit(1)
			}
			analysisWatcher := watcher.NewDragenAnalysisWatcher(time.Duration(analysisPollInterval)*time.Second, db, logger)
			defer analysisWatcher.Stop()
			analysisEvents := analysisWatcher.Start()

			go func() {
				for events := range analysisEvents {
					for _, e := range events {
						logger.Debug("analysis event", "analysis_id", e.Analysis.AnalysisId)
						if e.New {
							slog.Info("new analysis, adding", "path", e.Analysis.Path)
							if err := db.CreateAnalysis(e.Analysis); err != nil {
								logger.Error("failed to save analysis", "path", e.Analysis.Path, "analysis_id", e.Analysis.AnalysisId, "run_id", e.Analysis.AnalysisId, "error", err)
							}
							continue
						}
						if e.StateChanged {
							logger.Info("updating analysis state", "analysis_id", e.Analysis.AnalysisId, "path", e.Analysis.Path, "state", e.Analysis.StateHistory.LastState(), "new_state", e.State)
							e.Analysis.StateHistory.Add(e.Analysis.DetectState())
							if e.State == cleve.StateReady {
								logger.Info("updating analysis files", "analysis_id", e.Analysis.AnalysisId)
								if err := e.Analysis.UpdateOutputFiles(); err != nil {
									slog.Error("failed to update output files", "analysis_id", e.Analysis.AnalysisId, "error", err)
									continue
								}
							}
							if err := db.UpdateAnalysis(e.Analysis); err != nil {
								logger.Error("failed to update analysis", "analysis_id", e.Analysis.AnalysisId, "error", err)
							}
						}
					}
				}
			}()

			interrupt := make(chan os.Signal, 1)
			signal.Notify(interrupt, syscall.SIGINT, syscall.SIGTERM)
			go func() {
				s := <-interrupt
				slog.Error("signal received, shutting down", "signal", s)
				runWatcher.Stop()
				analysisWatcher.Stop()
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
	defaultPollInterval := 30
	serveCmd.Flags().BoolVar(&debug, "debug", false, "serve in debug mode")
	serveCmd.Flags().StringVar(&host, "host", "localhost", "host")
	serveCmd.Flags().IntVarP(&port, "port", "p", 8080, "port")
	serveCmd.Flags().StringVar(&logfile, "logfile", "", "file to write logs in")
	serveCmd.Flags().Int("poll-interval", defaultPollInterval, "how often, in seconds, that state changes to runs and analyses should be checked")
	_ = viper.BindPFlag("host", serveCmd.Flags().Lookup("host"))
	_ = viper.BindPFlag("port", serveCmd.Flags().Lookup("port"))
	_ = viper.BindPFlag("logfile", serveCmd.Flags().Lookup("logfile"))
	_ = viper.BindPFlag("run_poll_interval", serveCmd.Flags().Lookup("poll-interval"))
	_ = viper.BindPFlag("analysis_poll_interval", serveCmd.Flags().Lookup("poll-interval"))
	viper.SetDefault("run_poll_interval", defaultPollInterval)
}

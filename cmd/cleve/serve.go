package main

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gmc-norr/cleve"
	"github.com/gmc-norr/cleve/gin"
	"github.com/gmc-norr/cleve/interop"
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

			var webhook *cleve.Webhook
			webhook_url := viper.GetString("webhook_url")
			webhook_api_key := viper.GetString("webhook_api_key")
			if webhook_url != "" {
				api_key_header := ""
				api_key := ""
				if webhook_api_key != "" {
					parts := strings.SplitN(webhook_api_key, "=", 2)
					if len(parts) != 2 {
						slog.Error("failed to parse webhook api key")
						os.Exit(1)
					}
					api_key_header = parts[0]
					api_key = parts[1]
				}
				slog.Info("setting up webhook", "url", webhook_url)
				webhook = cleve.NewAuthWebhook(webhook_url, api_key, api_key_header)
				if err := webhook.Check(); err != nil {
					slog.Error("failed to set up webhook", "error", err)
					os.Exit(1)
				}
				slog.Info("set up webhook", "url", webhook.URL, "method", webhook.Method, "useauth", webhook.APIKey != "")
			} else {
				slog.Info("no webhook url given, won't send any webhook messages")
			}

			watcherLogPath := viper.GetString("watcher_logfile")

			var watcherLogWriter io.Writer
			if watcherLogPath != "" {
				f, err := os.OpenFile(watcherLogPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o666)
				cobra.CheckErr(err)
				watcherLogWriter = io.MultiWriter(os.Stderr, f)
			} else {
				watcherLogWriter = os.Stderr
			}
			watcherLogger := slog.New(slog.NewTextHandler(watcherLogWriter, &slog.HandlerOptions{Level: loglevel}))

			logger := slog.Default()

			runPollInterval := viper.GetInt("run_poll_interval")
			if runPollInterval < 1 {
				slog.Error("poll interval must be a positive, non-zero integer")
				os.Exit(1)
			}
			runWatcher := watcher.NewRunWatcher(time.Duration(runPollInterval)*time.Second, db, watcherLogger.With("watcher", "RunWatcher"))
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
							if webhook != nil {
								run, err := db.Run(e.Id)
								if err != nil {
									slog.Error("failed to get a run that should definitely exist", "run", e.Id, "error", err)
								} else {
									msg := cleve.NewRunMessage(run, "run state updated", cleve.MessageStateUpdate)
									if err := webhook.Send(msg); err != nil {
										slog.Error("failed to send webhook message", "error", err)
									}
								}
							}
						}
						if e.StateChanged && e.State == cleve.StateReady {
							slog.Info("loading qc data", "run", e.Id)
							qc, err := interop.InteropFromDir(e.Path)
							if err != nil {
								slog.Error("failed to read qc data", "run", e.Id, "error", err)
							}
							if err := db.UpdateRunQC(qc.Summarise()); err != nil {
								slog.Error("failed to load qc data", "run", e.Id, "error", err)
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
			analysisWatcher := watcher.NewDragenAnalysisWatcher(time.Duration(analysisPollInterval)*time.Second, db, watcherLogger.With("watcher", "DragenAnalysisWatcher"))
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
							if webhook != nil {
								msg := cleve.NewAnalysisMessage(e.Analysis, "analysis state updated", cleve.MessageStateUpdate)
								if err := webhook.Send(msg); err != nil {
									slog.Error("failed to send webhook message", "error", err)
								}
							}
							continue
						}
						if e.StateChanged {
							logger.Info("updating analysis state", "analysis_id", e.Analysis.AnalysisId, "path", e.Analysis.Path, "state", e.Analysis.StateHistory.LastState(), "new_state", e.State)
							e.Analysis.StateHistory.Add(e.Analysis.DetectState())
							if e.State == cleve.StateReady {
								logger.Info("updating analysis files", "analysis_id", e.Analysis.AnalysisId)
								if err := e.Analysis.UpdateOutputFiles(); err != nil {
									slog.Error("failed to update output files", "analysis_id", e.Analysis.AnalysisId, "path", e.Analysis.Path, "error", err)
									continue
								}
							}
							if err := db.UpdateAnalysis(e.Analysis); err != nil {
								logger.Error("failed to update analysis", "analysis_id", e.Analysis.AnalysisId, "error", err)
								continue
							}
							if webhook != nil {
								msg := cleve.NewAnalysisMessage(e.Analysis, "analysis state updated", cleve.MessageStateUpdate)
								if err := webhook.Send(msg); err != nil {
									slog.Error("failed to send webhook message", "error", err)
								}
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

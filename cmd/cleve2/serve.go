package main

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gmc-norr/cleve"
	"github.com/gmc-norr/cleve/cmd/cleve2/internal/cli"
	"github.com/gmc-norr/cleve/gin"
	"github.com/gmc-norr/cleve/internal/auth"
	"github.com/gmc-norr/cleve/interop"
	"github.com/gmc-norr/cleve/watcher"
	"github.com/spf13/cobra"
)

func NewServeCmd(cfg *CliConfig) *cobra.Command {
	var (
		debug   bool
		host    string
		port    int
		logfile string
	)
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Serve the Cleve API",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			_ = cfg.Viper.BindPFlag("host", cmd.Flag("host"))
			_ = cfg.Viper.BindPFlag("port", cmd.Flag("port"))
			_ = cfg.Viper.BindPFlag("logfile", cmd.Flag("logfile"))
			_ = cfg.Viper.BindPFlag("run_poll_interval", cmd.Flag("poll-interval"))
			_ = cfg.Viper.BindPFlag("analysis_poll_interval", cmd.Flag("poll-interval"))
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := cfg.DB()
			if err != nil {
				return err
			}
			host := cfg.Viper.GetString("host")
			port := cfg.Viper.GetInt("port")
			addr := fmt.Sprintf("%s:%d", host, port)

			webhookClient, err := cfg.WebhookClient()
			if err != nil {
				return err
			}

			watcherLogPath := cfg.Viper.GetString("watcher_logfile")

			var watcherLogWriter io.Writer
			if watcherLogPath != "" {
				f, err := os.OpenFile(watcherLogPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o666)
				cobra.CheckErr(err)
				watcherLogWriter = io.MultiWriter(os.Stderr, f)
			} else {
				watcherLogWriter = os.Stderr
			}
			watcherLogger := slog.New(slog.NewTextHandler(watcherLogWriter, &slog.HandlerOptions{Level: cfg.LogLevel}))

			runPollInterval := cfg.Viper.GetInt("run_poll_interval")
			if runPollInterval < 1 {
				return fmt.Errorf("poll interval must be a positive, non-zero integer")
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
							run, err := db.Run(e.Id)
							if err != nil {
								slog.Error("failed to get a run that should definitely exist", "run", e.Id, "error", err)
							} else {
								msg := cleve.NewRunMessage(run, "run state updated", cleve.MessageStateUpdate)
								_ = cli.SendWebhookMessage(cmd.Context(), webhookClient, msg)
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

			analysisPollInterval := cfg.Viper.GetInt("analysis_poll_interval")
			if analysisPollInterval < 1 {
				return fmt.Errorf("poll interval must be a positive, non-zero integer")
			}
			analysisWatcher := watcher.NewDragenAnalysisWatcher(time.Duration(analysisPollInterval)*time.Second, db, watcherLogger.With("watcher", "DragenAnalysisWatcher"))
			defer analysisWatcher.Stop()
			analysisEvents := analysisWatcher.Start()

			go func() {
				for events := range analysisEvents {
					for _, e := range events {
						cfg.Logger.Debug("analysis event", "analysis_id", e.Analysis.AnalysisId)
						if e.New {
							slog.Info("new analysis, adding", "path", e.Analysis.Path)
							if err := db.CreateAnalysis(e.Analysis); err != nil {
								cfg.Logger.Error("failed to save analysis", "path", e.Analysis.Path, "analysis_id", e.Analysis.AnalysisId, "run_id", e.Analysis.AnalysisId, "error", err)
								continue
							}
							msg := cleve.NewAnalysisMessage(e.Analysis, "analysis state updated", cleve.MessageStateUpdate)
							_ = cli.SendWebhookMessage(cmd.Context(), webhookClient, msg)
							continue
						}
						if e.StateChanged {
							cfg.Logger.Info("updating analysis state", "analysis_id", e.Analysis.AnalysisId, "path", e.Analysis.Path, "state", e.Analysis.StateHistory.LastState(), "new_state", e.State)
							e.Analysis.StateHistory.Add(e.Analysis.DetectState())
							if e.State == cleve.StateReady {
								cfg.Logger.Info("updating analysis files", "analysis_id", e.Analysis.AnalysisId)
								if err := e.Analysis.UpdateOutputFiles(); err != nil {
									slog.Error("failed to update output files", "analysis_id", e.Analysis.AnalysisId, "path", e.Analysis.Path, "error", err)
									continue
								}
							}
							if err := db.UpdateAnalysis(e.Analysis); err != nil {
								cfg.Logger.Error("failed to update analysis", "analysis_id", e.Analysis.AnalysisId, "error", err)
								continue
							}
							if webhookClient != nil {
								msg := cleve.NewAnalysisMessage(e.Analysis, "analysis state updated", cleve.MessageStateUpdate)
								_ = cli.SendWebhookMessage(cmd.Context(), webhookClient, msg)
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

			authType := auth.AuthType(cfg.Viper.GetString("auth.provider"))
			var authConfig *auth.Config
			switch authType {
			case auth.AuthNone:
				authConfig = auth.NoneConfig()
			case auth.AuthInternal:
				authConfig = auth.InternalConfig()
			case auth.AuthKeycloak:
				baseURL, err := url.Parse(cfg.Viper.GetString("url"))
				if err != nil {
					return fmt.Errorf("invalid application url %s: %w", cfg.Viper.GetString("url"), err)
				}
				clientID := cfg.Viper.GetString("auth.client")
				clientSecret := cfg.Viper.GetString("auth.secret")
				realm := cfg.Viper.GetString("auth.realm_url")
				internalRealm := cfg.Viper.GetString("auth.internal_realm_url")
				redirectPath := "/logincallback"
				authConfig = auth.KeycloakConfig(clientID, clientSecret, realm, internalRealm, redirectPath, baseURL.String())
			default:
				return fmt.Errorf("invalid authentication provider: %s", authType)
			}

			cfg.Logger.Info("authentiction", "provider", authConfig.AuthType)

			router := gin.NewRouter(db, authConfig, debug, webhookClient)
			cfg.Logger.Info("serving cleve", "address", addr)
			err = http.ListenAndServe(addr, router)
			if err != nil {
				return fmt.Errorf("server crashed: %w", err)
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&debug, "debug", false, "Serve in debug mode")
	cmd.Flags().StringVar(&host, "host", "localhost", "Host")
	cmd.Flags().IntVarP(&port, "port", "p", 8080, "Port")
	cmd.Flags().StringVar(&logfile, "logfile", "", "File to write logs to")
	cmd.Flags().Int("poll-interval", 0, "How often, in seconds, that state changes to runs and analyses should be checked")

	return cmd
}

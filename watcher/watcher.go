package watcher

import (
	"log/slog"
	"time"

	"github.com/gmc-norr/cleve"
)

type runHandler interface {
	Runs(filter cleve.RunFilter) (cleve.RunResult, error)
}

type analysisHandler interface {
	Analyses(filter cleve.AnalysisFilter) (cleve.AnalysisResult, error)
}

type RunWatcherEvent struct {
	Id           string
	Path         string
	State        cleve.State
	StateChanged bool
}

type RunWatcher struct {
	PollInterval time.Duration

	store     runHandler
	runFilter cleve.RunFilter
	logger    *slog.Logger

	quit chan struct{}
	done chan struct{}
	emit chan []RunWatcherEvent
}

// NewRunWatcher creates a new RunWatcher.
func NewRunWatcher(pollInterval time.Duration, db runHandler, logger *slog.Logger) RunWatcher {
	filter := cleve.NewRunFilter()
	filter.PageSize = 30
	return RunWatcher{
		PollInterval: pollInterval,
		store:        db,
		runFilter:    filter,
		logger:       logger,
		quit:         make(chan struct{}),
		done:         make(chan struct{}),
		emit:         make(chan []RunWatcherEvent, 1),
	}
}

func (w *RunWatcher) Start() chan []RunWatcherEvent {
	w.logger.Info("starting run watcher", "poll_interval", w.PollInterval)
	go w.start()
	return w.emit
}

func (w *RunWatcher) start() {
	defer close(w.done)

	ticker := time.NewTicker(w.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			w.Poll()
		case <-w.quit:
			close(w.emit)
			return
		}
	}
}

func (w *RunWatcher) Stop() {
	w.logger.Info("stopping run watcher, waiting for current poll (if any) finishes")
	close(w.quit)
	<-w.done
	w.logger.Info("run watcher stopped")
}

func (w *RunWatcher) Poll() {
	w.logger.Debug("run watcher start poll")
	w.runFilter.Page = 1
	events := make([]RunWatcherEvent, 0)
	for {
		w.logger.Debug("fetching runs", "page", w.runFilter.Page)
		runs, err := w.store.Runs(w.runFilter)
		if err != nil {
			w.logger.Error("failed to get runs", "error", err)
		}
		w.logger.Debug("got runs", "pagination", runs.PaginationMetadata)
		if runs.Count == 0 {
			w.logger.Debug("no runs, bail out")
			break
		}
		for _, r := range runs.Runs {
			knownState := r.StateHistory.LastState()
			if knownState.IsMoved() {
				// Nothing to do if the run is being moved, and we need an external
				// signal to update a moved case.
				continue
			}
			currentState := r.State(false)
			if currentState == knownState {
				continue
			}
			events = append(events, RunWatcherEvent{
				Id:           r.RunID,
				Path:         r.Path,
				State:        currentState,
				StateChanged: knownState != currentState,
			})
		}
		if w.runFilter.Page >= runs.TotalPages {
			break
		}
		w.runFilter.Page += 1
	}
	if len(events) > 0 {
		w.logger.Debug("emitting events", "count", len(events))
		w.emit <- events
	}
	w.logger.Debug("run watcher end poll")
}

type AnalysisWatcherEvent struct {
	Analysis     *cleve.Analysis
	New          bool
	StateChanged bool
}

type DragenAnalysisWatcher struct {
	PollInterval time.Duration

	store interface {
		runHandler
		analysisHandler
	}
	analysisRoot string
	runFilter    cleve.RunFilter
	logger       *slog.Logger

	quit chan struct{}
	done chan struct{}
	emit chan []AnalysisWatcherEvent
}

func NewDragenAnalysisWatcher(
	pollInterval time.Duration,
	db interface {
		runHandler
		analysisHandler
	},
	logger *slog.Logger,
) DragenAnalysisWatcher {
	filter := cleve.NewRunFilter()
	filter.PageSize = 30
	filter.State = cleve.StateReady.String()
	return DragenAnalysisWatcher{
		PollInterval: pollInterval,
		store:        db,
		analysisRoot: "Analysis",
		runFilter:    filter,
		logger:       logger,
		quit:         make(chan struct{}),
		done:         make(chan struct{}),
		emit:         make(chan []AnalysisWatcherEvent, 1),
	}
}

func (w *DragenAnalysisWatcher) Start() chan []AnalysisWatcherEvent {
	w.logger.Info("starting dragen analysis watcher", "poll_interval", w.PollInterval)
	go w.start()
	return w.emit
}

func (w *DragenAnalysisWatcher) start() {
	defer close(w.done)

	ticker := time.NewTicker(w.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			w.Poll()
		case <-w.quit:
			close(w.emit)
			return
		}
	}
}

func (w *DragenAnalysisWatcher) Stop() {
	w.logger.Info("stopping dragen analysis watcher, waiting for current poll (if any) finishes")
	close(w.quit)
	<-w.done
	w.logger.Info("dragen analysis watcher stopped")
}

func (w *DragenAnalysisWatcher) Poll() {
	w.logger.Debug("dragen analysis watcher start poll")
	w.logger.Debug("dragen analysis watcher end poll")
}

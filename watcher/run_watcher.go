package watcher

import (
	"log/slog"
	"slices"
	"time"

	"github.com/gmc-norr/cleve"
)

type runHandler interface {
	Runs(filter cleve.RunFilter) (cleve.RunResult, error)
	SetRunState(runId string, state cleve.RunState) error
}

type RunWatcher struct {
	PollInterval time.Duration

	store     runHandler
	runFilter cleve.RunFilter
	logger    *slog.Logger

	quit chan struct{}
	done chan struct{}
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
	}
}

func (w *RunWatcher) Start() {
	w.logger.Info("starting run watcher", "poll_interval", w.PollInterval)
	go w.start()
}

func (w *RunWatcher) start() {
	defer close(w.done)

	ticker := time.NewTicker(w.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			w.poll()
		case <-w.quit:
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

func (w *RunWatcher) poll() {
	w.updateStates()
}

func (w *RunWatcher) updateStates() {
	w.logger.Info("checking run states")
	w.runFilter.Page = 1
	for {
		w.logger.Debug("fetching runs", "page", w.runFilter.Page)
		runs, err := w.store.Runs(w.runFilter)
		if err != nil {
			w.logger.Error("failed to get runs", "error", err)
		}
		w.logger.Debug("got runs", "page", w.runFilter.Page, "pagination", runs.PaginationMetadata)
		if runs.Count == 0 {
			w.logger.Debug("no more runs, bail out")
			break
		}
		for _, r := range runs.Runs {
			knownState := r.StateHistory.LastState().State
			if slices.Contains([]cleve.RunState{cleve.StateMoving, cleve.StateMoved}, knownState) {
				// Nothing to do if the run is being moved, and we need an external
				// signal to update a moved case.
				continue
			}
			currentState := r.State(false)
			if knownState != currentState {
				if err := w.store.SetRunState(r.RunID, currentState); err != nil {
					w.logger.Error("failed to set run state", "run", r.RunID, "previous_state", knownState, "new_state", currentState)
				}
			}
		}
		if runs.TotalPages == w.runFilter.Page {
			break
		}
		w.runFilter.Page += 1
	}
}

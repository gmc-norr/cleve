package watcher

import (
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/gmc-norr/cleve"
	"github.com/gmc-norr/cleve/mock"
)

func assertTrue(t *testing.T, assertion func() bool, maxRetries int, waitTime time.Duration) {
	for range maxRetries {
		if assertion() {
			return
		}
		time.Sleep(waitTime)
	}
	t.Fail()
}

func assertFalse(t *testing.T, assertion func() bool, maxRetries int, waitTime time.Duration) {
	for range maxRetries {
		if assertion() {
			t.Fail()
		}
		time.Sleep(waitTime)
	}
}

func TestRunWatcher(t *testing.T) {
	testcases := []struct {
		name   string
		dbRuns cleve.RunResult
		events []WatcherEvent
	}{
		{
			name: "no runs",
			dbRuns: cleve.RunResult{
				PaginationMetadata: cleve.PaginationMetadata{},
			},
			events: []WatcherEvent{},
		},
		{
			name: "single run state unchanged",
			dbRuns: cleve.RunResult{
				PaginationMetadata: cleve.PaginationMetadata{},
				Runs: []*cleve.Run{
					{
						RunID:        "run1",
						Path:         t.TempDir(),
						StateHistory: cleve.StateHistory{{Time: time.Now(), State: cleve.StatePending}},
					},
				},
			},
			events: []WatcherEvent{},
		},
		{
			name: "single run state changed",
			dbRuns: cleve.RunResult{
				PaginationMetadata: cleve.PaginationMetadata{},
				Runs: []*cleve.Run{
					{
						RunID:        "run1",
						Path:         t.TempDir(),
						StateHistory: cleve.StateHistory{{Time: time.Now(), State: cleve.StateReady}},
					},
				},
			},
			events: []WatcherEvent{{Id: "run1", State: cleve.StatePending, StateChanged: true}},
		},
		{
			name: "two runs state changed for both",
			dbRuns: cleve.RunResult{
				PaginationMetadata: cleve.PaginationMetadata{
					Count: 2,
				},
				Runs: []*cleve.Run{
					{
						RunID:        "run1",
						Path:         t.TempDir(),
						StateHistory: cleve.StateHistory{{Time: time.Now(), State: cleve.StateReady}},
					},
					{
						RunID:        "run2",
						Path:         t.TempDir(),
						StateHistory: cleve.StateHistory{{Time: time.Now(), State: cleve.StateError}},
					},
				},
			},
			events: []WatcherEvent{
				{Id: "run1", State: cleve.StatePending, StateChanged: true},
				{Id: "run2", State: cleve.StatePending, StateChanged: true},
			},
		},
		{
			name: "single run with state moving",
			dbRuns: cleve.RunResult{
				PaginationMetadata: cleve.PaginationMetadata{
					Count: 1,
				},
				Runs: []*cleve.Run{
					{
						RunID:        "run1",
						Path:         t.TempDir(),
						StateHistory: cleve.StateHistory{{Time: time.Now(), State: cleve.StateMoving}},
					},
				},
			},
			events: []WatcherEvent{},
		},
	}

	db := mock.RunHandler{}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(logger)

	for _, c := range testcases {
		t.Run(c.name, func(t *testing.T) {
			db.RunsFn = func(filter cleve.RunFilter) (cleve.RunResult, error) {
				c.dbRuns.Page = filter.Page
				c.dbRuns.PageSize = filter.PageSize
				c.dbRuns.TotalCount = len(c.dbRuns.Runs)

				startIndex := (c.dbRuns.Page - 1) * c.dbRuns.PageSize
				endIndex := min(c.dbRuns.Page*c.dbRuns.PageSize, c.dbRuns.TotalCount)

				c.dbRuns.Count = endIndex - startIndex
				c.dbRuns.TotalPages = c.dbRuns.TotalCount / c.dbRuns.PageSize
				if c.dbRuns.TotalCount%c.dbRuns.PageSize > 0 {
					c.dbRuns.TotalPages += 1
				}
				return cleve.RunResult{
					PaginationMetadata: c.dbRuns.PaginationMetadata,
					Runs:               c.dbRuns.Runs[startIndex:endIndex],
				}, nil
			}
			w := NewRunWatcher(1*time.Minute, &db, logger)
			events := w.Start()
			defer w.Stop()

			for _, r := range c.dbRuns.Runs {
				t.Logf("current state of %s: %s", r.RunID, r.State(false))
			}

			go w.Poll()
			assertFunc := assertTrue
			if len(c.events) == 0 {
				assertFunc = assertFalse
			}
			var e []WatcherEvent
			assertFunc(t, func() bool {
				select {
				case e = <-events:
					return len(e) == len(c.events)
				case <-time.After(5 * time.Millisecond):
					return false
				}
			}, 10, 10*time.Millisecond)

			for i, event := range e {
				if event.Id != c.events[i].Id || event.State != c.events[i].State {
					t.Error("states mismatching")
				}
			}
		})
	}
}

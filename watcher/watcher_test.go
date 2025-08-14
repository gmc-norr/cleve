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
		name    string
		dbRuns  cleve.RunResult
		nEvents int
	}{
		{
			name: "no runs",
			dbRuns: cleve.RunResult{
				PaginationMetadata: cleve.PaginationMetadata{
					Count: 0,
				},
			},
			nEvents: 0,
		},
		{
			name: "single run",
			dbRuns: cleve.RunResult{
				PaginationMetadata: cleve.PaginationMetadata{
					Count: 1,
				},
				Runs: []*cleve.Run{
					{
						RunID:        "run1",
						Path:         t.TempDir(),
						StateHistory: cleve.StateHistory{{Time: time.Now(), State: cleve.StatePending}},
					},
				},
			},
			nEvents: 1,
		},
	}
	db := mock.RunHandler{}

	for _, c := range testcases {
		t.Run(c.name, func(t *testing.T) {
			db.RunsFn = func(filter cleve.RunFilter) (cleve.RunResult, error) {
				c.dbRuns.Page = filter.Page
				c.dbRuns.PageSize = filter.PageSize
				c.dbRuns.TotalCount = len(c.dbRuns.Runs)
				c.dbRuns.Count = min(c.dbRuns.Count, c.dbRuns.TotalCount)
				c.dbRuns.TotalPages = c.dbRuns.TotalCount / c.dbRuns.PageSize
				if c.dbRuns.TotalCount%c.dbRuns.PageSize == 0 {
					c.dbRuns.TotalPages++
				}
				return c.dbRuns, nil
			}
			logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
			w := NewRunWatcher(1*time.Minute, &db, logger)
			events := w.Start()
			defer w.Stop()

			go w.Poll()
			assertFunc := assertTrue
			if c.nEvents == 0 {
				assertFunc = assertFalse
			}
			assertFunc(t, func() bool {
				select {
				case e := <-events:
					return len(e) == c.nEvents
				case <-time.After(5 * time.Millisecond):
					return false
				}
			}, 10, 10*time.Millisecond)
		})
	}
}

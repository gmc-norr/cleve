package watcher

import (
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/gmc-norr/cleve"
	"github.com/gmc-norr/cleve/mock"
)

func TestRunWatcher(t *testing.T) {
	testcases := []struct {
		name   string
		dbRuns cleve.RunResult
	}{
		{
			name: "no runs",
			dbRuns: cleve.RunResult{
				PaginationMetadata: cleve.PaginationMetadata{
					Count: 0,
				},
			},
		},
	}
	db := mock.RunHandler{}

	for _, c := range testcases {
		t.Run(c.name, func(t *testing.T) {
			db.RunsFn = func(filter cleve.RunFilter) (cleve.RunResult, error) {
				c.dbRuns.Page = filter.Page
				c.dbRuns.PageSize = filter.PageSize
				return c.dbRuns, nil
			}
			db.SetRunStateFn = func(runId string, state cleve.State) error {
				t.Logf("changing state to %s for %s", state, runId)
				return nil
			}
			logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
			w := NewRunWatcher(5*time.Second, &db, logger)
			w.Start()
			defer w.Stop()
			w.poll()
		})
	}
}

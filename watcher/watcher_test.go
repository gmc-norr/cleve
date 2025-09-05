package watcher

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/gmc-norr/cleve"
	"github.com/gmc-norr/cleve/interop"
	"github.com/gmc-norr/cleve/mock"
)

// tryConsumeChannel will try to consume a single value from a channel.
// The resulting value is returned, but if the channel stays blocked for
// all retries, an error is returned.
func tryConsumeChannel[T any](ch chan T, maxRetries int, waitTime time.Duration) (T, error) {
	var v T
	for range maxRetries {
		select {
		case v = <-ch:
			return v, nil
		case <-time.After(waitTime):
			time.Sleep(waitTime)
		}
	}
	return v, fmt.Errorf("failed to read from channel")
}

func TestRunWatcher(t *testing.T) {
	testcases := []struct {
		name   string
		dbRuns cleve.RunResult
		events []RunWatcherEvent
	}{
		{
			name: "no runs",
			dbRuns: cleve.RunResult{
				PaginationMetadata: cleve.PaginationMetadata{},
			},
			events: []RunWatcherEvent{},
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
			events: []RunWatcherEvent{},
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
			events: []RunWatcherEvent{{Id: "run1", State: cleve.StatePending, StateChanged: true}},
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
			events: []RunWatcherEvent{
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
			events: []RunWatcherEvent{},
		},
		{
			name: "moved run",
			dbRuns: cleve.RunResult{
				PaginationMetadata: cleve.PaginationMetadata{
					Count: 1,
				},
				Runs: []*cleve.Run{
					{
						RunID:        "run1",
						StateHistory: cleve.StateHistory{{Time: time.Now(), State: cleve.StateReady}},
					},
				},
			},
			events: []RunWatcherEvent{{Id: "run1", State: cleve.StateMoved}},
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
			eventCh := w.Start()
			defer w.Stop()

			for _, r := range c.dbRuns.Runs {
				t.Logf("current state of %s: %s", r.RunID, r.State(false))
			}

			go w.Poll()
			events, err := tryConsumeChannel(eventCh, 10, 10*time.Millisecond)
			if err != nil && len(c.events) > 0 || len(events) != len(c.events) {
				t.Fatalf("expected %d events, got %d", len(c.events), len(events))
			}

			for i, e := range events {
				if e.Id != c.events[i].Id || e.State != c.events[i].State {
					t.Error("states mismatching")
				}
			}
		})
	}
}

func TestDragenAnalysisWatcher(t *testing.T) {
	type diskAnalysis struct {
		dir                       string
		copyComplete              bool
		secondaryAnalysisComplete bool
	}
	testcases := []struct {
		name         string
		dbRuns       []*cleve.Run
		dbAnalyses   []*cleve.Analysis
		events       []AnalysisWatcherEvent
		diskAnalyses [][]*diskAnalysis
	}{
		{
			name: "no runs",
		},
		{
			// Run is not ready, no events should be emitted
			name: "single pending run pending analysis",
			dbRuns: []*cleve.Run{
				{
					RunID:        "run1",
					Path:         t.TempDir(),
					StateHistory: cleve.StateHistory{{Time: time.Now(), State: cleve.StatePending}},
					RunParameters: interop.RunParameters{
						Software: []interop.Software{{Name: "Dragen", Version: "4.3.16"}},
					},
				},
			},
			diskAnalyses: [][]*diskAnalysis{
				{{dir: "Analysis/1", copyComplete: false, secondaryAnalysisComplete: false}},
			},
		},
		{
			// A new analysis event should be emitted
			name: "single run new analysis pending",
			dbRuns: []*cleve.Run{
				{
					RunID:        "run1",
					Path:         t.TempDir(),
					StateHistory: cleve.StateHistory{{Time: time.Now(), State: cleve.StateReady}},
					RunParameters: interop.RunParameters{
						Software: []interop.Software{{Name: "Dragen", Version: "4.3.16"}},
					},
				},
			},
			events: []AnalysisWatcherEvent{{New: true, State: cleve.StatePending}},
			diskAnalyses: [][]*diskAnalysis{
				{{dir: "Analysis/1", copyComplete: false, secondaryAnalysisComplete: false}},
			},
		},
		{
			// A state change event should be emitted
			name: "single run existing analysis pending to ready",
			dbRuns: []*cleve.Run{
				{
					RunID:        "run1",
					Path:         t.TempDir(),
					StateHistory: cleve.StateHistory{{Time: time.Now(), State: cleve.StateReady}},
					RunParameters: interop.RunParameters{
						Software: []interop.Software{{Name: "Dragen", Version: "4.3.16"}},
					},
				},
			},
			dbAnalyses: []*cleve.Analysis{
				{
					AnalysisId:   "run1_1_bclconvert",
					Runs:         []string{"run1"},
					Path:         "Analysis/1",
					Software:     "Dragen BCLConvert",
					StateHistory: cleve.StateHistory{{Time: time.Now(), State: cleve.StatePending}},
				},
			},
			events: []AnalysisWatcherEvent{{New: false, State: cleve.StateReady, StateChanged: true}},
			diskAnalyses: [][]*diskAnalysis{
				{{dir: "Analysis/1", copyComplete: true, secondaryAnalysisComplete: true}},
			},
		},
		{
			// Single analysis pointing to two individual runs. The last known state of the analysis
			// is pending, and the current state is also pending, so no events should be emitted.
			name: "two runs single pending analysis",
			dbRuns: []*cleve.Run{
				{
					RunID:        "run1",
					Path:         t.TempDir(),
					StateHistory: cleve.StateHistory{{Time: time.Now(), State: cleve.StateReady}},
					RunParameters: interop.RunParameters{
						Software: []interop.Software{{Name: "Dragen", Version: "4.3.16"}},
					},
				},
				{
					RunID:        "run2",
					Path:         t.TempDir(),
					StateHistory: cleve.StateHistory{{Time: time.Now(), State: cleve.StateReady}},
					RunParameters: interop.RunParameters{
						Software: []interop.Software{{Name: "Dragen", Version: "4.3.16"}},
					},
				},
			},
			dbAnalyses: []*cleve.Analysis{
				{
					AnalysisId:   "run2_1_bclconvert",
					Runs:         []string{"run1", "run2"},
					Path:         "Analysis/1",
					Software:     "Dragen BCLConvert",
					StateHistory: cleve.StateHistory{{Time: time.Now(), State: cleve.StatePending}},
				},
			},
			diskAnalyses: [][]*diskAnalysis{
				nil,
				{{dir: "Analysis/1", copyComplete: false, secondaryAnalysisComplete: false}},
			},
		},
	}

	db := struct {
		mock.RunHandler
		mock.AnalysesHandler
	}{
		RunHandler:      mock.RunHandler{},
		AnalysesHandler: mock.AnalysesHandler{},
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(logger)

	for _, c := range testcases {
		t.Run(c.name, func(t *testing.T) {
			db.RunsFn = func(filter cleve.RunFilter) (cleve.RunResult, error) {
				var filteredRuns []*cleve.Run
				// Set up analyses on disk
				for i, r := range c.dbRuns {
					diskAnalyses := c.diskAnalyses[i]
					for _, a := range diskAnalyses {
						if a == nil {
							continue
						}
						dir := filepath.Join(r.Path, a.dir)
						if err := os.MkdirAll(filepath.Join(dir, "Data", "summary", r.RunParameters.Software[0].Version), 0o755); err != nil {
							t.Fatal(err)
						}
						if a.copyComplete {
							path := filepath.Join(dir, "CopyComplete.txt")
							if _, err := os.Create(path); err != nil {
								t.Fatal(err)
							}
						}
						if a.secondaryAnalysisComplete {
							path := filepath.Join(dir, "Data", "Secondary_Analysis_Complete.txt")
							if _, err := os.Create(path); err != nil {
								t.Fatal(err)
							}
						}
					}
					if filter.State == r.StateHistory.LastState().String() {
						filteredRuns = append(filteredRuns, r)
					}
				}
				runResult := cleve.RunResult{
					PaginationMetadata: cleve.PaginationMetadata{
						Page:       filter.Page,
						PageSize:   filter.PageSize,
						TotalPages: 1,
						Count:      len(filteredRuns),
						TotalCount: len(filteredRuns),
					},
					Runs: filteredRuns,
				}
				return runResult, nil
			}
			db.AnalysesFn = func(filter cleve.AnalysisFilter) (cleve.AnalysisResult, error) {
				var runAnalyses []*cleve.Analysis
				for i, a := range c.dbAnalyses {
					if slices.Contains(a.Runs, filter.RunId) {
						a.Path = filepath.Join(c.dbRuns[i].Path, a.Path)
						runAnalyses = append(runAnalyses, a)
					}
				}
				analysisResult := cleve.AnalysisResult{
					PaginationMetadata: cleve.PaginationMetadata{
						Page:       1,
						PageSize:   filter.PageSize,
						TotalPages: 1,
						Count:      len(runAnalyses),
						TotalCount: len(runAnalyses),
					},
					Analyses: runAnalyses,
				}
				return analysisResult, nil
			}

			if len(c.dbRuns) != len(c.diskAnalyses) {
				t.Fatal("runs and disk analyses must have the same length, fix the test!")
			}

			w := NewDragenAnalysisWatcher(1*time.Minute, &db, logger)
			eventCh := w.Start()
			defer w.Stop()

			go w.Poll()
			events, err := tryConsumeChannel(eventCh, 10, 10*time.Millisecond)
			if err != nil && len(c.events) > 0 || len(events) != len(c.events) {
				for i, e := range events {
					t.Logf("event %d: %+v", i+1, e)
				}
				t.Fatalf("expected %d events, got %d", len(c.events), len(events))
			}

			for i, e := range events {
				if e.New != c.events[i].New || e.State != c.events[i].State || e.StateChanged != c.events[i].StateChanged {
					t.Errorf("states mismatching, expected %v, got %v", c.events[i], e)
				}
			}
		})
	}
}

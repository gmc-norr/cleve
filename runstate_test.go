package cleve

import (
	"testing"
	"time"
)

func TestRunState(t *testing.T) {
	type testCase struct {
		name string
		args RunState
		want string
	}
	tests := []testCase{
		{
			name: "new",
			args: StateNew,
			want: "new",
		},
		{
			name: "ready",
			args: StateReady,
			want: "ready",
		},
		{
			name: "pending",
			args: StatePending,
			want: "pending",
		},
		{
			name: "complete",
			args: StateComplete,
			want: "complete",
		},
		{
			name: "error",
			args: StateError,
			want: "error",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.args.String(); got != tt.want {
				t.Errorf("RunState.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInvalidState(t *testing.T) {
	var s RunState
	if err := s.Set("invalid"); err == nil { // No error
		t.Errorf("invalid state should return an error")
	}
}

func TestRunStateSet(t *testing.T) {
	var s RunState
	err := s.Set("invalid")
	if err == nil {
		t.Errorf("invalid state should return an error")
	}

	if err := s.Set("new"); err != nil {
		t.Fatalf("error setting state to new: %s", err)
	}
	actual := s.String()
	expected := "new"
	if actual != expected {
		t.Errorf("expected %s, got %s", expected, actual)
	}
}

func TestLastState(t *testing.T) {
	testcases := []struct {
		name      string
		history   StateHistory
		lastState RunState
	}{
		{
			name:      "empty history",
			history:   StateHistory{},
			lastState: StateUnknown,
		},
		{
			name: "single state",
			history: StateHistory{
				{
					Time:  time.Now(),
					State: StateReady,
				},
			},
			lastState: StateReady,
		},
		{
			name: "multiple states",
			history: StateHistory{
				{
					Time:  time.Now(),
					State: StateReady,
				},
				{
					Time:  time.Now().Add(-time.Hour),
					State: StatePending,
				},
			},
			lastState: StateReady,
		},
		{
			name: "multiple states reversed",
			history: StateHistory{
				{
					Time:  time.Now().Add(-time.Hour),
					State: StatePending,
				},
				{
					Time:  time.Now(),
					State: StateReady,
				},
			},
			lastState: StateReady,
		},
	}

	for _, c := range testcases {
		t.Run(c.name, func(t *testing.T) {
			if c.history.LastState().State != c.lastState {
				t.Errorf("expected last state to be %s, got %s", c.lastState, c.history.LastState())
			}
		})
	}
}

func TestIsMoved(t *testing.T) {
	testcases := []struct {
		name    string
		state   RunState
		isMoved bool
	}{
		{
			name:    "moved",
			state:   StateMoved,
			isMoved: true,
		},
		{
			name:    "moving",
			state:   StateMoving,
			isMoved: true,
		},
		{
			name:    "ready",
			state:   StateReady,
			isMoved: false,
		},
		{
			name:    "unknown",
			state:   StateUnknown,
			isMoved: false,
		},
	}

	for _, c := range testcases {
		t.Run(c.name, func(t *testing.T) {
			if c.state.IsMoved() != c.isMoved {
				t.Errorf("expected IsMoved to be %t, got %t", c.isMoved, c.state.IsMoved())
			}
		})
	}
}

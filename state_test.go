package cleve

import (
	"testing"
	"time"
)

func TestRunState(t *testing.T) {
	type testCase struct {
		name string
		args State
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
			name: "incomplete",
			args: StateIncomplete,
			want: "incomplete",
		},
		{
			name: "error",
			args: StateError,
			want: "error",
		},
		{
			name: "moved",
			args: StateMoved,
			want: "moved",
		},
		{
			name: "moving",
			args: StateMoving,
			want: "moving",
		},
		{
			name: "unknown",
			args: StateUnknown,
			want: "unknown",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.args.IsValid() {
				t.Errorf("state %s should be valid", tt.args)
			}
			if got := tt.args.String(); got != tt.want {
				t.Errorf("RunState.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInvalidState(t *testing.T) {
	var s State
	if err := s.Set("invalid"); err == nil { // No error
		t.Errorf("invalid state should return an error")
	}
}

func TestRunStateSet(t *testing.T) {
	var s State
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
		lastState State
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
			if c.history.LastState() != c.lastState {
				t.Errorf("expected last state to be %s, got %s", c.lastState, c.history.LastState())
			}
		})
	}
}

func TestIsMoved(t *testing.T) {
	testcases := []struct {
		name    string
		state   State
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

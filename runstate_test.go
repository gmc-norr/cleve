package cleve

import (
	"testing"
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

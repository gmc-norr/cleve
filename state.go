package cleve

import (
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsontype"
)

type State int

const (
	StateNew State = iota
	StateReady
	StatePending
	StateComplete
	StateIncomplete
	StateError
	StateMoved
	StateMoving
	StateUnknown
)

var ValidRunStates = map[string]State{
	"new":        StateNew,
	"ready":      StateReady,
	"pending":    StatePending,
	"complete":   StateComplete,
	"incomplete": StateIncomplete,
	"error":      StateError,
	"moved":      StateMoved,
	"moving":     StateMoving,
	"unknown":    StateUnknown,
}

// IsMoved returns true if the state is `moved` or `moving`. Otherwise
// it returns false.
func (s State) IsMoved() bool {
	return slices.Contains([]State{StateMoved, StateMoving}, s)
}

func (s State) String() string {
	for k, v := range ValidRunStates {
		if v == s {
			return k
		}
	}
	return ""
}

func (s *State) Set(v string) error {
	state, ok := ValidRunStates[v]
	if !ok {
		return fmt.Errorf("illegal state: %#v", v)
	}
	*s = state
	return nil
}

func (s *State) Type() string {
	return "RunState"
}

func (r *State) UnmarshalBSONValue(t bsontype.Type, data []byte) error {
	var s string
	var state State
	if err := bson.UnmarshalValue(bson.TypeString, data, &s); err != nil {
		return err
	}

	err := state.Set(s)
	if err != nil {
		return err
	}

	*r = state

	return nil
}

func (r State) MarshalBSONValue() (bsontype.Type, []byte, error) {
	state := r.String()
	return bson.MarshalValue(state)
}

func (r State) MarshalJSON() ([]byte, error) {
	return json.Marshal(r.String())
}

func (r *State) UnmarshalJSON(data []byte) error {
	var s string
	var state State
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	err := state.Set(s)
	if err != nil {
		return err
	}
	*r = state
	return nil
}

type TimedRunState struct {
	State State     `bson:"state" json:"state"`
	Time  time.Time `bson:"time" json:"time"`
}

// StateHistory represents a slice of TimedRunState
type StateHistory []TimedRunState

// LastState returns the most recent TimedRunState in the state history. If the
// history is empty, Unknown is returned with the current time.
func (h StateHistory) LastState() TimedRunState {
	if len(h) == 0 {
		return TimedRunState{
			Time:  time.Now(),
			State: StateUnknown,
		}
	}
	slices.SortFunc(h, func(a, b TimedRunState) int {
		return b.Time.Compare(a.Time)
	})
	return h[0]
}

// Add adds a new state to the state history with the current time.
func (h *StateHistory) Add(state State) {
	s := TimedRunState{
		Time:  time.Now(),
		State: state,
	}
	*h = append(*h, s)
}

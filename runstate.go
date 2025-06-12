package cleve

import (
	"encoding/json"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsontype"
)

type RunState int

const (
	New RunState = iota
	Ready
	Pending
	Complete
	Incomplete
	Error
	Moved
	Moving
	Unknown
)

var ValidRunStates = map[string]RunState{
	"new":        New,
	"ready":      Ready,
	"pending":    Pending,
	"complete":   Complete,
	"incomplete": Incomplete,
	"error":      Error,
	"moved":      Moved,
	"moving":     Moving,
	"unknown":    Unknown,
}

func (s RunState) String() string {
	for k, v := range ValidRunStates {
		if v == s {
			return k
		}
	}
	return ""
}

func (s *RunState) Set(v string) error {
	state, ok := ValidRunStates[v]
	if !ok {
		return fmt.Errorf("illegal state: %#v", v)
	}
	*s = state
	return nil
}

func (s *RunState) Type() string {
	return "RunState"
}

func (r *RunState) UnmarshalBSONValue(t bsontype.Type, data []byte) error {
	var s string
	var state RunState
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

func (r RunState) MarshalBSONValue() (bsontype.Type, []byte, error) {
	state := r.String()
	return bson.MarshalValue(state)
}

func (r RunState) MarshalJSON() ([]byte, error) {
	return json.Marshal(r.String())
}

func (r *RunState) UnmarshalJSON(data []byte) error {
	var s string
	var state RunState
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
	State RunState  `bson:"state" json:"state"`
	Time  time.Time `bson:"time" json:"time"`
}

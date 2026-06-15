package declaration

import (
	"errors"
)

var ErrUnsafeDelta = errors.New("unsafe declaration delta")

type Delta struct {
	Installable bool             `json:"installable"`
	Additions   []RecordRef      `json:"additions"`
	Removals    []RecordRef      `json:"removals"`
	Clearances  []RecordRef      `json:"routing_exclusion_clearances"`
	Conflicts   []RecordConflict `json:"conflicts"`
}

type RecordRef struct {
	Kind string `json:"kind"`
	Key  string `json:"key"`
}

type RecordConflict struct {
	Kind  string `json:"kind"`
	Key   string `json:"key"`
	State string `json:"state"`
}

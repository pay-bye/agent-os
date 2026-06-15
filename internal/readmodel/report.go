package readmodel

import (
	"context"
	"time"
)

const (
	Complete Result = "complete"
	Partial  Result = "partial"
)

const (
	QueueGroup         Group = "queue"
	LeasesGroup        Group = "leases"
	JournalGroup       Group = "journal"
	CommandsGroup      Group = "commands"
	RoutingGroup       Group = "routing"
	BuildGroup         Group = "build"
	CompatibilityGroup Group = "compatibility"
)

const AllChannels ChannelClass = "all"

type Result string

type Group string

type ChannelClass string

type Views struct {
	Queue         *Queue         `json:"queue,omitempty"`
	Leases        *Leases        `json:"leases,omitempty"`
	Journal       *Journal       `json:"journal,omitempty"`
	Commands      *Commands      `json:"commands,omitempty"`
	Routing       *Routing       `json:"routing,omitempty"`
	Build         *Build         `json:"build,omitempty"`
	Compatibility *Compatibility `json:"compatibility,omitempty"`
}

type Queue struct {
	ChannelClass              ChannelClass `json:"channel_class"`
	Depth                     int          `json:"depth"`
	Available                 int          `json:"available"`
	OldestAvailableAgeSeconds int          `json:"oldest_available_age_seconds"`
}

type Leases struct {
	ChannelClass ChannelClass `json:"channel_class"`
	Held         int          `json:"held"`
	Expired      int          `json:"expired"`
}

type Journal struct {
	AppendRatePerSecond float64 `json:"append_rate_per_second"`
	WindowSeconds       int     `json:"window_seconds"`
}

type Commands struct {
	Succeeded int `json:"succeeded"`
	Failed    int `json:"failed"`
}

type Routing struct {
	Routed   int `json:"routed"`
	Unrouted int `json:"unrouted"`
}

type Build struct {
	Version  string `json:"version"`
	Revision string `json:"revision"`
}

type Compatibility struct {
	ContractVersion string   `json:"contract_version"`
	Features        []string `json:"features"`
	Routes          []Route  `json:"routes"`
}

type Route struct {
	Method string `json:"method"`
	Path   string `json:"path"`
}

type Pressure struct {
	Depth                     int
	Available                 int
	Held                      int
	Expired                   int
	OldestAvailableAgeSeconds int
}

type Window struct {
	Appends int
	Seconds int
}

type Counters struct {
	CommandsSucceeded int
	CommandsFailed    int
	Routed            int
	Unrouted          int
}

type PressureSource interface {
	Pressure(context.Context, time.Time) (Pressure, error)
}

type JournalSource interface {
	Journal(context.Context, time.Time, time.Duration) (Window, error)
}

type CounterSource interface {
	Counters(context.Context) (Counters, error)
}

type BuildSource interface {
	Build(context.Context) (Build, error)
}

type CompatibilitySource interface {
	Compatibility(context.Context) (Compatibility, error)
}

type Clock interface {
	Now() time.Time
}

func groups() []Group {
	return []Group{
		QueueGroup,
		LeasesGroup,
		JournalGroup,
		CommandsGroup,
		RoutingGroup,
		BuildGroup,
		CompatibilityGroup,
	}
}

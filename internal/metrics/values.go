package metrics

import (
	"context"
	"time"
)

const SeriesBudget = 600

const (
	Submit        Operation = "submit"
	Claim         Operation = "claim"
	Ack           Operation = "ack"
	Nack          Operation = "nack"
	Extend        Operation = "extend"
	Heartbeat     Operation = "heartbeat"
	Instruction   Operation = "instruction"
	Compatibility Operation = "compatibility"
	Health        Operation = "health"
	Readyz        Operation = "readyz"
	Scrape        Operation = "metrics"
	Operations    Operation = "operations"
)

const (
	Completed Result = "completed"
	Failed    Result = "failed"
	Rejected  Result = "rejected"
	NotReady  Result = "not_ready"
	Succeeded Result = "succeeded"
)

const (
	Preview DeclarationOperation = "preview"
	Apply   DeclarationOperation = "apply"
)

const (
	ItemSubmitted    EventKind = "item_submitted"
	NeedDeclared     EventKind = "need_declared"
	ItemRouted       EventKind = "item_routed"
	NeedAcknowledged EventKind = "need_acknowledged"
	NeedRejected     EventKind = "need_rejected"
)

const (
	Routed        Outcome = "routed"
	Unrouted      Outcome = "unrouted"
	NoRoute       Outcome = "no_route"
	FailedOutcome Outcome = "failed"
)

type Operation string

type Result string

type DeclarationOperation string

type EventKind string

type Outcome string

type Build struct {
	Version  string
	Revision string
}

type Storage struct {
	QueueDepth    int
	LeasesHeld    int
	LeasesExpired int
}

type Store interface {
	Read(context.Context, time.Time) (Storage, error)
}

type Clock interface {
	Now() time.Time
}

type clock struct{}

func (clock) Now() time.Time {
	return time.Now().UTC()
}

func requestOperations() []Operation {
	return []Operation{
		Submit,
		Claim,
		Ack,
		Nack,
		Extend,
		Heartbeat,
		Instruction,
		Compatibility,
		Health,
		Readyz,
		Scrape,
		Operations,
	}
}

func requestResults() []Result {
	return []Result{Completed, Failed, Rejected, NotReady}
}

func operationResults() []Result {
	return []Result{Succeeded, Failed}
}

func declarationOperations() []DeclarationOperation {
	return []DeclarationOperation{Preview, Apply}
}

func eventKinds() []EventKind {
	return []EventKind{
		ItemSubmitted,
		NeedDeclared,
		ItemRouted,
		NeedAcknowledged,
		NeedRejected,
	}
}

func outcomes() []Outcome {
	return []Outcome{Routed, Unrouted, NoRoute, FailedOutcome}
}

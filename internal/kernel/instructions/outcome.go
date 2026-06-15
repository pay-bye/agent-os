package instructions

import (
	"github.com/pay-bye/agent-os/internal/channel"
	"github.com/pay-bye/agent-os/internal/journal"
	"github.com/pay-bye/agent-os/internal/journal/payloads"
	"github.com/pay-bye/agent-os/internal/registry"
	"github.com/pay-bye/agent-os/internal/workitem"
	"time"
)

type Application struct {
	Result Result

	Outcomes []Outcome
	Effects  Effects
	Routes   []RouteStep
}

type Outcome struct {
	Event      journal.EventID
	Coordinate journal.Coordinate
	Kind       registry.JournalEventKindKey
	AppendedAt time.Time
	Payload    []byte
}

type Effects struct {
	Exclusions []registry.NodeKey
	Releases   []channel.LeaseID
	Moves      []MoveEffect
	Deletes    []workitem.ID
}

type MoveEffect struct {
	Entries []channel.EntryID
	Target  registry.ChannelKey
}

type audit struct {
	fields        map[string]any
	preconditions []string
}

type applicationScope struct {
	record Record
	event  journal.EventID
	audit  audit
}

func newAudit(fields map[string]any, preconditions ...string) audit {
	return audit{
		fields:        fields,
		preconditions: copyStrings(preconditions),
	}
}

func newScope(record Record, event journal.EventID, audit audit) applicationScope {
	return applicationScope{record: record, event: event, audit: audit}
}

func applied(
	scope applicationScope,
	coordinate journal.Coordinate,
	affected []string,
	effects Effects,
) (Application, error) {
	outcome, err := outcome(scope, coordinate, payloads.InstructionAppliedKind, affected, "", Applied)
	if err != nil {
		return Application{}, err
	}
	return Application{
		Result:   result(scope.record, Applied, []string{scope.event.String()}, affected, ""),
		Outcomes: []Outcome{outcome},
		Effects:  effects,
	}, nil
}

func rejected(scope applicationScope, coordinate journal.Coordinate, precondition string) (Application, error) {
	outcome, err := outcome(
		scope,
		coordinate,
		payloads.InstructionRejectedKind,
		nil,
		precondition,
		PreconditionFailed,
	)
	if err != nil {
		return Application{}, err
	}
	return Application{
		Result:   result(scope.record, PreconditionFailed, []string{scope.event.String()}, nil, precondition),
		Outcomes: []Outcome{outcome},
	}, nil
}

func outcome(
	scope applicationScope,
	coordinate journal.Coordinate,
	kind registry.JournalEventKindKey,
	affected []string,
	failed string,
	value ResultValue,
) (Outcome, error) {
	body, err := outcomePayload(kind, payloads.InstructionOutcomeInput{
		ID:                 scope.record.ID.String(),
		Operation:          scope.record.Kind,
		AuditFields:        scope.audit.fields,
		AffectedIDs:        affected,
		FailedPrecondition: failed,
		Result:             string(value),
		Preconditions:      scope.audit.preconditions,
		AppendedAt:         scope.record.RecordedAt,
	})
	if err != nil {
		return Outcome{}, err
	}
	return Outcome{
		Event:      scope.event,
		Coordinate: coordinate,
		Kind:       kind,
		AppendedAt: scope.record.RecordedAt,
		Payload:    body,
	}, nil
}

func outcomePayload(kind registry.JournalEventKindKey, input payloads.InstructionOutcomeInput) ([]byte, error) {
	switch kind {
	case payloads.InstructionAppliedKind:
		return payloads.InstructionApplied(input)
	case payloads.InstructionRejectedKind:
		return payloads.InstructionRejected(input)
	case payloads.WorkItemDroppedKind:
		return payloads.WorkItemDropped(input)
	default:
		return nil, registry.JournalEventKindNotFound(kind)
	}
}

func result(record Record, value ResultValue, events []string, affected []string, failed string) Result {
	return Result{
		ID:                 record.ID,
		Result:             value,
		EventIDs:           copyStrings(events),
		AffectedIDs:        copyStrings(affected),
		FailedPrecondition: failed,
	}
}

func eventIDs(events []journal.EventID) []string {
	ids := make([]string, 0, len(events))
	for _, event := range events {
		ids = append(ids, event.String())
	}
	return ids
}

func workItemIDs(items []workitem.ID) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.String())
	}
	return ids
}

func entryIDs(entries []channel.EntryID) []string {
	ids := make([]string, 0, len(entries))
	for _, entry := range entries {
		ids = append(ids, entry.String())
	}
	return ids
}

func copyWorkItems(items []workitem.ID) []workitem.ID {
	return append([]workitem.ID(nil), items...)
}

func copyStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return append([]string(nil), values...)
}

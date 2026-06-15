package kernel

import (
	"context"
	"database/sql"
	coordination "github.com/pay-bye/agent-os/internal/channel"
	"github.com/pay-bye/agent-os/internal/journal"
	"github.com/pay-bye/agent-os/internal/journal/payloads"
	root "github.com/pay-bye/agent-os/internal/kernel"
	"github.com/pay-bye/agent-os/internal/kernel/resolution"
	"github.com/pay-bye/agent-os/internal/registry"
	"github.com/pay-bye/agent-os/internal/workitem"
)

var (
	ackedResolution  = resolutionEvent{kind: payloads.NeedAckedKind, codec: payloads.NeedAcked}
	nackedResolution = resolutionEvent{kind: payloads.NeedNackedKind, codec: payloads.NeedNacked}
)

type resolutionEvent struct {
	kind  registry.JournalEventKindKey
	codec func(payloads.Resolution) ([]byte, error)
}

func resolve(
	ctx context.Context,
	tx *sql.Tx,
	command resolution.Command,
	event resolutionEvent,
) (root.ResolutionResult, error) {
	lease, err := newChannel(tx).Heartbeat(
		ctx,
		command.Lease,
		command.TokenDigest,
		command.ResolvedAt,
	)
	if err != nil {
		return root.ResolutionResult{}, err
	}
	events := newJournal(tx)
	if err := appendResolution(ctx, events, command, lease, event); err != nil {
		return root.ResolutionResult{}, err
	}
	if err := appendResolutionNeeds(ctx, events, command, lease.WorkItem()); err != nil {
		return root.ResolutionResult{}, err
	}
	if err := cleanupLease(ctx, tx, lease); err != nil {
		return root.ResolutionResult{}, err
	}
	route, err := routeOutstanding(ctx, tx, routeCommand{
		WorkItem:  lease.WorkItem(),
		Event:     command.RouteEvent,
		Entry:     command.Entry,
		CreatedAt: command.ResolvedAt,
	})
	if err != nil {
		return root.ResolutionResult{}, err
	}
	return root.ResolutionResult{Resolved: true, Routed: route.Routed, Channel: route.Channel}, nil
}

func appendResolution(
	ctx context.Context,
	events *journalStore,
	command resolution.Command,
	lease coordination.Lease,
	event resolutionEvent,
) error {
	payload, err := event.codec(payloads.Resolution{
		Lease:          lease,
		FailurePayload: command.FailurePayload,
	})
	if err != nil {
		return err
	}
	_, err = events.Append(ctx, journal.EventInput{
		ID:         command.Event,
		Coordinate: journal.WorkItemCoordinate(lease.WorkItem()),
		Kind:       event.kind,
		AppendedAt: command.ResolvedAt,
		Payload:    payload,
	})
	return err
}

func appendResolutionNeeds(
	ctx context.Context,
	events *journalStore,
	command resolution.Command,
	id workitem.ID,
) error {
	for index, need := range command.DeclaredNeeds {
		payload, err := payloads.NeedDeclared(need)
		if err != nil {
			return err
		}
		if _, err := events.Append(ctx, journal.EventInput{
			ID:         command.NeedEvents[index],
			Coordinate: journal.WorkItemCoordinate(id),
			Kind:       payloads.NeedDeclaredKind,
			AppendedAt: command.ResolvedAt,
			Payload:    payload,
		}); err != nil {
			return err
		}
	}
	return nil
}

func cleanupLease(ctx context.Context, tx *sql.Tx, lease coordination.Lease) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM leases WHERE id = $1`, lease.ID().String()); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, `DELETE FROM channel_entries WHERE id = $1`, lease.Entry().String())
	return err
}

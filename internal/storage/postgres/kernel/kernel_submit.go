package kernel

import (
	"context"
	"database/sql"
	"github.com/pay-bye/agent-os/internal/journal"
	"github.com/pay-bye/agent-os/internal/journal/payloads"
	root "github.com/pay-bye/agent-os/internal/kernel"
	"github.com/pay-bye/agent-os/internal/kernel/submission"
)

func submit(ctx context.Context, tx *sql.Tx, command submission.Command) (root.SubmitResult, error) {
	events := newJournal(tx)
	if err := insertWorkItem(ctx, tx, command); err != nil {
		return root.SubmitResult{}, err
	}
	if err := appendItemSubmitted(ctx, events, command); err != nil {
		return root.SubmitResult{}, err
	}
	if err := appendNeeds(ctx, events, command); err != nil {
		return root.SubmitResult{}, err
	}
	result, err := routeOutstanding(ctx, tx, routeCommand{
		WorkItem:  command.Submission.ID(),
		Event:     command.RouteEvent,
		Entry:     command.Entry,
		CreatedAt: command.SubmittedAt,
	})
	if err != nil {
		return root.SubmitResult{}, err
	}
	return root.SubmitResult{
		WorkItem: command.Submission.ID(),
		Routed:   result.Routed,
		Channel:  result.Channel,
	}, nil
}

func insertWorkItem(ctx context.Context, tx *sql.Tx, command submission.Command) error {
	_, err := tx.ExecContext(ctx, `
INSERT INTO work_items (id, item_kind_key, payload, submitted_at)
VALUES ($1, $2, $3, $4)`,
		command.Submission.ID().String(),
		command.Submission.Kind().String(),
		command.Submission.Payload(),
		command.SubmittedAt,
	)
	return err
}

func appendItemSubmitted(ctx context.Context, events *journalStore, command submission.Command) error {
	payload, err := payloads.ItemSubmitted(command.Submission)
	if err != nil {
		return err
	}
	_, err = events.Append(ctx, journal.EventInput{
		ID:         command.ItemEvent,
		Coordinate: journal.WorkItemCoordinate(command.Submission.ID()),
		Kind:       payloads.ItemSubmittedKind,
		AppendedAt: command.SubmittedAt,
		Payload:    payload,
	})
	return err
}

func appendNeeds(ctx context.Context, events *journalStore, command submission.Command) error {
	needs := command.Submission.DeclaredNeeds()
	for index, need := range needs {
		payload, err := payloads.NeedDeclared(need)
		if err != nil {
			return err
		}
		if _, err := events.Append(ctx, journal.EventInput{
			ID:         command.NeedEvents[index],
			Coordinate: journal.WorkItemCoordinate(command.Submission.ID()),
			Kind:       payloads.NeedDeclaredKind,
			AppendedAt: command.SubmittedAt,
			Payload:    payload,
		}); err != nil {
			return err
		}
	}
	return nil
}

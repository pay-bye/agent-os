package kernel

import (
	"context"
	"database/sql"
	"errors"
	coordination "github.com/pay-bye/agent-os/internal/channel"
	root "github.com/pay-bye/agent-os/internal/kernel"
	"github.com/pay-bye/agent-os/internal/kernel/claiming"
	"github.com/pay-bye/agent-os/internal/workitem"
)

func claim(ctx context.Context, tx *sql.Tx, command claiming.Command) (root.ClaimResult, error) {
	lease, err := newChannel(tx).Dequeue(ctx, command.Channel, command.Lease)
	if errors.Is(err, coordination.ErrEmpty) {
		return root.ClaimResult{Empty: true}, nil
	}
	if err != nil {
		return root.ClaimResult{}, err
	}
	payload, err := workItemPayload(ctx, tx, lease.WorkItem())
	if err != nil {
		return root.ClaimResult{}, err
	}
	return root.ClaimResult{
		Lease:    lease,
		Payload:  payload,
		WorkItem: lease.WorkItem(),
	}, nil
}

func workItemPayload(ctx context.Context, tx *sql.Tx, id workitem.ID) ([]byte, error) {
	var payload []byte
	err := tx.QueryRowContext(ctx, `SELECT payload FROM work_items WHERE id = $1`, id.String()).Scan(&payload)
	if err != nil {
		return nil, err
	}
	return append([]byte(nil), payload...), nil
}

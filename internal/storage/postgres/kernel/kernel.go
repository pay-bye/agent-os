package kernel

import (
	"context"
	"database/sql"
	root "github.com/pay-bye/agent-os/internal/kernel"
	"github.com/pay-bye/agent-os/internal/kernel/claiming"
	"github.com/pay-bye/agent-os/internal/kernel/instructions"
	"github.com/pay-bye/agent-os/internal/kernel/leases"
	"github.com/pay-bye/agent-os/internal/kernel/pause"
	"github.com/pay-bye/agent-os/internal/kernel/resolution"
	"github.com/pay-bye/agent-os/internal/kernel/submission"
	channelstore "github.com/pay-bye/agent-os/internal/storage/postgres/channel"
)

type Option func(*Store)

type Store struct {
	db         *sql.DB
	searchPath string
}

func (k *Store) Submit(ctx context.Context, command submission.Command) (root.SubmitResult, error) {
	tx, err := k.begin(ctx)
	if err != nil {
		return root.SubmitResult{}, err
	}
	result, err := submit(ctx, tx, command)
	return result, finish(tx, err)
}

func (k *Store) begin(ctx context.Context) (*sql.Tx, error) {
	tx, err := k.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	if err := setSearchPath(ctx, tx, k.searchPath); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	return tx, nil
}

func (k *Store) Claim(ctx context.Context, command claiming.Command) (root.ClaimResult, error) {
	tx, err := k.begin(ctx)
	if err != nil {
		return root.ClaimResult{}, err
	}
	result, err := claim(ctx, tx, command)
	return result, finish(tx, err)
}

func (k *Store) Ack(ctx context.Context, command resolution.Command) (root.ResolutionResult, error) {
	return k.resolve(ctx, command, ackedResolution)
}

func (k *Store) resolve(
	ctx context.Context,
	command resolution.Command,
	event resolutionEvent,
) (root.ResolutionResult, error) {
	tx, err := k.begin(ctx)
	if err != nil {
		return root.ResolutionResult{}, err
	}
	result, err := resolve(ctx, tx, command, event)
	return result, finish(tx, err)
}

func (k *Store) Nack(ctx context.Context, command resolution.Command) (root.ResolutionResult, error) {
	return k.resolve(ctx, command, nackedResolution)
}

func (k *Store) Extend(ctx context.Context, command leases.ExtendCommand) (root.LeaseResult, error) {
	tx, err := k.begin(ctx)
	if err != nil {
		return root.LeaseResult{}, err
	}
	lease, err := channelstore.New(tx).Extend(
		ctx,
		command.Lease,
		command.TokenDigest,
		command.CheckedAt,
		command.ExpiresAt,
	)
	return root.LeaseResult{Lease: lease}, finish(tx, err)
}

func (k *Store) Heartbeat(ctx context.Context, command leases.HeartbeatCommand) (root.LeaseResult, error) {
	tx, err := k.begin(ctx)
	if err != nil {
		return root.LeaseResult{}, err
	}
	lease, err := channelstore.New(tx).Heartbeat(ctx, command.Lease, command.TokenDigest, command.CheckedAt)
	return root.LeaseResult{Lease: lease}, finish(tx, err)
}

func (k *Store) Pause(ctx context.Context, command pause.Command) (root.PauseResult, error) {
	tx, err := k.begin(ctx)
	if err != nil {
		return root.PauseResult{}, err
	}
	result, err := pauseCommand(ctx, tx, command)
	return result, finish(tx, err)
}

func (k *Store) PauseInstruction(
	ctx context.Context,
	command instructions.PauseCommand,
) (root.InstructionResult, error) {
	return k.instruction(ctx, command.Record, func(tx *sql.Tx) (root.InstructionResult, error) {
		return applyPauseInstruction(ctx, tx, command)
	})
}

func (k *Store) instruction(
	ctx context.Context,
	record instructions.Record,
	apply func(*sql.Tx) (root.InstructionResult, error),
) (root.InstructionResult, error) {
	tx, err := k.begin(ctx)
	if err != nil {
		return root.InstructionResult{}, err
	}
	replay, found, err := reserveInstruction(ctx, tx, record)
	if err != nil || found {
		return replay, finish(tx, err)
	}
	result, err := apply(tx)
	if err != nil {
		return root.InstructionResult{}, finish(tx, err)
	}
	err = finishInstruction(ctx, tx, result)
	return result, finish(tx, err)
}

func (k *Store) ReleaseExpiredLeaseInstruction(
	ctx context.Context,
	command instructions.LeaseCommand,
) (root.InstructionResult, error) {
	return k.instruction(ctx, command.Record, func(tx *sql.Tx) (root.InstructionResult, error) {
		return applyReleaseExpiredLeaseInstruction(ctx, tx, command)
	})
}

func (k *Store) ForceReleaseLeaseInstruction(
	ctx context.Context,
	command instructions.LeaseCommand,
) (root.InstructionResult, error) {
	return k.instruction(ctx, command.Record, func(tx *sql.Tx) (root.InstructionResult, error) {
		return applyForceReleaseLeaseInstruction(ctx, tx, command)
	})
}

func (k *Store) MoveItemInstruction(
	ctx context.Context,
	command instructions.MoveItemCommand,
) (root.InstructionResult, error) {
	return k.instruction(ctx, command.Record, func(tx *sql.Tx) (root.InstructionResult, error) {
		return applyMoveItemInstruction(ctx, tx, command)
	})
}

func (k *Store) MoveEntriesInstruction(
	ctx context.Context,
	command instructions.MoveEntriesCommand,
) (root.InstructionResult, error) {
	return k.instruction(ctx, command.Record, func(tx *sql.Tx) (root.InstructionResult, error) {
		return applyMoveEntriesInstruction(ctx, tx, command)
	})
}

func (k *Store) MoveAvailableInstruction(
	ctx context.Context,
	command instructions.MoveAvailableCommand,
) (root.InstructionResult, error) {
	return k.instruction(ctx, command.Record, func(tx *sql.Tx) (root.InstructionResult, error) {
		return applyMoveAvailableInstruction(ctx, tx, command)
	})
}

func (k *Store) DropInstruction(
	ctx context.Context,
	command instructions.ItemsCommand,
) (root.InstructionResult, error) {
	return k.instruction(ctx, command.Record, func(tx *sql.Tx) (root.InstructionResult, error) {
		return applyDropInstruction(ctx, tx, command)
	})
}

func (k *Store) RouteOutstandingInstruction(
	ctx context.Context,
	command instructions.ItemsCommand,
) (root.InstructionResult, error) {
	return k.instruction(ctx, command.Record, func(tx *sql.Tx) (root.InstructionResult, error) {
		return applyRouteOutstandingInstruction(ctx, tx, command)
	})
}

func New(db *sql.DB, options ...Option) *Store {
	store := &Store{db: db}
	for _, option := range options {
		option(store)
	}
	return store
}

func WithSearchPath(path string) Option {
	return func(store *Store) {
		store.searchPath = path
	}
}

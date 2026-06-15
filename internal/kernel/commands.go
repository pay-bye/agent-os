package kernel

import (
	"context"
	"github.com/pay-bye/agent-os/internal/channel"
	"github.com/pay-bye/agent-os/internal/kernel/claiming"
	"github.com/pay-bye/agent-os/internal/kernel/instructions"
	"github.com/pay-bye/agent-os/internal/kernel/leases"
	"github.com/pay-bye/agent-os/internal/kernel/pause"
	"github.com/pay-bye/agent-os/internal/kernel/resolution"
	"github.com/pay-bye/agent-os/internal/kernel/routing"
	"github.com/pay-bye/agent-os/internal/kernel/submission"
	"github.com/pay-bye/agent-os/internal/registry"
	"github.com/pay-bye/agent-os/internal/workitem"
	"time"
)

const (
	InstructionApplied            = instructions.Applied
	InstructionPreconditionFailed = instructions.PreconditionFailed
)

var (
	ErrInvalidLeaseDuration      = claiming.ErrInvalidLeaseDuration
	ErrUnsafePause               = pause.ErrUnsafe
	ErrEmptyInstructionID        = instructions.ErrEmptyID
	ErrInstructionConflict       = instructions.ErrConflict
	ErrInstructionSelectionEmpty = instructions.ErrSelectionEmpty
	ErrInstructionSelectionLimit = instructions.ErrSelectionLimit
	ErrInstructionDuplicateID    = instructions.ErrDuplicateID
	ErrInstructionLimit          = instructions.ErrLimit
	ErrRouteTargetAbsent         = routing.ErrRouteTargetAbsent
	ErrRouteTargetIncapable      = routing.ErrRouteTargetIncapable
	ErrRouteTargetExcluded       = routing.ErrRouteTargetExcluded
)

type Store interface {
	Submit(context.Context, submission.Command) (SubmitResult, error)
	Claim(context.Context, claiming.Command) (ClaimResult, error)
	Ack(context.Context, resolution.Command) (ResolutionResult, error)
	Nack(context.Context, resolution.Command) (ResolutionResult, error)
	Extend(context.Context, leases.ExtendCommand) (LeaseResult, error)
	Heartbeat(context.Context, leases.HeartbeatCommand) (LeaseResult, error)
	Pause(context.Context, pause.Command) (PauseResult, error)
	PauseInstruction(context.Context, instructions.PauseCommand) (InstructionResult, error)
	ReleaseExpiredLeaseInstruction(context.Context, instructions.LeaseCommand) (InstructionResult, error)
	ForceReleaseLeaseInstruction(context.Context, instructions.LeaseCommand) (InstructionResult, error)
	MoveItemInstruction(context.Context, instructions.MoveItemCommand) (InstructionResult, error)
	MoveEntriesInstruction(context.Context, instructions.MoveEntriesCommand) (InstructionResult, error)
	MoveAvailableInstruction(context.Context, instructions.MoveAvailableCommand) (InstructionResult, error)
	DropInstruction(context.Context, instructions.ItemsCommand) (InstructionResult, error)
	RouteOutstandingInstruction(context.Context, instructions.ItemsCommand) (InstructionResult, error)
}

type Clock interface {
	Now() time.Time
}

type IDGenerator interface {
	Next() string
}

type TokenGenerator interface {
	Next() (channel.Token, error)
}

type SubmitInput struct {
	ID            workitem.ID
	Kind          registry.ItemKindKey
	Payload       []byte
	DeclaredNeeds []workitem.DeclaredNeedInput
}

type SubmitResult struct {
	WorkItem workitem.ID
	Routed   bool
	Channel  registry.ChannelKey
}

type ClaimInput struct {
	Channel       registry.ChannelKey
	Lease         channel.LeaseID
	LeaseDuration time.Duration
}

type ClaimResult struct {
	Empty    bool
	Lease    channel.Lease
	Token    channel.Token
	Payload  []byte
	WorkItem workitem.ID
}

type ResolutionInput struct {
	Lease          channel.LeaseID
	Token          channel.Token
	DeclaredNeeds  []workitem.DeclaredNeedInput
	FailurePayload []byte
}

type ResolutionResult struct {
	Resolved bool
	Routed   bool
	Channel  registry.ChannelKey
}

type ExtendInput struct {
	Lease     channel.LeaseID
	Token     channel.Token
	ExpiresAt time.Time
}

type HeartbeatInput struct {
	Lease channel.LeaseID
	Token channel.Token
}

type LeaseResult struct {
	Lease channel.Lease
}

type PauseInput struct {
	Node registry.NodeKey
}

type PauseResult struct {
	Paused bool
}

type InstructionID = instructions.ID
type InstructionResultValue = instructions.ResultValue
type InstructionRecord = instructions.Record
type InstructionResult = instructions.Result
type PauseInstructionInput = instructions.PauseInput
type LeaseInstructionInput = instructions.LeaseInput
type MoveItemInstructionInput = instructions.MoveItemInput
type MoveEntriesInstructionInput = instructions.MoveEntriesInput
type MoveAvailableInstructionInput = instructions.MoveAvailableInput
type ItemsInstructionInput = instructions.ItemsInput

type Commands struct {
	store  Store
	clock  Clock
	ids    IDGenerator
	tokens TokenGenerator
}

func NewCommands(store Store, clock Clock, ids IDGenerator) Commands {
	return newCommandsWithTokenGenerator(store, clock, ids, secureTokens{})
}

func (c Commands) Submit(ctx context.Context, input SubmitInput) (SubmitResult, error) {
	command, err := submission.New(submission.Input{
		ID:            input.ID,
		Kind:          input.Kind,
		Payload:       input.Payload,
		DeclaredNeeds: input.DeclaredNeeds,
	}, c.clock.Now(), c.ids)
	if err != nil {
		return SubmitResult{}, err
	}
	return c.store.Submit(ctx, command)
}

func (c Commands) Claim(ctx context.Context, input ClaimInput) (ClaimResult, error) {
	command, token, err := claiming.New(claiming.Input{
		Channel:       input.Channel,
		Lease:         input.Lease,
		LeaseDuration: input.LeaseDuration,
	}, c.clock.Now(), c.tokens)
	if err != nil {
		return ClaimResult{}, err
	}
	result, err := c.store.Claim(ctx, command)
	if err != nil || result.Empty {
		return result, err
	}
	result.Token = token
	return result, nil
}

func (c Commands) Ack(ctx context.Context, input ResolutionInput) (ResolutionResult, error) {
	command, err := resolution.New(resolutionInput(input), c.clock.Now(), c.ids)
	if err != nil {
		return ResolutionResult{}, err
	}
	return c.store.Ack(ctx, command)
}

func (c Commands) Nack(ctx context.Context, input ResolutionInput) (ResolutionResult, error) {
	command, err := resolution.New(resolutionInput(input), c.clock.Now(), c.ids)
	if err != nil {
		return ResolutionResult{}, err
	}
	return c.store.Nack(ctx, command)
}

func (c Commands) Extend(ctx context.Context, input ExtendInput) (LeaseResult, error) {
	command, err := leases.Extend(leases.ExtendInput{
		Lease:     input.Lease,
		Token:     input.Token,
		ExpiresAt: input.ExpiresAt,
	}, c.clock.Now())
	if err != nil {
		return LeaseResult{}, err
	}
	return c.store.Extend(ctx, command)
}

func (c Commands) Heartbeat(ctx context.Context, input HeartbeatInput) (LeaseResult, error) {
	command, err := leases.Heartbeat(leases.HeartbeatInput{
		Lease: input.Lease,
		Token: input.Token,
	}, c.clock.Now())
	if err != nil {
		return LeaseResult{}, err
	}
	return c.store.Heartbeat(ctx, command)
}

func (c Commands) Pause(ctx context.Context, input PauseInput) (PauseResult, error) {
	command := pause.New(pause.Input{Node: input.Node}, c.clock.Now(), c.ids)
	return c.store.Pause(ctx, command)
}

func (c Commands) PauseInstruction(ctx context.Context, input PauseInstructionInput) (InstructionResult, error) {
	command, err := instructions.Pause(input, c.clock, c.ids)
	if err != nil {
		return InstructionResult{}, err
	}
	return c.store.PauseInstruction(ctx, command)
}

func (c Commands) ReleaseExpiredLeaseInstruction(
	ctx context.Context,
	input LeaseInstructionInput,
) (InstructionResult, error) {
	command, err := instructions.ReleaseExpiredLease(input, c.clock, c.ids)
	if err != nil {
		return InstructionResult{}, err
	}
	return c.store.ReleaseExpiredLeaseInstruction(ctx, command)
}

func (c Commands) ForceReleaseLeaseInstruction(
	ctx context.Context,
	input LeaseInstructionInput,
) (InstructionResult, error) {
	command, err := instructions.ForceReleaseLease(input, c.clock, c.ids)
	if err != nil {
		return InstructionResult{}, err
	}
	return c.store.ForceReleaseLeaseInstruction(ctx, command)
}

func (c Commands) MoveItemInstruction(ctx context.Context, input MoveItemInstructionInput) (InstructionResult, error) {
	command, err := instructions.MoveItem(input, c.clock, c.ids)
	if err != nil {
		return InstructionResult{}, err
	}
	return c.store.MoveItemInstruction(ctx, command)
}

func (c Commands) MoveEntriesInstruction(
	ctx context.Context,
	input MoveEntriesInstructionInput,
) (InstructionResult, error) {
	command, err := instructions.MoveEntries(input, c.clock, c.ids)
	if err != nil {
		return InstructionResult{}, err
	}
	return c.store.MoveEntriesInstruction(ctx, command)
}

func (c Commands) MoveAvailableInstruction(
	ctx context.Context,
	input MoveAvailableInstructionInput,
) (InstructionResult, error) {
	command, err := instructions.MoveAvailable(input, c.clock, c.ids)
	if err != nil {
		return InstructionResult{}, err
	}
	return c.store.MoveAvailableInstruction(ctx, command)
}

func (c Commands) DropInstruction(ctx context.Context, input ItemsInstructionInput) (InstructionResult, error) {
	command, err := instructions.Drop(input, c.clock, c.ids)
	if err != nil {
		return InstructionResult{}, err
	}
	return c.store.DropInstruction(ctx, command)
}

func (c Commands) RouteOutstandingInstruction(ctx context.Context, input ItemsInstructionInput) (InstructionResult, error) {
	command, err := instructions.RouteOutstanding(input, c.clock, c.ids)
	if err != nil {
		return InstructionResult{}, err
	}
	return c.store.RouteOutstandingInstruction(ctx, command)
}

func resolutionInput(input ResolutionInput) resolution.Input {
	return resolution.Input{
		Lease:          input.Lease,
		Token:          input.Token,
		DeclaredNeeds:  input.DeclaredNeeds,
		FailurePayload: input.FailurePayload,
	}
}

func newCommandsWithTokenGenerator(
	store Store,
	clock Clock,
	ids IDGenerator,
	tokens TokenGenerator,
) Commands {
	return Commands{store: store, clock: clock, ids: ids, tokens: tokens}
}

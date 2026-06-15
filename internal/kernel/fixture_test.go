package kernel

import (
	"context"
	"github.com/pay-bye/agent-os/internal/channel"
	"github.com/pay-bye/agent-os/internal/kernel/claiming"
	"github.com/pay-bye/agent-os/internal/kernel/instructions"
	"github.com/pay-bye/agent-os/internal/kernel/leases"
	"github.com/pay-bye/agent-os/internal/kernel/pause"
	"github.com/pay-bye/agent-os/internal/kernel/resolution"
	"github.com/pay-bye/agent-os/internal/kernel/submission"
	"time"
)

type recordingStore struct {
	submit    submission.Command
	claim     claiming.Command
	ack       resolution.Command
	nack      resolution.Command
	extend    leases.ExtendCommand
	heartbeat leases.HeartbeatCommand
	pause     pause.Command

	pauseInstruction            instructions.PauseCommand
	releaseExpiredInstruction   instructions.LeaseCommand
	forceReleaseInstruction     instructions.LeaseCommand
	moveItemInstruction         instructions.MoveItemCommand
	moveEntriesInstruction      instructions.MoveEntriesCommand
	moveAvailableInstruction    instructions.MoveAvailableCommand
	dropInstruction             instructions.ItemsCommand
	routeOutstandingInstruction instructions.ItemsCommand

	claimResult       ClaimResult
	instructionResult InstructionResult

	claimCalled bool
}

func (s *recordingStore) Submit(_ context.Context, command submission.Command) (SubmitResult, error) {
	s.submit = command
	return SubmitResult{}, nil
}

func (s *recordingStore) Claim(_ context.Context, command claiming.Command) (ClaimResult, error) {
	s.claimCalled = true
	s.claim = command
	return s.claimResult, nil
}

func (s *recordingStore) Ack(_ context.Context, command resolution.Command) (ResolutionResult, error) {
	s.ack = command
	return ResolutionResult{}, nil
}

func (s *recordingStore) Nack(_ context.Context, command resolution.Command) (ResolutionResult, error) {
	s.nack = command
	return ResolutionResult{}, nil
}

func (s *recordingStore) Extend(_ context.Context, command leases.ExtendCommand) (LeaseResult, error) {
	s.extend = command
	return LeaseResult{}, nil
}

func (s *recordingStore) Heartbeat(_ context.Context, command leases.HeartbeatCommand) (LeaseResult, error) {
	s.heartbeat = command
	return LeaseResult{}, nil
}

func (s *recordingStore) Pause(_ context.Context, command pause.Command) (PauseResult, error) {
	s.pause = command
	return PauseResult{Paused: true}, nil
}

func (s *recordingStore) PauseInstruction(
	_ context.Context,
	command instructions.PauseCommand,
) (InstructionResult, error) {
	s.pauseInstruction = command
	return s.instructionResult, nil
}

func (s *recordingStore) ReleaseExpiredLeaseInstruction(
	_ context.Context,
	command instructions.LeaseCommand,
) (InstructionResult, error) {
	s.releaseExpiredInstruction = command
	return s.instructionResult, nil
}

func (s *recordingStore) ForceReleaseLeaseInstruction(
	_ context.Context,
	command instructions.LeaseCommand,
) (InstructionResult, error) {
	s.forceReleaseInstruction = command
	return s.instructionResult, nil
}

func (s *recordingStore) MoveItemInstruction(
	_ context.Context,
	command instructions.MoveItemCommand,
) (InstructionResult, error) {
	s.moveItemInstruction = command
	return s.instructionResult, nil
}

func (s *recordingStore) MoveEntriesInstruction(
	_ context.Context,
	command instructions.MoveEntriesCommand,
) (InstructionResult, error) {
	s.moveEntriesInstruction = command
	return s.instructionResult, nil
}

func (s *recordingStore) MoveAvailableInstruction(
	_ context.Context,
	command instructions.MoveAvailableCommand,
) (InstructionResult, error) {
	s.moveAvailableInstruction = command
	return s.instructionResult, nil
}

func (s *recordingStore) DropInstruction(
	_ context.Context,
	command instructions.ItemsCommand,
) (InstructionResult, error) {
	s.dropInstruction = command
	return s.instructionResult, nil
}

func (s *recordingStore) RouteOutstandingInstruction(
	_ context.Context,
	command instructions.ItemsCommand,
) (InstructionResult, error) {
	s.routeOutstandingInstruction = command
	return s.instructionResult, nil
}

type fixedClock struct {
	now time.Time
}

func (c fixedClock) Now() time.Time {
	return c.now
}

type sequenceIDs struct {
	values []string
}

func (s *sequenceIDs) Next() string {
	if len(s.values) == 0 {
		return "x35"
	}
	value := s.values[0]
	s.values = s.values[1:]
	return value
}

func instant(offset int) time.Time {
	return time.Date(2026, 5, 18, 12, offset, 0, 0, time.UTC)
}

type fixedTokens struct {
	value channel.Token
	err   error
}

func (t fixedTokens) Next() (channel.Token, error) {
	return t.value, t.err
}

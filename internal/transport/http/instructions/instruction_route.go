package instructions

import (
	"context"
	nethttp "net/http"
	"time"

	"github.com/pay-bye/agent-os/internal/kernel"
	"github.com/pay-bye/agent-os/internal/metrics"
	"github.com/pay-bye/agent-os/internal/processlog"
	"github.com/pay-bye/agent-os/internal/transport/http/codec"
	"github.com/pay-bye/agent-os/internal/transport/http/diagnostics"
	"github.com/pay-bye/agent-os/internal/transport/http/security"
	"github.com/pay-bye/agent-os/internal/transport/http/security/credential"
)

type instructionDecode[T any] func(*nethttp.Request) (T, error)
type instructionCall[T any] func(context.Context, T) (kernel.InstructionResult, error)

type Commands interface {
	PauseInstruction(context.Context, kernel.PauseInstructionInput) (kernel.InstructionResult, error)
	ReleaseExpiredLeaseInstruction(context.Context, kernel.LeaseInstructionInput) (kernel.InstructionResult, error)
	ForceReleaseLeaseInstruction(context.Context, kernel.LeaseInstructionInput) (kernel.InstructionResult, error)
	MoveItemInstruction(context.Context, kernel.MoveItemInstructionInput) (kernel.InstructionResult, error)
	MoveEntriesInstruction(context.Context, kernel.MoveEntriesInstructionInput) (kernel.InstructionResult, error)
	MoveAvailableInstruction(context.Context, kernel.MoveAvailableInstructionInput) (kernel.InstructionResult, error)
	DropInstruction(context.Context, kernel.ItemsInstructionInput) (kernel.InstructionResult, error)
	RouteOutstandingInstruction(context.Context, kernel.ItemsInstructionInput) (kernel.InstructionResult, error)
}

type instructionResponse struct {
	ID                 string   `json:"instruction_id"`
	Result             string   `json:"result"`
	EventIDs           []string `json:"event_ids"`
	AffectedCount      int      `json:"affected_count"`
	AffectedIDs        []string `json:"affected_ids"`
	FailedPrecondition string   `json:"failed_precondition,omitempty"`
}

func Register(
	mux *nethttp.ServeMux,
	settings diagnostics.Settings,
	commands Commands,
	verifier credential.OperatorKeyVerifier,
) {
	registerInstructionRoute(mux, "/operations/instructions/pause", settings, verifier, decodePauseInstruction, commands.PauseInstruction)
	registerInstructionRoute(mux, "/operations/instructions/release-expired-lease", settings, verifier, decodeLeaseInstruction, commands.ReleaseExpiredLeaseInstruction)
	registerInstructionRoute(mux, "/operations/instructions/force-release-lease", settings, verifier, decodeLeaseInstruction, commands.ForceReleaseLeaseInstruction)
	registerInstructionRoute(mux, "/operations/instructions/move-item", settings, verifier, decodeMoveItemInstruction, commands.MoveItemInstruction)
	registerInstructionRoute(mux, "/operations/instructions/move-entries", settings, verifier, decodeMoveEntriesInstruction, commands.MoveEntriesInstruction)
	registerInstructionRoute(mux, "/operations/instructions/move-available", settings, verifier, decodeMoveAvailableInstruction, commands.MoveAvailableInstruction)
	registerInstructionRoute(mux, "/operations/instructions/drop", settings, verifier, decodeItemsInstruction, commands.DropInstruction)
	registerInstructionRoute(mux, "/operations/instructions/route-outstanding", settings, verifier, decodeItemsInstruction, commands.RouteOutstandingInstruction)
}

func registerInstructionRoute[T any](
	mux *nethttp.ServeMux,
	path string,
	settings diagnostics.Settings,
	verifier credential.OperatorKeyVerifier,
	decode instructionDecode[T],
	call instructionCall[T],
) {
	endpoint := instructionEndpoint(settings, decode, call)
	mux.Handle(path, security.RequireOperatorKey(verifier, endpoint, settings.Metrics))
}

func instructionEndpoint[T any](
	settings diagnostics.Settings,
	decode instructionDecode[T],
	call instructionCall[T],
) nethttp.HandlerFunc {
	return func(response nethttp.ResponseWriter, request *nethttp.Request) {
		start := time.Now()
		correlation := processlog.Correlation()
		diagnostics.Record(settings.Recorder, processlog.HTTPAccepted(correlation))
		if request.Method != nethttp.MethodPost || !codec.JSONMediaType(request.Header.Get("Content-Type")) {
			rejectInstruction(settings, start, correlation, codec.ErrInvalidInput)
			codec.WriteError(response, codec.ErrInvalidInput)
			return
		}
		input, err := decode(request)
		if err != nil {
			rejectInstruction(settings, start, correlation, err)
			codec.WriteError(response, err)
			return
		}
		result, err := call(request.Context(), input)
		if err != nil {
			failInstruction(settings, start, correlation, err)
			codec.WriteError(response, err)
			return
		}
		settings.Metrics.ObserveRequest(metrics.Instruction, metrics.Completed, time.Since(start))
		diagnostics.Record(settings.Recorder, processlog.KernelCommandWithCorrelation(correlation, processlog.Instruction, processlog.Succeeded, ""))
		diagnostics.Record(settings.Recorder, processlog.HTTPCompleted(correlation))
		codec.WriteBody(response, codec.WithCode(instructionStatus(result), newInstructionResponse(result)))
	}
}

func rejectInstruction(settings diagnostics.Settings, start time.Time, correlation string, err error) {
	settings.Metrics.ObserveRequest(metrics.Instruction, metrics.Rejected, time.Since(start))
	diagnostics.Record(settings.Recorder, processlog.HTTPRejected(correlation, codec.DiagnosticCode(err)))
}

func failInstruction(settings diagnostics.Settings, start time.Time, correlation string, err error) {
	settings.Metrics.ObserveRequest(metrics.Instruction, metrics.Failed, time.Since(start))
	diagnostics.Record(settings.Recorder, processlog.KernelCommandWithCorrelation(correlation, processlog.Instruction, processlog.Failed, codec.DiagnosticCode(err)))
	diagnostics.Record(settings.Recorder, processlog.HTTPFailed(correlation, codec.DiagnosticCode(err)))
}

func instructionStatus(result kernel.InstructionResult) int {
	if result.Result == kernel.InstructionPreconditionFailed {
		return nethttp.StatusConflict
	}
	return nethttp.StatusOK
}

func newInstructionResponse(result kernel.InstructionResult) instructionResponse {
	affected := append([]string(nil), result.AffectedIDs...)
	events := append([]string(nil), result.EventIDs...)
	if affected == nil {
		affected = []string{}
	}
	if events == nil {
		events = []string{}
	}
	return instructionResponse{
		ID:                 result.ID.String(),
		Result:             string(result.Result),
		EventIDs:           events,
		AffectedCount:      len(affected),
		AffectedIDs:        affected,
		FailedPrecondition: result.FailedPrecondition,
	}
}

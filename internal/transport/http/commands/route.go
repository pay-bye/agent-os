package commands

import (
	"context"
	"errors"
	"github.com/pay-bye/agent-os/internal/kernel"
	"github.com/pay-bye/agent-os/internal/metrics"
	"github.com/pay-bye/agent-os/internal/processlog"
	"github.com/pay-bye/agent-os/internal/registry"
	"github.com/pay-bye/agent-os/internal/transport/http/codec"
	"github.com/pay-bye/agent-os/internal/transport/http/diagnostics"
	nethttp "net/http"
	"time"
)

type Commands interface {
	Submit(context.Context, kernel.SubmitInput) (kernel.SubmitResult, error)
	Claim(context.Context, kernel.ClaimInput) (kernel.ClaimResult, error)
	Ack(context.Context, kernel.ResolutionInput) (kernel.ResolutionResult, error)
	Nack(context.Context, kernel.ResolutionInput) (kernel.ResolutionResult, error)
	Extend(context.Context, kernel.ExtendInput) (kernel.LeaseResult, error)
	Heartbeat(context.Context, kernel.HeartbeatInput) (kernel.LeaseResult, error)
}

type endpointCall func(context.Context, *nethttp.Request) (any, bool, error)

func Register(mux *nethttp.ServeMux, commands Commands, settings diagnostics.Settings) {
	mux.HandleFunc("/submit", submitEndpoint(commands, settings))
	mux.HandleFunc("/claim", claimEndpoint(commands, settings))
	mux.HandleFunc("/ack", ackEndpoint(commands, settings))
	mux.HandleFunc("/nack", nackEndpoint(commands, settings))
	mux.HandleFunc("/extend", extendEndpoint(commands, settings))
	mux.HandleFunc("/heartbeat", heartbeatEndpoint(commands, settings))
}

func submitEndpoint(commands Commands, settings diagnostics.Settings) nethttp.HandlerFunc {
	return commandEndpoint(metrics.Submit, processlog.Submit, settings, func(ctx context.Context, request *nethttp.Request) (any, bool, error) {
		input, err := decodeSubmit(request)
		if err != nil {
			return nil, false, err
		}
		result, err := commands.Submit(ctx, input)
		if err != nil {
			return nil, true, err
		}
		settings.Metrics.ObserveJournalAppend(metrics.ItemSubmitted)
		observeNeeds(settings.Metrics, len(input.DeclaredNeeds))
		observeRouting(settings.Metrics, result.Routed)
		return submitBody(result), true, nil
	})
}

func claimEndpoint(commands Commands, settings diagnostics.Settings) nethttp.HandlerFunc {
	return commandEndpoint(metrics.Claim, processlog.Claim, settings, func(ctx context.Context, request *nethttp.Request) (any, bool, error) {
		input, err := decodeClaim(request)
		if err != nil {
			return nil, false, err
		}
		result, err := commands.Claim(ctx, input)
		if err != nil {
			return nil, true, err
		}
		body, err := claimBody(result)
		return body, true, err
	})
}

func ackEndpoint(commands Commands, settings diagnostics.Settings) nethttp.HandlerFunc {
	return commandEndpoint(metrics.Ack, processlog.Ack, settings, func(ctx context.Context, request *nethttp.Request) (any, bool, error) {
		input, err := decodeAck(request)
		if err != nil {
			return nil, false, err
		}
		result, err := commands.Ack(ctx, input)
		if err != nil {
			return nil, true, err
		}
		settings.Metrics.ObserveJournalAppend(metrics.NeedAcknowledged)
		observeNeeds(settings.Metrics, len(input.DeclaredNeeds))
		observeRouting(settings.Metrics, result.Routed)
		return ackBody(result), true, nil
	})
}

func nackEndpoint(commands Commands, settings diagnostics.Settings) nethttp.HandlerFunc {
	return commandEndpoint(metrics.Nack, processlog.Nack, settings, func(ctx context.Context, request *nethttp.Request) (any, bool, error) {
		input, err := decodeNack(request)
		if err != nil {
			return nil, false, err
		}
		result, err := commands.Nack(ctx, input)
		if err != nil {
			return nil, true, err
		}
		settings.Metrics.ObserveJournalAppend(metrics.NeedRejected)
		observeNeeds(settings.Metrics, len(input.DeclaredNeeds))
		observeRouting(settings.Metrics, result.Routed)
		return nackBody(result), true, nil
	})
}

func extendEndpoint(commands Commands, settings diagnostics.Settings) nethttp.HandlerFunc {
	return commandEndpoint(metrics.Extend, processlog.Extend, settings, func(ctx context.Context, request *nethttp.Request) (any, bool, error) {
		input, err := decodeExtend(request)
		if err != nil {
			return nil, false, err
		}
		result, err := commands.Extend(ctx, input)
		if err != nil {
			return nil, true, err
		}
		return extendBody(result.Lease), true, nil
	})
}

func heartbeatEndpoint(commands Commands, settings diagnostics.Settings) nethttp.HandlerFunc {
	return commandEndpoint(metrics.Heartbeat, processlog.Heartbeat, settings, func(ctx context.Context, request *nethttp.Request) (any, bool, error) {
		input, err := decodeHeartbeat(request)
		if err != nil {
			return nil, false, err
		}
		result, err := commands.Heartbeat(ctx, input)
		if err != nil {
			return nil, true, err
		}
		return heartbeatBody(result.Lease), true, nil
	})
}

func commandEndpoint(
	operation metrics.Operation,
	family processlog.CommandFamily,
	settings diagnostics.Settings,
	call endpointCall,
) nethttp.HandlerFunc {
	return func(response nethttp.ResponseWriter, request *nethttp.Request) {
		start := time.Now()
		correlation := processlog.Correlation()
		diagnostics.Record(settings.Recorder, processlog.HTTPAccepted(correlation))
		if request.Method != nethttp.MethodPost || !codec.JSONMediaType(request.Header.Get("Content-Type")) {
			settings.Metrics.ObserveRequest(operation, metrics.Rejected, time.Since(start))
			diagnostics.Record(settings.Recorder, processlog.HTTPRejected(correlation, processlog.InvalidInput))
			codec.WriteError(response, codec.ErrInvalidInput)
			return
		}
		body, commandReached, err := call(request.Context(), request)
		if err != nil {
			observeRoutingError(settings.Metrics, operation, commandReached, err)
			settings.Metrics.ObserveRequest(operation, commandResult(commandReached), time.Since(start))
			recordFailure(settings.Recorder, correlation, family, commandReached, err)
			codec.WriteError(response, err)
			return
		}
		settings.Metrics.ObserveRequest(operation, metrics.Completed, time.Since(start))
		diagnostics.Record(settings.Recorder, processlog.KernelCommandWithCorrelation(correlation, family, processlog.Succeeded, ""))
		diagnostics.Record(settings.Recorder, processlog.HTTPCompleted(correlation))
		codec.WriteOK(response, body)
	}
}

func commandResult(commandReached bool) metrics.Result {
	if commandReached {
		return metrics.Failed
	}
	return metrics.Rejected
}

func observeNeeds(collector *metrics.Collector, count int) {
	for range count {
		collector.ObserveJournalAppend(metrics.NeedDeclared)
	}
}

func observeRouting(collector *metrics.Collector, routed bool) {
	if routed {
		collector.ObserveRouting(metrics.Routed)
		return
	}
	collector.ObserveRouting(metrics.Unrouted)
}

func observeRoutingError(collector *metrics.Collector, operation metrics.Operation, commandReached bool, err error) {
	if !commandReached || !routesWork(operation) {
		return
	}
	if errors.Is(err, registry.ErrNoRoute) {
		collector.ObserveRouting(metrics.NoRoute)
		return
	}
	collector.ObserveRouting(metrics.FailedOutcome)
}

func routesWork(operation metrics.Operation) bool {
	return operation == metrics.Submit ||
		operation == metrics.Ack ||
		operation == metrics.Nack
}

func recordFailure(
	recorder processlog.Recorder,
	correlation string,
	family processlog.CommandFamily,
	commandReached bool,
	err error,
) {
	code := codec.DiagnosticCode(err)
	if commandReached {
		diagnostics.Record(recorder, processlog.KernelCommandWithCorrelation(correlation, family, processlog.Failed, code))
		diagnostics.Record(recorder, processlog.HTTPFailed(correlation, code))
		return
	}
	diagnostics.Record(recorder, processlog.HTTPRejected(correlation, code))
}

package diagnostics

import (
	"github.com/pay-bye/agent-os/internal/metrics"
	"github.com/pay-bye/agent-os/internal/processlog"
	"github.com/pay-bye/agent-os/internal/transport/http/codec"
	nethttp "net/http"
	"time"
)

type Settings struct {
	Recorder processlog.Recorder
	Metrics  *metrics.Collector
}

type RequestCall func(*nethttp.Request) (codec.Response, error)
type RequestGuard func(*nethttp.Request) error

func RequestEndpoint(
	operation metrics.Operation,
	method string,
	settings Settings,
	call RequestCall,
	guards ...RequestGuard,
) nethttp.HandlerFunc {
	return func(response nethttp.ResponseWriter, request *nethttp.Request) {
		start := time.Now()
		correlation := processlog.Correlation()
		Record(settings.Recorder, processlog.HTTPAccepted(correlation))
		if request.Method != method {
			settings.Metrics.ObserveRequest(operation, metrics.Rejected, time.Since(start))
			Record(settings.Recorder, processlog.HTTPRejected(correlation, processlog.InvalidInput))
			codec.WriteError(response, codec.ErrInvalidInput)
			return
		}
		if err := validate(request, guards); err != nil {
			settings.Metrics.ObserveRequest(operation, metrics.Rejected, time.Since(start))
			Record(settings.Recorder, processlog.HTTPRejected(correlation, codec.DiagnosticCode(err)))
			codec.WriteError(response, err)
			return
		}
		result, err := call(request)
		if err != nil {
			code := codec.DiagnosticCode(err)
			settings.Metrics.ObserveRequest(operation, metrics.Failed, time.Since(start))
			Record(settings.Recorder, processlog.HTTPFailed(correlation, code))
			codec.WriteError(response, err)
			return
		}
		settings.Metrics.ObserveRequest(operation, requestResult(operation, result), time.Since(start))
		Record(settings.Recorder, processlog.HTTPCompleted(correlation))
		codec.WriteBody(response, result)
	}
}

func RejectBody(request *nethttp.Request) error {
	if hasBody(request) {
		return codec.ErrInvalidInput
	}
	return nil
}

func Record(recorder processlog.Recorder, record processlog.Record) {
	if recorder != nil {
		recorder.Record(record)
	}
}

func validate(request *nethttp.Request, guards []RequestGuard) error {
	for _, guard := range guards {
		if err := guard(request); err != nil {
			return err
		}
	}
	return nil
}

func hasBody(request *nethttp.Request) bool {
	return request.Body != nil && request.Body != nethttp.NoBody && request.ContentLength != 0
}

func requestResult(operation metrics.Operation, result codec.Response) metrics.Result {
	if operation == metrics.Readyz && result.StatusCode() == nethttp.StatusServiceUnavailable {
		return metrics.NotReady
	}
	return metrics.Completed
}

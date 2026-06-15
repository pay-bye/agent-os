package probes

import (
	"github.com/pay-bye/agent-os/internal/metrics"
	"github.com/pay-bye/agent-os/internal/transport/http/codec"
	"github.com/pay-bye/agent-os/internal/transport/http/diagnostics"
	nethttp "net/http"
)

func Register(mux *nethttp.ServeMux, settings diagnostics.Settings, readiness ReadinessFunc) {
	mux.HandleFunc("/health", healthEndpoint(settings))
	mux.HandleFunc("/readyz", readyzEndpoint(settings, readiness))
}

func healthEndpoint(settings diagnostics.Settings) nethttp.HandlerFunc {
	return diagnostics.RequestEndpoint(metrics.Health, nethttp.MethodGet, settings, func(*nethttp.Request) (codec.Response, error) {
		return codec.OK(healthBody()), nil
	}, diagnostics.RejectBody)
}

func readyzEndpoint(settings diagnostics.Settings, readiness ReadinessFunc) nethttp.HandlerFunc {
	return diagnostics.RequestEndpoint(metrics.Readyz, nethttp.MethodGet, settings, func(request *nethttp.Request) (codec.Response, error) {
		body := readyzBody(readiness(request.Context()))
		return codec.WithCode(readyzCode(body.Result), body), nil
	}, diagnostics.RejectBody)
}

func readyzCode(result Result) int {
	if result == Ready {
		return nethttp.StatusOK
	}
	return nethttp.StatusServiceUnavailable
}

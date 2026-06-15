package scrape

import (
	"github.com/pay-bye/agent-os/internal/metrics"
	"github.com/pay-bye/agent-os/internal/transport/http/codec"
	"github.com/pay-bye/agent-os/internal/transport/http/diagnostics"
	nethttp "net/http"
)

func Register(mux *nethttp.ServeMux, settings diagnostics.Settings) {
	mux.HandleFunc("/metrics", endpoint(settings))
}

func endpoint(settings diagnostics.Settings) nethttp.HandlerFunc {
	return diagnostics.RequestEndpoint(metrics.Scrape, nethttp.MethodGet, settings, func(request *nethttp.Request) (codec.Response, error) {
		return codec.TextOK(settings.Metrics.Text(request.Context())), nil
	}, diagnostics.RejectBody)
}

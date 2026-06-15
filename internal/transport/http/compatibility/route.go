package compatibility

import (
	"github.com/pay-bye/agent-os/internal/metrics"
	"github.com/pay-bye/agent-os/internal/transport/http/codec"
	"github.com/pay-bye/agent-os/internal/transport/http/diagnostics"
	nethttp "net/http"
)

func Register(mux *nethttp.ServeMux, settings diagnostics.Settings) {
	mux.HandleFunc("/compatibility", endpoint(settings))
}

func endpoint(settings diagnostics.Settings) nethttp.HandlerFunc {
	return diagnostics.RequestEndpoint(metrics.Compatibility, nethttp.MethodGet, settings, func(*nethttp.Request) (codec.Response, error) {
		return codec.OK(body()), nil
	})
}

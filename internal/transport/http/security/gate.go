package security

import (
	nethttp "net/http"
	"strings"

	"github.com/pay-bye/agent-os/internal/metrics"
	"github.com/pay-bye/agent-os/internal/processlog"
	"github.com/pay-bye/agent-os/internal/transport/http/diagnostics"
	"github.com/pay-bye/agent-os/internal/transport/http/security/credential"
)

const bearerPrefix = "Bearer "
const operatorKeyHeader = "Operator-Key"

func RequireCredential(
	verifier credential.Verifier,
	next nethttp.Handler,
	recorder processlog.Recorder,
	collector *metrics.Collector,
) nethttp.Handler {
	return nethttp.HandlerFunc(func(response nethttp.ResponseWriter, request *nethttp.Request) {
		credential, ok := presentedCredential(request)
		if !ok || !verifier.Accepts(credential) {
			observeAuthRejection(collector)
			diagnostics.Record(recorder, processlog.AuthRejectedRecord(processlog.Correlation()))
			writeUnauthorized(response)
			return
		}
		next.ServeHTTP(response, request)
	})
}

func RequireOperatorKey(
	verifier credential.OperatorKeyVerifier,
	next nethttp.Handler,
	collector *metrics.Collector,
) nethttp.Handler {
	return nethttp.HandlerFunc(func(response nethttp.ResponseWriter, request *nethttp.Request) {
		key, ok := presentedOperatorKey(request)
		if !ok || verifier == nil || !verifier.Accepts(key) {
			observeAuthRejection(collector)
			writeUnauthorized(response)
			return
		}
		next.ServeHTTP(response, request)
	})
}

func presentedCredential(request *nethttp.Request) (string, bool) {
	values := request.Header.Values("Authorization")
	if len(values) != 1 || !strings.HasPrefix(values[0], bearerPrefix) {
		return "", false
	}
	credential := strings.TrimPrefix(values[0], bearerPrefix)
	if credential == "" || strings.ContainsAny(credential, " \t\r\n") {
		return "", false
	}
	return credential, true
}

func presentedOperatorKey(request *nethttp.Request) (string, bool) {
	values := request.Header.Values(operatorKeyHeader)
	if len(values) != 1 {
		return "", false
	}
	value := values[0]
	if value == "" || strings.ContainsAny(value, " \t\r\n") {
		return "", false
	}
	return value, true
}

func observeAuthRejection(collector *metrics.Collector) {
	if collector != nil {
		collector.ObserveAuthRejection()
	}
}

func writeUnauthorized(response nethttp.ResponseWriter) {
	nethttp.Error(response, "Unauthorized", nethttp.StatusUnauthorized)
}

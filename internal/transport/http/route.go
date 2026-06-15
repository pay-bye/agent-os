package http

import (
	"github.com/pay-bye/agent-os/internal/metrics"
	"github.com/pay-bye/agent-os/internal/processlog"
	work "github.com/pay-bye/agent-os/internal/transport/http/commands"
	"github.com/pay-bye/agent-os/internal/transport/http/compatibility"
	"github.com/pay-bye/agent-os/internal/transport/http/diagnostics"
	"github.com/pay-bye/agent-os/internal/transport/http/instructions"
	"github.com/pay-bye/agent-os/internal/transport/http/operations"
	"github.com/pay-bye/agent-os/internal/transport/http/probes"
	"github.com/pay-bye/agent-os/internal/transport/http/scrape"
	"github.com/pay-bye/agent-os/internal/transport/http/security"
	"github.com/pay-bye/agent-os/internal/transport/http/security/credential"
	nethttp "net/http"
)

type Commands interface {
	work.Commands
	instructions.Commands
}

type Option func(*dependencies)

type dependencies struct {
	settings   diagnostics.Settings
	readiness  probes.ReadinessFunc
	operations operations.Operations
	operator   credential.OperatorKeyVerifier
}

func WithRecorder(recorder processlog.Recorder) Option {
	return func(dependencies *dependencies) {
		dependencies.settings.Recorder = recorder
	}
}

func WithReadiness(readiness probes.ReadinessFunc) Option {
	return func(dependencies *dependencies) {
		if readiness != nil {
			dependencies.readiness = readiness
		}
	}
}

func WithMetrics(collector *metrics.Collector) Option {
	return func(dependencies *dependencies) {
		if collector != nil {
			dependencies.settings.Metrics = collector
		}
	}
}

func WithOperations(reader operations.Operations) Option {
	return func(dependencies *dependencies) {
		if reader != nil {
			dependencies.operations = reader
		}
	}
}

func WithOperatorKey(verifier credential.OperatorKeyVerifier) Option {
	return func(dependencies *dependencies) {
		dependencies.operator = verifier
	}
}

// New returns a credential-gated HTTP handler for the accepted invocation contract routes.
func New(commands Commands, verifier credential.Verifier, options ...Option) (nethttp.Handler, error) {
	if verifier.Empty() {
		return nil, credential.ErrEmptyVerifier
	}
	dependencies := newDependencies(options)
	routes := nethttp.NewServeMux()
	work.Register(routes, commands, dependencies.settings)
	compatibility.Register(routes, dependencies.settings)
	probes.Register(routes, dependencies.settings, dependencies.readiness)
	scrape.Register(routes, dependencies.settings)
	operations.Register(routes, dependencies.settings, dependencies.operations)
	instructions.Register(routes, dependencies.settings, commands, dependencies.operator)
	return security.RequireCredential(verifier, routes, dependencies.settings.Recorder, dependencies.settings.Metrics), nil
}

func newDependencies(options []Option) dependencies {
	dependencies := dependencies{
		settings: diagnostics.Settings{
			Metrics: metrics.New(),
		},
		readiness:  probes.AllNotReady,
		operations: operations.Unavailable(),
	}
	for _, option := range options {
		option(&dependencies)
	}
	return dependencies
}

package main

import (
	"context"
	"database/sql"
	"errors"
	"github.com/pay-bye/agent-os/internal/config"
	"github.com/pay-bye/agent-os/internal/declaration"
	"github.com/pay-bye/agent-os/internal/declaration/execution"
	"github.com/pay-bye/agent-os/internal/kernel"
	"github.com/pay-bye/agent-os/internal/metrics"
	"github.com/pay-bye/agent-os/internal/process/ids"
	"github.com/pay-bye/agent-os/internal/processlog"
	"github.com/pay-bye/agent-os/internal/storage/postgres"
	metricstore "github.com/pay-bye/agent-os/internal/storage/postgres/metrics"
	transport "github.com/pay-bye/agent-os/internal/transport/http"
	"github.com/pay-bye/agent-os/internal/transport/http/probes"
	"github.com/pay-bye/agent-os/internal/transport/http/security/credential"
	"net"
	nethttp "net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

var errDeclarationDrift = errors.New("declaration_drift")

var (
	buildVersion  = "unknown"
	buildRevision = "unknown"
)

type runtime struct {
	config  runtimeConfig
	storage storage
}

func (r runtime) GenerateCredential() (credential.GeneratedCredential, error) {
	return credential.GenerateCredential()
}

func (r runtime) Init(input declaration.InitInput) error {
	return declaration.Init(input)
}

func (r runtime) Preview(ctx context.Context, input execution.Input) (declaration.Delta, error) {
	input.Recorder = fallbackRecorder(input.Recorder, r.storage.recorder)
	return execution.RunPreview(ctx, input)
}

func (r runtime) Apply(ctx context.Context, input execution.Input) (declaration.Delta, error) {
	input.Recorder = fallbackRecorder(input.Recorder, r.storage.recorder)
	return execution.RunApply(ctx, input)
}

func (r runtime) LookupEnv(key string) (string, bool) {
	return r.config.LookupEnv(key)
}

func (r runtime) Storage() execution.Storage {
	return r.storage
}

func (r runtime) Serve(ctx context.Context, input serverInput) error {
	migrationStarted := time.Now()
	db, err := openMigratedDatabase(ctx, input.config.DatabaseURL, input.recorder)
	if err != nil {
		return err
	}
	defer db.Close()
	collector := newCollector(db, migrationStarted)
	document, err := declaration.Read(input.config.Declaration)
	if err != nil {
		return err
	}
	if err := requireSteadyDeclaration(ctx, db, document, collector); err != nil {
		return err
	}
	handler, err := newHandler(db, input.verifier, input.operator, input.recorder, collector, document)
	if err != nil {
		return err
	}
	return serve(ctx, input.config, handler, input.recorder)
}

type serverInput struct {
	config   config.Values
	recorder processlog.Recorder
	verifier credential.Verifier
	operator credential.OperatorKeyVerifier
}

type clock struct{}

func (clock) Now() time.Time {
	return time.Now().UTC()
}

type metricStore struct {
	reader *metricstore.Store
}

func (s metricStore) Read(ctx context.Context, now time.Time) (metrics.Storage, error) {
	aggregates, err := s.reader.Aggregates(ctx, now)
	if err != nil {
		return metrics.Storage{}, err
	}
	return metrics.Storage{
		QueueDepth:    aggregates.AvailableDepth,
		LeasesHeld:    aggregates.LeasesHeld,
		LeasesExpired: aggregates.LeasesExpired,
	}, nil
}

func newRuntime(config runtimeConfig) runtime {
	return runtime{config: config, storage: storage{}}
}

func requireSteadyDeclaration(
	ctx context.Context,
	db *sql.DB,
	document declaration.Document,
	collector *metrics.Collector,
) error {
	store, err := openStore(ctx, db)
	if err != nil {
		return err
	}
	started := time.Now()
	delta, err := execution.Preview(ctx, store, document)
	collector.ObserveDeclaration(metrics.Preview, operationResult(err), time.Since(started))
	if err != nil {
		return err
	}
	if hasDeclarationDrift(delta) {
		return errDeclarationDrift
	}
	return nil
}

func newHandler(
	db *sql.DB,
	verifier credential.Verifier,
	operator credential.OperatorKeyVerifier,
	recorder processlog.Recorder,
	collector *metrics.Collector,
	document declaration.Document,
) (nethttp.Handler, error) {
	commands := kernel.NewCommands(
		postgres.NewKernel(db),
		clock{},
		ids.Random{},
	)
	return transport.New(
		commands,
		verifier,
		transport.WithRecorder(recorder),
		transport.WithReadiness(storageReadiness(db)),
		transport.WithMetrics(collector),
		transport.WithOperations(newOperationsView(db, collector)),
		transport.WithOperatorKey(operator),
	)
}

func newCollector(db *sql.DB, migrationStarted time.Time) *metrics.Collector {
	collector := metrics.New(
		metrics.WithBuild(metrics.Build{Version: buildVersion, Revision: buildRevision}),
		metrics.WithStore(metricStore{reader: metricstore.New(db)}),
	)
	collector.ObserveMigration(metrics.Succeeded, time.Since(migrationStarted))
	return collector
}

func operationResult(err error) metrics.Result {
	if err != nil {
		return metrics.Failed
	}
	return metrics.Succeeded
}

func storageReadiness(db *sql.DB) probes.ReadinessFunc {
	return func(ctx context.Context) probes.Readiness {
		readiness := probes.AllReady()
		if err := db.PingContext(ctx); err != nil {
			readiness.Storage = probes.NotReady
		}
		return readiness
	}
}

func fallbackRecorder(primary processlog.Recorder, fallback processlog.Recorder) processlog.Recorder {
	if primary != nil {
		return primary
	}
	return fallback
}

func serve(
	ctx context.Context,
	config config.Values,
	handler nethttp.Handler,
	recorder processlog.Recorder,
) error {
	listener, err := net.Listen("tcp", config.Listen)
	if err != nil {
		return err
	}
	server := &nethttp.Server{Handler: handler, ReadHeaderTimeout: 5 * time.Second}
	errs := make(chan error, 1)
	record(recorder, processlog.ProcessStarted())
	go func() {
		errs <- server.Serve(listener)
	}()
	select {
	case err := <-errs:
		result := serveError(err)
		recordProcessStop(recorder, result)
		return result
	case <-shutdownSignal(ctx):
		result := shutdown(server, config.Grace)
		recordProcessStop(recorder, result)
		return result
	}
}

func serveError(err error) error {
	if errors.Is(err, nethttp.ErrServerClosed) {
		return nil
	}
	return err
}

func recordProcessStop(recorder processlog.Recorder, err error) {
	record(recorder, processStop(err))
}

func processStop(err error) processlog.Record {
	if err != nil {
		return processlog.ProcessStopped(processlog.Failed, processlog.InternalError)
	}
	return processlog.ProcessStopped(processlog.Completed, "")
}

func shutdownSignal(ctx context.Context) <-chan struct{} {
	done := make(chan struct{})
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		defer signal.Stop(signals)
		select {
		case <-ctx.Done():
		case <-signals:
		}
		close(done)
	}()
	return done
}

func shutdown(server *nethttp.Server, grace time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), grace)
	defer cancel()
	return server.Shutdown(ctx)
}

package operations

import (
	"context"
	nethttp "net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/pay-bye/agent-os/internal/metrics"
	"github.com/pay-bye/agent-os/internal/processlog"
	"github.com/pay-bye/agent-os/internal/transport/http/codec"
	"github.com/pay-bye/agent-os/internal/transport/http/diagnostics"
)

const (
	defaultChannelLimit     = 50
	maxChannelLimit         = 200
	defaultChannelItemLimit = 50
	maxChannelItemLimit     = 100
	defaultJournalLimit     = 50
	maxJournalLimit         = 200
	defaultNodeLimit        = 100
	maxNodeLimit            = 200
)

type operationsReadCall func(context.Context, *nethttp.Request) (any, error)

func Register(mux *nethttp.ServeMux, settings diagnostics.Settings, operations Operations) {
	mux.HandleFunc("/operations", endpoint(settings, operations))
	mux.HandleFunc("/operations/channels", channelsEndpoint(settings, operations))
	mux.HandleFunc("/operations/channels/", channelItemsEndpoint(settings, operations))
	mux.HandleFunc("/operations/items/", itemEndpoint(settings, operations))
	mux.HandleFunc("/operations/nodes", nodesEndpoint(settings, operations))
}

func endpoint(settings diagnostics.Settings, operations Operations) nethttp.HandlerFunc {
	return func(response nethttp.ResponseWriter, request *nethttp.Request) {
		start := time.Now()
		correlation := processlog.Correlation()
		diagnostics.Record(settings.Recorder, processlog.HTTPAccepted(correlation))
		if request.Method != nethttp.MethodGet {
			observeRequest(settings, metrics.Rejected, start)
			diagnostics.Record(settings.Recorder, processlog.HTTPRejected(correlation, processlog.InvalidInput))
			codec.WriteError(response, codec.ErrInvalidInput)
			return
		}
		if err := diagnostics.RejectBody(request); err != nil {
			observeRequest(settings, metrics.Rejected, start)
			diagnostics.Record(settings.Recorder, processlog.HTTPRejected(correlation, processlog.InvalidInput))
			codec.WriteError(response, err)
			return
		}
		body := operations.Response(request.Context())
		observeRequest(settings, reportResult(body), start)
		diagnostics.Record(settings.Recorder, processlog.HTTPCompleted(correlation))
		codec.WriteBody(response, codec.WithCode(code(body), body))
	}
}

func code(body OperationsReport) int {
	if !body.Available() {
		return nethttp.StatusServiceUnavailable
	}
	return nethttp.StatusOK
}

func channelsEndpoint(settings diagnostics.Settings, operations Operations) nethttp.HandlerFunc {
	return operationsReadEndpoint(settings, func(ctx context.Context, request *nethttp.Request) (any, error) {
		query, err := channelQuery(request.URL.Query())
		if err != nil {
			return nil, err
		}
		return operations.Channels(ctx, query)
	})
}

func channelItemsEndpoint(settings diagnostics.Settings, operations Operations) nethttp.HandlerFunc {
	return operationsReadEndpoint(settings, func(ctx context.Context, request *nethttp.Request) (any, error) {
		channel, err := channelFromPath(request.URL.Path)
		if err != nil {
			return nil, err
		}
		query, err := channelItemQuery(channel, request.URL.Query())
		if err != nil {
			return nil, err
		}
		return operations.ChannelItems(ctx, query)
	})
}

func itemEndpoint(settings diagnostics.Settings, operations Operations) nethttp.HandlerFunc {
	return operationsReadEndpoint(settings, func(ctx context.Context, request *nethttp.Request) (any, error) {
		item, journal, err := itemPath(request.URL.Path)
		if err != nil {
			return nil, err
		}
		if journal {
			query, err := itemJournalQuery(item, request.URL.Query())
			if err != nil {
				return nil, err
			}
			return operations.ItemJournal(ctx, query)
		}
		if len(request.URL.Query()) > 0 {
			return nil, codec.ErrInvalidInput
		}
		return operations.Item(ctx, item)
	})
}

func nodesEndpoint(settings diagnostics.Settings, operations Operations) nethttp.HandlerFunc {
	return operationsReadEndpoint(settings, func(ctx context.Context, request *nethttp.Request) (any, error) {
		query, err := nodeQuery(request.URL.Query())
		if err != nil {
			return nil, err
		}
		return operations.Nodes(ctx, query)
	})
}

func operationsReadEndpoint(settings diagnostics.Settings, call operationsReadCall) nethttp.HandlerFunc {
	return func(response nethttp.ResponseWriter, request *nethttp.Request) {
		start := time.Now()
		correlation := processlog.Correlation()
		diagnostics.Record(settings.Recorder, processlog.HTTPAccepted(correlation))
		if request.Method != nethttp.MethodGet {
			observeRequest(settings, metrics.Rejected, start)
			diagnostics.Record(settings.Recorder, processlog.HTTPRejected(correlation, processlog.InvalidInput))
			codec.WriteError(response, codec.ErrInvalidInput)
			return
		}
		if err := diagnostics.RejectBody(request); err != nil {
			observeRequest(settings, metrics.Rejected, start)
			diagnostics.Record(settings.Recorder, processlog.HTTPRejected(correlation, processlog.InvalidInput))
			codec.WriteError(response, err)
			return
		}
		body, err := call(request.Context(), request)
		if err != nil {
			result := readErrorResult(err)
			diagnostic := codec.DiagnosticCode(err)
			observeRequest(settings, result, start)
			if result == metrics.Rejected {
				diagnostics.Record(settings.Recorder, processlog.HTTPRejected(correlation, diagnostic))
			} else {
				diagnostics.Record(settings.Recorder, processlog.HTTPFailed(correlation, diagnostic))
			}
			codec.WriteError(response, err)
			return
		}
		observeRequest(settings, metrics.Completed, start)
		diagnostics.Record(settings.Recorder, processlog.HTTPCompleted(correlation))
		codec.WriteOK(response, body)
	}
}

func observeRequest(settings diagnostics.Settings, result metrics.Result, start time.Time) {
	settings.Metrics.ObserveRequest(metrics.Operations, result, time.Since(start))
}

func reportResult(body OperationsReport) metrics.Result {
	if !body.Available() {
		return metrics.Failed
	}
	return metrics.Completed
}

func readErrorResult(err error) metrics.Result {
	if codec.DiagnosticCode(err) == processlog.InvalidInput {
		return metrics.Rejected
	}
	return metrics.Failed
}

func channelQuery(values url.Values) (ChannelQuery, error) {
	if err := rejectUnknownQuery(values, "limit", "after_channel_key", "older_than_seconds"); err != nil {
		return ChannelQuery{}, err
	}
	limit, err := boundedLimit(values, defaultChannelLimit, maxChannelLimit)
	if err != nil {
		return ChannelQuery{}, err
	}
	olderThan, err := optionalSeconds(values)
	if err != nil {
		return ChannelQuery{}, err
	}
	return ChannelQuery{
		Limit:            limit,
		After:            values.Get("after_channel_key"),
		OlderThanSeconds: olderThan,
	}, nil
}

func channelItemQuery(channel string, values url.Values) (ChannelItemQuery, error) {
	if err := rejectUnknownQuery(values, "limit", "older_than_seconds", "lease_view"); err != nil {
		return ChannelItemQuery{}, err
	}
	limit, err := boundedLimit(values, defaultChannelItemLimit, maxChannelItemLimit)
	if err != nil {
		return ChannelItemQuery{}, err
	}
	olderThan, err := optionalSeconds(values)
	if err != nil {
		return ChannelItemQuery{}, err
	}
	lease, err := leaseView(values)
	if err != nil {
		return ChannelItemQuery{}, err
	}
	return ChannelItemQuery{
		Channel:          channel,
		Limit:            limit,
		OlderThanSeconds: olderThan,
		Lease:            lease,
	}, nil
}

func itemJournalQuery(item string, values url.Values) (ItemJournalQuery, error) {
	if err := rejectUnknownQuery(values, "limit", "after_append_index"); err != nil {
		return ItemJournalQuery{}, err
	}
	limit, err := boundedLimit(values, defaultJournalLimit, maxJournalLimit)
	if err != nil {
		return ItemJournalQuery{}, err
	}
	index, err := optionalIndex(values)
	if err != nil {
		return ItemJournalQuery{}, err
	}
	return ItemJournalQuery{WorkItem: item, Limit: limit, AfterAppendIndex: index}, nil
}

func nodeQuery(values url.Values) (NodeQuery, error) {
	if err := rejectUnknownQuery(values, "limit", "after_node_key", "need_kind"); err != nil {
		return NodeQuery{}, err
	}
	limit, err := boundedLimit(values, defaultNodeLimit, maxNodeLimit)
	if err != nil {
		return NodeQuery{}, err
	}
	return NodeQuery{
		Limit:    limit,
		After:    values.Get("after_node_key"),
		NeedKind: values.Get("need_kind"),
	}, nil
}

func channelFromPath(path string) (string, error) {
	parts := strings.Split(strings.TrimPrefix(path, "/operations/channels/"), "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] != "items" {
		return "", codec.ErrInvalidInput
	}
	return parts[0], nil
}

func itemPath(path string) (string, bool, error) {
	parts := strings.Split(strings.TrimPrefix(path, "/operations/items/"), "/")
	switch {
	case len(parts) == 1 && parts[0] != "":
		return parts[0], false, nil
	case len(parts) == 2 && parts[0] != "" && parts[1] == "journal":
		return parts[0], true, nil
	default:
		return "", false, codec.ErrInvalidInput
	}
}

func rejectUnknownQuery(values url.Values, allowed ...string) error {
	for key := range values {
		if !contains(allowed, key) {
			return codec.ErrInvalidInput
		}
	}
	return nil
}

func boundedLimit(values url.Values, fallback int, maximum int) (int, error) {
	value := values.Get("limit")
	if value == "" {
		return fallback, nil
	}
	limit, err := strconv.Atoi(value)
	if err != nil || limit <= 0 || limit > maximum {
		return 0, codec.ErrInvalidInput
	}
	return limit, nil
}

func optionalSeconds(values url.Values) (int, error) {
	return optionalNonNegativeInt(values.Get("older_than_seconds"))
}

func optionalIndex(values url.Values) (int64, error) {
	value := values.Get("after_append_index")
	if value == "" {
		return 0, nil
	}
	index, err := strconv.ParseInt(value, 10, 64)
	if err != nil || index < 0 {
		return 0, codec.ErrInvalidInput
	}
	return index, nil
}

func optionalNonNegativeInt(value string) (int, error) {
	if value == "" {
		return 0, nil
	}
	number, err := strconv.Atoi(value)
	if err != nil || number < 0 {
		return 0, codec.ErrInvalidInput
	}
	return number, nil
}

func leaseView(values url.Values) (string, error) {
	value := values.Get("lease_view")
	if value == "" {
		return "all", nil
	}
	if contains([]string{"all", "held", "expired", "none"}, value) {
		return value, nil
	}
	return "", codec.ErrInvalidInput
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

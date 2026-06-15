package metrics

import (
	"strconv"
	"time"
)

func requestFamily() family {
	return family{
		name: "requests_total",
		kind: "counter",
		help: "Total credential-gated HTTP requests by bounded operation, result, and protocol.",
	}
}

func requestDurationFamily() family {
	return family{
		name: "request_duration_seconds",
		kind: "histogram",
		help: "Credential-gated HTTP request durations by bounded operation, result, and protocol.",
	}
}

func authFamily() family {
	return family{
		name: "auth_rejections_total",
		kind: "counter",
		help: "Total HTTP credential rejections by bounded family and protocol.",
	}
}

func journalFamily() family {
	return family{
		name: "journal_appends_total",
		kind: "counter",
		help: "Total observed Journal appends by bounded event family.",
	}
}

func routingFamily() family {
	return family{
		name: "routing_results_total",
		kind: "counter",
		help: "Total observed routing results by bounded outcome.",
	}
}

func declarationFamily() family {
	return family{
		name: "declaration_operations_total",
		kind: "counter",
		help: "Total declaration operations by bounded operation and result.",
	}
}

func declarationDurationFamily() family {
	return family{
		name: "declaration_duration_seconds",
		kind: "histogram",
		help: "Declaration operation durations by bounded operation and result.",
	}
}

func migrationFamily() family {
	return family{
		name: "migrations_total",
		kind: "counter",
		help: "Total migration applications by bounded result.",
	}
}

func migrationDurationFamily() family {
	return family{
		name: "migration_duration_seconds",
		kind: "histogram",
		help: "Migration application durations by bounded result.",
	}
}

func processFamily() family {
	return family{
		name: "process_start_time_seconds",
		kind: "gauge",
		help: "Unix timestamp when the process started.",
	}
}

func buildFamily() family {
	return family{
		name: "build_info",
		kind: "gauge",
		help: "Running binary build identity with bounded version and revision labels.",
	}
}

func queueFamily() family {
	return family{
		name: "queue_depth",
		kind: "gauge",
		help: "Available queue entries by bounded channel class.",
	}
}

func heldFamily() family {
	return family{
		name: "leases_held",
		kind: "gauge",
		help: "Unexpired held leases by bounded channel class.",
	}
}

func expiredFamily() family {
	return family{
		name: "leases_expired",
		kind: "gauge",
		help: "Expired lease records by bounded channel class.",
	}
}

func requestSamples(values map[requestKey]int) []sample {
	samples := []sample{}
	for key, value := range values {
		samples = append(samples, sample{
			name: "requests_total",
			labels: []label{
				{key: "operation", value: string(key.operation)},
				{key: "result", value: string(key.result)},
				{key: "protocol", value: "http"},
			},
			value: value,
		})
	}
	sortSamples(samples)
	return samples
}

func requestDurationSamples(values map[requestDurationKey]int) []sample {
	samples := []sample{}
	for key, value := range values {
		samples = append(samples, sample{
			name: "request_duration_seconds",
			labels: []label{
				{key: "operation", value: string(key.operation)},
				{key: "result", value: string(key.result)},
				{key: "protocol", value: "http"},
				{key: "le", value: key.bucket},
			},
			value: value,
		})
	}
	sortSamples(samples)
	return samples
}

func authSamples(value int) []sample {
	if value == 0 {
		return nil
	}
	return []sample{{
		name: "auth_rejections_total",
		labels: []label{
			{key: "family", value: "unauthorized"},
			{key: "protocol", value: "http"},
		},
		value: value,
	}}
}

func storageFamilies(storage *Storage) []gaugeFamily {
	if storage == nil {
		return nil
	}
	return []gaugeFamily{
		storageGauge(queueFamily(), "queue_depth", storage.QueueDepth),
		storageGauge(heldFamily(), "leases_held", storage.LeasesHeld),
		storageGauge(expiredFamily(), "leases_expired", storage.LeasesExpired),
	}
}

func storageGauge(metadata family, name string, value int) gaugeFamily {
	return gaugeFamily{
		metadata: metadata,
		samples: []gaugeSample{{
			name:   name,
			labels: []label{{key: "channel_class", value: "all"}},
			value:  strconv.Itoa(value),
		}},
	}
}

func journalSamples(values map[EventKind]int) []sample {
	samples := []sample{}
	for key, value := range values {
		samples = append(samples, sample{
			name:   "journal_appends_total",
			labels: []label{{key: "event_kind", value: string(key)}},
			value:  value,
		})
	}
	sortSamples(samples)
	return samples
}

func routingSamples(values map[Outcome]int) []sample {
	samples := []sample{}
	for key, value := range values {
		samples = append(samples, sample{
			name:   "routing_results_total",
			labels: []label{{key: "outcome", value: string(key)}},
			value:  value,
		})
	}
	sortSamples(samples)
	return samples
}

func declarationSamples(values map[declarationKey]int) []sample {
	samples := []sample{}
	for key, value := range values {
		samples = append(samples, sample{
			name: "declaration_operations_total",
			labels: []label{
				{key: "operation", value: string(key.operation)},
				{key: "result", value: string(key.result)},
			},
			value: value,
		})
	}
	sortSamples(samples)
	return samples
}

func declarationDurationSamples(values map[declarationDurationKey]int) []sample {
	samples := []sample{}
	for key, value := range values {
		samples = append(samples, sample{
			name: "declaration_duration_seconds",
			labels: []label{
				{key: "operation", value: string(key.operation)},
				{key: "result", value: string(key.result)},
				{key: "le", value: key.bucket},
			},
			value: value,
		})
	}
	sortSamples(samples)
	return samples
}

func migrationSamples(values map[Result]int) []sample {
	samples := []sample{}
	for key, value := range values {
		samples = append(samples, sample{
			name:   "migrations_total",
			labels: []label{{key: "result", value: string(key)}},
			value:  value,
		})
	}
	sortSamples(samples)
	return samples
}

func migrationDurationSamples(values map[migrationDurationKey]int) []sample {
	samples := []sample{}
	for key, value := range values {
		samples = append(samples, sample{
			name: "migration_duration_seconds",
			labels: []label{
				{key: "result", value: string(key.result)},
				{key: "le", value: key.bucket},
			},
			value: value,
		})
	}
	sortSamples(samples)
	return samples
}

func processSamples(start time.Time) []gaugeFamily {
	return []gaugeFamily{{
		metadata: processFamily(),
		samples: []gaugeSample{{
			name:  "process_start_time_seconds",
			value: strconv.FormatInt(start.Unix(), 10),
		}},
	}}
}

func buildSamples(build Build) []gaugeFamily {
	return []gaugeFamily{{
		metadata: buildFamily(),
		samples: []gaugeSample{{
			name: "build_info",
			labels: []label{
				{key: "version", value: build.Version},
				{key: "revision", value: build.Revision},
			},
			value: "1",
		}},
	}}
}

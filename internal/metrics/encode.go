package metrics

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

type snapshot struct {
	start time.Time
	build Build
	store Store
	now   time.Time

	storage              *Storage
	requests             map[requestKey]int
	requestDurations     map[requestDurationKey]int
	authRejections       int
	journalAppends       map[EventKind]int
	routingResults       map[Outcome]int
	declarations         map[declarationKey]int
	declarationDurations map[declarationDurationKey]int
	migrations           map[Result]int
	migrationDurations   map[migrationDurationKey]int
}

type gaugeFamily struct {
	metadata family
	samples  []gaugeSample
}

type family struct {
	name string
	kind string
	help string
}

type label struct {
	key   string
	value string
}

type sample struct {
	name   string
	labels []label
	value  int
}

type gaugeSample struct {
	name   string
	labels []label
	value  string
}

type bucket struct {
	label string
	value float64
}

func encode(s snapshot) string {
	var output strings.Builder
	writeCounters(&output, requestFamily(), requestSamples(s.requests))
	writeCounters(&output, requestDurationFamily(), requestDurationSamples(s.requestDurations))
	writeCounters(&output, authFamily(), authSamples(s.authRejections))
	writeGauges(&output, storageFamilies(s.storage))
	writeCounters(&output, journalFamily(), journalSamples(s.journalAppends))
	writeCounters(&output, routingFamily(), routingSamples(s.routingResults))
	writeCounters(&output, declarationFamily(), declarationSamples(s.declarations))
	writeCounters(&output, declarationDurationFamily(), declarationDurationSamples(s.declarationDurations))
	writeCounters(&output, migrationFamily(), migrationSamples(s.migrations))
	writeCounters(&output, migrationDurationFamily(), migrationDurationSamples(s.migrationDurations))
	writeGauges(&output, processSamples(s.start))
	writeGauges(&output, buildSamples(s.build))
	return output.String()
}

func withStorage(ctx context.Context, s snapshot) snapshot {
	if s.store == nil {
		return s
	}
	storage, err := s.store.Read(ctx, s.now)
	if err != nil {
		return s
	}
	s.storage = &storage
	return s
}

func writeCounters(output *strings.Builder, metadata family, samples []sample) {
	if len(samples) == 0 {
		return
	}
	writeHeader(output, metadata)
	for _, sample := range samples {
		writeSample(output, gaugeSample{
			name:   sample.name,
			labels: sample.labels,
			value:  strconv.Itoa(sample.value),
		})
	}
}

func writeGauges(output *strings.Builder, groups []gaugeFamily) {
	for _, group := range groups {
		if len(group.samples) == 0 {
			continue
		}
		writeHeader(output, group.metadata)
		for _, sample := range group.samples {
			writeSample(output, sample)
		}
	}
}

func writeHeader(output *strings.Builder, metadata family) {
	fmt.Fprintf(output, "# HELP %s %s\n", metadata.name, metadata.help)
	fmt.Fprintf(output, "# TYPE %s %s\n", metadata.name, metadata.kind)
}

func writeSample(output *strings.Builder, sample gaugeSample) {
	output.WriteString(sample.name)
	if len(sample.labels) > 0 {
		output.WriteString("{")
		output.WriteString(formatLabels(sample.labels))
		output.WriteString("}")
	}
	fmt.Fprintf(output, " %s\n", sample.value)
}

func formatLabels(labels []label) string {
	parts := make([]string, 0, len(labels))
	for _, label := range labels {
		parts = append(parts, fmt.Sprintf(`%s="%s"`, label.key, escape(label.value)))
	}
	return strings.Join(parts, ",")
}

func escape(value string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`)
	return replacer.Replace(value)
}

func buildVersionValue(value string) string {
	if !validVersion(value) {
		return "unknown"
	}
	return value
}

func buildRevisionValue(value string) string {
	if !validRevision(value) {
		return "unknown"
	}
	return value
}

func validVersion(value string) bool {
	rest, ok := strings.CutPrefix(value, "v")
	if !ok {
		return false
	}
	core, suffix, hasSuffix := strings.Cut(rest, "-")
	if !validVersionCore(core) {
		return false
	}
	return !hasSuffix || validVersionSuffix(suffix)
}

func validVersionCore(value string) bool {
	parts := strings.Split(value, ".")
	if len(parts) != 3 {
		return false
	}
	for _, part := range parts {
		if !digits(part) {
			return false
		}
	}
	return true
}

func validVersionSuffix(value string) bool {
	releaseNumber, ok := strings.CutPrefix(value, "rc.")
	if !ok {
		return false
	}
	return digits(releaseNumber)
}

func validRevision(value string) bool {
	if len(value) < 6 || len(value) > 40 {
		return false
	}
	for _, char := range value {
		if !hex(char) {
			return false
		}
	}
	return true
}

func digits(value string) bool {
	if value == "" {
		return false
	}
	for _, char := range value {
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}

func hex(char rune) bool {
	return (char >= '0' && char <= '9') ||
		(char >= 'a' && char <= 'f')
}

func sortSamples(samples []sample) {
	sort.Slice(samples, func(left int, right int) bool {
		return sampleKey(samples[left]) < sampleKey(samples[right])
	})
}

func sampleKey(sample sample) string {
	parts := []string{sample.name}
	for _, label := range sample.labels {
		parts = append(parts, label.key, label.value)
	}
	return strings.Join(parts, "\x00")
}

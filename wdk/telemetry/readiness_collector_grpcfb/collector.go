// Copyright 2026 PolitePixels Limited
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This project stands against fascism, authoritarianism, and all forms of
// oppression. We built this to empower people, not to enable those who would
// strip others of their rights and dignity.

package readiness_collector_grpcfb

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/clock"
	"piko.sh/piko/wdk/telemetry/readiness"
	"piko.sh/piko/wdk/telemetry/telemetry_grpcfb"
)

const (
	// DefaultInterval is the sampling cadence when none is supplied.
	DefaultInterval = 15 * time.Second

	// kindDependency is the MetricPoint.Kind sentinel that marks a readiness-dependency
	// sample, so the sink filters these out of its normal gauge handling.
	kindDependency = "dependency"

	// unitMs is the latency unit every dependency sample carries.
	unitMs = "ms"

	// labelStatus is the label key carrying the normalised health status.
	labelStatus = "status"

	// labelMessage is the label key carrying the dependency's human-readable message.
	labelMessage = "message"

	// labelIcon is the label key carrying the derived icon hint.
	labelIcon = "icon"

	// statusHealthy is the normalised status for a healthy dependency.
	statusHealthy = "healthy"

	// statusDegraded is the normalised status for a degraded dependency.
	statusDegraded = "degraded"

	// statusDown is the normalised status for an unhealthy (or unrecognised) dependency.
	// piko's UNHEALTHY maps here deliberately: the sink keys on healthy/degraded/down.
	statusDown = "down"

	// maxDeps caps the number of dependencies emitted per sample so a pathological tree
	// cannot flood the shared stream. Mirrors the bounding discipline of the sibling metric
	// collector.
	maxDeps = 256

	// maxStringLen bounds the Name and message label bytes per sample so off-box strings
	// stay within the wire frame's per-string budget.
	maxStringLen = 1024

	// defaultIcon is used when a dependency name matches no known icon hint.
	defaultIcon = "box"

	// maxInfoEntriesPerDep caps how many provider info labels one dependency contributes per
	// sample.
	//
	// With the 3 reserved labels (status/message/icon) this keeps a dependency at ~19
	// labels, well under the downstream sink's maxAttrPairs cap, so its order-dependent
	// truncation never severs an info subset we meant to keep. dep.Info is sorted BEFORE
	// this cap, so the kept subset is stable tick to tick.
	maxInfoEntriesPerDep = 16

	// maxInfoValueLen bounds each info label value's bytes (UTF-8 safe) so a verbose
	// provider value cannot blow the wire frame's per-string budget.
	maxInfoValueLen = 256

	// labelInfoPrefix is the key prefix every provider info label carries, namespacing it as
	// "info.<section>.<key>" so the sink can separate provider detail from reserved labels.
	labelInfoPrefix = "info."

	// labelInfoDropped is the label key carrying the number of provider info entries dropped
	// when a dependency's info exceeds maxInfoEntriesPerDep, so the truncation is observable
	// downstream rather than silent.
	labelInfoDropped = "info.dropped"
)

// Runtime samples piko's readiness tree and forwards each dependency as a MetricPoint to
// a telemetry client.
type Runtime struct {
	// client is the telemetry transport that dependency samples are enqueued onto.
	client *telemetry_grpcfb.Client

	// probe is the public readiness seam sampled on each tick.
	probe readiness.Probe

	// clock supplies timestamps and the sampling ticker for deterministic testing.
	clock clock.Clock

	// interval is the cadence between successive samples.
	interval time.Duration
}

// Option configures a Runtime.
type Option func(*Runtime)

// WithClock injects the clock used for timestamps and the sampling ticker, so the loop is
// deterministically testable. Defaults to the real system clock.
//
// Takes c (clock.Clock) which is the clock to drive timestamps and the ticker.
//
// Returns Option which sets the supplied clock on a Runtime.
func WithClock(c clock.Clock) Option {
	return func(r *Runtime) {
		if c != nil {
			r.clock = c
		}
	}
}

// NewReadiness wraps a shared telemetry client and the public readiness probe.
//
// The caller owns the client's lifecycle (ownsClient is implicitly false); Run launches
// the sampling loop and returns when its context is cancelled. Close is flush-only for
// the shared client.
//
// Takes client (*telemetry_grpcfb.Client) which is the shared telemetry transport.
// Takes probe (readiness.Probe) which is the readiness seam sampled each tick.
// Takes interval (time.Duration) which is the sampling cadence, defaulted when not
// positive.
// Takes opts (...Option) which adjust the Runtime before use.
//
// Returns *Runtime which samples the readiness tree over the shared client.
func NewReadiness(client *telemetry_grpcfb.Client, probe readiness.Probe, interval time.Duration, opts ...Option) *Runtime {
	if interval <= 0 {
		interval = DefaultInterval
	}
	r := &Runtime{client: client, probe: probe, clock: clock.RealClock(), interval: interval}
	for _, opt := range opts {
		if opt != nil {
			opt(r)
		}
	}
	return r
}

// Run samples immediately, then on each interval tick, until ctx is cancelled. It is
// non-blocking on the telemetry client (AddMetric enqueues, lossy by design).
func (r *Runtime) Run(ctx context.Context) {
	if r == nil || r.client == nil || r.probe == nil {
		return
	}
	r.collect(ctx)
	t := r.clock.NewTicker(r.interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C():
			r.collect(ctx)
		}
	}
}

// Close is a no-op for the shared-client case (NewReadiness).
//
// The caller owns the client's lifecycle and must not have it closed from under the other
// collectors sharing it. Stop Run by cancelling its context first; the shared client's
// own Close then flushes the final batch. The ctx is accepted for symmetry with the
// sibling collectors and lifecycle hooks.
//
// Returns error which is always nil, present to satisfy the lifecycle interface.
func (*Runtime) Close(context.Context) error {
	return nil
}

// collect takes one readiness sample and emits a MetricPoint per dependency child,
// bounded by maxDeps. A panic inside the probe or the client cannot crash the host: the
// deferred recover keeps the sampling goroutine (and any shared loop) alive.
func (r *Runtime) collect(ctx context.Context) {
	ctx, l := logger_domain.From(ctx, log)
	defer func() {
		if rec := recover(); rec != nil {
			l.Warn("recovered from panic in readiness collect", logger_domain.String("panic", fmt.Sprint(rec)))
		}
	}()

	snapshot := r.probe.CheckReadiness(ctx)
	nowMs := r.clock.Now().UnixMilli()

	deps := snapshot.Dependencies
	if len(deps) > maxDeps {
		l.Warn("readiness dependencies exceed cap, truncating sample",
			logger_domain.Int("total", len(deps)),
			logger_domain.Int("cap", maxDeps))
	}
	deps = deps[:min(len(deps), maxDeps)]
	for i := range deps {
		r.emit(ctx, &deps[i], nowMs)
	}
}

// emit forwards one dependency as a MetricPoint under the agreed field mapping, with the
// Name and message label bounded to the per-string byte budget. The 3 reserved labels
// (status/message/icon) are appended FIRST and are never counted as info nor truncated
// away; then up to maxInfoEntriesPerDep provider info labels follow.
//
// Takes dep (*readiness.Dependency) which is the dependency to forward.
// Takes nowMs (int64) which is the sample timestamp in Unix milliseconds.
func (r *Runtime) emit(ctx context.Context, dep *readiness.Dependency, nowMs int64) {
	name, _ := telemetry_grpcfb.TruncateUTF8(dep.Name, maxStringLen)
	message, _ := telemetry_grpcfb.TruncateUTF8(dep.Message, maxStringLen)
	labels := []telemetry_grpcfb.KV{
		{Key: labelStatus, Value: mapState(dep.State)},
		{Key: labelMessage, Value: message},
		{Key: labelIcon, Value: iconFor(dep)},
	}
	labels = appendInfoLabels(labels, dep.Info)
	r.client.AddMetric(ctx, telemetry_grpcfb.MetricPoint{
		Name:        name,
		Kind:        kindDependency,
		Unit:        unitMs,
		Value:       latencyMs(dep.Duration),
		TimestampMs: nowMs,
		Labels:      labels,
	})
}

// appendInfoLabels appends a dependency's sorted, capped provider info as namespaced
// labels.
//
// It sorts deterministically by (Section, Key), keeps the first maxInfoEntriesPerDep, and
// appends each as an "info.<section>.<key>" label with its value TruncateUTF8'd to
// maxInfoValueLen and its key to maxStringLen. The sort runs BEFORE the cap so the kept
// subset is identical tick to tick, keeping the downstream sink's order-dependent
// attribute truncation from dropping a different subset each sample. A copy is sorted so
// the caller's slice is left untouched.
//
// Takes labels ([]telemetry_grpcfb.KV) which is the existing label set to append onto.
// Takes info ([]readiness.InfoEntry) which is the dependency's provider info.
//
// Returns []telemetry_grpcfb.KV which is labels with the capped info labels appended.
func appendInfoLabels(labels []telemetry_grpcfb.KV, info []readiness.InfoEntry) []telemetry_grpcfb.KV {
	if len(info) == 0 {
		return labels
	}
	sorted := slices.Clone(info)
	slices.SortFunc(sorted, func(a, b readiness.InfoEntry) int {
		if c := cmp.Compare(a.Section, b.Section); c != 0 {
			return c
		}
		if c := cmp.Compare(a.Key, b.Key); c != 0 {
			return c
		}
		return cmp.Compare(a.Value, b.Value)
	})
	dropped := len(sorted) - min(len(sorted), maxInfoEntriesPerDep)
	sorted = sorted[:min(len(sorted), maxInfoEntriesPerDep)]
	for i := range sorted {
		key, _ := telemetry_grpcfb.TruncateUTF8(infoLabelKey(sorted[i].Section, sorted[i].Key), maxStringLen)
		value, _ := telemetry_grpcfb.TruncateUTF8(sorted[i].Value, maxInfoValueLen)
		labels = append(labels, telemetry_grpcfb.KV{Key: key, Value: value})
	}
	if dropped > 0 {
		labels = append(labels, telemetry_grpcfb.KV{Key: labelInfoDropped, Value: strconv.Itoa(dropped)})
	}
	return labels
}

// infoLabelKey builds the "info.<section>.<key>" label key for a provider info entry,
// lower casing the section and key so the namespace is stable regardless of the
// descriptor's display casing.
//
// Takes section (string) which is the provider info section title.
// Takes key (string) which is the provider info entry key.
//
// Returns string which is the namespaced "info.<section>.<key>" label key.
func infoLabelKey(section, key string) string {
	return labelInfoPrefix + strings.ToLower(section) + "." + strings.ToLower(key)
}

// mapState normalises piko's readiness state enum to the exact lowercase literals the
// sink keys on. UNHEALTHY (and any unrecognised state) maps to "down".
//
// Takes state (readiness.State) which is the readiness state to normalise.
//
// Returns string which is the normalised "healthy", "degraded" or "down" literal.
func mapState(state readiness.State) string {
	switch state {
	case readiness.StateHealthy:
		return statusHealthy
	case readiness.StateDegraded:
		return statusDegraded
	default:
		return statusDown
	}
}

// latencyMs parses a Go duration string (e.g. "1.2ms", "500us", "0s") into milliseconds.
//
// Takes duration (string) which is the Go duration text to parse.
//
// Returns float64 which is the duration in milliseconds, 0 when empty or unparseable.
func latencyMs(duration string) float64 {
	if duration == "" {
		return 0
	}
	d, err := time.ParseDuration(duration)
	if err != nil {
		return 0
	}
	return float64(d) / float64(time.Millisecond)
}

// iconHint pairs a lowercase name substring with the icon to use when a dependency's name
// contains it.
type iconHint struct {
	// substr is the lowercase dependency-name fragment that triggers this hint.
	substr string

	// icon is the icon emitted when substr is found in a dependency's name.
	icon string
}

var (
	// iconHints maps dependency-name fragments to a sensible icon.
	//
	// piko's readiness Status carries no icon field, so the icon is derived from the
	// dependency name. The first matching hint wins, so more specific hints come first.
	iconHints = []iconHint{
		{substr: "database", icon: "database"},
		{substr: "postgres", icon: "database"},
		{substr: "sql", icon: "database"},
		{substr: "db", icon: "database"},
		{substr: "redis", icon: "zap"},
		{substr: "cache", icon: "zap"},
		{substr: "http", icon: "globe"},
		{substr: "api", icon: "globe"},
	}
)

// iconFor derives an icon for a dependency from its name, falling back to defaultIcon
// when no hint matches.
//
// Takes dep (*readiness.Dependency) which is the dependency to derive an icon for.
//
// Returns string which is the matched icon, or defaultIcon when no hint matches.
func iconFor(dep *readiness.Dependency) string {
	name := strings.ToLower(dep.Name)
	for _, hint := range iconHints {
		if strings.Contains(name, hint.substr) {
			return hint.icon
		}
	}
	return defaultIcon
}

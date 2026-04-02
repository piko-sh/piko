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

package daemon_adapters

import (
	"sync"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"piko.sh/piko/internal/logger/logger_domain"
)

var (
	log = logger_domain.GetLogger("piko/internal/daemon/daemon_adapters")

	tracer = otel.Tracer("piko/internal/daemon/daemon_adapters")

	// meter provides OpenTelemetry metrics for the daemon adapters package.
	meter = otel.Meter("piko/internal/daemon/daemon_adapters")

	// httpResponseSize records the size of HTTP responses in bytes.
	httpResponseSize metric.Int64Histogram

	// artefactServeCount tracks how many artefacts have been served.
	artefactServeCount metric.Int64Counter

	// pageRequestCount tracks the number of page requests.
	pageRequestCount metric.Int64Counter

	// partialRequestCount tracks the number of partial requests.
	partialRequestCount metric.Int64Counter

	// actionRequestCount tracks the number of action requests.
	actionRequestCount metric.Int64Counter

	// requestErrorCount counts the number of failed requests.
	requestErrorCount metric.Int64Counter

	// cacheHitCount tracks the number of cache hits.
	cacheHitCount metric.Int64Counter

	// cacheMissCount counts the number of cache misses.
	cacheMissCount metric.Int64Counter

	// cacheGenerationDuration tracks the duration of cache generation in
	// milliseconds.
	cacheGenerationDuration metric.Float64Histogram

	// serverStartupDuration tracks the duration of server startup in milliseconds.
	serverStartupDuration metric.Float64Histogram

	// serverShutdownDuration records how long server shutdown takes in
	// milliseconds.
	serverShutdownDuration metric.Float64Histogram

	// serverErrorCount tracks the number of server errors.
	serverErrorCount metric.Int64Counter

	// lazyArtefactServeCount tracks how many artefacts were served while still in
	// PENDING state.
	lazyArtefactServeCount metric.Int64Counter

	// lazyVariantGenerationDuration tracks the time to generate the first variant
	// for PENDING artefacts.
	lazyVariantGenerationDuration metric.Float64Histogram

	// backgroundVariantQueueCount tracks the number of variants queued for
	// background generation.
	backgroundVariantQueueCount metric.Int64Counter

	// artefactMetadataCacheHitCount tracks the number of artefact metadata cache
	// hits.
	artefactMetadataCacheHitCount metric.Int64Counter

	// artefactMetadataCacheMissCount tracks the number of artefact metadata cache
	// misses.
	artefactMetadataCacheMissCount metric.Int64Counter

	// actionRateLimitedCount tracks requests rejected by per-action rate limiting.
	actionRateLimitedCount metric.Int64Counter

	// actionCacheHitCount tracks action response cache hits.
	actionCacheHitCount metric.Int64Counter

	// actionCacheMissCount tracks action response cache misses.
	actionCacheMissCount metric.Int64Counter

	// actionSlowCount tracks actions that exceeded their slow threshold.
	actionSlowCount metric.Int64Counter

	// tlsCertificateReloadCount tracks successful TLS certificate reloads.
	tlsCertificateReloadCount metric.Int64Counter

	// tlsCertificateReloadErrorCount tracks failed TLS certificate reload
	// attempts.
	tlsCertificateReloadErrorCount metric.Int64Counter

	// metricAttrMu guards metricAttrCache.
	//
	// The cache stores metric.MeasurementOption by path and method
	// to avoid re-creating the attribute.Set on every request. The
	// number of unique path+method combinations is bounded by the
	// route table (typically < 100), so this cache is small and
	// never needs eviction.
	metricAttrMu sync.RWMutex

	metricAttrCache = make(map[string]map[string]metric.MeasurementOption)
)

// cachedMetricOption returns a metric.MeasurementOption with the given
// path and method attributes. Results are cached so repeat calls with
// the same arguments produce zero allocations.
//
// Takes path (string) which identifies the route path for the metric.
// Takes method (string) which identifies the HTTP method for the metric.
//
// Returns metric.MeasurementOption which contains the cached path and
// method attributes.
//
// Safe for concurrent use. Uses a read-write mutex to protect the shared
// cache map.
func cachedMetricOption(path, method string) metric.MeasurementOption {
	metricAttrMu.RLock()
	if methods, ok := metricAttrCache[path]; ok {
		if opt, ok := methods[method]; ok {
			metricAttrMu.RUnlock()
			return opt
		}
	}
	metricAttrMu.RUnlock()

	opt := metric.WithAttributeSet(attribute.NewSet(
		attribute.String("path", path),
		attribute.String("method", method),
	))

	metricAttrMu.Lock()
	methods := metricAttrCache[path]
	if methods == nil {
		methods = make(map[string]metric.MeasurementOption, 4)
		metricAttrCache[path] = methods
	}
	methods[method] = opt
	metricAttrMu.Unlock()

	return opt
}

func init() {
	var err error

	httpResponseSize, err = meter.Int64Histogram(
		"daemon.http_response_size",
		metric.WithDescription("Size of HTTP responses"),
		metric.WithUnit("bytes"),
	)
	if err != nil {
		otel.Handle(err)
	}

	artefactServeCount, err = meter.Int64Counter(
		"daemon.artefact_serve_count",
		metric.WithDescription("Number of artefacts served"),
	)
	if err != nil {
		otel.Handle(err)
	}

	pageRequestCount, err = meter.Int64Counter(
		"daemon.page_request_count",
		metric.WithDescription("Number of page requests"),
	)
	if err != nil {
		otel.Handle(err)
	}

	partialRequestCount, err = meter.Int64Counter(
		"daemon.partial_request_count",
		metric.WithDescription("Number of partial requests"),
	)
	if err != nil {
		otel.Handle(err)
	}

	actionRequestCount, err = meter.Int64Counter(
		"daemon.action_request_count",
		metric.WithDescription("Number of action requests"),
	)
	if err != nil {
		otel.Handle(err)
	}

	requestErrorCount, err = meter.Int64Counter(
		"daemon.request_error_count",
		metric.WithDescription("Number of request errors"),
	)
	if err != nil {
		otel.Handle(err)
	}

	cacheHitCount, err = meter.Int64Counter(
		"daemon.cache_hit_count",
		metric.WithDescription("Number of cache hits"),
	)
	if err != nil {
		otel.Handle(err)
	}

	cacheMissCount, err = meter.Int64Counter(
		"daemon.cache_miss_count",
		metric.WithDescription("Number of cache misses"),
	)
	if err != nil {
		otel.Handle(err)
	}

	cacheGenerationDuration, err = meter.Float64Histogram(
		"daemon.cache_generation_duration",
		metric.WithDescription("Duration of cache generation"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	serverStartupDuration, err = meter.Float64Histogram(
		"daemon.server_startup_duration",
		metric.WithDescription("Duration of server startup"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	serverShutdownDuration, err = meter.Float64Histogram(
		"daemon.server_shutdown_duration",
		metric.WithDescription("Duration of server shutdown"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	serverErrorCount, err = meter.Int64Counter(
		"daemon.server_error_count",
		metric.WithDescription("Number of server errors"),
	)
	if err != nil {
		otel.Handle(err)
	}

	lazyArtefactServeCount, err = meter.Int64Counter(
		"daemon.http.lazy_artefact_serve_count",
		metric.WithDescription("Number of artefacts served in PENDING state (lazy generation)"),
	)
	if err != nil {
		otel.Handle(err)
	}

	lazyVariantGenerationDuration, err = meter.Float64Histogram(
		"daemon.http.lazy_variant_generation_duration",
		metric.WithDescription("Time to generate first variant for PENDING artefacts"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	backgroundVariantQueueCount, err = meter.Int64Counter(
		"daemon.http.background_variant_queue_count",
		metric.WithDescription("Number of variants queued for background generation"),
	)
	if err != nil {
		otel.Handle(err)
	}

	artefactMetadataCacheHitCount, err = meter.Int64Counter(
		"daemon.http.artefact_metadata_cache_hit_count",
		metric.WithDescription("Number of artefact metadata cache hits"),
	)
	if err != nil {
		otel.Handle(err)
	}

	artefactMetadataCacheMissCount, err = meter.Int64Counter(
		"daemon.http.artefact_metadata_cache_miss_count",
		metric.WithDescription("Number of artefact metadata cache misses"),
	)
	if err != nil {
		otel.Handle(err)
	}

	actionRateLimitedCount, err = meter.Int64Counter(
		"daemon.action_rate_limited_count",
		metric.WithDescription("Number of action requests rejected by per-action rate limiting"),
	)
	if err != nil {
		otel.Handle(err)
	}

	actionCacheHitCount, err = meter.Int64Counter(
		"daemon.action_cache_hit_count",
		metric.WithDescription("Number of action response cache hits"),
	)
	if err != nil {
		otel.Handle(err)
	}

	actionCacheMissCount, err = meter.Int64Counter(
		"daemon.action_cache_miss_count",
		metric.WithDescription("Number of action response cache misses"),
	)
	if err != nil {
		otel.Handle(err)
	}

	actionSlowCount, err = meter.Int64Counter(
		"daemon.action_slow_count",
		metric.WithDescription("Number of action executions that exceeded the slow threshold"),
	)
	if err != nil {
		otel.Handle(err)
	}

	tlsCertificateReloadCount, err = meter.Int64Counter(
		"daemon.tls.certificate_reload_count",
		metric.WithDescription("Number of successful TLS certificate reloads"),
	)
	if err != nil {
		otel.Handle(err)
	}

	tlsCertificateReloadErrorCount, err = meter.Int64Counter(
		"daemon.tls.certificate_reload_error_count",
		metric.WithDescription("Number of failed TLS certificate reload attempts"),
	)
	if err != nil {
		otel.Handle(err)
	}
}

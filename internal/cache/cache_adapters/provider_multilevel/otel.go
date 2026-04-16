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

package provider_multilevel

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"piko.sh/piko/internal/logger/logger_domain"
)

var (
	// log is the package-level logger for the provider_multilevel package.
	log = logger_domain.GetLogger("piko/internal/cache/cache_adapters/provider_multilevel")

	// meter is the OpenTelemetry meter for the multi-level cache provider metrics.
	meter = otel.Meter("piko/internal/cache/cache_adapters/provider_multilevel")

	// l1HitsTotal counts cache hits served from the L1 (in-memory) cache.
	l1HitsTotal metric.Int64Counter

	// l2HitsTotal counts cache hits served from the L2 (distributed) cache after
	// an L1 miss.
	l2HitsTotal metric.Int64Counter

	// totalMissesTotal counts cache misses on both L1 and L2 that need a loader
	// call.
	totalMissesTotal metric.Int64Counter

	// l2ErrorsTotal counts errors encountered when communicating with the L2 cache.
	l2ErrorsTotal metric.Int64Counter

	// backPopulations counts items from L2 written back into L1.
	backPopulations metric.Int64Counter
)

func init() {
	var err error

	l1HitsTotal, err = meter.Int64Counter(
		"piko.cache.provider.multilevel.l1.hits.total",
		metric.WithDescription("Total number of cache hits served directly from the L1 (in-memory) cache."),
	)
	if err != nil {
		otel.Handle(err)
	}

	l2HitsTotal, err = meter.Int64Counter(
		"piko.cache.provider.multilevel.l2.hits.total",
		metric.WithDescription("Total number of cache hits served from the L2 (distributed) cache after an L1 miss."),
	)
	if err != nil {
		otel.Handle(err)
	}

	totalMissesTotal, err = meter.Int64Counter(
		"piko.cache.provider.multilevel.misses.total",
		metric.WithDescription("Total number of cache misses on both L1 and L2, requiring a loader call."),
	)
	if err != nil {
		otel.Handle(err)
	}

	l2ErrorsTotal, err = meter.Int64Counter(
		"piko.cache.provider.multilevel.l2.errors.total",
		metric.WithDescription("Total number of errors encountered when communicating with the L2 cache."),
	)
	if err != nil {
		otel.Handle(err)
	}

	backPopulations, err = meter.Int64Counter(
		"piko.cache.provider.multilevel.back_populations.total",
		metric.WithDescription("Total number of times an item from L2 was written back into L1."),
	)
	if err != nil {
		otel.Handle(err)
	}

}

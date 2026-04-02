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

package daemon_frontend

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"piko.sh/piko/internal/logger/logger_domain"
)

var (
	log = logger_domain.GetLogger("piko/internal/daemon/daemon_frontend")

	// meter is the OpenTelemetry meter for daemon frontend metrics.
	meter = otel.Meter("piko/internal/daemon/daemon_frontend")

	// assetCacheInitCount tracks how many times the asset cache has been set up.
	assetCacheInitCount metric.Int64Counter

	// assetCacheInitErrorCount counts errors that occur when the asset cache
	// starts up.
	assetCacheInitErrorCount metric.Int64Counter

	// assetCacheSize tracks the number of assets held in the cache.
	assetCacheSize metric.Int64Counter

	// assetCacheReadCount tracks the total number of asset reads.
	assetCacheReadCount metric.Int64Counter

	// assetCacheReadErrorCount counts errors that occur when reading from the
	// asset cache.
	assetCacheReadErrorCount metric.Int64Counter

	// assetCacheMissCount tracks cache misses when an asset is not found.
	assetCacheMissCount metric.Int64Counter
)

func init() {
	var err error

	assetCacheInitCount, err = meter.Int64Counter(
		"daemon.frontend.asset_cache_init_count",
		metric.WithDescription("Number of times the asset cache is initialised"),
	)
	if err != nil {
		otel.Handle(err)
	}

	assetCacheInitErrorCount, err = meter.Int64Counter(
		"daemon.frontend.asset_cache_init_error_count",
		metric.WithDescription("Number of errors during asset cache initialisation"),
	)
	if err != nil {
		otel.Handle(err)
	}

	assetCacheSize, err = meter.Int64Counter(
		"daemon.frontend.asset_cache_size",
		metric.WithDescription("Size of the asset cache (number of assets)"),
	)
	if err != nil {
		otel.Handle(err)
	}

	assetCacheReadCount, err = meter.Int64Counter(
		"daemon.frontend.asset_cache_read_count",
		metric.WithDescription("Number of asset reads"),
	)
	if err != nil {
		otel.Handle(err)
	}

	assetCacheReadErrorCount, err = meter.Int64Counter(
		"daemon.frontend.asset_cache_read_error_count",
		metric.WithDescription("Number of asset read errors"),
	)
	if err != nil {
		otel.Handle(err)
	}

	assetCacheMissCount, err = meter.Int64Counter(
		"daemon.frontend.asset_cache_miss_count",
		metric.WithDescription("Number of asset cache misses"),
	)
	if err != nil {
		otel.Handle(err)
	}
}

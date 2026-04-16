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

package provider_otter

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"piko.sh/piko/internal/logger/logger_domain"
)

var (
	// log is the package-level logger for the provider_otter package.
	log = logger_domain.GetLogger("piko/internal/cache/cache_adapters/provider_otter")

	// meter is the OpenTelemetry meter for otter cache provider metrics.
	meter = otel.Meter("piko/internal/cache/cache_adapters/provider_otter")

	// tagInvalidationsTotal counts tag-based invalidation operations.
	tagInvalidationsTotal metric.Int64Counter

	// invalidatedKeysByTagTotal counts keys invalidated via tag-based operations.
	invalidatedKeysByTagTotal metric.Int64Counter
)

func init() {
	var err error

	tagInvalidationsTotal, err = meter.Int64Counter(
		"piko.cache.provider.otter.tags.invalidations.total",
		metric.WithDescription("Total number of tag-based invalidation operations."),
	)
	if err != nil {
		otel.Handle(err)
	}

	invalidatedKeysByTagTotal, err = meter.Int64Counter(
		"piko.cache.provider.otter.tags.invalidated_keys.total",
		metric.WithDescription("Total number of keys invalidated via tag-based operations."),
	)
	if err != nil {
		otel.Handle(err)
	}
}

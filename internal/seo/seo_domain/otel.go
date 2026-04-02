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

package seo_domain

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"piko.sh/piko/internal/logger/logger_domain"
)

var (
	log = logger_domain.GetLogger("piko/internal/seo/seo_domain")

	// meter is the package-level meter for OpenTelemetry metrics.
	meter = otel.Meter("piko/internal/seo/seo_domain")

	// sitemapGenerationDuration tracks how long it takes to generate a complete
	// sitemap.
	sitemapGenerationDuration metric.Float64Histogram

	// sitemapURLCount tracks the number of URLs included in generated sitemaps.
	sitemapURLCount metric.Int64Counter

	// robotsTxtGenerationCount tracks how many times robots.txt has been
	// generated.
	robotsTxtGenerationCount metric.Int64Counter
)

func init() {
	var err error

	sitemapGenerationDuration, err = meter.Float64Histogram(
		"seo.sitemap.generation.duration",
		metric.WithDescription("Duration of complete sitemap generation in milliseconds"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	sitemapURLCount, err = meter.Int64Counter(
		"seo.sitemap.url.count",
		metric.WithDescription("Number of URLs included in the generated sitemap"),
	)
	if err != nil {
		otel.Handle(err)
	}

	robotsTxtGenerationCount, err = meter.Int64Counter(
		"seo.robots.generation.count",
		metric.WithDescription("Number of times robots.txt has been generated"),
	)
	if err != nil {
		otel.Handle(err)
	}

}

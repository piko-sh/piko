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

package video_domain

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"piko.sh/piko/internal/logger/logger_domain"
)

const (
	// metricAttributeProvider is the attribute key for the video provider name.
	metricAttributeProvider = "provider"

	// metricAttributeCodec is the attribute key for the video codec name.
	metricAttributeCodec = "codec"
)

var (
	// log is the package-level logger for the video domain.
	log = logger_domain.GetLogger("piko/internal/video/video_domain")

	// meter is the OpenTelemetry meter for video domain metrics.
	meter = otel.Meter("piko/internal/video/video_domain")

	// transcodeDuration records the duration of video transcode operations.
	transcodeDuration metric.Float64Histogram

	// transcodeCount counts the total number of video transcode operations.
	transcodeCount metric.Int64Counter

	// transcodeErrorCount counts the number of failed transcode operations.
	transcodeErrorCount metric.Int64Counter
)

func init() {
	var err error

	transcodeDuration, err = meter.Float64Histogram(
		"video.domain.transcode_duration",
		metric.WithDescription("Duration of video transcode operations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	transcodeCount, err = meter.Int64Counter(
		"video.domain.transcode_count",
		metric.WithDescription("Total number of video transcode operations"),
	)
	if err != nil {
		otel.Handle(err)
	}

	transcodeErrorCount, err = meter.Int64Counter(
		"video.domain.transcode_error_count",
		metric.WithDescription("Number of failed transcode operations"),
	)
	if err != nil {
		otel.Handle(err)
	}

}

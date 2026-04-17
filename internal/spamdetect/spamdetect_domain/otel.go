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

package spamdetect_domain

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"piko.sh/piko/internal/logger/logger_domain"
)

const (
	// attributeKeyOperation is the OTel attribute key for the operation name.
	attributeKeyOperation = "operation"

	// attributeKeyDetector is the OTel attribute key for the detector name.
	attributeKeyDetector = "detector"

	// attributeKeyStatus is the OTel attribute key for the outcome status.
	attributeKeyStatus = "status"

	// attributeKeyIsSpam is the OTel attribute key for the spam verdict.
	attributeKeyIsSpam = "is_spam"

	// attributeKeyDurationMS is the OTel attribute key for duration in milliseconds.
	attributeKeyDurationMS = "duration_ms"

	// opAnalyse is the operation name for analysis metrics.
	opAnalyse = "analyse"

	// statusSuccess is the outcome status for successful operations.
	statusSuccess = "success"

	// statusError is the outcome status for failed operations.
	statusError = "error"
)

var (
	// log is the logger for the spam detection domain.
	log = logger_domain.GetLogger("piko/internal/spamdetect/spamdetect_domain")

	// meter is the OTel meter for spam detection metrics.
	meter = otel.Meter("piko/internal/spamdetect/spamdetect_domain")

	// spamDetectCheckCount tracks the number of spam detection analyses.
	spamDetectCheckCount metric.Int64Counter

	// spamDetectCheckDuration tracks analysis durations in milliseconds.
	spamDetectCheckDuration metric.Float64Histogram
)

// metricAttributes builds an OTel measurement option from key-value
// pairs.
//
// Takes keyValues (...string) which are alternating key-value pairs.
//
// Returns metric.MeasurementOption which contains the attributes.
func metricAttributes(keyValues ...string) metric.MeasurementOption {
	if len(keyValues)%2 != 0 {
		return metric.WithAttributes()
	}

	attrs := make([]attribute.KeyValue, 0, len(keyValues)/2)
	for index := 0; index+1 < len(keyValues); index += 2 {
		attrs = append(attrs, attribute.String(keyValues[index], keyValues[index+1]))
	}

	return metric.WithAttributes(attrs...)
}

func init() {
	var err error

	spamDetectCheckCount, err = meter.Int64Counter(
		"spamdetect.analyse.count",
		metric.WithDescription("Number of spam detection analysis attempts"),
		metric.WithUnit("{operation}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	spamDetectCheckDuration, err = meter.Float64Histogram(
		"spamdetect.analyse.duration",
		metric.WithDescription("Duration of spam detection analysis operations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}
}

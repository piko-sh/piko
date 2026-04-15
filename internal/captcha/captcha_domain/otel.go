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

package captcha_domain

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"piko.sh/piko/internal/logger/logger_domain"
)

const (
	// attributeKeyOperation is the metric attribute key for the operation name.
	attributeKeyOperation = "operation"

	// attributeKeyProvider is the attribute key for the captcha provider type.
	attributeKeyProvider = "provider"

	// attributeKeyStatus is the metric attribute key for the operation outcome.
	attributeKeyStatus = "status"

	// attributeKeyDurationMS is the logging attribute key for operation duration
	// in milliseconds.
	attributeKeyDurationMS = "duration_ms"

	// opVerify is the operation name for verification metrics.
	opVerify = "verify"

	// statusSuccess is the status value recorded when verification succeeds.
	statusSuccess = "success"

	// statusError is the metric status value for failed verifications.
	statusError = "error"
)

var (
	// log is the package-level logger for the captcha_domain package.
	log = logger_domain.GetLogger("piko/internal/captcha/captcha_domain")

	// meter is the package-level OTel meter for the captcha_domain package.
	meter = otel.Meter("piko/internal/captcha/captcha_domain")

	// captchaVerifyCount counts the number of captcha verification attempts.
	captchaVerifyCount metric.Int64Counter

	// captchaVerifyDuration tracks the duration of captcha verifications.
	captchaVerifyDuration metric.Float64Histogram
)

// metricAttributes creates metric attributes from key-value pairs.
//
// Takes keyVals (...string) which are alternating key-value pairs for
// attributes.
//
// Returns metric.MeasurementOption which contains the constructed attributes.
func metricAttributes(keyVals ...string) metric.MeasurementOption {
	if len(keyVals)%2 != 0 {
		return metric.WithAttributes()
	}

	attrs := make([]attribute.KeyValue, 0, len(keyVals)/2)
	for i := 0; i+1 < len(keyVals); i += 2 {
		attrs = append(attrs, attribute.String(keyVals[i], keyVals[i+1]))
	}

	return metric.WithAttributes(attrs...)
}

func init() {
	var err error

	captchaVerifyCount, err = meter.Int64Counter(
		"captcha.verify.count",
		metric.WithDescription("Number of captcha verification attempts"),
		metric.WithUnit("{operation}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	captchaVerifyDuration, err = meter.Float64Histogram(
		"captcha.verify.duration",
		metric.WithDescription("Duration of captcha verification operations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}
}

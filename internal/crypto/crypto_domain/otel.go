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

package crypto_domain

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"piko.sh/piko/internal/logger/logger_domain"
)

const (
	// attributeKeyOperation is the metric attribute key for the operation name.
	attributeKeyOperation = "operation"

	// attributeKeyProvider is the attribute key for the encryption provider type.
	attributeKeyProvider = "provider"

	// attributeKeyStatus is the metric attribute key for the operation outcome.
	attributeKeyStatus = "status"

	// attributeKeyDurationMS is the logging attribute key for operation duration in
	// milliseconds.
	attributeKeyDurationMS = "duration_ms"

	// attributeKeyBatchSize is the metric attribute key for batch size.
	attributeKeyBatchSize = "batch_size"

	// attributeKeyCount is the logging attribute key for item counts
	// in batch operations.
	attributeKeyCount = "count"

	// opEncrypt is the operation name for encryption metrics and error tracking.
	opEncrypt = "encrypt"

	// opDecrypt is the operation name used for decryption metrics and error tracking.
	opDecrypt = "decrypt"

	// opEncryptBatch is the metric name for batch encryption operations.
	opEncryptBatch = "encrypt_batch"

	// opDecryptBatch is the operation name for batch decryption metrics.
	opDecryptBatch = "decrypt_batch"

	// opKeyRotation is the operation name for key rotation metrics.
	opKeyRotation = "key_rotation"

	// opAutoReencrypt is the operation name for automatic re-encryption metrics.
	opAutoReencrypt = "auto_reencrypt"

	// opEncryptStream is the operation name for streaming encryption metrics.
	opEncryptStream = "encrypt_stream"

	// opDecryptStream is the operation name for streaming decryption metrics.
	opDecryptStream = "decrypt_stream"

	// statusSuccess is the status value recorded when an operation completes
	// without error.
	statusSuccess = "success"

	// statusError is the metric status value for failed operations.
	statusError = "error"

	// statusPartialFailure is the metric status for batch operations that
	// completed with some errors.
	statusPartialFailure = "partial_failure"

	// statusInitiated is the status for newly started operations.
	statusInitiated = "initiated"
)

var (
	// log is the package-level logger for the crypto_domain package.
	log = logger_domain.GetLogger("piko/internal/crypto/crypto_domain")

	// meter is the OpenTelemetry meter for the crypto_domain package.
	meter = otel.Meter("piko/internal/crypto/crypto_domain")

	// cryptoOperationCount counts the number of cryptographic operations performed.
	cryptoOperationCount metric.Int64Counter

	// cryptoOperationDuration tracks the duration of cryptographic operations.
	cryptoOperationDuration metric.Float64Histogram
)

// metricAttributes creates metric attributes from key-value pairs.
// This is a helper function to make metric recording more ergonomic.
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
	for i := 0; i < len(keyVals); i += 2 {
		attrs = append(attrs, attribute.String(keyVals[i], keyVals[i+1])) //nolint:gosec // bounds checked above
	}

	return metric.WithAttributes(attrs...)
}

func init() {
	var err error

	cryptoOperationCount, err = meter.Int64Counter(
		"crypto.operation.count",
		metric.WithDescription("Number of cryptographic operations performed"),
		metric.WithUnit("{operation}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	cryptoOperationDuration, err = meter.Float64Histogram(
		"crypto.operation.duration",
		metric.WithDescription("Duration of cryptographic operations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

}

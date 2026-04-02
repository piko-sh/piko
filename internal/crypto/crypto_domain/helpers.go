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
	"context"
	"fmt"
	"sync/atomic"
)

// recordOperationMetrics records both count and duration metrics for a crypto
// operation.
//
// Takes provider (EncryptionProvider) which identifies the provider type for
// metrics.
// Takes op (string) which identifies the operation type being measured.
// Takes status (string) which indicates the operation result status.
// Takes durationMs (int64) which specifies the operation duration in
// milliseconds.
func (*cryptoService) recordOperationMetrics(ctx context.Context, provider EncryptionProvider, op, status string, durationMs int64) {
	providerType := string(provider.Type())

	cryptoOperationDuration.Record(ctx, float64(durationMs),
		metricAttributes(attributeKeyOperation, op, attributeKeyProvider, providerType),
	)

	cryptoOperationCount.Add(ctx, 1,
		metricAttributes(attributeKeyOperation, op, attributeKeyProvider, providerType, attributeKeyStatus, status),
	)
}

// recordOperationError records a failure metric for a crypto operation.
//
// Takes provider (EncryptionProvider) which identifies the provider type for
// metrics.
// Takes op (string) which identifies the operation that failed.
func (*cryptoService) recordOperationError(ctx context.Context, provider EncryptionProvider, op string) {
	providerType := "unknown"
	if provider != nil {
		providerType = string(provider.Type())
	}
	cryptoOperationCount.Add(ctx, 1,
		metricAttributes(
			attributeKeyOperation, op,
			attributeKeyProvider, providerType,
			attributeKeyStatus, statusError,
		),
	)
}

// recordBatchMetrics records metrics for a batch operation.
//
// Takes provider (EncryptionProvider) which identifies the provider for
// metrics.
// Takes op (string) which specifies the operation type being measured.
// Takes status (string) which indicates the outcome of the operation.
// Takes batchSize (int) which specifies the number of items in the batch.
func (*cryptoService) recordBatchMetrics(ctx context.Context, provider EncryptionProvider, op, status string, batchSize int) {
	cryptoOperationCount.Add(ctx, 1,
		metricAttributes(
			attributeKeyOperation, op,
			attributeKeyProvider, string(provider.Type()),
			attributeKeyStatus, status,
			attributeKeyBatchSize, fmt.Sprintf("%d", batchSize),
		),
	)
}

// zeroBytes overwrites a byte slice with zeros to remove sensitive data
// from memory.
//
// This is a critical security measure for ephemeral encryption keys. The
// memory barrier prevents the compiler from optimising away the writes.
//
// Takes data ([]byte) which is the slice to be zeroed in place.
func zeroBytes(data []byte) {
	for i := range data {
		data[i] = 0
	}
	atomic.StoreInt32(new(int32), 0)
}

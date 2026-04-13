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

package cache_provider_firestore

import (
	"time"

	"piko.sh/piko/wdk/cache"
)

// Config holds all Firestore-specific configuration.
//
// Fields are ordered for optimal struct alignment.
type Config struct {
	// KeyRegistry specifies an EncodingRegistry for complex key types such as
	// structs. If nil, keys are encoded using fmt.Sprintf, which is suitable for
	// primitive types.
	KeyRegistry *cache.EncodingRegistry

	// Registry specifies the encoding registry for cache values; required.
	Registry *cache.EncodingRegistry

	// ProjectID is the Google Cloud project ID that contains the Firestore
	// database.
	ProjectID string

	// DatabaseID is the Firestore database ID. Defaults to "(default)".
	DatabaseID string

	// CollectionPrefix is the top-level Firestore collection name used as the
	// root for all cache namespaces. Defaults to "piko_cache".
	CollectionPrefix string

	// EmulatorHost is the address of a Firestore emulator (e.g.,
	// "localhost:8080"). When set, the provider connects to the emulator
	// instead of the production service.
	EmulatorHost string

	// CredentialsFile is the path to a Google Cloud service account JSON key
	// file. If empty, Application Default Credentials are used.
	CredentialsFile string

	// Namespace is a prefix added to all keys (e.g., "myapp:").
	// Recommended for shared Firestore databases to prevent key collisions.
	Namespace string

	// CredentialsJSON is the raw Google Cloud service account JSON key data.
	// Takes precedence over CredentialsFile when both are set.
	CredentialsJSON []byte

	// DefaultTTL is how long cache entries are kept before expiry. Default is
	// 1 hour.
	DefaultTTL time.Duration

	// OperationTimeout specifies the timeout for standard Firestore operations.
	// Default is 2 seconds.
	OperationTimeout time.Duration

	// AtomicOperationTimeout is the maximum time allowed for atomic operations
	// such as Compute functions. Default is 5 seconds.
	AtomicOperationTimeout time.Duration

	// BulkOperationTimeout is the maximum duration for bulk operations such as
	// BulkGet and BulkSet. Default is 10 seconds.
	BulkOperationTimeout time.Duration

	// FlushTimeout is the maximum time allowed for flush operations. Default is
	// 30s.
	FlushTimeout time.Duration

	// SearchTimeout is the timeout for search operations. Default is 5 seconds.
	SearchTimeout time.Duration

	// MaxComputeRetries is the maximum number of transaction retry attempts.
	// Default is 10.
	MaxComputeRetries int

	// BatchSize is the maximum number of documents to process in a single
	// batch write or bulk read. Defaults to 500.
	BatchSize int

	// EnableTTLClientCheck controls whether the adapter checks the __ttl field
	// client-side on reads and treats expired documents as misses. This provides
	// consistent behaviour even before Firestore's server-side TTL policy
	// removes the document. Defaults to true.
	EnableTTLClientCheck bool
}

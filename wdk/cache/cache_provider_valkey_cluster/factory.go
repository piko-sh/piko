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

package cache_provider_valkey_cluster

import (
	"crypto/tls"
	"time"

	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/wdk/cache"
)

// Config holds all Valkey Cluster-specific configuration.
type Config struct {
	// Registry is the encoding registry for cache values; required.
	Registry *cache.EncodingRegistry

	// KeyRegistry is an optional encoding registry for complex key types such as
	// structs. If nil, keys are encoded using fmt.Sprintf, which works for simple
	// types like strings and integers.
	KeyRegistry *cache.EncodingRegistry

	// Namespace is a prefix added to all keys (e.g., "myapp:").
	// Recommended to prevent key collisions with other
	// applications sharing the cluster.
	Namespace string

	// Password is the credential used to connect to Valkey.
	// Empty string means no password is needed.
	Password string

	// Username is the ACL username for Valkey authentication.
	// Empty string means default user.
	Username string

	// ClientName is used with the CLIENT SETNAME command to identify connections.
	ClientName string

	// TLSConfig configures TLS for the Valkey connection.
	// Nil means no TLS.
	TLSConfig *tls.Config

	// IndexPrefix is the prefix for Valkey Search index names; default is "index:".
	IndexPrefix string

	// InitAddress is the list of seed nodes for the
	// cluster (e.g., ["localhost:7000", "localhost:7001"]).
	// REQUIRED: At least one address must be provided.
	InitAddress []string

	// AtomicOperationTimeout specifies the timeout for atomic operations such as
	// Compute. Default is 5 seconds.
	AtomicOperationTimeout time.Duration

	// OperationTimeout is the maximum time for standard operations. Default is 2s.
	OperationTimeout time.Duration

	// DefaultTTL specifies how long cache entries are kept before they expire;
	// defaults to 1 hour.
	DefaultTTL time.Duration

	// BulkOperationTimeout is the maximum time for bulk operations such as
	// BulkGet and BulkSet. Default is 10 seconds.
	BulkOperationTimeout time.Duration

	// FlushTimeout is the longest time allowed for flush
	// operations. Default is 30s.
	FlushTimeout time.Duration

	// SearchTimeout is the time limit for FT.SEARCH
	// operations; default is 5 seconds.
	SearchTimeout time.Duration

	// MaxComputeRetries is the maximum number of retries when an optimistic lock
	// fails. Default is 10.
	MaxComputeRetries int

	// AllowUnsafeFLUSHDB permits InvalidateAll to use FLUSHDB when Namespace is
	// empty. WARNING: In cluster mode, this clears ALL data on ALL master nodes.
	AllowUnsafeFLUSHDB bool

	// SendToReplicas enables routing read commands to
	// replica nodes for read scaling.
	SendToReplicas bool
}

// applyConfigDefaults sets sensible defaults for any zero-valued fields in the
// given Config.
//
// Takes config (*Config) which is the configuration to populate with defaults.
func applyConfigDefaults(config *Config) {
	cache_domain.ApplyProviderDefaults(cache_domain.ProviderDefaultsParams{
		DefaultTTL:             &config.DefaultTTL,
		OperationTimeout:       &config.OperationTimeout,
		AtomicOperationTimeout: &config.AtomicOperationTimeout,
		BulkOperationTimeout:   &config.BulkOperationTimeout,
		FlushTimeout:           &config.FlushTimeout,
		MaxComputeRetries:      &config.MaxComputeRetries,
		SearchTimeout:          &config.SearchTimeout,
		IndexPrefix:            &config.IndexPrefix,
	})
}

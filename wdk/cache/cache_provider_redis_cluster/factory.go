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

package cache_provider_redis_cluster

import (
	"time"

	"piko.sh/piko/wdk/cache"
)

// Config holds all Redis Cluster-specific configuration.
type Config struct {
	// Registry is the encoding registry for cache values; required.
	Registry *cache.EncodingRegistry

	// KeyRegistry is an optional encoding registry for complex key types such as
	// structs. If nil, keys are encoded using fmt.Sprintf, which works for simple
	// types like strings and integers.
	KeyRegistry *cache.EncodingRegistry

	// Namespace is a prefix added to all keys (e.g., "myapp:").
	// Recommended to prevent key collisions with other applications
	// sharing the cluster.
	Namespace string

	// Password is the credential used to connect to Redis.
	// Empty string means no password is needed.
	Password string

	// IndexPrefix is the prefix for RediSearch index names; default is "index:".
	IndexPrefix string

	// Addrs is the list of seed nodes for the cluster
	// (e.g., ["localhost:7000", "localhost:7001"]).
	// REQUIRED: At least one address must be provided.
	Addrs []string

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

	// FlushTimeout is the longest time allowed for flush operations. Default is 30s.
	FlushTimeout time.Duration

	// SearchTimeout is the time limit for FT.SEARCH operations; default is 5 seconds.
	SearchTimeout time.Duration

	// MaxComputeRetries is the maximum number of retries when an optimistic lock
	// fails. Default is 10.
	MaxComputeRetries int

	// MaxRedirects is the maximum number of cluster slot redirects. Default: 3.
	MaxRedirects int

	// AllowUnsafeFLUSHDB permits InvalidateAll to use FLUSHDB when Namespace is
	// empty. WARNING: In cluster mode, this clears ALL data on ALL master nodes.
	AllowUnsafeFLUSHDB bool

	// ReadOnly enables reading from replica nodes.
	ReadOnly bool

	// RouteByLatency enables routing of read requests to the node with
	// the lowest latency.
	RouteByLatency bool

	// RouteRandomly enables random routing of read requests across cluster nodes.
	RouteRandomly bool
}

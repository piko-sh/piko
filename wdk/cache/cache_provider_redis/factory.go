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

package cache_provider_redis

import (
	"time"

	"piko.sh/piko/wdk/cache"
)

// Config holds all Redis-specific configuration.
type Config struct {
	// KeyRegistry specifies an EncodingRegistry for complex key types such as
	// structs. If nil, keys are encoded using fmt.Sprintf, which is suitable for
	// primitive types.
	KeyRegistry *cache.EncodingRegistry

	// Registry specifies the encoding registry for cache values; required.
	Registry *cache.EncodingRegistry

	// Password is the Redis authentication password; empty means no
	// authentication.
	Password string

	// Namespace is a prefix added to all keys (e.g., "myapp:").
	// Recommended for shared Redis instances to prevent key collisions.
	Namespace string

	// Address is the Redis server address (for example, "localhost:6379").
	Address string

	// IndexPrefix is the prefix for RediSearch index names (default: "index:").
	IndexPrefix string

	// DefaultTTL is how long cache entries are kept before expiry. Default is 1 hour.
	DefaultTTL time.Duration

	// DB is the Redis database number, ranging from 0 to 15.
	DB int

	// OperationTimeout specifies the timeout for standard Redis operations.
	// Default is 2 seconds.
	OperationTimeout time.Duration

	// AtomicOperationTimeout is the maximum time allowed for atomic operations
	// such as Compute functions. Default is 5 seconds.
	AtomicOperationTimeout time.Duration

	// BulkOperationTimeout is the maximum duration for bulk operations such as
	// BulkGet and BulkSet. Default is 10 seconds.
	BulkOperationTimeout time.Duration

	// FlushTimeout is the maximum time allowed for flush operations. Default is 30s.
	FlushTimeout time.Duration

	// SearchTimeout is the timeout for FT.SEARCH operations. Default is 5 seconds.
	SearchTimeout time.Duration

	// MaxComputeRetries is the maximum number of optimistic lock retries.
	// Default is 10.
	MaxComputeRetries int

	// AllowUnsafeFLUSHDB enables the use of FLUSHDB in InvalidateAll when
	// Namespace is empty. WARNING: This deletes ALL keys in the database, not
	// just cache keys.
	AllowUnsafeFLUSHDB bool
}

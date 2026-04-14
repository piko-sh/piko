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

package cache_provider_dynamodb

import (
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"piko.sh/piko/wdk/cache"
)

// Config holds all DynamoDB-specific configuration.
type Config struct {
	// KeyRegistry specifies an EncodingRegistry for complex key types such as
	// structs. If nil, keys are encoded using fmt.Sprintf, which is suitable for
	// primitive types.
	KeyRegistry *cache.EncodingRegistry

	// Registry specifies the encoding registry for cache values; required.
	Registry *cache.EncodingRegistry

	// AWSConfig provides the AWS SDK v2 configuration. If nil, the provider
	// loads the default configuration from the environment.
	AWSConfig *aws.Config

	// TableName is the DynamoDB table used for cache storage. Default is
	// "piko_cache".
	TableName string

	// Namespace is a prefix added to all partition keys (e.g., "myapp:").
	// Recommended for shared tables to prevent key collisions.
	Namespace string

	// Region is the AWS region for the DynamoDB client. If empty, the region
	// from AWSConfig or the environment is used.
	Region string

	// EndpointURL overrides the DynamoDB endpoint, useful for LocalStack or
	// DynamoDB Local during development.
	EndpointURL string

	// BillingMode controls the billing mode used when AutoCreateTable creates
	// the table. Default is PAY_PER_REQUEST.
	BillingMode types.BillingMode

	// DefaultTTL is how long cache entries are kept before expiry. Default is
	// 1 hour.
	DefaultTTL time.Duration

	// OperationTimeout specifies the timeout for standard DynamoDB operations.
	// Default is 2 seconds.
	OperationTimeout time.Duration

	// AtomicOperationTimeout is the maximum time allowed for atomic operations
	// such as Compute functions. Default is 5 seconds.
	AtomicOperationTimeout time.Duration

	// BulkOperationTimeout is the maximum duration for bulk operations such as
	// BulkGet and BulkSet. Default is 10 seconds.
	BulkOperationTimeout time.Duration

	// FlushTimeout is the maximum time allowed for flush operations. Default is
	// 30 seconds.
	FlushTimeout time.Duration

	// SearchTimeout is the timeout for search operations. Default is 5 seconds.
	SearchTimeout time.Duration

	// ReadCapacityUnits sets the provisioned read capacity when AutoCreateTable
	// is true and BillingMode is PROVISIONED.
	ReadCapacityUnits int64

	// WriteCapacityUnits sets the provisioned write capacity when
	// AutoCreateTable is true and BillingMode is PROVISIONED.
	WriteCapacityUnits int64

	// MaxComputeRetries is the maximum number of optimistic lock retries.
	// Default is 10.
	MaxComputeRetries int

	// AutoCreateTable causes the provider to create the DynamoDB table and GSI
	// if they do not already exist.
	AutoCreateTable bool

	// ConsistentReads enables strongly consistent reads for all GetItem and
	// Query operations. By default, eventually consistent reads are used.
	ConsistentReads bool

	// CreateFieldGSIs causes AutoCreateTable to create Global Secondary
	// Indexes for TAG and sortable NUMERIC fields, enabling efficient range
	// queries without full-table scans when AutoCreateTable is true.
	CreateFieldGSIs bool
}

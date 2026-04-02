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

package cache_provider_valkey

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestApplyConfigDefaults_AllZero(t *testing.T) {
	t.Parallel()

	config := &Config{}
	applyConfigDefaults(config)

	assert.Equal(t, 1*time.Hour, config.DefaultTTL)
	assert.Equal(t, 2*time.Second, config.OperationTimeout)
	assert.Equal(t, 5*time.Second, config.AtomicOperationTimeout)
	assert.Equal(t, 10*time.Second, config.BulkOperationTimeout)
	assert.Equal(t, 30*time.Second, config.FlushTimeout)
	assert.Equal(t, 10, config.MaxComputeRetries)
	assert.Equal(t, 5*time.Second, config.SearchTimeout)
	assert.Equal(t, "index:", config.IndexPrefix)
}

func TestApplyConfigDefaults_PreservesSetValues(t *testing.T) {
	t.Parallel()

	config := &Config{
		DefaultTTL:        5 * time.Minute,
		OperationTimeout:  10 * time.Second,
		MaxComputeRetries: 3,
		IndexPrefix:       "custom:",
	}
	applyConfigDefaults(config)

	assert.Equal(t, 5*time.Minute, config.DefaultTTL)
	assert.Equal(t, 10*time.Second, config.OperationTimeout)
	assert.Equal(t, 3, config.MaxComputeRetries)
	assert.Equal(t, "custom:", config.IndexPrefix)

	assert.Equal(t, 5*time.Second, config.AtomicOperationTimeout)
	assert.Equal(t, 10*time.Second, config.BulkOperationTimeout)
	assert.Equal(t, 30*time.Second, config.FlushTimeout)
	assert.Equal(t, 5*time.Second, config.SearchTimeout)
}

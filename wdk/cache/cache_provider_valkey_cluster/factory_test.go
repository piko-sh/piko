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
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestApplyConfigDefaults_SetsZeroValues(t *testing.T) {
	t.Parallel()

	config := Config{}
	applyConfigDefaults(&config)

	require.NotZero(t, config.DefaultTTL)
	require.NotZero(t, config.OperationTimeout)
	require.NotZero(t, config.AtomicOperationTimeout)
	require.NotZero(t, config.BulkOperationTimeout)
	require.NotZero(t, config.FlushTimeout)
	require.NotZero(t, config.SearchTimeout)
	require.Greater(t, config.MaxComputeRetries, 0)
	require.NotEmpty(t, config.IndexPrefix)
}

func TestApplyConfigDefaults_DoesNotOverrideExplicitValues(t *testing.T) {
	t.Parallel()

	config := Config{
		DefaultTTL:             7 * time.Minute,
		OperationTimeout:       11 * time.Second,
		AtomicOperationTimeout: 3 * time.Second,
		BulkOperationTimeout:   13 * time.Second,
		FlushTimeout:           17 * time.Second,
		SearchTimeout:          19 * time.Second,
		MaxComputeRetries:      99,
		IndexPrefix:            "custom-prefix:",
	}
	applyConfigDefaults(&config)

	require.Equal(t, 7*time.Minute, config.DefaultTTL)
	require.Equal(t, 11*time.Second, config.OperationTimeout)
	require.Equal(t, 3*time.Second, config.AtomicOperationTimeout)
	require.Equal(t, 13*time.Second, config.BulkOperationTimeout)
	require.Equal(t, 17*time.Second, config.FlushTimeout)
	require.Equal(t, 19*time.Second, config.SearchTimeout)
	require.Equal(t, 99, config.MaxComputeRetries)
	require.Equal(t, "custom-prefix:", config.IndexPrefix)
}

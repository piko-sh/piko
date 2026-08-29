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

package cache_domain

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type weighedNamespace struct {
	weighted bool
	size     uint64
	maximum  uint64
}

func (w weighedNamespace) IsWeightBounded() bool { return w.weighted }
func (w weighedNamespace) WeightedSize() uint64  { return w.size }
func (w weighedNamespace) GetMaximum() uint64    { return w.maximum }

type enumerableProvider struct {
	namespaces map[string]any
}

func (e enumerableProvider) ListNamespaces() map[string]any { return e.namespaces }

type opaqueProvider struct{}

func TestTotalWeightedSize_SumsOnlyWeightBoundedNamespaces(t *testing.T) {
	t.Parallel()

	svc := &service{providers: map[string]any{
		"otter": enumerableProvider{namespaces: map[string]any{
			"sessions": weighedNamespace{weighted: true, size: 300, maximum: 1000},
			"blobs":    weighedNamespace{weighted: true, size: 700, maximum: 4000},
			"counters": weighedNamespace{weighted: false},
		}},
	}}

	aggregate := svc.TotalWeightedSize(context.Background())

	assert.Equal(t, uint64(1000), aggregate.TotalWeight)
	assert.Equal(t, uint64(5000), aggregate.DeclaredMaximum)
	assert.Equal(t, 2, aggregate.WeightedCaches)
	assert.Equal(t, 1, aggregate.UnweightedCaches,
		"a count-bounded cache contributes no bytes and must be reported separately, not summed in")
}

func TestTotalWeightedSize_CountsProvidersItCannotInspect(t *testing.T) {
	t.Parallel()

	svc := &service{providers: map[string]any{
		"otter": enumerableProvider{namespaces: map[string]any{
			"sessions": weighedNamespace{weighted: true, size: 128, maximum: 512},
		}},
		"redis": opaqueProvider{},
	}}

	aggregate := svc.TotalWeightedSize(context.Background())

	assert.Equal(t, uint64(128), aggregate.TotalWeight)
	assert.Equal(t, 1, aggregate.OpaqueProviders,
		"a total that silently omits a provider reads as complete when it is not")
}

func TestSetWeightBudget_IsReportedInTheAggregate(t *testing.T) {
	t.Parallel()

	svc := &service{providers: map[string]any{
		"otter": enumerableProvider{namespaces: map[string]any{
			"sessions": weighedNamespace{weighted: true, size: 10, maximum: 100},
		}},
	}}

	svc.SetWeightBudget(context.Background(), 4096)

	assert.Equal(t, uint64(4096), svc.TotalWeightedSize(context.Background()).Budget)
}

func TestSetWeightBudget_WarnsWhenTheDeclaredMaximaExceedIt(t *testing.T) {
	t.Parallel()

	svc := &service{providers: map[string]any{
		"otter": enumerableProvider{namespaces: map[string]any{
			"a": weighedNamespace{weighted: true, maximum: 3000},
			"b": weighedNamespace{weighted: true, maximum: 3000},
		}},
	}}

	svc.SetWeightBudget(context.Background(), 4096)

	aggregate := svc.TotalWeightedSize(context.Background())
	require.Greater(t, aggregate.DeclaredMaximum, aggregate.Budget,
		"this is the over-commitment the warning exists to catch")
}

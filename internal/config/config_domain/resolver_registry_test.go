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

package config_domain

import (
	"context"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type registryTestResolver struct {
	resolveFunc func(ctx context.Context, value string) (string, error)
	prefix      string
}

func (m *registryTestResolver) GetPrefix() string {
	return m.prefix
}

func (m *registryTestResolver) Resolve(ctx context.Context, value string) (string, error) {
	if m.resolveFunc != nil {
		return m.resolveFunc(ctx, value)
	}
	return value, nil
}

func TestResolverRegistryNew(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		test func(t *testing.T)
		name string
	}{
		{
			name: "creates new registry with empty resolvers",
			test: func(t *testing.T) {
				rr := newResolverRegistry()
				require.NotNil(t, rr)
				assert.Equal(t, 0, rr.Count())
			},
		},
		{
			name: "each call creates independent registry",
			test: func(t *testing.T) {
				rr1 := newResolverRegistry()
				rr2 := newResolverRegistry()
				assert.NotSame(t, rr1, rr2)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, tc.test)
	}
}

func TestResolverRegistryGlobalSingleton(t *testing.T) {
	testCases := []struct {
		test func(t *testing.T)
		name string
	}{
		{
			name: "returns same instance on multiple calls",
			test: func(t *testing.T) {
				ResetGlobalResolverRegistry()
				defer ResetGlobalResolverRegistry()

				rr1 := GetGlobalResolverRegistry()
				rr2 := GetGlobalResolverRegistry()
				assert.Same(t, rr1, rr2)
			},
		},
		{
			name: "reset creates new instance",
			test: func(t *testing.T) {
				ResetGlobalResolverRegistry()
				rr1 := GetGlobalResolverRegistry()

				ResetGlobalResolverRegistry()
				rr2 := GetGlobalResolverRegistry()

				assert.NotSame(t, rr1, rr2)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, tc.test)
	}
}

func TestResolverRegistryRegister(t *testing.T) {
	testCases := []struct {
		resolver   Resolver
		setup      func(*ResolverRegistry)
		name       string
		errMessage string
		wantErr    bool
	}{
		{
			name:     "registers resolver with valid prefix",
			resolver: &registryTestResolver{prefix: "test:"},
		},
		{
			name: "registers multiple resolvers",
			setup: func(rr *ResolverRegistry) {
				_ = rr.Register(&registryTestResolver{prefix: "first:"})
			},
			resolver: &registryTestResolver{prefix: "second:"},
		},
		{
			name: "replaces resolver with same prefix",
			setup: func(rr *ResolverRegistry) {
				_ = rr.Register(&registryTestResolver{prefix: "test:"})
			},
			resolver: &registryTestResolver{prefix: "test:"},
		},
		{
			name:       "fails with nil resolver",
			resolver:   nil,
			wantErr:    true,
			errMessage: "resolver cannot be nil",
		},
		{
			name:       "fails with empty prefix",
			resolver:   &registryTestResolver{prefix: ""},
			wantErr:    true,
			errMessage: "resolver prefix cannot be empty",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rr := newResolverRegistry()

			if tc.setup != nil {
				tc.setup(rr)
			}

			err := rr.Register(tc.resolver)

			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errMessage)
			} else {
				require.NoError(t, err)
				assert.True(t, rr.Has(tc.resolver.GetPrefix()))
			}
		})
	}
}

func TestResolverRegistryUnregister(t *testing.T) {
	testCases := []struct {
		name     string
		setup    func(*ResolverRegistry)
		prefix   string
		expected bool
	}{
		{
			name: "removes existing resolver",
			setup: func(rr *ResolverRegistry) {
				_ = rr.Register(&registryTestResolver{prefix: "test:"})
			},
			prefix:   "test:",
			expected: true,
		},
		{
			name:     "returns false for non-existent resolver",
			prefix:   "nonexistent:",
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rr := newResolverRegistry()

			if tc.setup != nil {
				tc.setup(rr)
			}

			removed := rr.Unregister(tc.prefix)
			assert.Equal(t, tc.expected, removed)

			if tc.expected {
				assert.False(t, rr.Has(tc.prefix))
			}
		})
	}
}

func TestResolverRegistryGet(t *testing.T) {
	testCases := []struct {
		name    string
		setup   func(*ResolverRegistry) Resolver
		prefix  string
		wantNil bool
	}{
		{
			name: "retrieves existing resolver",
			setup: func(rr *ResolverRegistry) Resolver {
				r := &registryTestResolver{prefix: "test:"}
				_ = rr.Register(r)
				return r
			},
			prefix:  "test:",
			wantNil: false,
		},
		{
			name:    "returns nil for non-existent resolver",
			prefix:  "nonexistent:",
			wantNil: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rr := newResolverRegistry()

			var expected Resolver
			if tc.setup != nil {
				expected = tc.setup(rr)
			}

			got := rr.Get(tc.prefix)

			if tc.wantNil {
				assert.Nil(t, got)
			} else {
				assert.Same(t, expected, got)
			}
		})
	}
}

func TestResolverRegistryGetAll(t *testing.T) {
	testCases := []struct {
		setup       func(*ResolverRegistry)
		name        string
		expectedLen int
	}{
		{
			name:        "returns empty slice when no resolvers",
			expectedLen: 0,
		},
		{
			name: "returns all registered resolvers",
			setup: func(rr *ResolverRegistry) {
				_ = rr.Register(&registryTestResolver{prefix: "first:"})
				_ = rr.Register(&registryTestResolver{prefix: "second:"})
				_ = rr.Register(&registryTestResolver{prefix: "third:"})
			},
			expectedLen: 3,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rr := newResolverRegistry()

			if tc.setup != nil {
				tc.setup(rr)
			}

			resolvers := rr.GetAll()
			assert.Len(t, resolvers, tc.expectedLen)
		})
	}
}

func TestResolverRegistryGetPrefixes(t *testing.T) {
	testCases := []struct {
		name             string
		setup            func(*ResolverRegistry)
		expectedPrefixes []string
	}{
		{
			name:             "returns empty slice when no resolvers",
			expectedPrefixes: []string{},
		},
		{
			name: "returns all registered prefixes",
			setup: func(rr *ResolverRegistry) {
				_ = rr.Register(&registryTestResolver{prefix: "alpha:"})
				_ = rr.Register(&registryTestResolver{prefix: "beta:"})
			},
			expectedPrefixes: []string{"alpha:", "beta:"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rr := newResolverRegistry()

			if tc.setup != nil {
				tc.setup(rr)
			}

			prefixes := rr.GetPrefixes()

			sort.Strings(prefixes)
			sort.Strings(tc.expectedPrefixes)

			assert.Equal(t, tc.expectedPrefixes, prefixes)
		})
	}
}

func TestResolverRegistryHas(t *testing.T) {
	testCases := []struct {
		name     string
		setup    func(*ResolverRegistry)
		prefix   string
		expected bool
	}{
		{
			name: "returns true for existing resolver",
			setup: func(rr *ResolverRegistry) {
				_ = rr.Register(&registryTestResolver{prefix: "test:"})
			},
			prefix:   "test:",
			expected: true,
		},
		{
			name:     "returns false for non-existent resolver",
			prefix:   "nonexistent:",
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rr := newResolverRegistry()

			if tc.setup != nil {
				tc.setup(rr)
			}

			assert.Equal(t, tc.expected, rr.Has(tc.prefix))
		})
	}
}

func TestResolverRegistryClear(t *testing.T) {
	rr := newResolverRegistry()

	_ = rr.Register(&registryTestResolver{prefix: "first:"})
	_ = rr.Register(&registryTestResolver{prefix: "second:"})
	assert.Equal(t, 2, rr.Count())

	rr.Clear()
	assert.Equal(t, 0, rr.Count())
}

func TestResolverRegistryCount(t *testing.T) {
	rr := newResolverRegistry()
	assert.Equal(t, 0, rr.Count())

	_ = rr.Register(&registryTestResolver{prefix: "first:"})
	assert.Equal(t, 1, rr.Count())

	_ = rr.Register(&registryTestResolver{prefix: "second:"})
	assert.Equal(t, 2, rr.Count())

	rr.Unregister("first:")
	assert.Equal(t, 1, rr.Count())
}

func TestResolverRegistryConcurrentAccess(t *testing.T) {
	t.Parallel()

	rr := newResolverRegistry()

	done := make(chan bool)
	for i := range 10 {
		go func(index int) {
			prefix := string(rune('a'+index)) + ":"
			_ = rr.Register(&registryTestResolver{prefix: prefix})
			_ = rr.Has(prefix)
			_ = rr.Get(prefix)
			_ = rr.GetAll()
			_ = rr.GetPrefixes()
			_ = rr.Count()
			done <- true
		}(i)
	}

	for range 10 {
		<-done
	}

	assert.True(t, rr.Count() <= 10)
}

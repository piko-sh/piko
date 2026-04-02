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

package config_resolver_gcp

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"piko.sh/piko/wdk/config"
)

func TestGetPrefix(t *testing.T) {
	t.Parallel()

	r := &Resolver{}

	assert.Equal(t, "gcp-secret:", r.GetPrefix())
}

func TestResolve_ValidationErrors(t *testing.T) {
	t.Parallel()

	r := &Resolver{}

	t.Run("empty value", func(t *testing.T) {
		t.Parallel()

		_, err := r.Resolve(context.Background(), "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "must not be empty")
	})

	t.Run("only json key separator", func(t *testing.T) {
		t.Parallel()

		_, err := r.Resolve(context.Background(), "#mykey")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "must not be empty")
	})
}

var _ config.Resolver = (*Resolver)(nil)

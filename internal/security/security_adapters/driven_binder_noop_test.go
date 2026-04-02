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

package security_adapters

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNoOpBinderAdapter_GetBindingIdentifier(t *testing.T) {
	t.Parallel()

	adapter := NewNoOpBinderAdapter()

	t.Run("valid request returns context_none", func(t *testing.T) {
		t.Parallel()

		r := httptest.NewRequest("GET", "/", nil)
		result := adapter.GetBindingIdentifier(r)
		assert.Equal(t, "context_none", result)
	})

	t.Run("nil request returns context_none", func(t *testing.T) {
		t.Parallel()

		result := adapter.GetBindingIdentifier(nil)
		assert.Equal(t, "context_none", result)
	})
}

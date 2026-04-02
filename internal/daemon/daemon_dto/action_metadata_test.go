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

package daemon_dto

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestActionMetadata_Request(t *testing.T) {
	t.Parallel()

	t.Run("nil by default", func(t *testing.T) {
		t.Parallel()

		m := &ActionMetadata{}
		assert.Nil(t, m.Request())
	})

	t.Run("after set", func(t *testing.T) {
		t.Parallel()

		request := &RequestMetadata{Method: "POST", Path: "/api/test"}
		m := &ActionMetadata{}
		m.SetRequest(request)

		assert.Equal(t, "POST", m.Request().Method)
		assert.Equal(t, "/api/test", m.Request().Path)
	})
}

func TestActionMetadata_Response(t *testing.T) {
	t.Parallel()

	t.Run("nil by default", func(t *testing.T) {
		t.Parallel()

		m := &ActionMetadata{}
		assert.Nil(t, m.Response())
	})

	t.Run("after set", func(t *testing.T) {
		t.Parallel()

		response := NewResponseWriter()
		m := &ActionMetadata{}
		m.SetResponse(response)

		assert.NotNil(t, m.Response())
	})
}

func TestActionMetadata_InheritMetadata(t *testing.T) {
	t.Parallel()

	request := &RequestMetadata{Method: "GET"}
	response := NewResponseWriter()

	source := &ActionMetadata{}
	source.SetRequest(request)
	source.SetResponse(response)

	target := &ActionMetadata{}
	target.InheritMetadata(source)

	assert.Equal(t, "GET", target.Request().Method)
	assert.Equal(t, response, target.Response())
}

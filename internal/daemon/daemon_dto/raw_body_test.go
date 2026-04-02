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
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRawBody(t *testing.T) {
	t.Parallel()

	data := []byte(`{"key":"value"}`)
	body := NewRawBody("application/json", data)

	assert.Equal(t, "application/json", body.ContentType)
	assert.Equal(t, int64(15), body.Size)
	assert.Equal(t, data, body.Bytes())
}

func TestRawBody_Bytes(t *testing.T) {
	t.Parallel()

	data := []byte("hello world")
	body := NewRawBody("text/plain", data)

	assert.Equal(t, data, body.Bytes())
}

func TestRawBody_String(t *testing.T) {
	t.Parallel()

	body := NewRawBody("text/plain", []byte("hello world"))
	assert.Equal(t, "hello world", body.String())
}

func TestRawBody_Reader(t *testing.T) {
	t.Parallel()

	data := []byte("reader test")
	body := NewRawBody("text/plain", data)

	reader := body.Reader()
	require.NotNil(t, reader)

	read, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, data, read)
}

func TestRawBody_Empty(t *testing.T) {
	t.Parallel()

	body := NewRawBody("", nil)

	assert.Equal(t, int64(0), body.Size)
	assert.Empty(t, body.Bytes())
	assert.Empty(t, body.String())
}

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

package capabilities_functions

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/capabilities/capabilities_domain"
)

func TestFatalReader_NormalRead(t *testing.T) {
	t.Parallel()

	inner := strings.NewReader("hello world")
	fr := newFatalReader(inner)

	buffer := make([]byte, 64)
	n, err := fr.Read(buffer)

	require.NoError(t, err)
	assert.Equal(t, "hello world", string(buffer[:n]))
}

func TestFatalReader_EOF(t *testing.T) {
	t.Parallel()

	inner := strings.NewReader("")
	fr := newFatalReader(inner)

	buffer := make([]byte, 64)
	n, err := fr.Read(buffer)

	assert.Equal(t, 0, n)
	assert.ErrorIs(t, err, io.EOF, "io.EOF should be passed through unchanged, not wrapped as fatal")
	assert.False(t, capabilities_domain.IsFatalError(err), "io.EOF must not be recognised as a fatal error")
}

func TestFatalReader_NonEOFError(t *testing.T) {
	t.Parallel()

	readErr := errors.New("disk failure")
	fr := newFatalReader(&failingReader{err: readErr})

	buffer := make([]byte, 64)
	_, err := fr.Read(buffer)

	require.Error(t, err)
	assert.True(t, capabilities_domain.IsFatalError(err), "non-EOF read errors should be wrapped as fatal")
	assert.True(t, errors.Is(err, readErr), "the original error should remain in the chain")
}

func TestFatalReader_ErrorPreservesBytes(t *testing.T) {
	t.Parallel()

	fr := newFatalReader(&partialReader{
		data: []byte("partial"),
		err:  errors.New("mid-stream failure"),
	})

	buffer := make([]byte, 64)
	n, err := fr.Read(buffer)

	require.Error(t, err)
	assert.True(t, capabilities_domain.IsFatalError(err), "the error should be wrapped as fatal")
	assert.Equal(t, "partial", string(buffer[:n]), "bytes read before the error should still be returned")
}

type failingReader struct {
	err error
}

func (r *failingReader) Read(_ []byte) (int, error) {
	return 0, r.err
}

type partialReader struct {
	err  error
	data []byte
	done bool
}

func (r *partialReader) Read(p []byte) (int, error) {
	if r.done {
		return 0, io.EOF
	}
	r.done = true
	n := copy(p, r.data)
	return n, r.err
}

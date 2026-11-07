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

package cli

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type closeRecorder struct {
	io.Reader
	closed bool
}

func (c *closeRecorder) Close() error {
	c.closed = true
	return nil
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) {
	return 0, errors.New("read failed")
}

func TestReadBoundedBody(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("too large")

	data, err := readBoundedBody(strings.NewReader("abcd"), 4, sentinel)
	require.NoError(t, err)
	assert.Equal(t, "abcd", string(data))

	_, err = readBoundedBody(strings.NewReader("abcde"), 4, sentinel)
	require.ErrorIs(t, err, sentinel)

	_, err = readBoundedBody(failingReader{}, 4, sentinel)
	require.Error(t, err)
	assert.NotErrorIs(t, err, sentinel)
}

func TestDrainAndClose(t *testing.T) {
	t.Parallel()

	remaining := strings.NewReader(strings.Repeat("x", 128))
	body := &closeRecorder{Reader: remaining}

	drainAndClose(body)

	assert.True(t, body.closed)
	assert.Zero(t, remaining.Len(), "the unread remainder is discarded")

	large := strings.NewReader(strings.Repeat("x", 2*maxDrainBytes))
	largeBody := &closeRecorder{Reader: large}
	drainAndClose(largeBody)
	assert.True(t, largeBody.closed)
	assert.Equal(t, maxDrainBytes, large.Len(), "draining stops at the cap")
}

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

//go:build !bench

package logger_domain_test

import (
	"bytes"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/logger/logger_domain"
)

func TestLockedWriter_Write(t *testing.T) {
	t.Parallel()

	t.Run("WritesDataToUnderlyingWriter", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		var mu sync.Mutex
		writer := logger_domain.NewLockedWriter(&buf, &mu)

		n, err := writer.Write([]byte("hello"))

		require.NoError(t, err)
		assert.Equal(t, 5, n)
		assert.Equal(t, "hello", buf.String())
	})

	t.Run("SerialisesWrites", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		var mu sync.Mutex
		writer := logger_domain.NewLockedWriter(&buf, &mu)

		var wg sync.WaitGroup
		for range 100 {
			wg.Go(func() {
				_, _ = writer.Write([]byte("x"))
			})
		}
		wg.Wait()

		assert.Equal(t, 100, buf.Len())
	})
}

func TestLockedWriter_HoldWrites(t *testing.T) {
	t.Parallel()

	t.Run("BlocksWritesUntilReleased", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		var mu sync.Mutex
		writer := logger_domain.NewLockedWriter(&buf, &mu)

		release := writer.HoldWrites()

		started := make(chan struct{})
		done := make(chan struct{})
		go func() {
			close(started)
			_, _ = writer.Write([]byte("after"))
			close(done)
		}()

		<-started

		buf.WriteString("banner")
		release()

		<-done

		assert.Equal(t, "bannerafter", buf.String())
	})
}

func TestLockedWriter_SharedMutex(t *testing.T) {
	t.Parallel()

	t.Run("TwoWritersSameMutexSerialise", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		var mu sync.Mutex
		writer1 := logger_domain.NewLockedWriter(&buf, &mu)
		writer2 := logger_domain.NewLockedWriter(&buf, &mu)

		var wg sync.WaitGroup
		for range 50 {
			wg.Go(func() {
				_, _ = writer1.Write([]byte("a"))
			})
			wg.Go(func() {
				_, _ = writer2.Write([]byte("b"))
			})
		}
		wg.Wait()

		assert.Equal(t, 100, buf.Len())
	})
}

func TestStderrWriter(t *testing.T) {
	t.Parallel()

	t.Run("ReturnsSameInstance", func(t *testing.T) {
		t.Parallel()

		writer1 := logger_domain.StderrWriter()
		writer2 := logger_domain.StderrWriter()

		assert.Same(t, writer1, writer2)
	})

	t.Run("IsNotNil", func(t *testing.T) {
		t.Parallel()

		assert.NotNil(t, logger_domain.StderrWriter())
	})
}

func TestStdoutWriter(t *testing.T) {
	t.Parallel()

	t.Run("ReturnsSameInstance", func(t *testing.T) {
		t.Parallel()

		writer1 := logger_domain.StdoutWriter()
		writer2 := logger_domain.StdoutWriter()

		assert.Same(t, writer1, writer2)
	})

	t.Run("IsNotNil", func(t *testing.T) {
		t.Parallel()

		assert.NotNil(t, logger_domain.StdoutWriter())
	})

	t.Run("SharesMutexWithStderrWriter", func(t *testing.T) {
		// Holding the stderr writer's lock must block a stdout write. This
		// guarantees the startup banner (stderr) cannot be interleaved with
		// log output (stdout) at the kernel TTY level.
		release := logger_domain.StderrWriter().HoldWrites()

		done := make(chan struct{})
		go func() {
			_, _ = logger_domain.StdoutWriter().Write([]byte{})
			close(done)
		}()

		select {
		case <-done:
			release()
			t.Fatal("stdout write completed while stderr HoldWrites was active; mutex is not shared")
		case <-time.After(50 * time.Millisecond):
		}

		release()
		<-done
	})
}

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

package llm_provider_ollama

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/logger"
)

const (
	fakeOllamaModeVariable        = "OLLAMA_PIKO_TEST_MODE"
	fakeOllamaModeServe           = "serve"
	fakeOllamaModeExit            = "exit"
	fakeOllamaModeIgnoreInterrupt = "ignore-interrupt"
	fakeOllamaExitCode            = 3
	fakeOllamaGreeting            = "fake ollama starting"
)

type capturedRecord struct {
	attributes map[string]string
	message    string
	level      slog.Level
}

type captureHandler struct {
	records *[]capturedRecord
	mu      *sync.Mutex
}

func (*captureHandler) Enabled(context.Context, slog.Level) bool { return true }

func (h *captureHandler) Handle(_ context.Context, record slog.Record) error {
	captured := capturedRecord{
		attributes: map[string]string{},
		message:    record.Message,
		level:      record.Level,
	}
	record.Attrs(func(attribute slog.Attr) bool {
		captured.attributes[attribute.Key] = attribute.Value.String()
		return true
	})

	h.mu.Lock()
	defer h.mu.Unlock()
	*h.records = append(*h.records, captured)
	return nil
}

func (h *captureHandler) WithAttrs([]slog.Attr) slog.Handler { return h }

func (h *captureHandler) WithGroup(string) slog.Handler { return h }

func (h *captureHandler) snapshot() []capturedRecord {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]capturedRecord(nil), *h.records...)
}

func init() {
	mode := os.Getenv(fakeOllamaModeVariable)
	if mode == "" || len(os.Args) < 2 || os.Args[1] != "serve" {
		return
	}
	os.Exit(runFakeOllama(mode))
}

func runFakeOllama(mode string) int {
	_, _ = fmt.Fprintln(os.Stderr, fakeOllamaGreeting)
	_, _ = fmt.Fprintln(os.Stderr, strings.Repeat("x", maxOutputLineBytes+100))

	if mode == fakeOllamaModeExit {
		return fakeOllamaExitCode
	}

	host, err := url.Parse(os.Getenv("OLLAMA_HOST"))
	if err != nil {
		return 1
	}
	listener, err := net.Listen("tcp", host.Host)
	if err != nil {
		return 1
	}

	interrupts := make(chan os.Signal, 1)
	signal.Notify(interrupts, os.Interrupt)

	mux := http.NewServeMux()
	mux.HandleFunc(ollamaVersionPath, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"version":"0.0.0"}`)
	})
	server := &http.Server{Handler: mux, ReadHeaderTimeout: time.Second}
	go func() {
		_ = server.Serve(listener)
	}()

	if mode == fakeOllamaModeIgnoreInterrupt {
		for range interrupts {
			_, _ = fmt.Fprintln(os.Stderr, "ignoring interrupt")
		}
	}
	<-interrupts
	return 0
}

func newCaptureContext(t *testing.T) (context.Context, *captureHandler) {
	t.Helper()

	handler := &captureHandler{records: &[]capturedRecord{}, mu: &sync.Mutex{}}
	captureLogger := logger_domain.New(slog.New(handler), "ollama-test")
	return logger.WithLogger(t.Context(), captureLogger), handler
}

func freeLoopbackHost(t *testing.T) string {
	t.Helper()

	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	require.NoError(t, err)
	address := listener.Addr().String()
	require.NoError(t, listener.Close())
	return "http://" + address
}

func fakeOllamaConfig(t *testing.T, mode string) Config {
	t.Helper()

	t.Setenv(fakeOllamaModeVariable, mode)

	executable, err := os.Executable()
	require.NoError(t, err)

	return Config{
		Host:            freeLoopbackHost(t),
		BinaryPath:      executable,
		StartupTimeout:  20 * time.Second,
		ProbeTimeout:    time.Second,
		StopGracePeriod: 10 * time.Second,
	}.WithDefaults()
}

func requireExited(t *testing.T, process *managedProcess) {
	t.Helper()

	select {
	case <-process.done:
	case <-time.After(10 * time.Second):
		require.FailNow(t, "managed process was not reaped")
	}
}

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
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildManagedOllamaEnv_PreservesOnlyRelevantVariables(t *testing.T) {
	t.Parallel()

	env := []string{
		"PATH=/usr/bin",
		"HOME=/home/test",
		"TMPDIR=/tmp/piko",
		"HTTP_PROXY=http://proxy.internal:8080",
		"SSL_CERT_FILE=/etc/ssl/custom.pem",
		"OLLAMA_MODELS=/srv/ollama/models",
		"OLLAMA_HOST=http://old-host:11434",
		"AWS_SECRET_ACCESS_KEY=super-secret",
		"GITHUB_TOKEN=top-secret",
		"MALFORMED",
		"=no-key",
	}

	got := buildManagedOllamaEnv(env, "http://127.0.0.1:11434")

	assert.Contains(t, got, "OLLAMA_HOST=http://127.0.0.1:11434")
	assert.Contains(t, got, "PATH=/usr/bin")
	assert.Contains(t, got, "HOME=/home/test")
	assert.Contains(t, got, "TMPDIR=/tmp/piko")
	assert.Contains(t, got, "HTTP_PROXY=http://proxy.internal:8080")
	assert.Contains(t, got, "SSL_CERT_FILE=/etc/ssl/custom.pem")
	assert.Contains(t, got, "OLLAMA_MODELS=/srv/ollama/models")

	assert.NotContains(t, got, "OLLAMA_HOST=http://old-host:11434")
	assert.NotContains(t, got, "AWS_SECRET_ACCESS_KEY=super-secret")
	assert.NotContains(t, got, "GITHUB_TOKEN=top-secret")
	assert.NotContains(t, got, "MALFORMED")
	assert.Equal(t, 1, countEnvKey(got, "OLLAMA_HOST"))
}

func TestNewManagedOllamaCommand_UsesExplicitEnvironmentAndExplicitIO(t *testing.T) {
	t.Parallel()

	command := newManagedOllamaCommand("/usr/bin/ollama", "http://127.0.0.1:11434", []string{
		"PATH=/usr/bin",
		"HOME=/home/test",
		"AWS_SECRET_ACCESS_KEY=super-secret",
	})

	require.NotNil(t, command)
	assert.Equal(t, "/usr/bin/ollama", command.Path)
	assert.Equal(t, []string{"/usr/bin/ollama", "serve"}, command.Args)
	assert.Contains(t, command.Env, "OLLAMA_HOST=http://127.0.0.1:11434")
	assert.Contains(t, command.Env, "PATH=/usr/bin")
	assert.Contains(t, command.Env, "HOME=/home/test")
	assert.NotContains(t, command.Env, "AWS_SECRET_ACCESS_KEY=super-secret")

	assert.Nil(t, command.Stdin)
	assert.Equal(t, io.Discard, command.Stdout)
}

func TestManagedProcessStop(t *testing.T) {
	t.Parallel()

	errDenied := errors.New("operation denied")

	testCases := []struct {
		interruptErr    error
		killErr         error
		wantErr         string
		name            string
		gracePeriod     time.Duration
		reapTimeout     time.Duration
		wantInterrupts  int32
		wantKills       int32
		alreadyExited   bool
		exitOnInterrupt bool
		exitOnKill      bool
		cancelContext   bool
	}{
		{
			name:            "exits after interrupt without being killed",
			gracePeriod:     time.Hour,
			reapTimeout:     time.Hour,
			exitOnInterrupt: true,
			exitOnKill:      true,
			wantInterrupts:  1,
			wantKills:       0,
		},
		{
			name:           "kills immediately when the interrupt cannot be delivered",
			interruptErr:   errors.New("not supported by windows"),
			gracePeriod:    time.Hour,
			reapTimeout:    time.Hour,
			exitOnKill:     true,
			wantInterrupts: 1,
			wantKills:      1,
		},
		{
			name:           "kills once the grace period elapses",
			gracePeriod:    10 * time.Millisecond,
			reapTimeout:    time.Hour,
			exitOnKill:     true,
			wantInterrupts: 1,
			wantKills:      1,
		},
		{
			name:           "kills immediately when the context is cancelled",
			gracePeriod:    time.Hour,
			reapTimeout:    time.Hour,
			exitOnKill:     true,
			cancelContext:  true,
			wantInterrupts: 1,
			wantKills:      1,
		},
		{
			name:           "does nothing when the process already exited",
			gracePeriod:    time.Hour,
			reapTimeout:    time.Hour,
			alreadyExited:  true,
			wantInterrupts: 0,
			wantKills:      0,
		},
		{
			name:           "reports a kill failure",
			interruptErr:   errDenied,
			killErr:        errDenied,
			gracePeriod:    time.Hour,
			reapTimeout:    time.Hour,
			wantErr:        "killing managed ollama server",
			wantInterrupts: 1,
			wantKills:      1,
		},
		{
			name:           "treats a vanished process as stopped",
			interruptErr:   os.ErrProcessDone,
			killErr:        os.ErrProcessDone,
			gracePeriod:    time.Hour,
			reapTimeout:    time.Hour,
			exitOnKill:     true,
			wantInterrupts: 1,
			wantKills:      1,
		},
		{
			name:           "reports a process that is not reaped in time",
			interruptErr:   errDenied,
			gracePeriod:    time.Hour,
			reapTimeout:    10 * time.Millisecond,
			wantErr:        "was not reaped",
			wantInterrupts: 1,
			wantKills:      1,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			var interrupts, kills atomic.Int32
			process := &managedProcess{
				command:         &exec.Cmd{Process: &os.Process{Pid: os.Getpid()}},
				done:            make(chan struct{}),
				stopGracePeriod: testCase.gracePeriod,
				reapTimeout:     testCase.reapTimeout,
			}
			process.interrupt = func(*exec.Cmd) error {
				interrupts.Add(1)
				if testCase.exitOnInterrupt {
					close(process.done)
				}
				return testCase.interruptErr
			}
			process.kill = func(*exec.Cmd) error {
				kills.Add(1)
				if testCase.exitOnKill {
					close(process.done)
				}
				return testCase.killErr
			}
			if testCase.alreadyExited {
				close(process.done)
			}

			ctx := t.Context()
			if testCase.cancelContext {
				cancelled, cancel := context.WithCancelCause(ctx)
				cancel(errors.New("shutting down"))
				ctx = cancelled
			}

			err := process.Stop(ctx)

			if testCase.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), testCase.wantErr)
			}
			assert.Equal(t, testCase.wantInterrupts, interrupts.Load())
			assert.Equal(t, testCase.wantKills, kills.Load())
			assert.True(t, process.stopping.Load())
		})
	}
}

func TestManagedProcessStop_NilProcess(t *testing.T) {
	t.Parallel()

	var missing *managedProcess
	require.NoError(t, missing.Stop(t.Context()))
	require.NoError(t, (&managedProcess{}).Stop(t.Context()))
	require.NoError(t, (&managedProcess{command: &exec.Cmd{}}).Stop(t.Context()))
}

func TestManagedProcessExitError(t *testing.T) {
	t.Parallel()

	clean := &managedProcess{}
	assert.Equal(t, errManagedProcessExited, clean.exitError())

	cause := errors.New("exit status 3")
	failed := &managedProcess{exitErr: cause}
	assert.ErrorIs(t, failed.exitError(), errManagedProcessExited)
	assert.ErrorIs(t, failed.exitError(), cause)
}

func TestResolveOllamaBinary(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	configured, err := resolveOllamaBinary("/opt/ollama/bin/ollama")
	require.NoError(t, err)
	assert.Equal(t, "/opt/ollama/bin/ollama", configured)

	_, err = resolveOllamaBinary("")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ollama binary not found")
}

func TestStartOllama_StartsAndStopsManagedServer(t *testing.T) {
	config := fakeOllamaConfig(t, fakeOllamaModeServe)
	ctx, records := newCaptureContext(t)

	process, err := startOllama(ctx, &config)
	require.NoError(t, err)
	require.NotNil(t, process)
	assert.True(t, process.healthy.Load())

	healthURL, err := healthURLFor(config.Host)
	require.NoError(t, err)
	require.NoError(t, probeOnce(ctx, healthURL, time.Second))

	require.NoError(t, process.Stop(ctx))
	requireExited(t, process)
	assert.NoError(t, process.exitErr)

	var sawGreeting, sawTruncatedLine bool
	for _, record := range records.snapshot() {
		if record.message != "Ollama output" {
			continue
		}
		if record.attributes["output"] == fakeOllamaGreeting {
			sawGreeting = true
		}
		if record.attributes["truncated_bytes"] == strconv.Itoa(100) {
			sawTruncatedLine = true
			assert.Len(t, record.attributes["output"], maxOutputLineBytes)
		}
	}
	assert.True(t, sawGreeting, "the server's output is relayed to the logger")
	assert.True(t, sawTruncatedLine, "an over-long output line is cut and reported")
}

func TestStartOllama_ReportsEarlyExit(t *testing.T) {
	config := fakeOllamaConfig(t, fakeOllamaModeExit)

	process, err := startOllama(t.Context(), &config)
	require.Error(t, err)
	assert.Nil(t, process)
	assert.ErrorIs(t, err, errManagedProcessExited)
	exitErr, ok := errors.AsType[*exec.ExitError](err)
	require.True(t, ok, "the exit status is kept in the error chain")
	assert.Equal(t, fakeOllamaExitCode, exitErr.ExitCode())
}

func TestStartOllama_KillsServerThatIgnoresInterrupt(t *testing.T) {
	config := fakeOllamaConfig(t, fakeOllamaModeIgnoreInterrupt)
	config.StopGracePeriod = 200 * time.Millisecond

	process, err := startOllama(t.Context(), &config)
	require.NoError(t, err)

	require.NoError(t, process.Stop(t.Context()))
	requireExited(t, process)
	assert.Error(t, process.exitErr)
}

func TestStartOllama_FailsWhenBinaryCannotStart(t *testing.T) {
	t.Parallel()

	config := Config{Host: "http://127.0.0.1:1", BinaryPath: "/nonexistent/ollama"}.WithDefaults()

	process, err := startOllama(t.Context(), &config)
	require.Error(t, err)
	assert.Nil(t, process)
	assert.Contains(t, err.Error(), "starting ollama serve")
}

func TestWaitForHealth(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		wantErrText    string
		failuresBefore int32
		alwaysFail     bool
		processExited  bool
	}{
		{name: "healthy immediately", failuresBefore: 0},
		{name: "healthy after failed probes", failuresBefore: 2},
		{name: "times out keeping the last probe failure", alwaysFail: true, wantErrText: "unexpected status 503"},
		{name: "stops when the process exits", alwaysFail: true, processExited: true, wantErrText: "managed ollama process exited"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			var probes atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				if testCase.alwaysFail || probes.Add(1) <= testCase.failuresBefore {
					w.WriteHeader(http.StatusServiceUnavailable)
					return
				}
				w.WriteHeader(http.StatusOK)
			}))
			t.Cleanup(server.Close)

			config := Config{
				Host:           server.URL,
				StartupTimeout: 2 * time.Second,
				ProbeTimeout:   time.Second,
			}.WithDefaults()

			exited := make(chan struct{})
			if testCase.processExited {
				close(exited)
			}

			err := waitForHealth(t.Context(), &config, exited)

			if testCase.wantErrText == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), testCase.wantErrText)
		})
	}
}

func TestWaitForHealth_KeepsCancellationCause(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	t.Cleanup(server.Close)

	config := Config{Host: server.URL, StartupTimeout: time.Hour}.WithDefaults()
	cause := errors.New("application shutting down")
	ctx, cancel := context.WithCancelCause(t.Context())
	cancel(cause)

	err := waitForHealth(ctx, &config, make(chan struct{}))
	require.Error(t, err)
	assert.ErrorIs(t, err, cause)
}

func TestWaitForHealth_RejectsInvalidHost(t *testing.T) {
	t.Parallel()

	config := Config{Host: "://missing-scheme"}.WithDefaults()

	err := waitForHealth(t.Context(), &config, make(chan struct{}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "building ollama health URL")
}

func TestProbeServer(t *testing.T) {
	t.Parallel()

	healthy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"version":"0.0.0"}`)
	}))
	t.Cleanup(healthy.Close)

	failing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(failing.Close)

	stopped := httptest.NewServer(http.NotFoundHandler())
	stoppedURL := stopped.URL
	stopped.Close()

	testCases := []struct {
		name        string
		host        string
		wantErrText string
	}{
		{name: "healthy server", host: healthy.URL},
		{name: "server error", host: failing.URL, wantErrText: "unexpected status 500"},
		{name: "unreachable server", host: stoppedURL, wantErrText: "probing ollama"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			healthURL, err := healthURLFor(testCase.host)
			require.NoError(t, err)

			err = probeOnce(t.Context(), healthURL, time.Second)
			if testCase.wantErrText == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), testCase.wantErrText)
		})
	}
}

func TestProbeServer_RejectsInvalidURL(t *testing.T) {
	t.Parallel()

	err := probeServer(t.Context(), newProbeClient(time.Second), "http://[::1]:namedport/")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "building ollama probe request")
}

func TestNewProbeClient(t *testing.T) {
	t.Parallel()

	client := newProbeClient(3 * time.Second)

	assert.Equal(t, 3*time.Second, client.Timeout)
	transport, ok := client.Transport.(*http.Transport)
	require.True(t, ok)
	assert.True(t, transport.DisableKeepAlives)
}

func TestEnsureServerRunning(t *testing.T) {
	t.Parallel()

	reachable := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(reachable.Close)

	unreachable := httptest.NewServer(http.NotFoundHandler())
	unreachableHost := unreachable.URL
	unreachable.Close()

	testCases := []struct {
		autoStart   *bool
		name        string
		host        string
		binaryPath  string
		wantErrText string
	}{
		{name: "reachable server needs no managed instance", host: reachable.URL},
		{
			name:        "unreachable server with auto-start disabled",
			host:        unreachableHost,
			autoStart:   new(false),
			wantErrText: "AutoStart is disabled",
		},
		{
			name:        "unreachable server with unset auto-start tries to start one",
			host:        unreachableHost,
			binaryPath:  "/nonexistent/ollama",
			wantErrText: "auto-starting ollama",
		},
		{
			name:        "invalid host",
			host:        "://missing-scheme",
			wantErrText: "building ollama health URL",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			config := Config{
				Host:         testCase.host,
				BinaryPath:   testCase.binaryPath,
				ProbeTimeout: time.Second,
			}
			config.AutoStart = testCase.autoStart

			process, err := ensureServerRunning(t.Context(), &config)
			assert.Nil(t, process)
			if testCase.wantErrText == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), testCase.wantErrText)
		})
	}
}

func TestEnsureServerRunning_StartsManagedServer(t *testing.T) {
	config := fakeOllamaConfig(t, fakeOllamaModeServe)

	process, err := ensureServerRunning(t.Context(), &config)
	require.NoError(t, err)
	require.NotNil(t, process)

	require.NoError(t, process.Stop(t.Context()))
	requireExited(t, process)
}

func countEnvKey(env []string, key string) int {
	count := 0
	for _, entry := range env {
		if len(entry) > len(key) && entry[:len(key)+1] == key+"=" {
			count++
		}
	}
	return count
}

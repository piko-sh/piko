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
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"sync/atomic"
	"time"

	"piko.sh/piko/wdk/goroutine"
	"piko.sh/piko/wdk/logger"
)

const (
	// healthPollInterval is the interval between health status checks.
	healthPollInterval = 250 * time.Millisecond

	// killWaitTimeout bounds the wait for a killed managed process to be reaped.
	killWaitTimeout = 5 * time.Second

	// outputWaitDelay bounds how long Wait keeps reading output pipes after the managed
	// process exits, in case a child process still holds them open.
	outputWaitDelay = 2 * time.Second

	// ollamaVersionPath is the Ollama endpoint used for health and reachability probes.
	ollamaVersionPath = "/api/version"
)

var (
	// errManagedProcessExited is returned when the managed Ollama process exits before it
	// becomes healthy.
	errManagedProcessExited = errors.New("managed ollama process exited")

	// managedOllamaEnvKeys holds the set of environment variable names that are forwarded to
	// the managed Ollama process.
	managedOllamaEnvKeys = map[string]struct{}{
		"APPDATA":           {},
		"COMSPEC":           {},
		"DYLD_LIBRARY_PATH": {},
		"HOME":              {},
		"HTTPS_PROXY":       {},
		"HTTP_PROXY":        {},
		"LANG":              {},
		"LC_ALL":            {},
		"LC_CTYPE":          {},
		"LD_LIBRARY_PATH":   {},
		"LOCALAPPDATA":      {},
		"NO_PROXY":          {},
		"PATH":              {},
		"PROGRAMDATA":       {},
		"SSL_CERT_DIR":      {},
		"SSL_CERT_FILE":     {},
		"SYSTEMROOT":        {},
		"TEMP":              {},
		"TMP":               {},
		"TMPDIR":            {},
		"USERPROFILE":       {},
		"XDG_CACHE_HOME":    {},
		"XDG_CONFIG_HOME":   {},
		"XDG_DATA_HOME":     {},
	}
)

// managedProcess tracks an Ollama subprocess spawned by the provider.
type managedProcess struct {
	// command is the running ollama serve process.
	command *exec.Cmd

	// done is closed once the process has exited and been reaped.
	done chan struct{}

	// interrupt sends a graceful shutdown signal to the managed process tree.
	interrupt func(*exec.Cmd) error

	// kill forcefully terminates the managed process tree.
	kill func(*exec.Cmd) error

	// exitErr is the result of waiting for the process; it is written before done is closed
	// and must only be read after done is closed.
	exitErr error

	// stopGracePeriod is how long Stop waits after an interrupt before killing.
	stopGracePeriod time.Duration

	// reapTimeout bounds the wait for the process to be reaped after it is killed.
	reapTimeout time.Duration

	// healthy is set once the process has answered a health check.
	healthy atomic.Bool

	// stopping is set once Stop has been called, so the exit is not reported as unexpected.
	stopping atomic.Bool
}

// Stop terminates the managed Ollama process, gracefully when possible.
//
// The process is interrupted and given its grace period to exit. It is killed straight
// away when the interrupt cannot be delivered (as on Windows), and also when the grace
// period elapses or ctx is cancelled first.
//
// Returns error when the process cannot be killed or is not reaped in time.
func (p *managedProcess) Stop(ctx context.Context) error {
	if p == nil || p.command == nil || p.command.Process == nil {
		return nil
	}
	p.stopping.Store(true)

	select {
	case <-p.done:
		return nil
	default:
	}

	ctx, l := logger.From(ctx, log)
	if err := p.interrupt(p.command); err != nil {
		l.Internal("Interrupting managed Ollama server failed, killing it",
			logger.Error(err),
		)
		return p.forceStop()
	}

	graceContext, cancel := context.WithTimeoutCause(ctx, p.stopGracePeriod,
		fmt.Errorf("managed ollama server did not stop within %s", p.stopGracePeriod))
	defer cancel()

	select {
	case <-p.done:
		return nil
	case <-graceContext.Done():
		l.Warn("Managed Ollama server did not stop gracefully, killing it",
			logger.String("reason", context.Cause(graceContext).Error()),
		)
		return p.forceStop()
	}
}

// forceStop kills the managed process tree and waits a bounded time for it to be reaped.
//
// Returns error when the kill fails or the process is not reaped in time.
func (p *managedProcess) forceStop() error {
	p.stopping.Store(true)

	if err := p.kill(p.command); err != nil && !errors.Is(err, os.ErrProcessDone) {
		return fmt.Errorf("killing managed ollama server: %w", err)
	}

	timer := time.NewTimer(p.reapTimeout)
	defer timer.Stop()

	select {
	case <-p.done:
		return nil
	case <-timer.C:
		return fmt.Errorf("managed ollama server was not reaped within %s of being killed", p.reapTimeout)
	}
}

// exitError describes why the process exited.
//
// Returns error which wraps errManagedProcessExited and the wait result, and must only be
// called after done is closed.
func (p *managedProcess) exitError() error {
	if p.exitErr == nil {
		return errManagedProcessExited
	}
	return fmt.Errorf("%w: %w", errManagedProcessExited, p.exitErr)
}

// supervise waits for the process to exit, flushes its output and records the result.
//
// The context only supplies the logger; it does not cancel supervision.
//
// Takes output (*outputRelay) which is flushed once the process has exited.
func (p *managedProcess) supervise(ctx context.Context, output *outputRelay) {
	defer close(p.done)
	defer goroutine.RecoverPanic(ctx, "llm.ollamaManagedProcess.supervise")

	waitErr := p.command.Wait()
	output.Flush()
	p.exitErr = waitErr

	_, l := logger.From(ctx, log)
	attributes := []logger.Attr{logger.Int("pid", p.command.Process.Pid)}
	if waitErr != nil {
		attributes = append(attributes, logger.Error(waitErr))
	}
	if p.healthy.Load() && !p.stopping.Load() {
		l.Warn("Managed Ollama server exited unexpectedly", attributes...)
		return
	}
	l.Internal("Managed Ollama server exited", attributes...)
}

// startOllama spawns `ollama serve` and waits for it to become healthy.
//
// The process is not tied to ctx; ctx only bounds the health wait, and the process is
// killed when that wait fails.
//
// Takes config (*Config) which supplies the binary path, host and process timeouts.
//
// Returns *managedProcess which manages the subprocess lifecycle.
// Returns error when the binary cannot be found or the server fails to start.
//
// Spawns a supervising goroutine that reaps the subprocess; it exits once the subprocess
// has exited and its output pipes are closed.
func startOllama(ctx context.Context, config *Config) (*managedProcess, error) {
	ctx, l := logger.From(ctx, log)

	binaryPath, err := resolveOllamaBinary(config.BinaryPath)
	if err != nil {
		return nil, err
	}

	command := newManagedOllamaCommand(binaryPath, config.Host, os.Environ())
	configureManagedOllamaCommand(command)

	supervisionContext := context.WithoutCancel(ctx)
	output := newOutputRelay(supervisionContext)
	command.Stderr = output
	command.WaitDelay = outputWaitDelay

	if err := command.Start(); err != nil {
		return nil, fmt.Errorf("starting ollama serve: %w", err)
	}

	process := newManagedProcess(command, config.StopGracePeriod)
	go process.supervise(supervisionContext, output)

	if err := waitForHealth(ctx, config, process.done); err != nil {
		if errors.Is(err, errManagedProcessExited) {
			err = process.exitError()
		}
		return nil, errors.Join(
			fmt.Errorf("ollama failed to become healthy: %w", err),
			process.forceStop(),
		)
	}
	process.healthy.Store(true)

	l.Notice("Started managed Ollama server",
		logger.String("binary", binaryPath),
		logger.String("host", config.Host),
		logger.Int("pid", command.Process.Pid),
	)

	return process, nil
}

// newManagedProcess wraps a started command with the platform stop handlers.
//
// Takes command (*exec.Cmd) which is the started ollama serve process.
// Takes stopGracePeriod (time.Duration) which is how long Stop waits after an interrupt
// before killing.
//
// Returns *managedProcess which is ready to be supervised.
func newManagedProcess(command *exec.Cmd, stopGracePeriod time.Duration) *managedProcess {
	return &managedProcess{
		command:         command,
		done:            make(chan struct{}),
		interrupt:       interruptManagedOllamaCommand,
		kill:            killManagedOllamaCommand,
		exitErr:         nil,
		stopGracePeriod: stopGracePeriod,
		reapTimeout:     killWaitTimeout,
		healthy:         atomic.Bool{},
		stopping:        atomic.Bool{},
	}
}

// resolveOllamaBinary returns the configured binary path, or the ollama binary found on
// PATH when none is configured.
//
// Takes binaryPath (string) which is the configured path, or empty to search PATH.
//
// Returns string which is the binary to run.
// Returns error when no binary is configured and none is found on PATH.
func resolveOllamaBinary(binaryPath string) (string, error) {
	if binaryPath != "" {
		return binaryPath, nil
	}
	found, err := exec.LookPath("ollama")
	if err != nil {
		return "", fmt.Errorf("ollama binary not found on $PATH - install from https://ollama.com: %w", err)
	}
	return found, nil
}

// newManagedOllamaCommand creates the managed `ollama serve` command with an explicit
// environment and explicit stdio handling.
//
// Takes binaryPath (string) which is the path to the ollama binary.
// Takes host (string) which is the Ollama API endpoint.
// Takes currentEnv ([]string) which is the parent process environment to filter.
//
// Returns *exec.Cmd which is the configured command ready to start.
func newManagedOllamaCommand(binaryPath, host string, currentEnv []string) *exec.Cmd {
	command := exec.Command(binaryPath, "serve")
	command.Env = buildManagedOllamaEnv(currentEnv, host)

	command.Stdin = nil
	command.Stdout = io.Discard

	return command
}

// buildManagedOllamaEnv constructs a minimal child environment for the managed Ollama
// process.
//
// Takes currentEnv ([]string) which is the parent environment to filter.
// Takes host (string) which is the Ollama API endpoint to set.
//
// Returns []string which is the filtered environment for the child process.
func buildManagedOllamaEnv(currentEnv []string, host string) []string {
	env := []string{"OLLAMA_HOST=" + host}

	for _, entry := range currentEnv {
		key, value, ok := strings.Cut(entry, "=")
		if !ok || key == "" {
			continue
		}
		if strings.EqualFold(key, "OLLAMA_HOST") {
			continue
		}
		if !shouldPreserveManagedOllamaEnv(key) {
			continue
		}
		env = append(env, key+"="+value)
	}

	return env
}

// shouldPreserveManagedOllamaEnv reports whether an environment variable is required for
// a managed Ollama process to behave correctly.
//
// Takes key (string) which is the environment variable name to check.
//
// Returns bool which is true when the key should be preserved.
func shouldPreserveManagedOllamaEnv(key string) bool {
	normalised := strings.ToUpper(key)
	if strings.HasPrefix(normalised, "OLLAMA_") {
		return true
	}
	_, ok := managedOllamaEnvKeys[normalised]
	return ok
}

// waitForHealth polls the Ollama health endpoint until it answers, the startup timeout
// elapses, ctx is cancelled, or the managed process exits.
//
// Takes config (*Config) which supplies the host, startup timeout and probe timeout.
// Takes exited (<-chan struct{}) which is closed when the managed process exits.
//
// Returns error which carries the cancellation cause and the last probe failure when the
// server does not become healthy, or wraps errManagedProcessExited when the process exits
// first.
func waitForHealth(ctx context.Context, config *Config, exited <-chan struct{}) error {
	healthURL, err := healthURLFor(config.Host)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeoutCause(ctx, config.StartupTimeout,
		fmt.Errorf("ollama did not become healthy within %s", config.StartupTimeout))
	defer cancel()

	client := newProbeClient(config.ProbeTimeout)
	ticker := time.NewTicker(healthPollInterval)
	defer ticker.Stop()

	var lastProbeErr error
	for {
		probeErr := probeServer(ctx, client, healthURL)
		if probeErr == nil {
			return nil
		}
		if ctx.Err() == nil {
			lastProbeErr = probeErr
		}

		select {
		case <-exited:
			return fmt.Errorf("%w before answering at %s", errManagedProcessExited, config.Host)
		case <-ctx.Done():
			return healthWaitError(ctx, config.Host, lastProbeErr)
		case <-ticker.C:
		}
	}
}

// healthWaitError describes a health wait that ended before the server answered.
//
// Takes host (string) which is the Ollama host that was polled.
// Takes lastProbeErr (error) which is the last probe failure seen before ctx ended, or
// nil when there was none.
//
// Returns error which carries the cancellation cause and the last probe failure.
func healthWaitError(ctx context.Context, host string, lastProbeErr error) error {
	if lastProbeErr == nil {
		return fmt.Errorf("waiting for ollama at %s: %w", host, context.Cause(ctx))
	}
	return fmt.Errorf("waiting for ollama at %s: %w (last probe: %w)", host, context.Cause(ctx), lastProbeErr)
}

// probeOnce sends a single health probe bounded by timeout.
//
// Takes healthURL (string) which is the Ollama version endpoint.
// Takes timeout (time.Duration) which bounds the probe.
//
// Returns error when the server does not answer with 200 OK in time.
func probeOnce(ctx context.Context, healthURL string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeoutCause(ctx, timeout,
		fmt.Errorf("ollama reachability probe exceeded %s", timeout))
	defer cancel()

	return probeServer(ctx, newProbeClient(timeout), healthURL)
}

// probeServer sends one health request to the Ollama server.
//
// Takes client (*http.Client) which sends the probe.
// Takes healthURL (string) which is the Ollama version endpoint.
//
// Returns error when the request fails or the status is not 200 OK.
func probeServer(ctx context.Context, client *http.Client, healthURL string) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, healthURL, nil)
	if err != nil {
		return fmt.Errorf("building ollama probe request: %w", err)
	}

	response, err := client.Do(request)
	if err != nil {
		if cause := context.Cause(ctx); cause != nil {
			return fmt.Errorf("probing ollama: %w", cause)
		}
		return fmt.Errorf("probing ollama: %w", err)
	}
	defer func() {
		_ = drainAndClose(response.Body)
	}()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("probing ollama: unexpected status %d", response.StatusCode)
	}
	return nil
}

// healthURLFor builds the health endpoint URL for an Ollama host.
//
// Takes host (string) which is the Ollama API base URL.
//
// Returns string which is the version endpoint URL.
// Returns error when host cannot be joined with the endpoint path.
func healthURLFor(host string) (string, error) {
	healthURL, err := url.JoinPath(host, ollamaVersionPath)
	if err != nil {
		return "", fmt.Errorf("building ollama health URL: %w", err)
	}
	return healthURL, nil
}

// newProbeClient returns an HTTP client for one-shot Ollama health probes.
//
// Keep-alives are disabled so a probe never leaves a pooled connection, and its read and
// write goroutines, behind after the provider has been closed.
//
// Takes timeout (time.Duration) which bounds each probe request.
//
// Returns *http.Client which closes each connection once its response is read.
func newProbeClient(timeout time.Duration) *http.Client {
	transport := cloneTransport(http.DefaultTransport)
	transport.DisableKeepAlives = true
	return &http.Client{Timeout: timeout, Transport: transport}
}

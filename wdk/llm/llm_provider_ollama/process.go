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
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"piko.sh/piko/wdk/logger"
)

const (
	// healthPollInterval is the interval between health status checks.
	healthPollInterval = 250 * time.Millisecond

	// healthTimeout is the maximum duration to wait for health check responses.
	healthTimeout = 15 * time.Second

	// stopGracePeriod is the time to wait for graceful shutdown.
	stopGracePeriod = 5 * time.Second
)

var managedOllamaEnvKeys = map[string]struct{}{
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

// managedProcess tracks an Ollama subprocess spawned by the provider.
type managedProcess struct {
	// command is the running ollama serve process.
	command *exec.Cmd

	// done is closed when the process exits.
	done chan struct{}

	// interrupt sends a graceful shutdown signal to the managed process tree.
	interrupt func(*exec.Cmd) error

	// kill forcefully terminates the managed process tree.
	kill func(*exec.Cmd) error
}

// Stop terminates the managed Ollama process gracefully.
//
// Returns error when the process cannot be stopped.
func (p *managedProcess) Stop() error {
	if p == nil || p.command == nil || p.command.Process == nil {
		return nil
	}

	if p.interrupt != nil {
		_ = p.interrupt(p.command)
	} else {
		_ = p.command.Process.Signal(os.Interrupt)
	}

	select {
	case <-p.done:
		return nil
	case <-time.After(stopGracePeriod):
		_, l := logger.From(context.Background(), log)
		l.Warn("Ollama did not stop gracefully, killing process")
		if p.kill != nil {
			return p.kill(p.command)
		}
		return p.command.Process.Kill()
	}
}

// startOllama spawns `ollama serve` and waits for it to become healthy.
//
// Takes binaryPath (string) which is the path to the ollama binary.
// Takes host (string) which is the Ollama API endpoint to wait for.
//
// Returns *managedProcess which manages the subprocess lifecycle.
// Returns error when the binary cannot be found or the server fails to start.
//
// Spawns a goroutine to read stderr output from the subprocess. The goroutine
// terminates when the subprocess exits.
func startOllama(binaryPath, host string) (*managedProcess, error) {
	_, l := logger.From(context.Background(), log)
	if binaryPath == "" {
		var err error
		binaryPath, err = exec.LookPath("ollama")
		if err != nil {
			return nil, fmt.Errorf(
				"ollama binary not found on $PATH - install from https://ollama.com: %w",
				err,
			)
		}
	}

	command := newManagedOllamaCommand(binaryPath, host, os.Environ())
	configureManagedOllamaCommand(command)

	stderr, err := command.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("creating stderr pipe: %w", err)
	}

	if err := command.Start(); err != nil {
		return nil, fmt.Errorf("starting ollama serve: %w", err)
	}

	done := make(chan struct{})

	go func() {
		defer close(done)
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			l.Debug("ollama",
				logger.String("output", scanner.Text()),
			)
		}
		_ = command.Wait()
	}()

	if err := waitForHealth(host, healthTimeout); err != nil {
		_ = killManagedOllamaCommand(command)
		return nil, fmt.Errorf("ollama failed to become healthy: %w", err)
	}

	l.Info("Started managed Ollama server",
		logger.String("binary", binaryPath),
		logger.String("host", host),
	)

	return &managedProcess{
		command:   command,
		done:      done,
		interrupt: interruptManagedOllamaCommand,
		kill:      killManagedOllamaCommand,
	}, nil
}

// newManagedOllamaCommand creates the managed `ollama serve` command with an
// explicit environment and explicit stdio handling.
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

// buildManagedOllamaEnv constructs a minimal child environment for the managed
// Ollama process.
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

// shouldPreserveManagedOllamaEnv reports whether an environment variable is
// required for a managed Ollama process to behave correctly.
//
// Takes key (string) which is the environment variable name to check.
//
// Returns bool which is true when the key should be preserved.
func shouldPreserveManagedOllamaEnv(key string) bool {
	normalized := strings.ToUpper(key)
	if strings.HasPrefix(normalized, "OLLAMA_") {
		return true
	}
	_, ok := managedOllamaEnvKeys[normalized]
	return ok
}

// waitForHealth polls the Ollama health endpoint until it responds or the
// timeout elapses.
//
// Takes host (string) which is the Ollama API base URL.
// Takes timeout (time.Duration) which is the maximum time to wait.
//
// Returns error when the server does not become healthy within the timeout.
func waitForHealth(host string, timeout time.Duration) error {
	healthURL, err := url.JoinPath(host, "/api/version")
	if err != nil {
		return fmt.Errorf("building health URL: %w", err)
	}

	ctx, cancel := context.WithTimeoutCause(context.Background(), timeout, fmt.Errorf("ollama process startup exceeded %s timeout", timeout))
	defer cancel()

	client := &http.Client{Timeout: 2 * time.Second}

	for {
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, healthURL, nil)
		if err != nil {
			return err
		}

		response, err := client.Do(request)
		if err == nil {
			_ = response.Body.Close()
			if response.StatusCode == http.StatusOK {
				return nil
			}
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for Ollama at %s", host)
		case <-time.After(healthPollInterval):
		}
	}
}

// isServerReachable checks whether the Ollama server is responding.
//
// Takes host (string) which is the Ollama API base URL.
//
// Returns bool which is true if the server is reachable.
func isServerReachable(host string) bool {
	healthURL, err := url.JoinPath(host, "/api/version")
	if err != nil {
		return false
	}

	ctx, cancel := context.WithTimeoutCause(context.Background(), 2*time.Second, errors.New("ollama process shutdown exceeded 2s timeout"))
	defer cancel()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, healthURL, nil)
	if err != nil {
		return false
	}

	response, err := (&http.Client{Timeout: 2 * time.Second}).Do(request)
	if err != nil {
		return false
	}
	_ = response.Body.Close()
	return response.StatusCode == http.StatusOK
}

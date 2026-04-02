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

package llm_provider_ollama

import (
	"io"
	"os"
	"os/exec"
	"testing"

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

func TestManagedProcessStop_UsesInjectedHandlers(t *testing.T) {
	t.Parallel()

	interruptCalls := 0
	killCalls := 0

	var process *managedProcess
	process = &managedProcess{
		command: &exec.Cmd{
			Process: &os.Process{Pid: os.Getpid()},
		},
		done: make(chan struct{}),
		interrupt: func(*exec.Cmd) error {
			interruptCalls++
			return nil
		},
		kill: func(*exec.Cmd) error {
			killCalls++
			close(process.done)
			return nil
		},
	}

	err := process.Stop()
	require.NoError(t, err)
	assert.Equal(t, 1, interruptCalls)
	assert.Equal(t, 1, killCalls)
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

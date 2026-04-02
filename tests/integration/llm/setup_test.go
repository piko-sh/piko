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

//go:build integration

package llm_integration_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	defaultOllamaHost       = "http://localhost:11434"
	defaultCompletionModel  = "qwen2.5:0.5b"
	defaultToolModel        = "qwen3:0.6b"
	defaultEmbeddingModel   = "all-minilm"
	defaultCompletionDigest = "a8b0c5157701"
	defaultToolDigest       = "7df6b6e09427"
	defaultEmbeddingDigest  = "1b226e2802db"
)

func setupTestEnvironment(ctx context.Context) (*testEnv, error) {
	env := &testEnv{
		ollamaHost:       defaultOllamaHost,
		completionModel:  defaultCompletionModel,
		completionDigest: defaultCompletionDigest,
		toolModel:        defaultToolModel,
		toolDigest:       defaultToolDigest,
		embeddingModel:   defaultEmbeddingModel,
		embeddingDigest:  defaultEmbeddingDigest,
	}

	if host := os.Getenv("OLLAMA_HOST"); host != "" {
		env.ollamaHost = host
	}
	if model := os.Getenv("OLLAMA_COMPLETION_MODEL"); model != "" {
		env.completionModel = model
		env.completionDigest = ""
	}
	if model := os.Getenv("OLLAMA_TOOL_MODEL"); model != "" {
		env.toolModel = model
		env.toolDigest = ""
	}
	if model := os.Getenv("OLLAMA_EMBEDDING_MODEL"); model != "" {
		env.embeddingModel = model
		env.embeddingDigest = ""
	}

	if !isOllamaReachable(env.ollamaHost) {
		_, _ = fmt.Fprintf(os.Stderr,
			"WARN: Ollama server not reachable at %s (Ollama tests will be skipped).\n"+
				"  Install with: curl -fsSL https://ollama.com/install.sh | sh\n",
			env.ollamaHost,
		)
	} else if missing := findMissingModels(env.ollamaHost, env.completionModel, env.toolModel, env.embeddingModel); len(missing) > 0 {
		_, _ = fmt.Fprintf(os.Stderr,
			"WARN: Required Ollama model(s) not found (Ollama tests will be skipped).\n"+
				"  Missing: %s\n"+
				"  Pull them with:\n",
			strings.Join(missing, ", "),
		)
		for _, m := range missing {
			_, _ = fmt.Fprintf(os.Stderr, "    ollama pull %s\n", m)
		}
	} else {
		env.ollamaAvailable = true
	}

	var cleanups []func()

	if addr := os.Getenv("REDIS_STACK_ADDR"); addr != "" {
		env.redisStackAddr = addr
	} else {
		ctr, addr, err := startRedisStackContainer(ctx)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "WARN: could not start Redis Stack container (Redis vector tests will be skipped): %v\n", err)
		} else {
			env.redisStackAddr = addr
			if ctr != nil {
				cleanups = append(cleanups, func() { _ = ctr.Terminate(context.Background()) })
			}
		}
	}

	if addr := os.Getenv("VALKEY_ADDR"); addr != "" {
		env.valkeyAddr = addr
	} else {
		ctr, addr, err := startValkeyContainer(ctx)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "WARN: could not start Valkey container (Valkey vector tests will be skipped): %v\n", err)
		} else {
			env.valkeyAddr = addr
			if ctr != nil {
				cleanups = append(cleanups, func() { _ = ctr.Terminate(context.Background()) })
			}
		}
	}

	env.cleanup = func() {
		for i := len(cleanups) - 1; i >= 0; i-- {
			cleanups[i]()
		}
	}

	return env, nil
}

func isOllamaReachable(host string) bool {
	client := &http.Client{Timeout: 3 * time.Second}

	response, err := client.Get(host + "/api/version")
	if err != nil {
		return false
	}
	_ = response.Body.Close()
	return response.StatusCode == http.StatusOK
}

func findMissingModels(host string, required ...string) []string {
	client := &http.Client{Timeout: 5 * time.Second}

	response, err := client.Get(host + "/api/tags")
	if err != nil {
		return required
	}
	defer func() { _ = response.Body.Close() }()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return required
	}

	var result struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return required
	}

	available := make(map[string]bool, len(result.Models))
	for _, m := range result.Models {
		available[m.Name] = true
		if index := strings.LastIndex(m.Name, ":"); index > 0 {
			available[m.Name[:index]] = true
		}
	}

	var missing []string
	for _, r := range required {
		if !available[r] {
			missing = append(missing, r)
		}
	}
	return missing
}

func startRedisStackContainer(ctx context.Context) (testcontainers.Container, string, error) {
	request := testcontainers.ContainerRequest{
		Image:        "redis/redis-stack-server:latest",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor: wait.ForLog("Ready to accept connections").
			WithStartupTimeout(60 * time.Second),
	}

	ctr, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: request,
		Started:          true,
	})
	if err != nil {
		return nil, "", fmt.Errorf("creating Redis Stack container: %w", err)
	}

	host, err := ctr.Host(ctx)
	if err != nil {
		_ = ctr.Terminate(ctx)
		return nil, "", fmt.Errorf("getting host: %w", err)
	}

	port, err := ctr.MappedPort(ctx, "6379/tcp")
	if err != nil {
		_ = ctr.Terminate(ctx)
		return nil, "", fmt.Errorf("getting port: %w", err)
	}

	return ctr, fmt.Sprintf("%s:%s", host, port.Port()), nil
}

func startValkeyContainer(ctx context.Context) (testcontainers.Container, string, error) {
	request := testcontainers.ContainerRequest{
		Image:        "valkey/valkey-bundle:8-alpine",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor: wait.ForLog("Ready to accept connections").
			WithStartupTimeout(60 * time.Second),
	}

	ctr, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: request,
		Started:          true,
	})
	if err != nil {
		return nil, "", fmt.Errorf("creating Valkey container: %w", err)
	}

	host, err := ctr.Host(ctx)
	if err != nil {
		_ = ctr.Terminate(ctx)
		return nil, "", fmt.Errorf("getting host: %w", err)
	}

	port, err := ctr.MappedPort(ctx, "6379/tcp")
	if err != nil {
		_ = ctr.Terminate(ctx)
		return nil, "", fmt.Errorf("getting port: %w", err)
	}

	return ctr, fmt.Sprintf("%s:%s", host, port.Port()), nil
}

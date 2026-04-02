//go:build integration

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

package wasm_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"piko.sh/piko/internal/wasm/wasm_dto"
)

type WASMExecutor struct {
	wasmDir string
	timeout time.Duration
}

func NewWASMExecutor(wasmDir string) *WASMExecutor {
	return &WASMExecutor{
		wasmDir: wasmDir,
		timeout: 60 * time.Second,
	}
}

func (e *WASMExecutor) WithTimeout(d time.Duration) *WASMExecutor {
	e.timeout = d
	return e
}

type wasmRequest struct {
	Args     any    `json:"args"`
	Function string `json:"function"`
}

func (e *WASMExecutor) Generate(ctx context.Context, request *wasm_dto.GenerateFromSourcesRequest) (*wasm_dto.GenerateFromSourcesResponse, error) {
	result, err := e.callWASM(ctx, "generate", request)
	if err != nil {
		return nil, err
	}

	var response wasm_dto.GenerateFromSourcesResponse
	if err := mapToStruct(result, &response); err != nil {
		return nil, fmt.Errorf("failed to parse generate response: %w", err)
	}

	return &response, nil
}

func (e *WASMExecutor) Analyse(ctx context.Context, request *wasm_dto.AnalyseRequest) (*wasm_dto.AnalyseResponse, error) {
	result, err := e.callWASM(ctx, "analyse", request)
	if err != nil {
		return nil, err
	}

	var response wasm_dto.AnalyseResponse
	if err := mapToStruct(result, &response); err != nil {
		return nil, fmt.Errorf("failed to parse analyse response: %w", err)
	}

	return &response, nil
}

func (e *WASMExecutor) Render(ctx context.Context, request *wasm_dto.RenderFromSourcesRequest) (*wasm_dto.RenderFromSourcesResponse, error) {
	result, err := e.callWASM(ctx, "render", request)
	if err != nil {
		return nil, err
	}

	var response wasm_dto.RenderFromSourcesResponse
	if err := mapToStruct(result, &response); err != nil {
		return nil, fmt.Errorf("failed to parse render response: %w", err)
	}

	return &response, nil
}

func (e *WASMExecutor) callWASM(ctx context.Context, function string, arguments any) (map[string]any, error) {

	ctx, cancel := context.WithTimeoutCause(ctx, e.timeout, fmt.Errorf("test: WASM execution exceeded %s timeout", e.timeout))
	defer cancel()

	request := wasmRequest{
		Function: function,
		Args:     arguments,
	}
	reqJSON, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	command := exec.CommandContext(ctx, "node", "run_wasm.js")
	command.Dir = e.wasmDir
	command.Stdin = bytes.NewReader(reqJSON)

	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr

	if err := command.Run(); err != nil {

		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return nil, fmt.Errorf("WASM execution timed out after %v", e.timeout)
		}

		stderrString := stderr.String()
		if stderrString != "" {
			return nil, fmt.Errorf("node execution failed: %w\nstderr: %s", err, stderrString)
		}
		return nil, fmt.Errorf("node execution failed: %w", err)
	}

	jsonLine := extractJSONLine(stdout.String())
	if jsonLine == "" {
		return nil, fmt.Errorf("no JSON response found in WASM output:\n%s", stdout.String())
	}

	var response map[string]any
	if err := json.Unmarshal([]byte(jsonLine), &response); err != nil {
		return nil, fmt.Errorf("failed to parse WASM response: %w\noutput: %s", err, stdout.String())
	}

	if errMessage, ok := response["error"].(string); ok && errMessage != "" {
		if success, ok := response["success"].(bool); ok && !success {
			return nil, fmt.Errorf("WASM error: %s", errMessage)
		}
	}

	return response, nil
}

func mapToStruct(m map[string]any, target any) error {
	jsonBytes, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return json.Unmarshal(jsonBytes, target)
}

func extractJSONLine(output string) string {
	lines := strings.Split(output, "\n")

	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if strings.HasPrefix(line, "{") && strings.HasSuffix(line, "}") {
			return line
		}
	}
	return ""
}

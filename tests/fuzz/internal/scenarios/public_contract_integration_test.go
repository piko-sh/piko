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

package scenarios

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/stretchr/testify/require"
	"piko.sh/json"
	"piko.sh/piko/tests/fuzz/internal/casefile"
	"piko.sh/piko/tests/fuzz/internal/publiccontracts"
)

var (
	errNoObservedAssertion = errors.New("provider executor returned no observed assertion")
	errProviderTimeout     = errors.New("provider contract timed out")
)

type providerOutcome struct {
	Family  string
	Test    string
	Status  string
	Problem string
	Receipt publiccontracts.Receipt
	Seconds float64
}

type providerReport struct {
	Cgroup     map[string]string
	Go         string
	StartedUTC string
	Results    []providerOutcome
}

func TestFinitePublicProviderContracts(t *testing.T) {
	repository, err := filepath.Abs("../../../..")
	require.NoError(t, err)
	families, err := publiccontracts.Providers(os.DirFS(repository))
	require.NoError(t, err)
	report := providerReport{Go: runtime.Version(), StartedUTC: time.Now().UTC().Format(time.RFC3339), Cgroup: cgroupLimits()}
	t.Setenv("GOPROXY", "off")
	t.Setenv("GOSUMDB", "off")
	t.Setenv("GOTOOLCHAIN", "local")
	for index, provider := range families {
		t.Run(provider.Family, func(t *testing.T) {
			start := time.Now()
			ctx, cancel := context.WithTimeoutCause(t.Context(), 60*time.Second, errProviderTimeout)
			defer cancel()
			got, err := RunPublic(ctx, repository, t.TempDir(), &casefile.Case{Target: "provider-contracts", Index: uint64(index)})
			if err == nil && (got.Contract == "" || got.Assertions == 0) {
				err = errNoObservedAssertion
			}
			outcome := providerOutcome{Family: provider.Family, Test: provider.Test, Status: "pass", Receipt: got, Seconds: time.Since(start).Seconds()}
			if err != nil {
				outcome.Problem = truncateMiddle(err.Error(), 4096)
				switch {
				case strings.Contains(outcome.Problem, "module lookup disabled by GOPROXY=off"), strings.Contains(outcome.Problem, "missing go.sum entry"), strings.Contains(outcome.Problem, "no required module provides package"):
					outcome.Status = "prerequisite"
				default:
					outcome.Status = "failed"
				}
			}
			report.Results = append(report.Results, outcome)
			if destination := os.Getenv("PIKO_FUZZ_PROVIDER_RECEIPT"); destination != "" {
				encoded, err := json.MarshalIndent(report, "", "  ")
				require.NoError(t, err)
				require.NoError(t, os.WriteFile(destination, append(encoded, '\n'), 0o600))
			}
			switch outcome.Status {
			case "prerequisite":
				t.Skipf("offline prerequisite: %s", outcome.Problem)
			case "failed":
				require.Fail(t, outcome.Problem)
			}
		})
	}
}

func cgroupLimits() map[string]string {
	data, err := os.ReadFile("/proc/self/cgroup")
	if err != nil {
		return nil
	}
	var limits map[string]string
	for line := range strings.Lines(string(data)) {
		path, ok := strings.CutPrefix(strings.TrimSpace(line), "0::")
		if !ok {
			continue
		}
		limits = map[string]string{}
		for _, key := range []string{"cpu.max", "memory.max", "memory.swap.max", "pids.max"} {
			if contents, readErr := os.ReadFile(filepath.Join("/sys/fs/cgroup", path, key)); readErr == nil {
				limits[key] = strings.TrimSpace(string(contents))
			}
		}
	}
	return limits
}

func truncateMiddle(text string, limit int) string {
	if len(text) <= limit {
		return text
	}
	half := limit / 2
	head := half
	for head > 0 && !utf8.RuneStart(text[head]) {
		head--
	}
	tail := len(text) - half
	for tail < len(text) && !utf8.RuneStart(text[tail]) {
		tail++
	}
	return text[:head] + "\n... [truncated] ...\n" + text[tail:]
}

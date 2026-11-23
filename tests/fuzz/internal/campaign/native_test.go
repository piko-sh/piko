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

package campaign

import (
	"bytes"
	"cmp"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/json"
	"piko.sh/piko/tests/fuzz/internal/resource"
	"piko.sh/piko/tests/fuzz/internal/targets"
)

type sequencedBudget struct {
	samples []resource.Sample
	calls   int
}

func (*sequencedBudget) Admit(context.Context, uint64) error { return nil }

func (budget *sequencedBudget) Sample(ctx context.Context) (resource.Sample, error) {
	return budget.SampleNow(ctx)
}

func (budget *sequencedBudget) SampleNow(context.Context) (resource.Sample, error) {
	sample := budget.samples[min(budget.calls, len(budget.samples)-1)]
	budget.calls++
	return sample, nil
}

func TestLoadNativeValidatesTheSavedReceipt(t *testing.T) {
	testCases := []struct {
		name    string
		content string
		message string
	}{
		{name: "valid", content: `{"Fuzzer":"binder","Cases":3,"Duration":0}`, message: ""},
		{name: "malformed", content: "{", message: "decoding native campaign receipt"},
		{name: "missing fuzzer", content: `{"Cases":3}`, message: "invalid native campaign receipt"},
		{name: "no cases", content: `{"Fuzzer":"binder"}`, message: "invalid native campaign receipt"},
		{name: "negative duration", content: `{"Fuzzer":"binder","Cases":1,"Duration":-1}`, message: "invalid native campaign receipt"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			dir := t.TempDir()
			require.NoError(t, os.WriteFile(filepath.Join(dir, "native.json"), []byte(testCase.content), 0o600))

			n, err := LoadNative(dir)

			if testCase.message != "" {
				require.ErrorContains(t, err, testCase.message)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, "binder", n.Fuzzer)
			assert.Equal(t, uint64(3), n.Cases)
		})
	}
	_, err := LoadNative(t.TempDir())
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestNativeStatusClassifiesTheWorkerOutcome(t *testing.T) {
	cancelled, cancel := context.WithCancel(t.Context())
	cancel()
	expired, expire := context.WithTimeout(t.Context(), 0)
	defer expire()
	<-expired.Done()
	failure := errors.New("exit status 1")
	testCases := []struct {
		run  context.Context
		err  error
		name string
		out  string
		want string
	}{
		{name: "pass", run: t.Context(), err: nil, out: "", want: "pass"},
		{name: "cancelled", run: cancelled, err: failure, out: "", want: "paused"},
		{name: "deadline", run: expired, err: failure, out: "", want: kindResource},
		{name: "resource limit", run: t.Context(), err: resource.ErrLimit, out: "", want: kindResource},
		{name: "unrelated failure", run: t.Context(), err: failure, out: "panic: setup", want: "harness"},
		{name: "fuzz failure", run: t.Context(), err: failure, out: "--- FAIL: FuzzBinder (0.01s)", want: StatusFailed},
		{name: "written input", run: t.Context(), err: failure, out: "Failing input written to testdata/fuzz/FuzzBinder/x", want: StatusFailed},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.want, nativeStatus(testCase.run, testCase.err, testCase.out, "FuzzBinder"))
		})
	}
}

func TestCopyParserSeedsCopiesOnlyPageSeeds(t *testing.T) {
	source := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(source, "one.pk"), []byte("<template></template>"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(source, "notes.txt"), []byte("ignored"), 0o600))
	destination := filepath.Join(t.TempDir(), "testdata")

	require.NoError(t, copyParserSeeds(source, destination))

	data, err := os.ReadFile(filepath.Join(destination, "one.pk"))
	require.NoError(t, err)
	assert.Equal(t, "<template></template>", string(data))
	assert.NoFileExists(t, filepath.Join(destination, "notes.txt"))
}

func TestCopyParserSeedsCreatesAnEmptyCorpusWithoutSeeds(t *testing.T) {
	destination := filepath.Join(t.TempDir(), "testdata")

	require.NoError(t, copyParserSeeds(filepath.Join(t.TempDir(), "missing"), destination))

	assert.DirExists(t, destination)
}

func TestNativeArgumentsSelectFuzzingOrReplay(t *testing.T) {
	h := Harness{State: "/state"}
	fuzzer := targets.GoFuzzer{Package: "example.test/fuzz", Function: "FuzzThing"}

	args, timeout, err := h.nativeArguments(t.TempDir(), &NativeRun{Fuzzer: "thing", Duration: time.Minute}, &fuzzer, false)
	require.NoError(t, err)
	assert.Equal(t, time.Minute+nativeReportingAllowance, timeout)
	assert.Contains(t, args, "-test.fuzztime=1m0s")
	assert.Contains(t, args, "-test.fuzz=^FuzzThing$")
	assert.Contains(t, args, "-test.fuzzcachedir="+filepath.Join("/state", ".cache", "fuzz", "thing"))

	args, timeout, err = h.nativeArguments(t.TempDir(), &NativeRun{Fuzzer: "thing", Cases: 7}, &fuzzer, false)
	require.NoError(t, err)
	assert.Equal(t, nativeBuildTimeout, timeout)
	assert.Contains(t, args, "-test.fuzztime=7x")

	dir := t.TempDir()
	_, _, err = h.nativeArguments(dir, &NativeRun{Fuzzer: "thing"}, &fuzzer, true)
	require.ErrorIs(t, err, os.ErrNotExist)
	corpus := filepath.Join(dir, "testdata", "fuzz", "FuzzThing")
	require.NoError(t, os.MkdirAll(corpus, 0o700))
	_, _, err = h.nativeArguments(dir, &NativeRun{Fuzzer: "thing"}, &fuzzer, true)
	require.ErrorContains(t, err, "no retained failure corpus for FuzzThing")
	require.NoError(t, os.WriteFile(filepath.Join(corpus, "input"), []byte("go test fuzz v1\n"), 0o600))
	args, _, err = h.nativeArguments(dir, &NativeRun{Fuzzer: "thing"}, &fuzzer, true)
	require.NoError(t, err)
	assert.Equal(t, []string{"-test.run=^FuzzThing$", "-test.parallel=1", "-test.timeout=1m"}, args)
}

func TestWriteNativeHeaderDescribesLimitsAndProvenance(t *testing.T) {
	output := &bytes.Buffer{}
	h := Harness{Output: output}
	n := NativeRun{
		Duration:   time.Minute,
		Resources:  resource.Sample{QuotaCPUs: 2, CPUs: []int{0, 1}, MemoryMax: 100, SwapMax: 5},
		Provenance: Provenance{Go: "go1.27.0", WorkspaceSHA256: "abc"},
	}

	require.NoError(t, h.writeNativeHeader(&n, "FuzzThing", "/state/native.json", false))

	assert.Contains(t, output.String(), "Go fuzz target FuzzThing, CPU quota 2.00, allowed CPUs [0 1], memory cap 100 bytes, swap cap 5 bytes")
	assert.Contains(t, output.String(), "followed by 1m0s of Go fuzzing")
	assert.Contains(t, output.String(), "go1.27.0, workspace SHA-256 abc, 0 local source receipts; full provenance in /state/native.json")
	output.Reset()
	require.NoError(t, h.writeNativeHeader(&n, "FuzzThing", "/state/replay.json", true))
	assert.NotContains(t, output.String(), "Go fuzzing")
}

func TestExecuteNativeClassifiesAndRecordsTheWorkerRun(t *testing.T) {
	testCases := []struct {
		budget  *sequencedBudget
		name    string
		script  string
		status  string
		timeout time.Duration
		fails   bool
	}{
		{name: "pass", script: "echo ok\n", status: "pass", fails: false},
		{name: "finding", script: "echo '--- FAIL: FuzzThing (0.00s)'\nexit 1\n", status: StatusFailed, fails: true},
		{name: "harness", script: "echo 'cannot start'\nexit 2\n", status: "harness", fails: true},
		{name: "deadline", script: "exec sleep 30\n", status: kindResource, timeout: 50 * time.Millisecond, fails: true},
		{
			name: "out of memory", script: "echo ok\n", status: kindResource, fails: true,
			budget: &sequencedBudget{samples: []resource.Sample{{Cgroup: "/fuzz"}, {Cgroup: "/fuzz", OOMKills: 1}}},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			dir := t.TempDir()
			binary := filepath.Join(dir, "worker.test")
			require.NoError(t, os.WriteFile(binary, []byte("#!/bin/sh\n"+testCase.script), 0o700))
			record := filepath.Join(dir, "native.json")
			budget := testCase.budget
			if budget == nil {
				budget = &sequencedBudget{samples: []resource.Sample{{}}}
			}
			timeout := cmp.Or(testCase.timeout, time.Minute)
			h := Harness{Budget: budget}
			n := NativeRun{Fuzzer: "thing", Cases: 1}
			fuzzer := targets.GoFuzzer{Package: "example.test/fuzz", Function: "FuzzThing"}

			err := h.executeNative(t.Context(), &n, &fuzzer, &nativeWorker{binary: binary, dir: dir, record: record, timeout: timeout})

			if testCase.fails {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, testCase.status, n.Status)
			var saved NativeRun
			data, readErr := os.ReadFile(record)
			require.NoError(t, readErr)
			require.NoError(t, json.Unmarshal(data, &saved))
			assert.Equal(t, testCase.status, saved.Status)
		})
	}
}

func TestNativeRejectsAnUnknownFuzzer(t *testing.T) {
	h := Harness{Budget: testBudget{}}

	err := h.Native(t.Context(), t.TempDir(), &NativeRun{Fuzzer: "no-such-fuzzer"}, false)

	require.ErrorContains(t, err, `unknown Go fuzzer "no-such-fuzzer"`)
}

func TestNativeStopsWhenAdmissionIsRefused(t *testing.T) {
	refused := errors.New("admission refused")
	h := Harness{Budget: scriptedBudget{admitErr: refused}}
	dir := filepath.Join(t.TempDir(), "native")

	err := h.Native(t.Context(), dir, &NativeRun{Fuzzer: "project", Cases: 1}, false)

	require.ErrorIs(t, err, refused)
	assert.NoDirExists(t, dir)
}

func TestNativeRecordsABuildFailure(t *testing.T) {
	repository := nativeFixture(t)
	output := &bytes.Buffer{}
	h := Harness{Repo: repository, State: t.TempDir(), Budget: testBudget{}, Output: output, Progress: &bytes.Buffer{}}
	dir := t.TempDir()
	requested := NativeRun{Fuzzer: "binder", Cases: 1}

	err := h.Native(t.Context(), dir, &requested, false)

	require.ErrorContains(t, err, "build ")
	n, loadErr := LoadNative(dir)
	require.NoError(t, loadErr)
	assert.Equal(t, "build-error", n.Status)
	assert.NotEmpty(t, n.Output)
	assert.Contains(t, n.RequestedLimits, "CPUS")
	assert.Empty(t, requested.Status, "the requested run was modified")
	assert.Contains(t, output.String(), "Go fuzz target FuzzBinder")
}

func TestNativeFuzzesAndReplaysARetainedFailure(t *testing.T) {
	repository := nativeFixture(t)
	h := Harness{Repo: repository, State: t.TempDir(), Budget: testBudget{}, Output: &bytes.Buffer{}, Progress: &bytes.Buffer{}}
	dir := t.TempDir()

	require.NoError(t, h.Native(t.Context(), dir, &NativeRun{Fuzzer: "project", Cases: 2}, false))

	n, err := LoadNative(dir)
	require.NoError(t, err)
	assert.Equal(t, "pass", n.Status)
	assert.Contains(t, n.Provenance.WorkerModules, "worker.test")
	assert.NoFileExists(t, filepath.Join(dir, "worker.test"), "the worker binary was retained")

	corpus := filepath.Join(dir, "testdata", "fuzz", "FuzzProject")
	require.NoError(t, os.MkdirAll(corpus, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(corpus, "boom"), []byte("go test fuzz v1\n[]byte(\"boom\")\n"), 0o600))

	err = h.Native(t.Context(), dir, &NativeRun{Fuzzer: "project", Cases: 2}, true)

	require.ErrorContains(t, err, "FuzzProject")
	data, readErr := os.ReadFile(filepath.Join(dir, "replay.json"))
	require.NoError(t, readErr)
	var replay NativeRun
	require.NoError(t, json.Unmarshal(data, &replay))
	assert.Equal(t, StatusFailed, replay.Status)
	assert.Contains(t, replay.Output, "--- FAIL: FuzzProject")
}

func nativeFixture(t *testing.T) string {
	t.Helper()
	parent := t.TempDir()
	repository := filepath.Join(parent, "generate")
	require.NoError(t, os.Mkdir(repository, 0o700))
	writeFixture(t, filepath.Join(repository, "go.mod"), "module piko.sh/piko/tests/fuzz/internal/generate\n\ngo 1.27.0\n")
	writeFixture(t, filepath.Join(repository, "doc.go"), "package generate\n")
	writeFixture(t, filepath.Join(repository, "fuzz_test.go"), "package generate\n\nimport \"testing\"\n\n"+
		"func FuzzProject(f *testing.F) {\n\tf.Fuzz(func(t *testing.T, data []byte) {\n\t\tif string(data) == \"boom\" {\n\t\t\tt.Fatal(\"boom\")\n\t\t}\n\t})\n}\n")
	work := filepath.Join(parent, "go.work")
	writeFixture(t, work, "go 1.27.0\n\nuse ./generate\n")
	t.Setenv("GOWORK", work)
	t.Setenv("GOFLAGS", "")
	t.Setenv("GOPROXY", "off")
	t.Setenv("PIKO_FUZZ_STATE", filepath.Join(parent, "state"))
	return repository
}

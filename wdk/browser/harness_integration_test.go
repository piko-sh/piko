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

package browser

import (
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/safedisk"
)

const (
	harnessServerEnv = "WDK_BROWSER_HARNESS_SERVER"
)

func TestHarnessServerProcess(t *testing.T) {
	if os.Getenv(harnessServerEnv) != "1" {
		t.Skip("serves the fixture only when started by a harness")
	}
	listener, err := net.Listen("tcp", "localhost:"+os.Getenv("PIKO_PORT"))
	require.NoError(t, err)
	server := &http.Server{Handler: fixtureHandler(), ReadHeaderTimeout: 5 * time.Second}
	_ = server.Serve(listener)
}

func writeSpec(t *testing.T, directory, name, content string) string {
	t.Helper()
	path := filepath.Join(directory, name)
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}

func TestHarnessLifecycle(t *testing.T) {
	if !chromeAvailable() {
		t.Skip("no local Chrome or Chromium executable")
	}
	executable, err := os.Executable()
	require.NoError(t, err)
	root := t.TempDir()
	projectDirectory := filepath.Join(root, "project")
	outputDirectory := filepath.Join(root, "output")
	require.NoError(t, os.MkdirAll(projectDirectory, 0o750))
	require.NoError(t, os.MkdirAll(outputDirectory, 0o750))
	factory, err := safedisk.NewFactory(safedisk.FactoryConfig{CWD: root, AllowedPaths: []string{root}, Enabled: true})
	require.NoError(t, err)

	harness := NewHarness(
		WithProjectDir(projectDirectory),
		WithOutputDir(outputDirectory),
		WithSandboxFactory(factory),
		WithSkipBuild(true),
		WithHeadless(true),
		WithBuildTimeout(time.Minute),
		WithServerCommand(executable, "-test.run=^TestHarnessServerProcess$"),
		WithServerArgs("-test.count=1"),
		WithEnv(harnessServerEnv, "1"),
	)
	t.Cleanup(harness.Cleanup)
	require.NoError(t, harness.Setup())
	require.NoError(t, harness.Setup(), "setup runs once")
	assert.False(t, harness.IsInteractive())
	assert.Contains(t, harness.ServerURL(), "http://localhost:")

	page := New(t)
	page.Navigate("/")
	assert.Equal(t, "Fixture", page.Title())
	page.Close()

	session, err := harness.NewSession()
	require.NoError(t, err)
	require.NoError(t, session.Navigate(t.Context(), "/second"))
	title, err := session.EvalReturn(t.Context(), "document.title")
	require.NoError(t, err)
	assert.Equal(t, "Second", title)
	require.NoError(t, session.Close())

	specs := t.TempDir()
	passing := writeSpec(t, specs, "passing.json", `{"requestURL":"/","browserSteps":[
		{"action":"click","selector":"#enabled"},
		{"action":"checkText","selector":"#title","expected":"Hello"}]}`)
	RunSpec(t, passing, WithUpdateGoldens(false), WithSpecTimeout(time.Minute))

	recorder := &recordingTB{TB: t}
	failingAssertion := writeSpec(t, specs, "assertion.json", `{"requestURL":"/","browserSteps":[
		{"action":"checkText","selector":"#title","expected":"Goodbye"}]}`)
	RunSpec(recorder, failingAssertion)
	failures := recorder.take()
	require.Len(t, failures, 1)
	assert.Contains(t, failures[0], `step 1: checkText "#title"`)

	failingStep := writeSpec(t, specs, "step.json", `{"requestURL":"/","browserSteps":[
		{"action":"noSuchAction","selector":"#title"}]}`)
	failures = recorder.attempt(func() { RunSpec(recorder, failingStep) })
	require.Len(t, failures, 1)
	assert.Contains(t, failures[0], "step 1: noSuchAction")

	failures = recorder.attempt(func() { RunSpec(recorder, filepath.Join(specs, "absent.json")) })
	require.Len(t, failures, 1)
	assert.Contains(t, failures[0], "failed to load spec")

	harness.Cleanup()
	assert.Nil(t, getGlobalHarness())
	_, err = harness.NewSession()
	require.Error(t, err)
}

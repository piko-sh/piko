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

package scenarios

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/tests/fuzz/internal/casefile"
)

func TestPrivateOverlayFailureIdentity(t *testing.T) {
	t.Setenv("GOWORK", "off")
	t.Setenv("GOPROXY", "off")
	repository := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(repository, "go.mod"), []byte("module example.org/replay\n\ngo 1.27\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(repository, "doc.go"), []byte("package local\n"), 0o600))
	directory := filepath.Join(repository, "testdata")
	require.NoError(t, os.Mkdir(directory, 0o700))
	for _, testCase := range []struct {
		detail  string
		subject string
	}{
		{detail: "missing observation", subject: "assertion-162"},
		{detail: "PIKO_CHECK sampled-count: wrong count", subject: "sampled-count"},
	} {
		source := "package local\nimport \"testing\"\nfunc TestRecorded(t *testing.T){\n//line piko_fuzz_scenario_test.go:162\n t.Fatal(" + strconv.Quote(testCase.detail) + ")\n}\n"
		require.NoError(t, os.WriteFile(filepath.Join(directory, "release.go"), []byte(source), 0o600))
		err := RunSuite(t.Context(), repository, t.TempDir(), &casefile.Case{Target: "release"}, SuiteTest{Package: ".", Source: "testdata/release.go", TestName: "TestRecorded"})
		require.Error(t, err)
		assert.True(t, strings.HasPrefix(err.Error(), "release/release/"+testCase.subject+":"), err.Error())
	}
}

func TestAssertionIDFallbacks(t *testing.T) {
	assert.Equal(t, "sampled-count", assertionID("PIKO_CHECK sampled-count: wrong"))
	assert.Equal(t, "assertion-9", assertionID("piko_fuzz_scenario_test.go:9: boom"))
	assert.Equal(t, "assertion", assertionID("nothing useful"))
}

func TestScenarioFilesSelectsMinimalSources(t *testing.T) {
	assert.Equal(t, []string{"piko_fuzz_scenario_test.go"}, scenarioFiles("database"))
	assert.Equal(t, []string{"harness_test.go", "piko_fuzz_scenario_test.go"}, scenarioFiles("orchestrator"))
	assert.Nil(t, scenarioFiles("assets"))
}

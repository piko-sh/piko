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

package generator_adapters

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/gosource"
)

func TestGeneratedPikoScenario(t *testing.T) {
	const pkg = "piko.sh/piko/tests/fuzz/internal/scenarios/actions_fixture/actionimpl"
	input := &annotator_dto.TypeSpec{Name: "Input", PackagePath: pkg}
	spec := annotator_dto.ActionSpec{
		Name: "fuzz.collect", HTTPMethod: "POST", PackagePath: pkg, PackageName: "actionimpl",
		StructName: "CaptureAction", ReturnType: input, HasError: true,
		CallParams: []annotator_dto.ParamSpec{{Name: "input", JSONName: "input", GoType: "Input", Struct: input}},
	}
	emitted, err := NewActionWrapperEmitter().EmitWrappers(t.Context(), []annotator_dto.ActionSpec{spec})
//line piko_fuzz_scenario_test.go:41
	require.NoError(t, err)

	fixture, err := filepath.Abs("../../../tests/fuzz/internal/scenarios/actions_fixture")
	require.NoError(t, err)
	scratch := t.TempDir()
	generated := filepath.Join(scratch, "wrappers.go")
	wrapper, err := gosource.Render("wrappers.go", emitted, nil)
	require.NoError(t, err, "PIKO_CHECK wrapper-format: emitted action source must parse")
//line piko_fuzz_scenario_test.go:47
	require.NoError(t, os.WriteFile(generated, wrapper, 0600))
	overlay := map[string]string{
		filepath.Join(fixture, "actionimpl/action.go"):        os.Getenv("PIKO_FUZZ_ACTION_IMPL"),
		filepath.Join(fixture, "wrappers/wrappers.go"):        generated,
		filepath.Join(fixture, "wrappers/invocation_test.go"): os.Getenv("PIKO_FUZZ_ACTION_TEST"),
	}
	data, err := json.Marshal(struct{ Replace map[string]string }{overlay})
	require.NoError(t, err)
	overlayPath := filepath.Join(scratch, "overlay.json")
	require.NoError(t, os.WriteFile(overlayPath, data, 0600))
	cmd := exec.CommandContext(t.Context(), "go", "test", "-overlay", overlayPath, "-buildvcs=false", "-count=1", "-run=^TestInvokeGeneratedBinding$", "./wrappers")
	cmd.Dir = fixture
	result := filepath.Join(scratch, "passed")
	cmd.Env = append(os.Environ(), "PIKO_FUZZ_WRAPPER_RESULT="+result)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	require.NoError(t, cmd.Run(), "PIKO_CHECK invoke: compilation and invocation of emitted action wrapper")
	passed, err := os.ReadFile(result)
	require.NoError(t, err, "PIKO_CHECK invocation-missing: generated wrapper test must run")
	require.Equal(t, os.Getenv("PIKO_FUZZ_CASE_TOKEN"), string(passed))
}

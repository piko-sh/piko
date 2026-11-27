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

package runner

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/daemon/daemon_frontend"
	"piko.sh/piko/tests/fuzz/internal/generate"
)

var (
	errTypeScriptCheckTimeout = errors.New("generated artefacts: TypeScript check exceeded its deadline")
)

func TestGeneratedActionTypeAndRegistrationControls(t *testing.T) {
	for _, compiler := range []string{"tsc", "node"} {
		if _, err := exec.LookPath(compiler); err != nil {
			t.Skip(err)
		}
	}
	source := `import {createActionBuilder, registerActionFunction} from "` + daemon_frontend.CoreModuleImportURL() + `";
const Echo = (input:{label:string})=>createActionBuilder<{label:string}>("contract.Echo",input);
registerActionFunction("contract.Echo",Echo); export const action = { contract: { Echo } };`
	for _, testCase := range []struct {
		name, previous, replacement string
		reject                      bool
	}{
		{name: "valid"},
		{name: "wrong-registration", previous: `registerActionFunction("contract.Echo"`, replacement: `registerActionFunction("other"`, reject: true},
		{name: "input-widened", previous: "input:{label:string}", replacement: "input:{label:any}", reject: true},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			text := source
			if testCase.previous != "" {
				text = strings.Replace(text, testCase.previous, testCase.replacement, 1)
			}
			ctx, cancel := context.WithTimeoutCause(t.Context(), 30*time.Second, errTypeScriptCheckTimeout)
			defer cancel()
			err := checkTypeScriptAction(ctx, t.TempDir(), nil, []byte(text), "example")
			assert.Equal(t, testCase.reject, err != nil, "%v", err)
		})
	}
}

func TestGeneratedPageModelsListsBothPages(t *testing.T) {
	c := generate.New("generated-artifacts", 7, 0)

	models := generatedPageModels(&c)

	assert.Len(t, models, 2)
	assert.Contains(t, models, "/artifact")
	assert.Contains(t, models, "/unaffected")
}

func TestCheckCoerceSymbolRoundTripsTheCount(t *testing.T) {
	assert.NoError(t, checkCoerceSymbol(5))
}

func TestArtefactMismatchNamesTheField(t *testing.T) {
	err := artefactMismatch{field: "edit/manifest", detail: "missing"}

	assert.Equal(t, "generated-artifact: edit/manifest missing", err.Error())
	assert.Equal(t, "edit/manifest", err.FailureSubject())
}

func TestLocalTransportSourceAcceptsAnyRuntimeVersion(t *testing.T) {
	for _, testCase := range []struct {
		name     string
		source   string
		want     string
		mismatch bool
	}{
		{
			name:   "this build's runtime version",
			source: `import {x} from "` + daemon_frontend.CoreModuleImportURL() + `";`,
			want:   `import {x} from "./transport.js";`,
		},
		{
			name:   "another build's runtime version",
			source: `import {x} from "/_piko/dist/0123456789abcdef/ppframework.core.es.js";`,
			want:   `import {x} from "./transport.js";`,
		},
		{
			name:   "unversioned runtime",
			source: `import {x} from "/_piko/dist/ppframework.core.es.js";`,
			want:   `import {x} from "./transport.js";`,
		},
		{
			name:     "no runtime import",
			source:   `import {x} from "./other.js";`,
			mismatch: true,
		},
		{
			name:     "two runtime imports",
			source:   `import {x} from "/_piko/dist/ppframework.core.es.js"; import {y} from "/_piko/dist/0123456789abcdef/ppframework.core.es.js";`,
			mismatch: true,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			got, err := localTransportSource([]byte(testCase.source))
			if testCase.mismatch {
				_, isMismatch := errors.AsType[artefactMismatch](err)
				assert.True(t, isMismatch, "%v", err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, testCase.want, got)
		})
	}
}

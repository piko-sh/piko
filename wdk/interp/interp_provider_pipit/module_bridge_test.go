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

package interp_provider_pipit

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/wdk/modules"
	"pipit.sh/pipit"
	pipitmodule "pipit.sh/pipit/sdk/module"
)

func TestFingerprintMatchesInterpreter(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name       string
		descriptor modules.ModuleDescriptor
	}{
		{
			name: "minimal descriptor",
			descriptor: modules.ModuleDescriptor{
				SchemaVersion: modules.DescriptorVersion,
				Ref:           modules.ModuleRef{Path: "example.com/minimal", Version: "v1.0.0"},
			},
		},
		{
			name: "descriptor with every field",
			descriptor: modules.ModuleDescriptor{
				Entrypoints:     map[string]string{"main": "Run", "init": "Setup"},
				Annotations:     map[string]string{"owner": "team", "tier": "gold"},
				Ref:             modules.ModuleRef{Path: "example.com/full", Version: "v2.3.4"},
				StdlibVersion:   "v1",
				Capabilities:    modules.CapabilitySet{{Axis: "network", Scope: "example.com"}, {Axis: "filesystem", Scope: "/tmp"}},
				SymbolAllowlist: []string{"Zeta", "Alpha"},
				SchemaVersion:   modules.DescriptorVersion,
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			bundle, err := PackageModuleForPiko(context.Background(), pipit.NewInterpreter(), testCase.descriptor, testCase.descriptor.Ref.Path,
				map[string]map[string]string{"": {"lib.go": "package lib\n\nfunc Run() int { return 1 }\n\nfunc Setup() {}\n"}},
				pipit.PackCompiledFileSetToBytes)
			require.NoError(t, err)

			pikoCanonical, err := bundle.Descriptor.MarshalCanonicalJSON()
			require.NoError(t, err)
			pipitCanonical, err := toPipitDescriptor(bundle.Descriptor).MarshalCanonicalJSON()
			require.NoError(t, err)
			assert.JSONEq(t, string(pipitCanonical), string(pikoCanonical))
			assert.Equal(t, string(pipitCanonical), string(pikoCanonical))

			pikoFingerprint, err := bundle.Fingerprint()
			require.NoError(t, err)
			pipitFingerprint, err := toPipitBundle(bundle).Fingerprint()
			require.NoError(t, err)
			assert.Equal(t, pipitFingerprint, pikoFingerprint)
			assert.Equal(t, bundle.Descriptor.Ref.Pin, pikoFingerprint)

			require.NoError(t, bundle.VerifyAgainstRef(bundle.Descriptor.Ref))
		})
	}
}

func TestTranslateModuleErrorMatchesBothVocabularies(t *testing.T) {
	t.Parallel()
	for _, pair := range moduleSentinelPairs {
		t.Run(pair.pikoErr.Error(), func(t *testing.T) {
			t.Parallel()
			original := fmt.Errorf("loading example.com/mod: %w", pair.pipitErr)
			translated := translateModuleError(original)
			require.ErrorIs(t, translated, pair.pikoErr)
			require.ErrorIs(t, translated, pair.pipitErr)
			assert.Equal(t, original.Error(), translated.Error())
		})
	}
}

func TestTranslateModuleErrorPassesOtherErrorsThrough(t *testing.T) {
	t.Parallel()
	unrelated := errors.New("unrelated failure")
	assert.Same(t, unrelated, translateModuleError(unrelated))
	assert.NoError(t, translateModuleError(nil))
}

func TestPackageModuleForPikoReportsFailures(t *testing.T) {
	t.Parallel()
	descriptor := modules.ModuleDescriptor{
		SchemaVersion: modules.DescriptorVersion,
		Ref:           modules.ModuleRef{Path: "example.com/broken", Version: "v1"},
	}
	testCases := []struct {
		interpreter *pipit.Interpreter
		wantErr     error
		name        string
		source      string
	}{
		{name: "compile error", interpreter: pipit.NewInterpreter(), source: "package lib\n\nfunc Broken() int { return \"x\" }\n"},
		{name: "interpreter panic", interpreter: nil, source: "package lib\n", wantErr: errInterpreterPanic},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			bundle, err := PackageModuleForPiko(context.Background(), testCase.interpreter, descriptor, "example.com/broken",
				map[string]map[string]string{"": {"lib.go": testCase.source}}, pipit.PackCompiledFileSetToBytes)
			require.Error(t, err)
			assert.Nil(t, bundle)
			if testCase.wantErr != nil {
				require.ErrorIs(t, err, testCase.wantErr)
			}
		})
	}
}

func TestModuleConversionsHandleNil(t *testing.T) {
	t.Parallel()
	assert.Nil(t, toPipitBundle(nil))
	assert.Nil(t, toPipitDescriptor(nil))
	assert.Nil(t, fromPipitBundle(nil))
	assert.Nil(t, fromPipitDescriptor(nil))
}

func TestModuleConversionsRoundTrip(t *testing.T) {
	t.Parallel()
	original := &pipitmodule.Bundle{
		Descriptor: &pipitmodule.Descriptor{
			Entrypoints:     map[string]string{"main": "Run"},
			Annotations:     map[string]string{"owner": "team"},
			Ref:             pipitmodule.Ref{Path: "example.com/mod", Version: "v1", Pin: "sha256:abc"},
			StdlibVersion:   "v1",
			Capabilities:    pipitmodule.CapabilitySet{{Axis: "network", Scope: "example.com"}},
			SymbolAllowlist: []string{"Run"},
			SchemaVersion:   pipitmodule.DescriptorVersion,
		},
		Bytecode:    []byte("bytecode"),
		TypesExport: []byte("types"),
	}
	assert.Equal(t, original, toPipitBundle(fromPipitBundle(original)))
}

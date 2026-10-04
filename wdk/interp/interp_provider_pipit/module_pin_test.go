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

package interp_provider_pipit_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/wdk/interp/interp_provider_pipit"
	"piko.sh/piko/wdk/modules"
	"piko.sh/piko/wdk/modules/modules_provider_filesystem"
	"pipit.sh/pipit"
)

const (
	pinnedModulePath    = "example.com/pinned"
	pinnedModuleVersion = "v0.0.0"
)

func packageTestModule(t *testing.T) *modules.ModuleBundle {
	t.Helper()
	descriptor := modules.ModuleDescriptor{
		SchemaVersion: modules.DescriptorVersion,
		Ref:           modules.ModuleRef{Path: pinnedModulePath, Version: pinnedModuleVersion},
	}
	bundle, err := interp_provider_pipit.PackageModuleForPiko(context.Background(), pipit.NewInterpreter(), descriptor, pinnedModulePath,
		map[string]map[string]string{"": {"lib.go": "package lib\n\nfunc Answer() int { return 42 }\n"}},
		pipit.PackCompiledFileSetToBytes)
	require.NoError(t, err)
	return bundle
}

func loadModuleIntoPool(t *testing.T, bundle *modules.ModuleBundle, ref modules.ModuleRef, options ...interp_provider_pipit.ProviderOption) (interp_provider_pipit.InterpreterPoolPort, error) {
	t.Helper()
	provider := interp_provider_pipit.NewProvider(options...)
	provider.LoadModule(bundle, ref)
	pool := provider.NewInterpreterPool()
	return pool, pool.LoadModules(context.Background())
}

func TestLoadModulesVerifiesPins(t *testing.T) {
	t.Parallel()
	bundle := packageTestModule(t)
	pikoFingerprint, err := bundle.Fingerprint()
	require.NoError(t, err)

	testCases := []struct {
		wantErr error
		name    string
		pin     string
		options []interp_provider_pipit.ProviderOption
	}{
		{name: "unpinned ref rejected by default", pin: "", wantErr: modules.ErrUnpinnedModuleRef},
		{name: "unpinned ref allowed when opted in", pin: "", options: []interp_provider_pipit.ProviderOption{interp_provider_pipit.WithUnpinnedModuleLoads()}},
		{name: "pin produced by the interpreter accepted", pin: bundle.Descriptor.Ref.Pin},
		{name: "pin computed by piko accepted", pin: pikoFingerprint},
		{name: "wrong pin rejected", pin: "sha256:" + strings.Repeat("0", 64), wantErr: modules.ErrIntegrityMismatch},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			ref := modules.ModuleRef{Path: pinnedModulePath, Version: pinnedModuleVersion, Pin: testCase.pin}
			pool, loadErr := loadModuleIntoPool(t, bundle, ref, testCase.options...)
			_, getErr := pool.Get()
			if testCase.wantErr == nil {
				require.NoError(t, loadErr)
				require.NoError(t, getErr)
				return
			}
			require.ErrorIs(t, loadErr, testCase.wantErr)
			require.ErrorIs(t, getErr, testCase.wantErr)
		})
	}
}

func TestLoadModulesErrorDoesNotRepeatSentinelText(t *testing.T) {
	t.Parallel()
	ref := modules.ModuleRef{Path: pinnedModulePath, Version: pinnedModuleVersion, Pin: "sha256:" + strings.Repeat("0", 64)}
	_, err := loadModuleIntoPool(t, packageTestModule(t), ref)
	require.ErrorIs(t, err, modules.ErrIntegrityMismatch)
	assert.Equal(t, 1, strings.Count(err.Error(), "integrity mismatch"), err.Error())
}

func TestPoolGetRequiresLoadModulesWhenModulesQueued(t *testing.T) {
	t.Parallel()
	bundle := packageTestModule(t)
	provider := interp_provider_pipit.NewProvider()
	provider.LoadModule(bundle, modules.ModuleRef{Path: pinnedModulePath, Version: pinnedModuleVersion, Pin: bundle.Descriptor.Ref.Pin})
	pool := provider.NewInterpreterPool()

	_, err := pool.Get()
	require.ErrorContains(t, err, "LoadModules")

	require.NoError(t, pool.LoadModules(context.Background()))
	require.NoError(t, pool.LoadModules(context.Background()), "a second LoadModules returns the first result")
	_, err = pool.Get()
	require.NoError(t, err)
}

func TestPoolGetWithoutModulesNeedsNoLoad(t *testing.T) {
	t.Parallel()
	_, err := interp_provider_pipit.NewProvider().NewInterpreterPool().Get()
	require.NoError(t, err)
}

func TestLoadModulesHonoursCancellation(t *testing.T) {
	t.Parallel()
	bundle := packageTestModule(t)
	provider := interp_provider_pipit.NewProvider()
	provider.LoadModule(bundle, modules.ModuleRef{Path: pinnedModulePath, Version: pinnedModuleVersion, Pin: bundle.Descriptor.Ref.Pin})
	pool := provider.NewInterpreterPool()

	cause := errors.New("startup abandoned")
	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(cause)
	require.ErrorIs(t, pool.LoadModules(ctx), cause)
	_, err := pool.Get()
	require.ErrorIs(t, err, cause)
}

func TestLoadedModuleIsImportable(t *testing.T) {
	t.Parallel()
	bundle := packageTestModule(t)
	pool, err := loadModuleIntoPool(t, bundle, modules.ModuleRef{Path: pinnedModulePath, Version: pinnedModuleVersion, Pin: bundle.Descriptor.Ref.Pin})
	require.NoError(t, err)
	interpreter, err := pool.Get()
	require.NoError(t, err)

	err = interpreter.CompileAndExecute(context.Background(), "app", map[string]map[string]string{
		"": {"main.go": "package app\n\nimport lib \"" + pinnedModulePath + "\"\n\nvar Answer = lib.Answer()\n"},
	})
	require.NoError(t, err)
}

func TestPinnedModuleLoadsThroughFilesystemProvider(t *testing.T) {
	t.Parallel()
	bundle := packageTestModule(t)
	require.NotEmpty(t, bundle.TypesExport)

	store, err := modules_provider_filesystem.New(t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, store.Close()) })
	require.NoError(t, store.Write(bundle))

	ref := modules.ModuleRef{Path: pinnedModulePath, Version: pinnedModuleVersion, Pin: bundle.Descriptor.Ref.Pin}
	resolved, err := store.Resolve(context.Background(), ref)
	require.NoError(t, err)
	require.NoError(t, resolved.VerifyAgainstRef(ref))

	_, err = loadModuleIntoPool(t, resolved, ref)
	require.NoError(t, err)
}

func TestRestrictedSurfacePermitsLoadedModule(t *testing.T) {
	t.Parallel()
	bundle := packageTestModule(t)
	pool, err := loadModuleIntoPool(t, bundle,
		modules.ModuleRef{Path: pinnedModulePath, Version: pinnedModuleVersion, Pin: bundle.Descriptor.Ref.Pin},
		interp_provider_pipit.WithRestrictedSymbolSurface())
	require.NoError(t, err)
	interpreter, err := pool.Get()
	require.NoError(t, err)

	err = interpreter.CompileAndExecute(context.Background(), "app", map[string]map[string]string{
		"": {"main.go": "package app\n\nimport lib \"" + pinnedModulePath + "\"\n\nvar Answer = lib.Answer()\n"},
	})
	require.NoError(t, err)

	err = interpreter.CompileAndExecute(context.Background(), "app", map[string]map[string]string{
		"": {"main.go": "package app\n\nimport _ \"os\"\n"},
	})
	require.ErrorContains(t, err, "not permitted")
}

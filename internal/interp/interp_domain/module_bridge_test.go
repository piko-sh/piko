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

package interp_domain_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/interp/interp_adapters"
	"piko.sh/piko/internal/interp/interp_domain"
	"piko.sh/piko/internal/modules/modules_domain"
)

func TestLoadModule_BytecodeBridgesSymbols(t *testing.T) {
	t.Parallel()

	libSources := map[string]map[string]string{
		"": {
			"lib.go": `package lib

func Add(a, b int) int {
	return a + b
}
`,
		},
	}

	builder := interp_domain.NewService()
	descriptor := modules_domain.ModuleDescriptor{
		SchemaVersion: modules_domain.DescriptorVersion,
		Ref:           modules_domain.ModuleRef{Path: "example.com/lib", Version: "v0.0.0"},
	}
	bundle, err := builder.PackageModule(context.Background(),
		descriptor,
		"example.com/lib",
		libSources,
		interp_adapters.PackCompiledFileSetToBytes,
	)
	require.NoError(t, err)
	require.NotEmpty(t, bundle.Bytecode, "bundle should carry bytecode")
	require.NotEmpty(t, bundle.TypesExport, "bundle should carry TypesExport so downstream imports resolve")

	consumer := interp_domain.NewService()
	ref := modules_domain.ModuleRef{
		Path:    descriptor.Ref.Path,
		Version: descriptor.Ref.Version,
		Pin:     bundle.Descriptor.Ref.Pin,
	}
	loaded, err := consumer.LoadModule(context.Background(), bundle, ref, nil, interp_adapters.LoadCompiledFromBytes)
	require.NoError(t, err)
	require.Equal(t, bundle.Descriptor.Ref.Pin, loaded.Fingerprint)

	mainSources := map[string]map[string]string{
		"": {
			"main.go": `package main

import "example.com/lib"

func entrypoint() int {
	return lib.Add(2, 5)
}

func main() {}
`,
		},
	}
	mainCfs, err := consumer.CompileProgram(context.Background(), "main", mainSources)
	require.NoError(t, err, "main should compile against the bytecode-loaded lib")

	result, err := consumer.ExecuteEntrypoint(context.Background(), mainCfs, "entrypoint")
	require.NoError(t, err)
	require.Equal(t, "7", fmt.Sprint(result))
}

func TestLoadModule_MissingTypesExport_DownstreamCompileFails(t *testing.T) {
	t.Parallel()

	libSources := map[string]map[string]string{
		"": {
			"lib.go": `package lib

func Greet() string { return "hi" }
`,
		},
	}

	builder := interp_domain.NewService()
	descriptor := modules_domain.ModuleDescriptor{
		SchemaVersion: modules_domain.DescriptorVersion,
		Ref:           modules_domain.ModuleRef{Path: "example.com/lib2", Version: "v0.0.0"},
	}
	bundle, err := builder.PackageModule(context.Background(),
		descriptor,
		"example.com/lib2",
		libSources,
		interp_adapters.PackCompiledFileSetToBytes,
	)
	require.NoError(t, err)

	bundle.TypesExport = nil
	fingerprint, err := bundle.Fingerprint()
	require.NoError(t, err)
	bundle.Descriptor.Ref.Pin = fingerprint

	consumer := interp_domain.NewService()
	ref := modules_domain.ModuleRef{
		Path:    descriptor.Ref.Path,
		Version: descriptor.Ref.Version,
		Pin:     fingerprint,
	}
	_, err = consumer.LoadModule(context.Background(), bundle, ref, nil, interp_adapters.LoadCompiledFromBytes)
	require.NoError(t, err, "LoadModule should still succeed for legacy bundles (no TypesExport)")

	mainSources := map[string]map[string]string{
		"": {
			"main.go": `package main

import "example.com/lib2"

func entrypoint() string { return lib2.Greet() }

func main() {}
`,
		},
	}
	_, err = consumer.CompileProgram(context.Background(), "main", mainSources)
	require.Error(t, err, "CompileProgram should fail when importing a bundle without TypesExport")
}

func TestLoadModule_VarExportRoundtrip(t *testing.T) {
	t.Parallel()

	libSources := map[string]map[string]string{
		"": {
			"lib.go": `package varlib

var Answer = 40 + 2

var Greeting = "hello"
`,
		},
	}

	builder := interp_domain.NewService()
	descriptor := modules_domain.ModuleDescriptor{
		SchemaVersion: modules_domain.DescriptorVersion,
		Ref:           modules_domain.ModuleRef{Path: "example.com/varlib", Version: "v0.0.0"},
	}
	bundle, err := builder.PackageModule(context.Background(),
		descriptor,
		"example.com/varlib",
		libSources,
		interp_adapters.PackCompiledFileSetToBytes,
	)
	require.NoError(t, err)
	require.NotEmpty(t, bundle.Bytecode)
	require.NotEmpty(t, bundle.TypesExport)

	consumer := interp_domain.NewService()
	ref := modules_domain.ModuleRef{
		Path:    descriptor.Ref.Path,
		Version: descriptor.Ref.Version,
		Pin:     bundle.Descriptor.Ref.Pin,
	}
	_, err = consumer.LoadModule(context.Background(), bundle, ref, nil, interp_adapters.LoadCompiledFromBytes)
	require.NoError(t, err)

	mainSources := map[string]map[string]string{
		"": {
			"main.go": `package main

import "example.com/varlib"

func entrypoint() int {
	return varlib.Answer + len(varlib.Greeting)
}

func main() {}
`,
		},
	}
	mainCfs, err := consumer.CompileProgram(context.Background(), "main", mainSources)
	require.NoError(t, err)

	require.NoError(t, consumer.ExecuteInits(context.Background(), mainCfs))

	result, err := consumer.ExecuteEntrypoint(context.Background(), mainCfs, "entrypoint")
	require.NoError(t, err)

	require.Equal(t, "47", fmt.Sprint(result))
}

func TestLoadModule_VarSlotsDoNotCollide(t *testing.T) {
	t.Parallel()

	makeLib := func(value int) ([]byte, []byte, string, string) {
		libSources := map[string]map[string]string{
			"": {
				"lib.go": fmt.Sprintf(`package twolib%d

var Counter = %d
`, value, value),
			},
		}
		builder := interp_domain.NewService()
		path := fmt.Sprintf("example.com/twolib%d", value)
		descriptor := modules_domain.ModuleDescriptor{
			SchemaVersion: modules_domain.DescriptorVersion,
			Ref:           modules_domain.ModuleRef{Path: path, Version: "v0.0.0"},
		}
		bundle, err := builder.PackageModule(context.Background(),
			descriptor, path, libSources, interp_adapters.PackCompiledFileSetToBytes,
		)
		require.NoError(t, err)
		return bundle.Bytecode, bundle.TypesExport, path, bundle.Descriptor.Ref.Pin
	}

	bcA, teA, pathA, pinA := makeLib(10)
	bcB, teB, pathB, pinB := makeLib(20)

	consumer := interp_domain.NewService()
	bundleA := &modules_domain.ModuleBundle{
		Descriptor:  &modules_domain.ModuleDescriptor{SchemaVersion: modules_domain.DescriptorVersion, Ref: modules_domain.ModuleRef{Path: pathA, Version: "v0.0.0"}},
		Bytecode:    bcA,
		TypesExport: teA,
	}
	bundleA.Descriptor.Ref.Pin = pinA
	bundleB := &modules_domain.ModuleBundle{
		Descriptor:  &modules_domain.ModuleDescriptor{SchemaVersion: modules_domain.DescriptorVersion, Ref: modules_domain.ModuleRef{Path: pathB, Version: "v0.0.0"}},
		Bytecode:    bcB,
		TypesExport: teB,
	}
	bundleB.Descriptor.Ref.Pin = pinB

	_, err := consumer.LoadModule(context.Background(), bundleA, modules_domain.ModuleRef{Path: pathA, Version: "v0.0.0", Pin: pinA}, nil, interp_adapters.LoadCompiledFromBytes)
	require.NoError(t, err)
	_, err = consumer.LoadModule(context.Background(), bundleB, modules_domain.ModuleRef{Path: pathB, Version: "v0.0.0", Pin: pinB}, nil, interp_adapters.LoadCompiledFromBytes)
	require.NoError(t, err)

	mainSources := map[string]map[string]string{
		"": {
			"main.go": fmt.Sprintf(`package main

import (
	"%s"
	"%s"
)

func entrypoint() int {
	return twolib10.Counter*100 + twolib20.Counter
}

func main() {}
`, pathA, pathB),
		},
	}
	mainCfs, err := consumer.CompileProgram(context.Background(), "main", mainSources)
	require.NoError(t, err)

	require.NoError(t, consumer.ExecuteInits(context.Background(), mainCfs))

	result, err := consumer.ExecuteEntrypoint(context.Background(), mainCfs, "entrypoint")
	require.NoError(t, err)

	require.Equal(t, "1020", fmt.Sprint(result))
}

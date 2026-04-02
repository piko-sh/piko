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

package driver_symbols_extract

import (
	"errors"
	"fmt"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"piko.sh/piko/wdk/safedisk"
)

// FunctionConfig holds per-function type overrides for generic
// dispatch wrapper generation.
type FunctionConfig struct {
	// ElementTypes lists the concrete element types for
	// single-type-parameter functions.
	ElementTypes []string `yaml:"element_types"`

	// KeyTypes lists the concrete key types for map-type-parameter functions.
	KeyTypes []string `yaml:"key_types"`

	// ValueTypes lists the concrete value types for map-type-parameter functions.
	ValueTypes []string `yaml:"value_types"`
}

// PackageConfig holds the configuration for a single package to
// extract, including optional generic type configuration.
type PackageConfig struct {
	// Functions holds per-function type overrides that take
	// precedence over package defaults.
	Functions map[string]FunctionConfig

	// ImportPath is the Go import path of the package.
	ImportPath string

	// BuildTag is an optional Go build constraint to emit in the
	// generated file header (e.g. "!js" to exclude from WASM builds).
	BuildTag string

	// ElementTypes lists the default concrete element types for
	// generic functions.
	ElementTypes []string

	// KeyTypes lists the default concrete key types for generic map functions.
	KeyTypes []string

	// ValueTypes lists the default concrete value types for generic map functions.
	ValueTypes []string
}

// IsGeneric returns true if the package has generic type
// configuration.
//
// Returns true when any element types, key types, or per-function
// overrides are configured.
func (packageConfig PackageConfig) IsGeneric() bool {
	return len(packageConfig.ElementTypes) > 0 || len(packageConfig.KeyTypes) > 0 || len(packageConfig.Functions) > 0
}

// TypesForFunc returns the element, key, and value types to use for
// a specific function, with per-function overrides taking
// precedence.
//
// Takes name (string) which specifies the function name to look up.
//
// Returns the element, key, and value type slices for the
// function.
func (packageConfig PackageConfig) TypesForFunc(name string) (elemTypes, keyTypes, valTypes []string) {
	fc, ok := packageConfig.Functions[name]
	if !ok {
		return packageConfig.ElementTypes, packageConfig.KeyTypes, packageConfig.ValueTypes
	}

	elemTypes = packageConfig.ElementTypes
	if len(fc.ElementTypes) > 0 {
		elemTypes = fc.ElementTypes
	}

	keyTypes = packageConfig.KeyTypes
	if len(fc.KeyTypes) > 0 {
		keyTypes = fc.KeyTypes
	}

	valTypes = packageConfig.ValueTypes
	if len(fc.ValueTypes) > 0 {
		valTypes = fc.ValueTypes
	}

	return elemTypes, keyTypes, valTypes
}

// Manifest describes which packages to extract and where to write the
// generated symbol files.
type Manifest struct {
	// Package is the Go package name for the generated files.
	Package string `yaml:"package"`

	// Output is the directory path for generated files, relative to
	// the repository root.
	Output string `yaml:"output"`

	// Packages lists the packages to extract. Supports both simple
	// strings and objects with generic configuration.
	Packages []PackageConfig `yaml:"-"`
}

// UnmarshalYAML implements custom YAML parsing to support mixed
// package list formats: simple strings and maps with generic config.
//
// Takes value (*yaml.Node) which provides the YAML node to decode.
//
// Returns an error if the YAML structure is invalid.
func (m *Manifest) UnmarshalYAML(value *yaml.Node) error {
	var raw struct {
		Package  string    `yaml:"package"`
		Output   string    `yaml:"output"`
		Packages yaml.Node `yaml:"packages"`
	}
	if err := value.Decode(&raw); err != nil {
		return err
	}
	m.Package = raw.Package
	m.Output = raw.Output

	if raw.Packages.Kind != yaml.SequenceNode {
		return errors.New("'packages' must be a sequence")
	}

	for _, item := range raw.Packages.Content {
		switch item.Kind {
		case yaml.ScalarNode:
			m.Packages = append(m.Packages, PackageConfig{ImportPath: item.Value})

		case yaml.MappingNode:
			packageConfig, err := parseGenericPackageNode(item)
			if err != nil {
				return err
			}
			m.Packages = append(m.Packages, packageConfig)

		default:
			return fmt.Errorf("unexpected node kind in packages list: %v", item.Kind)
		}
	}

	return nil
}

// ImportPaths returns the list of import paths from all package
// configs.
//
// Returns a string slice of all configured import paths.
func (m *Manifest) ImportPaths() []string {
	paths := make([]string, len(m.Packages))
	for i, p := range m.Packages {
		paths[i] = p.ImportPath
	}
	return paths
}

// GenericConfigs returns a map from import path to PackageConfig for
// packages with generic configuration.
//
// Returns a map keyed by import path containing only generic package configs.
func (m *Manifest) GenericConfigs() map[string]PackageConfig {
	configs := make(map[string]PackageConfig)
	for _, p := range m.Packages {
		if p.IsGeneric() {
			configs[p.ImportPath] = p
		}
	}
	return configs
}

// LoadManifest reads and parses a YAML manifest file.
//
// Takes path (string) which specifies the filesystem path to the
// manifest.
//
// Returns the parsed Manifest or an error if reading or parsing
// fails.
func LoadManifest(path string) (*Manifest, error) {
	data, err := readManifestFile(path, nil)
	if err != nil {
		return nil, fmt.Errorf("reading manifest %s: %w", path, err)
	}

	return parseManifest(path, data)
}

// LoadManifestWithFactory reads and parses a YAML manifest file using the
// given sandbox factory.
//
// Takes path (string) which specifies the filesystem path to the manifest.
// Takes factory (safedisk.Factory) which creates sandboxes when non-nil.
//
// Returns the parsed Manifest or an error if reading or parsing fails.
func LoadManifestWithFactory(path string, factory safedisk.Factory) (*Manifest, error) {
	data, err := readManifestFile(path, factory)
	if err != nil {
		return nil, fmt.Errorf("reading manifest %s: %w", path, err)
	}

	return parseManifest(path, data)
}

// parseManifest parses manifest data and validates required fields.
//
// Takes path (string) which identifies the manifest for error messages.
// Takes data ([]byte) which contains the raw YAML content.
//
// Returns the parsed Manifest or an error if parsing or validation fails.
func parseManifest(path string, data []byte) (*Manifest, error) {
	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parsing manifest %s: %w", path, err)
	}

	if m.Package == "" {
		return nil, fmt.Errorf("manifest %s: 'package' is required", path)
	}
	if m.Output == "" {
		return nil, fmt.Errorf("manifest %s: 'output' is required", path)
	}
	if len(m.Packages) == 0 {
		return nil, fmt.Errorf("manifest %s: 'packages' must list at least one package", path)
	}

	return &m, nil
}

// parseGenericPackageNode parses a YAML mapping node into a
// PackageConfig with generic type configuration.
//
// Takes node (*yaml.Node) which provides the YAML mapping node to
// parse.
//
// Returns the parsed PackageConfig or an error if the node is
// invalid.
func parseGenericPackageNode(node *yaml.Node) (PackageConfig, error) {
	const minMappingNodeChildren = 2
	if len(node.Content) < minMappingNodeChildren {
		return PackageConfig{}, errors.New("empty package mapping")
	}

	importPath := node.Content[0].Value

	var config struct {
		Functions    map[string]yaml.Node `yaml:"functions"`
		BuildTag     string               `yaml:"build_tag"`
		ElementTypes []string             `yaml:"element_types"`
		KeyTypes     []string             `yaml:"key_types"`
		ValueTypes   []string             `yaml:"value_types"`
	}
	if err := node.Content[1].Decode(&config); err != nil {
		return PackageConfig{}, fmt.Errorf("parsing config for %s: %w", importPath, err)
	}

	packageConfig := PackageConfig{
		ImportPath:   importPath,
		BuildTag:     config.BuildTag,
		ElementTypes: config.ElementTypes,
		KeyTypes:     config.KeyTypes,
		ValueTypes:   config.ValueTypes,
	}

	if len(config.Functions) > 0 {
		packageConfig.Functions = make(map[string]FunctionConfig, len(config.Functions))
		for name := range config.Functions {
			fc, err := parseFunctionConfigNode(name, new(config.Functions[name]))
			if err != nil {
				return PackageConfig{}, fmt.Errorf("parsing function %s.%s: %w", importPath, name, err)
			}
			packageConfig.Functions[name] = fc
		}
	}

	return packageConfig, nil
}

// parseFunctionConfigNode parses a per-function config node
// supporting both sequence and mapping forms.
//
// Takes name (string) which specifies the function name for error
// messages.
// Takes node (*yaml.Node) which provides the YAML node to parse.
//
// Returns the parsed FunctionConfig or an error if the node is
// invalid.
func parseFunctionConfigNode(name string, node *yaml.Node) (FunctionConfig, error) {
	switch node.Kind {
	case yaml.SequenceNode:
		var types []string
		if err := node.Decode(&types); err != nil {
			return FunctionConfig{}, fmt.Errorf("decoding type list for %s: %w", name, err)
		}
		return FunctionConfig{ElementTypes: types}, nil

	case yaml.MappingNode:
		var fc FunctionConfig
		if err := node.Decode(&fc); err != nil {
			return FunctionConfig{}, fmt.Errorf("decoding config for %s: %w", name, err)
		}
		return fc, nil

	default:
		return FunctionConfig{}, fmt.Errorf("function %s: expected sequence or mapping, got %v", name, node.Kind)
	}
}

// readManifestFile reads a manifest file using a sandboxed reader
// scoped to the file's parent directory.
//
// Takes path (string) which specifies the filesystem path to read.
//
// Returns the file contents as bytes or an error if reading fails.
func readManifestFile(path string, factory safedisk.Factory) ([]byte, error) {
	directory := filepath.Dir(path)
	base := filepath.Base(path)

	var sandbox safedisk.Sandbox
	var err error
	if factory != nil {
		sandbox, err = factory.Create("manifest", directory, safedisk.ModeReadOnly)
	} else {
		sandbox, err = safedisk.NewNoOpSandbox(directory, safedisk.ModeReadOnly)
	}
	if err != nil {
		return nil, err
	}
	defer func() { _ = sandbox.Close() }()

	return sandbox.ReadFile(base)
}

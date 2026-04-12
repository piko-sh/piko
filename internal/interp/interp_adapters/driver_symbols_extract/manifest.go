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
	"go/parser"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"piko.sh/piko/wdk/safedisk"
)

var (
	// errManifestTypeArg is returned when a manifest type-argument string is not a parseable
	// Go type expression. A malformed string such as "map[int" would otherwise reach the
	// codegen and produce a non-compiling generated file with no clear diagnostic.
	errManifestTypeArg = errors.New("invalid type argument")

	// errManifestUnknownKey is returned when a per-package config mapping contains a key
	// outside the recognised allow-list, surfacing manifest typos (for example
	// "generic_funcs" instead of "generic_functions") that would otherwise be silently
	// dropped.
	errManifestUnknownKey = errors.New("unknown package config key")

	// errManifestEmptyImportPath is returned when a package mapping node supplies an empty
	// import path.
	errManifestEmptyImportPath = errors.New("empty import path")
)

// FunctionConfig holds per-function type overrides for generic dispatch wrapper
// generation.
type FunctionConfig struct {
	// ElementTypes lists the concrete element types for single-type-parameter functions.
	ElementTypes []string `yaml:"element_types"`

	// KeyTypes lists the concrete key types for map-type-parameter functions.
	KeyTypes []string `yaml:"key_types"`

	// ValueTypes lists the concrete value types for map-type-parameter functions.
	ValueTypes []string `yaml:"value_types"`
}

// PackageConfig holds the configuration for a single package to extract, including
// optional generic type configuration.
type PackageConfig struct {
	// Functions holds per-function type overrides that take precedence over package
	// defaults.
	Functions map[string]FunctionConfig

	// GenericInstantiations holds explicit type-argument instantiations.
	//
	// Each map key is a generic top-level function name and each value is a list of
	// instantiations; an instantiation is itself a list of type-argument names, one per type
	// parameter. For example, OnceValues -> [["int", "error"]] instantiates
	// sync.OnceValues[int, error].
	GenericInstantiations map[string][][]string

	// GenericTypes opts generic stdlib types into the native-backed pipeline.
	//
	// Each map key is a generic type name and each value is the canonical erased
	// type-argument list (one entry per type parameter) used to bake the real method-bearing
	// reflect.Type into the generated file. For example, Pointer -> ["struct{}"] registers
	// atomic.Pointer via the erased atomic.Pointer[struct{}].
	GenericTypes map[string][]string

	// ImportPath is the Go import path of the package.
	ImportPath string

	// BuildTag is an optional Go build constraint to emit in the generated file header (e.g.
	// "!js" to exclude from WASM builds).
	BuildTag string

	// GenericFallback selects the default-clause behaviour for the generated type-switch
	// dispatch when the runtime element type is not in the enumerated fast-path set.
	//
	// Accepted values:
	//   - "" (default) = "auto"; emit a reflective fallback when a hand-written helper
	//     exists for the function, otherwise panic.
	//   - "auto" = same as default.
	//   - "reflect" = force the reflective fallback path. Equivalent to "auto" while the
	//     only helpers live in the reflectiveFallbackTable.
	//   - "panic" = force the panic("unsupported type %T") behaviour even when a reflective
	//     helper exists. Use when host policy requires strict typing and prefers a panic
	//     over a slower silent fallback.
	GenericFallback string

	// ElementTypes lists the default concrete element types for generic functions.
	ElementTypes []string

	// KeyTypes lists the default concrete key types for generic map functions.
	KeyTypes []string

	// ValueTypes lists the default concrete value types for generic map functions.
	ValueTypes []string
}

// IsGeneric returns true if the package has generic type configuration.
//
// Returns true when any element types, key types, per-function overrides, or explicit
// generic instantiations are configured.
func (packageConfig PackageConfig) IsGeneric() bool {
	return len(packageConfig.ElementTypes) > 0 ||
		len(packageConfig.KeyTypes) > 0 ||
		len(packageConfig.Functions) > 0 ||
		len(packageConfig.GenericInstantiations) > 0 ||
		len(packageConfig.GenericTypes) > 0
}

// TypesForFunc returns the element, key, and value types to use for a specific function,
// with per-function overrides taking precedence.
//
// Takes name (string) which specifies the function name to look up.
//
// Returns the element, key, and value type slices for the function.
func (packageConfig PackageConfig) TypesForFunc(name string) (elemTypes, keyTypes, valTypes []string) {
	functionConfig, ok := packageConfig.Functions[name]
	if !ok {
		return packageConfig.ElementTypes, packageConfig.KeyTypes, packageConfig.ValueTypes
	}

	elemTypes = packageConfig.ElementTypes
	if len(functionConfig.ElementTypes) > 0 {
		elemTypes = functionConfig.ElementTypes
	}

	keyTypes = packageConfig.KeyTypes
	if len(functionConfig.KeyTypes) > 0 {
		keyTypes = functionConfig.KeyTypes
	}

	valTypes = packageConfig.ValueTypes
	if len(functionConfig.ValueTypes) > 0 {
		valTypes = functionConfig.ValueTypes
	}

	return elemTypes, keyTypes, valTypes
}

// Manifest describes which packages to extract and where to write the generated symbol
// files.
type Manifest struct {
	// Package is the Go package name for the generated files.
	Package string `yaml:"package"`

	// Output is the directory path for generated files, relative to the repository root.
	Output string `yaml:"output"`

	// Packages lists the packages to extract. Supports both simple strings and objects with
	// generic configuration.
	Packages []PackageConfig `yaml:"-"`
}

// UnmarshalYAML implements custom YAML parsing to support mixed package list formats:
// simple strings and maps with generic config.
//
// Takes value (*yaml.Node) which provides the YAML node to decode.
//
// Returns an error if the YAML structure is invalid.
func (manifest *Manifest) UnmarshalYAML(value *yaml.Node) error {
	var raw struct {
		Package  string    `yaml:"package"`
		Output   string    `yaml:"output"`
		Packages yaml.Node `yaml:"packages"`
	}
	if err := value.Decode(&raw); err != nil {
		return err
	}
	manifest.Package = raw.Package
	manifest.Output = raw.Output

	if raw.Packages.Kind != yaml.SequenceNode {
		return errors.New("'packages' must be a sequence")
	}

	for _, item := range raw.Packages.Content {
		switch item.Kind {
		case yaml.ScalarNode:
			manifest.Packages = append(manifest.Packages, PackageConfig{ImportPath: item.Value})

		case yaml.MappingNode:
			packageConfig, err := parseGenericPackageNode(item)
			if err != nil {
				return err
			}
			manifest.Packages = append(manifest.Packages, packageConfig)

		default:
			return fmt.Errorf("unexpected node kind in packages list: %v", item.Kind)
		}
	}

	return nil
}

// ImportPaths returns the list of import paths from all package configs.
//
// Returns a string slice of all configured import paths.
func (manifest *Manifest) ImportPaths() []string {
	paths := make([]string, len(manifest.Packages))
	for i := range manifest.Packages {
		paths[i] = manifest.Packages[i].ImportPath
	}
	return paths
}

// GenericConfigs returns a map from import path to PackageConfig for packages with
// generic configuration.
//
// Returns a map keyed by import path containing only generic package configs.
func (manifest *Manifest) GenericConfigs() map[string]PackageConfig {
	configs := make(map[string]PackageConfig)
	for i := range manifest.Packages {
		packageConfig := &manifest.Packages[i]
		if packageConfig.IsGeneric() {
			configs[packageConfig.ImportPath] = *packageConfig
		}
	}
	return configs
}

// LoadManifest reads and parses a YAML manifest file.
//
// Takes path (string) which specifies the filesystem path to the manifest.
//
// Returns the parsed Manifest or an error if reading or parsing fails.
func LoadManifest(path string) (*Manifest, error) {
	data, err := readManifestFile(path, nil)
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
	var manifest Manifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("parsing manifest %s: %w", path, err)
	}

	if manifest.Package == "" {
		return nil, fmt.Errorf("manifest %s: 'package' is required", path)
	}
	if manifest.Output == "" {
		return nil, fmt.Errorf("manifest %s: 'output' is required", path)
	}
	if len(manifest.Packages) == 0 {
		return nil, fmt.Errorf("manifest %s: 'packages' must list at least one package", path)
	}

	return &manifest, nil
}

var (
	// knownPackageConfigKeys is the allow-list of recognised keys in a per-package config
	// mapping.
	//
	// Decoding with this allow-list surfaces manifest typos rather than silently dropping
	// the offending key.
	knownPackageConfigKeys = map[string]struct{}{
		"functions":         {},
		"generic_functions": {},
		"generic_types":     {},
		"build_tag":         {},
		"element_types":     {},
		"key_types":         {},
		"value_types":       {},
		"generic_fallback":  {},
	}
)

// parseGenericPackageNode parses a YAML node into a PackageConfig.
//
// The per-package config keys are validated against an allow-list so a manifest typo
// surfaces as an error, and every generic type-argument string is validated as a Go type
// expression so a malformed entry is rejected at the manifest boundary rather than
// producing non-compiling generated code.
//
// Takes node (*yaml.Node) which provides the YAML mapping node to parse.
//
// Returns PackageConfig which holds the parsed configuration.
// Returns error when the node is invalid.
func parseGenericPackageNode(node *yaml.Node) (PackageConfig, error) {
	const minMappingNodeChildren = 2
	if len(node.Content) < minMappingNodeChildren {
		return PackageConfig{}, errors.New("empty package mapping")
	}

	importPath := node.Content[0].Value
	if importPath == "" {
		return PackageConfig{}, errManifestEmptyImportPath
	}

	configNode := node.Content[1]
	if err := checkKnownPackageConfigKeys(importPath, configNode); err != nil {
		return PackageConfig{}, err
	}

	var config struct {
		Functions        map[string]yaml.Node  `yaml:"functions"`
		GenericFunctions map[string][][]string `yaml:"generic_functions"`
		GenericTypes     map[string][]string   `yaml:"generic_types"`
		BuildTag         string                `yaml:"build_tag"`
		GenericFallback  string                `yaml:"generic_fallback"`
		ElementTypes     []string              `yaml:"element_types"`
		KeyTypes         []string              `yaml:"key_types"`
		ValueTypes       []string              `yaml:"value_types"`
	}
	if err := configNode.Decode(&config); err != nil {
		return PackageConfig{}, fmt.Errorf("parsing config for %s: %w", importPath, err)
	}

	if err := validateGenericTypeArgs(importPath, config.GenericFunctions, config.GenericTypes); err != nil {
		return PackageConfig{}, err
	}

	if err := validateGenericFallback(importPath, config.GenericFallback); err != nil {
		return PackageConfig{}, err
	}

	packageConfig := PackageConfig{
		ImportPath:            importPath,
		BuildTag:              config.BuildTag,
		ElementTypes:          config.ElementTypes,
		KeyTypes:              config.KeyTypes,
		ValueTypes:            config.ValueTypes,
		GenericInstantiations: config.GenericFunctions,
		GenericTypes:          config.GenericTypes,
		GenericFallback:       config.GenericFallback,
	}

	if len(config.Functions) > 0 {
		packageConfig.Functions = make(map[string]FunctionConfig, len(config.Functions))
		for name := range config.Functions {
			functionConfig, err := parseFunctionConfigNode(name, new(config.Functions[name]))
			if err != nil {
				return PackageConfig{}, fmt.Errorf("parsing function %s.%s: %w", importPath, name, err)
			}
			packageConfig.Functions[name] = functionConfig
		}
	}

	return packageConfig, nil
}

// checkKnownPackageConfigKeys rejects any key in a per-package config mapping that is not
// on the recognised allow-list.
//
// Takes importPath (string) which identifies the package for error messages.
// Takes configNode (*yaml.Node) which is the mapping node holding the per-package
// configuration.
//
// Returns an error wrapping errManifestUnknownKey when an unrecognised key is present.
func checkKnownPackageConfigKeys(importPath string, configNode *yaml.Node) error {
	if configNode.Kind != yaml.MappingNode {
		return nil
	}
	for keyIndex := 0; keyIndex+1 < len(configNode.Content); keyIndex += 2 {
		key := configNode.Content[keyIndex].Value
		if _, known := knownPackageConfigKeys[key]; !known {
			return fmt.Errorf("package %s: %w: %q", importPath, errManifestUnknownKey, key)
		}
	}
	return nil
}

// validateGenericFallback ensures the generic_fallback value is one of the accepted
// modes.
//
// Takes importPath (string) for error messages.
// Takes value (string) which is the raw YAML value.
//
// Returns an error when the value is non-empty and not one of the known modes; nil
// otherwise.
func validateGenericFallback(importPath, value string) error {
	switch value {
	case "", "auto", "reflect", "panic":
		return nil
	default:
		return fmt.Errorf("package %s: generic_fallback %q must be one of: auto, reflect, panic", importPath, value)
	}
}

// validateGenericTypeArgs checks that every type-argument string in the generic_functions
// and generic_types config parses as a Go type expression.
//
// Takes importPath (string) which identifies the package for error messages.
// Takes genericFunctions (map[string][][]string) which holds the per-function
// instantiation type-argument lists.
// Takes genericTypes (map[string][]string) which holds the per-type erasure type-argument
// lists.
//
// Returns an error wrapping errManifestTypeArg when a type-argument string is malformed.
func validateGenericTypeArgs(importPath string, genericFunctions map[string][][]string, genericTypes map[string][]string) error {
	for name, instantiations := range genericFunctions {
		for _, typeArgs := range instantiations {
			if err := checkTypeArgList(typeArgs); err != nil {
				return fmt.Errorf("package %s: generic_functions %s: %w", importPath, name, err)
			}
		}
	}
	for name, typeArgs := range genericTypes {
		if err := checkTypeArgList(typeArgs); err != nil {
			return fmt.Errorf("package %s: generic_types %s: %w", importPath, name, err)
		}
	}
	return nil
}

// checkTypeArgList reports the first type-argument string in the list that is not a
// parseable Go type expression.
//
// Takes typeArgs ([]string) which is one manifest type-argument list.
//
// Returns an error wrapping errManifestTypeArg when an entry is malformed, or nil when
// every entry parses.
func checkTypeArgList(typeArgs []string) error {
	for _, typeArg := range typeArgs {
		if err := checkTypeArgExpr(typeArg); err != nil {
			return err
		}
	}
	return nil
}

// checkTypeArgExpr reports whether a manifest type-argument string is a parseable Go type
// expression.
//
// Takes typeArg (string) which is the candidate type-argument string.
//
// Returns an error wrapping errManifestTypeArg when the string is empty or does not parse
// as a Go expression.
func checkTypeArgExpr(typeArg string) error {
	if typeArg == "" {
		return fmt.Errorf("%w: empty type argument", errManifestTypeArg)
	}
	if _, err := parser.ParseExpr(typeArg); err != nil {
		return fmt.Errorf("%w: %q: %w", errManifestTypeArg, typeArg, err)
	}
	return nil
}

// parseFunctionConfigNode parses a per-function config node supporting both sequence and
// mapping forms.
//
// Takes name (string) which specifies the function name for error messages.
// Takes node (*yaml.Node) which provides the YAML node to parse.
//
// Returns the parsed FunctionConfig or an error if the node is invalid.
func parseFunctionConfigNode(name string, node *yaml.Node) (FunctionConfig, error) {
	switch node.Kind {
	case yaml.SequenceNode:
		var types []string
		if err := node.Decode(&types); err != nil {
			return FunctionConfig{}, fmt.Errorf("decoding type list for %s: %w", name, err)
		}
		return FunctionConfig{ElementTypes: types}, nil

	case yaml.MappingNode:
		var functionConfig FunctionConfig
		if err := node.Decode(&functionConfig); err != nil {
			return FunctionConfig{}, fmt.Errorf("decoding config for %s: %w", name, err)
		}
		return functionConfig, nil

	default:
		return FunctionConfig{}, fmt.Errorf("function %s: expected sequence or mapping, got %v", name, node.Kind)
	}
}

// readManifestFile reads a manifest file through a read-only sandbox scoped to the file's
// parent directory. When no factory is supplied a genuine os.Root-backed sandbox is
// created, so a symlinked or traversing manifest path cannot read content outside that
// directory.
//
// Takes path (string) which specifies the filesystem path to read.
// Takes factory (safedisk.Factory) which creates the sandbox when non-nil; otherwise a
// confined sandbox is created directly.
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
		sandbox, err = safedisk.NewSandbox(directory, safedisk.ModeReadOnly)
	}
	if err != nil {
		return nil, err
	}
	defer func() { _ = sandbox.Close() }()

	return sandbox.ReadFile(base)
}

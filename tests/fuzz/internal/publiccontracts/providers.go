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

package publiccontracts

import (
	_ "embed"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"maps"
	"path"
	"slices"
	"strings"
)

const (
	// PackageTest runs the named test in its own package unchanged.
	PackageTest ExecutionKind = "package-test"

	// OverlayTest renders the contract source and runs it as a private package overlay.
	OverlayTest ExecutionKind = "overlay"
)

const (
	// columnFamily is the registry column naming the WDK provider family.
	columnFamily = iota

	// columnPackage is the registry column naming the package that runs the contract.
	columnPackage

	// columnSource is the registry column naming the file declaring the contract test.
	columnSource

	// columnTest is the registry column naming the contract test function.
	columnTest

	// columnOperation is the registry column describing what the contract exercises.
	columnOperation

	// columnPrerequisite is the registry column describing the services the contract needs.
	columnPrerequisite

	// columnExecution is the registry column selecting how the contract runs.
	columnExecution

	// contractColumns is the number of tab-separated columns in a registry row.
	contractColumns
)

var (
	// contracts embeds the public provider contract registry.
	//
	//go:embed contracts.tsv
	contracts string

	// ErrInvalidRegistry reports a public-provider registry that no longer matches the
	// source tree.
	ErrInvalidRegistry = errors.New("invalid public contract registry")

	// prerequisiteRules maps adapter names to their required services, first match wins.
	prerequisiteRules = []prerequisiteRule{
		{family: "", words: []string{"grpc", "otel_http"}, except: nil, text: "private loopback telemetry receiver for transport I/O"},
		{family: "", words: []string{"redis", "valkey", "nats"}, except: nil, text: "private service matching the named protocol and standalone or cluster topology"},
		{family: "", words: []string{"sqlite_cgo"}, except: nil, text: "local C compiler and private SQLite database"},
		{family: "", words: []string{"sqlite", "duckdb"}, except: nil, text: "local engine and private database file"},
		{family: "db", words: []string{"engine", "schema"}, except: nil, text: "private database server for the named dialect"},
		{family: "", words: []string{"seagull"}, except: nil, text: "local Seagull generator and matching dialect"},
		{
			family: "",
			words: []string{"d1", "hcaptcha", "rev3", "turnstile", "ga4", "plausible", "webhook", "sentry", "_kms", "resolver_",
				"provider_gcs", "provider_s3", "provider_r2"},
			except: nil,
			text:   "private protocol fixture; live integration requires its service endpoint and credentials",
		},
		{family: "email", words: nil, except: []string{"stdout", "disk", "mock"}, text: "private mail server or protocol capture fixture"},
		{family: "llm", words: []string{"ollama"}, except: nil, text: "local Ollama server with a preloaded model for inference"},
		{family: "llm", words: nil, except: nil, text: "private API fixture; live inference requires a configured model endpoint"},
		{family: "", words: []string{"event", "gochannel"}, except: nil, text: "in-process event channel"},
		{family: "", words: []string{"vips"}, except: nil, text: "locally installed libvips and compatible native bindings"},
		{family: "", words: []string{"astiav"}, except: nil, text: "locally installed FFmpeg libraries and compatible native bindings"},
		{family: "", words: []string{"pipit"}, except: nil, text: "locally resolved Pipit interpreter"},
		{family: "system", words: nil, except: nil, text: "supported OS and a detectable system or cgroup memory limit"},
		{family: "", words: []string{"filesystem", "disk", "output_file"}, except: nil, text: "writable private directory"},
	}
)

// ExecutionKind identifies how Go executes the named contract.
type ExecutionKind string

// Provider describes a public provider contract and its adapters.
type Provider struct {
	// Family is the WDK provider family.
	Family string

	// Package is the repository-relative package that runs the contract.
	Package string

	// Source is the repository-relative file declaring the contract test.
	Source string

	// Test is the contract test function name.
	Test string

	// Operation describes what the contract exercises.
	Operation string

	// Prerequisite describes the services the contract needs.
	Prerequisite string

	// Execution selects the package test or private rendered overlay.
	Execution ExecutionKind

	// Adapters lists public provider implementations.
	Adapters []Adapter
}

// prerequisiteRule matches adapters to the services they need.
type prerequisiteRule struct {
	// family restricts the rule to one WDK family, or applies to all when empty.
	family string

	// text describes the required tools or services.
	text string

	// words lists name fragments of which one must appear, or matches any name when empty.
	words []string

	// except lists name fragments that exclude the rule.
	except []string
}

// Adapter describes a provider implementation and its prerequisites.
type Adapter struct {
	// Name is the adapter directory name.
	Name string

	// Source is the repository-relative adapter directory.
	Source string

	// Prerequisite describes the services the adapter needs.
	Prerequisite string

	// SeparateModule reports whether the adapter has its own Go module.
	SeparateModule bool
}

// matches reports whether the rule applies to an adapter.
//
// Takes family (string) which identifies the WDK provider family.
// Takes name (string) which identifies the adapter within its WDK family.
//
// Returns bool which reports whether the rule applies.
func (r prerequisiteRule) matches(family, name string) bool {
	if r.family != "" && r.family != family {
		return false
	}
	if len(r.words) > 0 && !containsAny(name, r.words) {
		return false
	}
	return !containsAny(name, r.except)
}

// Providers requires a representative contract for each current family and resolves its
// exact test declaration.
//
// Takes repository (fs.FS) which exposes the source tree to inventory.
//
// Returns []Provider which lists provider values in their recorded order.
// Returns error which reports an invalid embedded public-provider registry, or nil on
// success.
func Providers(repository fs.FS) ([]Provider, error) {
	families, err := fs.ReadDir(repository, "wdk")
	if err != nil {
		return nil, fmt.Errorf("read wdk families: %w", err)
	}
	wanted := map[string]bool{}
	for _, entry := range families {
		if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
			wanted[entry.Name()] = true
		}
	}
	var providers []Provider
	for row := range strings.SplitSeq(strings.TrimSpace(contracts), "\n") {
		provider, err := parseProvider(row)
		if err != nil {
			return nil, err
		}
		if !wanted[provider.Family] {
			return nil, fmt.Errorf("%w: stale or duplicate public family %s", ErrInvalidRegistry, provider.Family)
		}
		delete(wanted, provider.Family)
		adapters, err := providerAdapters(repository, provider)
		if err != nil {
			return nil, err
		}
		provider.Adapters = adapters
		providers = append(providers, provider)
	}
	if missing := slices.Sorted(maps.Keys(wanted)); len(missing) > 0 {
		return nil, fmt.Errorf("%w: public WDK families lack a contract: %s", ErrInvalidRegistry, strings.Join(missing, ", "))
	}
	return providers, nil
}

// providerAdapters checks a provider's contract test and lists its adapters.
//
// Takes repository (fs.FS) which exposes the source tree to inventory.
// Takes provider (Provider) which is the registry row.
//
// Returns []Adapter which lists the family's adapter directories.
// Returns error which reports a missing contract test or an unreadable source tree.
func providerAdapters(repository fs.FS, provider Provider) ([]Adapter, error) {
	source, err := fs.ReadFile(repository, provider.Source)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", provider.Source, err)
	}
	file, err := parser.ParseFile(token.NewFileSet(), provider.Source, source, parser.SkipObjectResolution)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", provider.Source, err)
	}
	if !hasTest(file, provider.Test) {
		return nil, fmt.Errorf("%w: %s no longer defines %s", ErrInvalidRegistry, provider.Source, provider.Test)
	}
	base := path.Join("wdk", provider.Family)
	entries, err := fs.ReadDir(repository, base)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", base, err)
	}
	var adapters []Adapter
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == "internal" || entry.Name() == "testdata" || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		adapter, found, err := inspectAdapter(repository, provider.Family, path.Join(base, entry.Name()), entry.Name())
		if err != nil {
			return nil, err
		}
		if found {
			adapters = append(adapters, adapter)
		}
	}
	return adapters, nil
}

// parseProvider decodes one registry row.
//
// Takes row (string) which is the tab-separated registry row.
//
// Returns Provider which holds the row without its adapters.
// Returns error which reports a malformed row or an unknown execution kind.
func parseProvider(row string) (Provider, error) {
	fields := strings.Split(row, "\t")
	if len(fields) != contractColumns {
		return Provider{}, fmt.Errorf("%w: malformed public contract %q", ErrInvalidRegistry, row)
	}
	provider := Provider{
		Family:       fields[columnFamily],
		Package:      fields[columnPackage],
		Source:       fields[columnSource],
		Test:         fields[columnTest],
		Operation:    fields[columnOperation],
		Prerequisite: fields[columnPrerequisite],
		Execution:    ExecutionKind(fields[columnExecution]),
		Adapters:     nil,
	}
	if provider.Execution != OverlayTest && provider.Execution != PackageTest {
		return Provider{}, fmt.Errorf("%w: invalid execution kind %q for %s", ErrInvalidRegistry, provider.Execution, provider.Family)
	}
	return provider, nil
}

// inspectAdapter describes one adapter directory when it holds Go sources or a module.
//
// Takes repository (fs.FS) which exposes the source tree to inventory.
// Takes family (string) which identifies the WDK provider family.
// Takes directory (string) which is the repository-relative adapter directory.
// Takes name (string) which is the adapter directory name.
//
// Returns Adapter which describes the adapter.
// Returns bool which reports whether the directory is an adapter.
// Returns error which reports an unreadable adapter directory.
func inspectAdapter(repository fs.FS, family, directory, name string) (Adapter, bool, error) {
	sources, err := fs.Glob(repository, directory+"/*.go")
	if err != nil {
		return Adapter{}, false, fmt.Errorf("list %s sources: %w", directory, err)
	}
	_, moduleErr := fs.Stat(repository, directory+"/go.mod")
	if moduleErr != nil && !errors.Is(moduleErr, fs.ErrNotExist) {
		return Adapter{}, false, fmt.Errorf("inspect %s/go.mod: %w", directory, moduleErr)
	}
	if len(sources) == 0 && moduleErr != nil {
		return Adapter{}, false, nil
	}
	return Adapter{Name: name, Source: directory, Prerequisite: adapterPrerequisite(family, name), SeparateModule: moduleErr == nil}, true, nil
}

// hasTest reports whether a file declares a top-level function with a body.
//
// Takes file (*ast.File) which is the parsed source.
// Takes name (string) which is the function name.
//
// Returns bool which reports whether the function is declared.
func hasTest(file *ast.File, name string) bool {
	for _, decl := range file.Decls {
		if function, ok := decl.(*ast.FuncDecl); ok && function.Recv == nil && function.Name.Name == name && function.Body != nil {
			return true
		}
	}
	return false
}

// adapterPrerequisite describes the selected provider adapter's required service.
//
// Takes family (string) which identifies the WDK provider family.
// Takes name (string) which identifies the adapter within its WDK family.
//
// Returns string which describes the adapter's required tools or services.
func adapterPrerequisite(family, name string) string {
	for _, rule := range prerequisiteRules {
		if rule.matches(family, name) {
			return rule.text
		}
	}
	return "local Go package, its build dependencies and in-process input fixtures"
}

// containsAny reports whether name contains any of the words.
//
// Takes name (string) which is the adapter name.
// Takes words ([]string) which lists the fragments to look for.
//
// Returns bool which reports whether any fragment appears.
func containsAny(name string, words []string) bool {
	return slices.ContainsFunc(words, func(word string) bool { return strings.Contains(name, word) })
}

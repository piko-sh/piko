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

package inventory

import (
	_ "embed"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"maps"
	"path"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/tests/fuzz/internal/targets"
)

const (
	// componentBehaviourKind is the capability kind of a shipped component behaviour.
	componentBehaviourKind = "component-behaviour"

	// cliSource is the repository path of the piko command's subcommand dispatch.
	cliSource = "cmd/piko/main.go"

	// targetsSource is the repository path that registers fuzz targets.
	targetsSource = "tests/fuzz/internal/targets/targets.go"

	// assignmentColumns is the number of tab-separated columns in one assignment rule.
	assignmentColumns = 3
)

var (
	// dispositions embeds source-capability assignments.
	//
	//go:embed capabilities.tsv
	dispositions string

	// specialTag recognises Piko's special element names.
	specialTag = regexp.MustCompile(`^piko:[a-z][a-z0-9-]*$`)

	// sourceGroups lists directories whose child directories are capabilities.
	sourceGroups = []sourceGroup{
		{root: "examples/scenarios", kind: "example"},
		{root: "wdk", kind: "wdk"},
		{root: "tests/integration", kind: "suite"},
	}

	// filePatterns lists globs whose matched files identify capabilities.
	filePatterns = []filePattern{
		{pattern: "wdk/*/*/go.mod", kind: "adapter-module"},
		{pattern: "frontend/extensions/components/src/behaviours/*.ts", kind: componentBehaviourKind},
	}

	// referencePages lists the reference documents whose headings are capabilities.
	referencePages = []string{"template-syntax", "routing-rules", "client-components", "pk-file-format", "directives"}
)

// sourceGroup selects a directory whose child directories are capabilities.
type sourceGroup struct {
	// root is the directory whose children are inventoried.
	root string

	// kind is the capability kind assigned to each child directory.
	kind string
}

// filePattern selects files whose paths identify capabilities.
type filePattern struct {
	// pattern is the glob that matches capability files.
	pattern string

	// kind is the capability kind assigned to each matched file.
	kind string
}

// Capability assigns a source capability to its responsible subsystem.
type Capability struct {
	// Kind is the capability category.
	Kind string

	// Name identifies the capability within its kind.
	Name string

	// Source is the repository path that declares the capability.
	Source string

	// Owner is the responsible subsystem, or empty when unassigned.
	Owner string
}

// Catalogue compares current capabilities with a recorded baseline.
type Catalogue struct {
	// Capabilities lists current source capabilities.
	Capabilities []Capability

	// Unassigned lists capabilities without a responsible subsystem.
	Unassigned []string `json:",omitempty"`

	// Removed lists baseline assignments whose capability no longer exists.
	Removed []string `json:",omitempty"`
}

// capabilitySet maps capability identities to their source records.
type capabilitySet map[string]Capability

// add records one capability.
//
// Takes kind (string) which is the capability category.
// Takes name (string) which identifies the capability within its kind.
// Takes source (string) which is the repository path that declares the capability.
func (entries capabilitySet) add(kind, name, source string) {
	entries[kind+"\t"+name] = Capability{Kind: kind, Name: name, Source: source, Owner: ""}
}

// addRegistered records the directives and fuzz targets registered in code.
func (entries capabilitySet) addRegistered() {
	for tag := range ast_domain.DirectiveNameToType {
		entries.add("directive", tag, "internal/ast/ast_domain/directive.go")
	}
	for _, name := range targets.ApplicationNames() {
		entries.add("app-target", name, targetsSource)
	}
	for _, name := range targets.List() {
		entries.add("package-target", name, targetsSource)
	}
	for name := range targets.Fuzzers() {
		entries.add("native-target", name, targetsSource)
	}
}

// Discover discovers source capabilities and their subsystem assignments.
//
// Takes repo (fs.FS) which exposes the source tree to inventory.
//
// Returns Catalogue which holds capabilities, their owners and missing or stale
// assignments.
// Returns error which reports an unreadable capability registry or invalid subsystem
// assignment, or nil on success.
func Discover(repo fs.FS) (Catalogue, error) {
	entries := capabilitySet{}
	entries.addRegistered()
	for _, discover := range []func(fs.FS, capabilitySet) error{
		addSourceGroups, addFilePatterns, addReferenceHeadings, addCLISubcommands, addCompilerSources,
	} {
		if err := discover(repo, entries); err != nil {
			return Catalogue{}, err
		}
	}
	return classify(entries, dispositions)
}

// addSourceGroups records the visible child directories of each source group.
//
// Takes repo (fs.FS) which exposes the source tree to inventory.
// Takes entries (capabilitySet) which receives the capabilities.
//
// Returns error which reports an unreadable group directory, or nil on success.
func addSourceGroups(repo fs.FS, entries capabilitySet) error {
	for _, group := range sourceGroups {
		items, err := fs.ReadDir(repo, group.root)
		if err != nil {
			return fmt.Errorf("reading %s: %w", group.root, err)
		}
		for _, entry := range items {
			if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
				continue
			}
			directory := path.Join(group.root, entry.Name())
			authored, authoredErr := hasAuthoredFiles(repo, directory)
			if authoredErr != nil {
				return authoredErr
			}
			if authored {
				entries.add(group.kind, entry.Name(), directory)
			}
		}
	}
	return nil
}

// addFilePatterns records the files matched by each capability file pattern.
//
// Takes repo (fs.FS) which exposes the source tree to inventory.
// Takes entries (capabilitySet) which receives the capabilities.
//
// Returns error which reports an invalid pattern, or nil on success.
func addFilePatterns(repo fs.FS, entries capabilitySet) error {
	for _, group := range filePatterns {
		files, err := fs.Glob(repo, group.pattern)
		if err != nil {
			return fmt.Errorf("matching %s: %w", group.pattern, err)
		}
		for _, file := range files {
			name := path.Base(path.Dir(file))
			if group.kind == componentBehaviourKind {
				name = strings.TrimSuffix(path.Base(file), ".ts")
				if name == "index" || name == "registry" || strings.HasSuffix(name, ".spec") {
					continue
				}
			}
			entries.add(group.kind, name, file)
		}
	}
	return nil
}

// addReferenceHeadings records the section headings of each reference document.
//
// Takes repo (fs.FS) which exposes the source tree to inventory.
// Takes entries (capabilitySet) which receives the capabilities.
//
// Returns error which reports an unreadable reference document, or nil on success.
func addReferenceHeadings(repo fs.FS, entries capabilitySet) error {
	for _, name := range referencePages {
		file := "docs/reference/" + name + ".md"
		data, err := readRepoFile(repo, file)
		if err != nil {
			return err
		}
		for line := range strings.SplitSeq(string(data), "\n") {
			title, ok := strings.CutPrefix(line, "## ")
			if !ok {
				title, ok = strings.CutPrefix(line, "### ")
			}
			if ok && title != "See also" && title != "" {
				entries.add(name, title, file)
			}
		}
	}
	return nil
}

// addCLISubcommands records the piko command's dispatched subcommands.
//
// Takes repo (fs.FS) which exposes the source tree to inventory.
// Takes entries (capabilitySet) which receives the capabilities.
//
// Returns error which reports an unreadable or unparsable command source, or nil on
// success.
func addCLISubcommands(repo fs.FS, entries capabilitySet) error {
	source, err := readRepoFile(repo, cliSource)
	if err != nil {
		return err
	}
	unit, err := parser.ParseFile(token.NewFileSet(), cliSource, source, 0)
	if err != nil {
		return fmt.Errorf("parsing %s: %w", cliSource, err)
	}
	var inspectErr error
	ast.Inspect(unit, func(node ast.Node) bool {
		if inspectErr != nil {
			return false
		}
		statement, ok := node.(*ast.SwitchStmt)
		if !ok {
			return true
		}
		name, ok := statement.Tag.(*ast.Ident)
		if !ok || name.Name != "subcommand" {
			return true
		}
		inspectErr = addSubcommands(statement, func(subcommand string) { entries.add("cli", subcommand, cliSource) })
		return false
	})
	if inspectErr != nil {
		return fmt.Errorf("%s: %w", cliSource, inspectErr)
	}
	return nil
}

// addCompilerSources records the special element names every compiler source mentions.
//
// Takes repo (fs.FS) which exposes the source tree to inventory.
// Takes entries (capabilitySet) which receives the capabilities.
//
// Returns error which reports an unreadable or unparsable compiler source, or nil on
// success.
func addCompilerSources(repo fs.FS, entries capabilitySet) error {
	files, err := fs.Glob(repo, "internal/compiler/compiler_domain/*.go")
	if err != nil {
		return fmt.Errorf("matching compiler sources: %w", err)
	}
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		if err := addCompilerTags(repo, file, func(tag string) { entries.add("compiler-tag", tag, file) }); err != nil {
			return err
		}
	}
	return nil
}

// addSubcommands records the string-literal arms of a subcommand switch.
//
// Takes statement (*ast.SwitchStmt) which is the subcommand dispatch switch.
// Takes record (func(string)) which receives each subcommand name.
//
// Returns error which reports an unexpected switch arm or an invalid literal, or nil on
// success.
func addSubcommands(statement *ast.SwitchStmt, record func(string)) error {
	for _, arm := range statement.Body.List {
		clause, ok := arm.(*ast.CaseClause)
		if !ok {
			return fmt.Errorf("unexpected subcommand switch arm %T", arm)
		}
		for _, expression := range clause.List {
			value, err := subcommandLiteral(expression)
			if err != nil {
				return err
			}
			if value != "" && !strings.HasPrefix(value, "-") {
				record(value)
			}
		}
	}
	return nil
}

// subcommandLiteral extracts the string value of one subcommand switch expression.
//
// Takes expression (ast.Expr) which is one case-clause expression.
//
// Returns string which is the unquoted literal, or empty for a non-string expression.
// Returns error which reports an invalid string literal, or nil on success.
func subcommandLiteral(expression ast.Expr) (string, error) {
	literal, ok := expression.(*ast.BasicLit)
	if !ok || literal.Kind != token.STRING {
		return "", nil
	}
	value, err := strconv.Unquote(literal.Value)
	if err != nil {
		return "", fmt.Errorf("unquoting subcommand %s: %w", literal.Value, err)
	}
	return value, nil
}

// readRepoFile reads one repository file.
//
// Takes repo (fs.FS) which exposes the source tree to inventory.
// Takes file (string) which is the repository path to read.
//
// Returns []byte which holds the file contents.
// Returns error which reports an unreadable file, wrapped with its path, or nil on
// success.
func readRepoFile(repo fs.FS, file string) ([]byte, error) {
	data, err := fs.ReadFile(repo, file)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", file, err)
	}
	return data, nil
}

// addCompilerTags records the special element names a compiler source mentions.
//
// Takes repo (fs.FS) which exposes the source tree to inventory.
// Takes file (string) which is the compiler source path.
// Takes record (func(string)) which receives each special element name.
//
// Returns error which reports an unreadable or unparsable source, or nil on success.
func addCompilerTags(repo fs.FS, file string, record func(string)) error {
	data, err := readRepoFile(repo, file)
	if err != nil {
		return err
	}
	unit, err := parser.ParseFile(token.NewFileSet(), file, data, 0)
	if err != nil {
		return fmt.Errorf("parsing %s: %w", file, err)
	}
	var unquoteErr error
	ast.Inspect(unit, func(node ast.Node) bool {
		literal, ok := node.(*ast.BasicLit)
		if !ok || literal.Kind != token.STRING || unquoteErr != nil {
			return unquoteErr == nil
		}
		value, err := strconv.Unquote(literal.Value)
		if err != nil {
			unquoteErr = fmt.Errorf("%s: unquoting %s: %w", file, literal.Value, err)
			return false
		}
		if specialTag.MatchString(value) {
			record(value)
		}
		return true
	})
	return unquoteErr
}

// classify assigns discovered capabilities to their declared subsystems.
//
// Takes entries (map[string]Capability) which maps capability identities to source
// records.
// Takes assignments (string) which holds the subsystem assignment rules.
//
// Returns Catalogue which holds capabilities, their owners and missing or stale
// assignments.
// Returns error which reports an unreadable capability registry or invalid subsystem
// assignment, or nil on success.
func classify(entries map[string]Capability, assignments string) (Catalogue, error) {
	result := Catalogue{}
	known := map[string]string{}
	for line := range strings.SplitSeq(assignments, "\n") {
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		columns := strings.Split(line, "\t")
		if len(columns) != assignmentColumns || columns[2] == "" {
			return result, fmt.Errorf("invalid capability assignment %q", line)
		}
		identity := columns[0] + "\t" + columns[1]
		if _, duplicate := known[identity]; duplicate {
			return result, fmt.Errorf("duplicate capability %q", identity)
		}
		known[identity] = columns[2]
		if _, ok := entries[identity]; !ok {
			result.Removed = append(result.Removed, identity)
		}
	}
	for _, identity := range slices.Sorted(maps.Keys(entries)) {
		entry := entries[identity]
		entry.Owner = known[identity]
		if entry.Owner == "" {
			result.Unassigned = append(result.Unassigned, identity)
		}
		result.Capabilities = append(result.Capabilities, entry)
	}
	slices.Sort(result.Removed)
	return result, nil
}

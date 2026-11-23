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
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"io/fs"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"piko.sh/piko/internal/gosource"
	"piko.sh/piko/tests/fuzz/internal/casefile"
	"piko.sh/safedisk"
)

const (
	// scenarioDir is the repository-relative scenarios package directory.
	scenarioDir = "tests/fuzz/internal/scenarios"

	// scenarioTestFile is the overlay file name of a rendered scenario test.
	scenarioTestFile = "piko_fuzz_scenario_test.go"

	// privateFileMode is the permission of every staged file.
	privateFileMode fs.FileMode = 0o600

	// privateDirectoryMode is the permission of every staged directory.
	privateDirectoryMode fs.FileMode = 0o700

	// schemaChoices is the number of choice bytes the database schema draws.
	schemaChoices = 3

	// graphPage is the page that marks an incremental graph case.
	graphPage = "pages/partial-graph.pk"

	// graphTextHelper is the transitive helper of the current graph layout.
	graphTextHelper = "pkg/graphtext/text.go"

	// graphLeaf is the partial whose label the edited edition changes.
	graphLeaf = "partials/graph/leaf.pk"
)

var (
	// errFixtureShape reports a scenario fixture whose editable source no longer matches.
	errFixtureShape = errors.New("unexpected scenario fixture shape")

	// errGraphSource reports an incremental graph case that cannot be staged.
	errGraphSource = errors.New("unsupported graph source")

	// companions lists the extra files each scenario overlay needs.
	companions = map[string][]companion{
		"database": {
			{key: "PIKO_FUZZ_DATABASE_RUNTIME", name: "runtime_test.go", relative: "testdata/database_runtime.go"},
		},
		"actions": {
			{key: "PIKO_FUZZ_ACTION_IMPL", name: "action.go", relative: "actions_fixture/actionimpl/testdata/action.go"},
			{key: "PIKO_FUZZ_ACTION_TEST", name: "invocation_test.go", relative: "actions_fixture/wrappers/testdata/invocation_test.go"},
		},
	}
)

// companion names one rendered file staged beside a scenario overlay.
type companion struct {
	// key is the environment variable naming the staged file.
	key string

	// name is the staged file name.
	name string

	// relative is the fixture path beneath the scenarios package.
	relative string
}

// renderSuite renders a scenario fixture with its case-specific edits applied.
//
// Takes repository (string) which is the repository source directory.
// Takes source (string) which is the repository-relative fixture path.
// Takes data ([]byte) which holds the replayable choice bytes.
//
// Returns []byte which is the rendered Go source.
// Returns error which reports a read, edit or formatting failure.
func renderSuite(repository, source string, data []byte) ([]byte, error) {
	fixture := filepath.Join(repository, source)
	contents, err := readFileIn(filepath.Dir(fixture), filepath.Base(fixture))
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", source, err)
	}
	name := filepath.Base(source)
	var edit func(*token.FileSet, *ast.File) error
	if name == "database.go" || name == "database_runtime.go" {
		table, direction, kind := databaseSchema(data)
		edit = func(_ *token.FileSet, file *ast.File) error {
			if name == "database.go" {
				return setFixtureStrings(file, map[string]string{"table": table, "direction": direction, "kind": kind})
			}
			return editDatabaseRuntime(file, table, kind)
		}
	}
	if name == "callbacks.go" {
		seed := sha256.Sum256(data)
		units := 1 + int(seed[0]%6)
		expression := &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   &ast.CallExpr{Fun: &ast.SelectorExpr{X: ast.NewIdent("library"), Sel: ast.NewIdent("Ledger")}, Args: []ast.Expr{&ast.BasicLit{Kind: token.INT, Value: strconv.Itoa(units)}}},
				Sel: ast.NewIdent("Advance"),
			},
			Args: []ast.Expr{ast.NewIdent("delta")},
		}
		var output bytes.Buffer
		if err := format.Node(&output, token.NewFileSet(), expression); err != nil {
			return nil, fmt.Errorf("format callback expression: %w", err)
		}
		edit = func(_ *token.FileSet, file *ast.File) error {
			return setFixtureStrings(file, map[string]string{"invocation": output.String()})
		}
	}
	return gosource.Render(source, contents, edit)
}

// databaseSchema selects the generated table, sort direction and column kind.
//
// Takes data ([]byte) which holds the replayable choice bytes.
//
// Returns table (string) which is the generated table name.
// Returns direction (string) which is the SQL sort direction.
// Returns kind (string) which is the SQL column type.
func databaseSchema(data []byte) (table, direction, kind string) {
	tables := [...]string{"readings", "records", "entries"}
	directions := [...]string{"ASC", "DESC"}
	kinds := [...]string{"INTEGER", "TEXT"}
	var choices [schemaChoices]byte
	copy(choices[:], data)
	return tables[int(choices[0])%len(tables)], directions[int(choices[1])%len(directions)], kinds[int(choices[2])%len(kinds)]
}

// fixtureString finds a named constant with one string initialiser.
//
// A moved declaration survives lookup, and a removed or ambiguous edit fails loudly.
//
// Takes file (*ast.File) which is the parsed fixture.
// Takes name (string) which is the constant name.
//
// Returns *ast.BasicLit which is the constant's string literal.
// Returns error which reports a missing, ambiguous or non-string constant.
func fixtureString(file *ast.File, name string) (*ast.BasicLit, error) {
	var match *ast.BasicLit
	for node := range ast.Preorder(file) {
		group, ok := node.(*ast.GenDecl)
		if !ok || group.Tok != token.CONST {
			continue
		}
		found, err := groupStrings(group, name)
		if err != nil {
			return nil, err
		}
		for _, literal := range found {
			if match != nil {
				return nil, fmt.Errorf("%w: expected one constant initialiser for %s", errFixtureShape, name)
			}
			match = literal
		}
	}
	if match == nil {
		return nil, fmt.Errorf("%w: missing string constant %s", errFixtureShape, name)
	}
	return match, nil
}

// groupStrings finds the string initialisers of a named constant in one const group.
//
// Takes group (*ast.GenDecl) which is the const declaration group.
// Takes name (string) which is the constant name.
//
// Returns []*ast.BasicLit which lists the matching string literals.
// Returns error which reports a non-value spec or a constant without exactly one string
// initialiser.
func groupStrings(group *ast.GenDecl, name string) ([]*ast.BasicLit, error) {
	var found []*ast.BasicLit
	for _, raw := range group.Specs {
		spec, ok := raw.(*ast.ValueSpec)
		if !ok {
			return nil, fmt.Errorf("%w: constant group holds %T", errFixtureShape, raw)
		}
		if len(spec.Names) != 1 || spec.Names[0].Name != name {
			continue
		}
		if len(found) > 0 || len(spec.Values) != 1 {
			return nil, fmt.Errorf("%w: expected one constant initialiser for %s", errFixtureShape, name)
		}
		literal, ok := spec.Values[0].(*ast.BasicLit)
		if !ok || literal.Kind != token.STRING {
			return nil, fmt.Errorf("%w: expected a string literal for %s", errFixtureShape, name)
		}
		found = append(found, literal)
	}
	return found, nil
}

// setFixtureStrings replaces named string constants in a fixture.
//
// Takes file (*ast.File) which is the parsed fixture.
// Takes values (map[string]string) which maps constant names to new values.
//
// Returns error which reports a missing or ambiguous constant.
func setFixtureStrings(file *ast.File, values map[string]string) error {
	for name, value := range values {
		literal, err := fixtureString(file, name)
		if err != nil {
			return err
		}
		literal.Value = strconv.Quote(value)
	}
	return nil
}

// editDatabaseRuntime rewrites the database runtime fixture for the drawn schema.
//
// Takes file (*ast.File) which is the parsed runtime fixture.
// Takes table (string) which is the generated table name.
// Takes kind (string) which is the SQL column type.
//
// Returns error which reports a fixture that no longer has the expected shape.
func editDatabaseRuntime(file *ast.File, table, kind string) error {
	if err := retargetQueries(file, table); err != nil {
		return err
	}
	hits := 0
	for node := range ast.Preorder(file) {
		literal, ok := node.(*ast.CompositeLit)
		if !ok || !identIs(literal.Type, "ChangeParams") {
			continue
		}
		count, err := convertQuantity(literal, kind)
		if err != nil {
			return err
		}
		hits += count
	}
	if hits != 1 {
		return fmt.Errorf("%w: expected one ChangeParams.Quantity conversion; found %d", errFixtureShape, hits)
	}
	return nil
}

// retargetQueries points the authored SQL literals at the drawn table.
//
// The two authored SQL literals are the source of truth. Their table reference is
// substituted without modifying unrelated Go strings or query clauses.
//
// Takes file (*ast.File) which is the parsed runtime fixture.
// Takes table (string) which is the generated table name.
//
// Returns error which reports a query without exactly one readings table reference.
func retargetQueries(file *ast.File, table string) error {
	const from = " FROM readings "
	for _, name := range []string{"countParentsAndNotes", "readParentNote"} {
		literal, err := fixtureString(file, name)
		if err != nil {
			return err
		}
		sql, err := strconv.Unquote(literal.Value)
		if err != nil {
			return fmt.Errorf("%w: unquote %s: %w", errFixtureShape, name, err)
		}
		if strings.Count(sql, from) != 1 {
			return fmt.Errorf("%w: %s requires one readings table reference", errFixtureShape, name)
		}
		literal.Value = strconv.Quote(strings.Replace(sql, from, " FROM "+table+" ", 1))
	}
	return nil
}

// convertQuantity rewrites the Quantity conversion of one ChangeParams literal.
//
// Takes literal (*ast.CompositeLit) which is the ChangeParams literal.
// Takes kind (string) which is the SQL column type.
//
// Returns int which counts the rewritten Quantity fields.
// Returns error which reports an unexpected Quantity conversion.
func convertQuantity(literal *ast.CompositeLit, kind string) (int, error) {
	hits := 0
	for _, raw := range literal.Elts {
		field, ok := raw.(*ast.KeyValueExpr)
		if !ok || !identIs(field.Key, "Quantity") {
			continue
		}
		call, ok := field.Value.(*ast.CallExpr)
		if !ok || !identIs(call.Fun, "int32") || len(call.Args) != 1 || !identIs(call.Args[0], "q") {
			return 0, fmt.Errorf("%w: unexpected ChangeParams.Quantity conversion", errFixtureShape)
		}
		if kind == "TEXT" {
			call.Fun = &ast.SelectorExpr{X: ast.NewIdent("fmt"), Sel: ast.NewIdent("Sprint")}
		}
		hits++
	}
	return hits, nil
}

// identIs reports whether an expression is the named identifier.
//
// Takes expression (ast.Expr) which is the expression to test.
// Takes name (string) which is the expected identifier.
//
// Returns bool which reports whether the expression is that identifier.
func identIs(expression ast.Expr, name string) bool {
	identifier, ok := expression.(*ast.Ident)
	return ok && identifier.Name == name
}

// stageCompanions writes the extra files a scenario overlay needs.
//
// Takes repository (string) which is the repository source directory.
// Takes directory (string) which is the staging directory.
// Takes id (string) which names the scenario.
// Takes fuzzCase (*casefile.Case) which holds the saved sources, choices and expected
// results.
//
// Returns []string which lists environment entries naming the staged files.
// Returns error which reports a render or write failure.
func stageCompanions(repository, directory, id string, fuzzCase *casefile.Case) ([]string, error) {
	var environment []string
	for _, staged := range companions[id] {
		data, err := renderSuite(repository, filepath.Join(scenarioDir, staged.relative), fuzzCase.Data)
		if err != nil {
			return nil, err
		}
		if err := writeFileIn(directory, staged.name, data); err != nil {
			return nil, fmt.Errorf("write %s: %w", staged.name, err)
		}
		environment = append(environment, staged.key+"="+filepath.Join(directory, staged.name))
	}
	if _, ok := fuzzCase.Files[graphPage]; id == "incremental" && ok {
		root := filepath.Join(directory, "graph")
		if err := stageGraph(root, fuzzCase.Files); err != nil {
			return nil, err
		}
		environment = append(environment, "PIKO_FUZZ_GRAPH_SOURCE="+root)
	}
	return environment, nil
}

// stageGraph writes base and edited editions of an incremental graph case.
//
// Takes root (string) which is the graph staging directory.
// Takes files (map[string]string) which holds the case's source files.
//
// Returns error which reports an unsupported graph or a write failure.
func stageGraph(root string, files map[string]string) error {
	helper, names, err := graphSources(files)
	if err != nil {
		return err
	}
	for _, name := range names {
		if !filepath.IsLocal(name) || filepath.ToSlash(filepath.Clean(name)) != name {
			return fmt.Errorf("%w: invalid path %q", errGraphSource, name)
		}
		original, ok := files[name]
		if !ok {
			return fmt.Errorf("%w: missing %s", errGraphSource, name)
		}
		if err := writeGraphEdition(root, "base", name, original, false); err != nil {
			return err
		}
		if name != helper && name != graphLeaf {
			continue
		}
		if err := writeGraphEdition(root, "edited", name, original, true); err != nil {
			return err
		}
	}
	return nil
}

// graphSources selects the label helper and the sorted sources a graph case stages.
//
// Takes files (map[string]string) which holds the case's source files.
//
// Returns string which is the label helper path.
// Returns []string which lists the staged source paths in sorted order.
// Returns error which reports a graph without a supported label helper.
func graphSources(files map[string]string) (string, []string, error) {
	helper := ""
	for _, name := range []string{graphTextHelper, "pkg/graphmodel/label.go", "lib/graphmodel/label.go"} {
		if _, ok := files[name]; ok {
			helper = name
			break
		}
	}
	if helper == "" {
		return "", nil, fmt.Errorf("%w: no supported label helper", errGraphSource)
	}
	names := []string{graphPage, helper}
	if helper == graphTextHelper {
		names = append(names, "pkg/graphmodel/label.go", "pkg/graphmodel/format.go")
	}
	for name := range files {
		if strings.HasPrefix(name, "partials/graph/") && strings.HasSuffix(name, ".pk") {
			names = append(names, name)
		}
	}
	slices.Sort(names)
	return helper, names, nil
}

// writeGraphEdition renders and writes one edition of a graph source file.
//
// Takes root (string) which is the graph staging directory.
// Takes edition (string) which names the edition directory.
// Takes name (string) which is the source path.
// Takes original (string) which is the original source.
// Takes change (bool) which selects the edited edition.
//
// Returns error which reports a render or write failure.
func writeGraphEdition(root, edition, name, original string, change bool) error {
	contents, err := graphEdition(name, original, change)
	if err != nil {
		return err
	}
	relative := filepath.Join(edition, filepath.FromSlash(name))
	if err := writeFileIn(root, relative, []byte(contents)); err != nil {
		return fmt.Errorf("write %s: %w", filepath.Join(root, relative), err)
	}
	return nil
}

// graphEdition renders one edition of a graph source file.
//
// Takes name (string) which is the source path.
// Takes source (string) which is the original source.
// Takes change (bool) which selects the edited edition.
//
// Returns string which is the rendered edition.
// Returns error which reports a malformed Go block or a failed edit.
func graphEdition(name, source string, change bool) (string, error) {
	const opener = `<script type="application/x-go">`
	prefix, code, suffix := "", source, ""
	if strings.HasSuffix(name, ".pk") {
		before, rest, hasGo := strings.Cut(source, opener)
		if !hasGo {
			if change {
				return "", fmt.Errorf("%w: %s has no Go block to edit", errGraphSource, name)
			}
			return source, nil
		}
		var closed bool
		code, suffix, closed = strings.Cut(rest, "</script>")
		if !closed {
			return "", fmt.Errorf("%w: %s has an unterminated Go block", errGraphSource, name)
		}
		prefix = before + opener + "\n"
		suffix = "</script>" + suffix
	}
	output, err := gosource.Render(name, []byte(code), func(_ *token.FileSet, file *ast.File) error {
		if _, err := gosource.ReplaceModuleImports(file, casefile.Module, "testcase_cache_partial_07"); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		if !change {
			return nil
		}
		if strings.HasSuffix(name, ".pk") {
			return editLeafLabel(file)
		}
		return editHelperPrefix(file)
	})
	if err != nil {
		return "", err
	}
	return prefix + string(output) + suffix, nil
}

// editLeafLabel prefixes the leaf partial's rendered Text field.
//
// Takes file (*ast.File) which is the parsed leaf Go block.
//
// Returns error which reports a leaf without exactly one Text field.
func editLeafLabel(file *ast.File) error {
	hits := 0
	for _, decl := range file.Decls {
		function, ok := decl.(*ast.FuncDecl)
		if !ok || function.Name.Name != "Render" || function.Body == nil {
			continue
		}
		for node := range ast.Preorder(function.Body) {
			if literal, ok := node.(*ast.CompositeLit); ok && identIs(literal.Type, "Response") {
				hits += prefixResponseText(literal)
			}
		}
	}
	if hits != 1 {
		return fmt.Errorf("%w: expected one Render response Text field; found %d", errFixtureShape, hits)
	}
	return nil
}

// prefixResponseText prefixes the Text fields of one Response literal.
//
// Takes literal (*ast.CompositeLit) which is the Response literal.
//
// Returns int which counts the prefixed Text fields.
func prefixResponseText(literal *ast.CompositeLit) int {
	hits := 0
	for _, raw := range literal.Elts {
		field, ok := raw.(*ast.KeyValueExpr)
		if !ok || !identIs(field.Key, "Text") {
			continue
		}
		field.Value = &ast.BinaryExpr{X: &ast.BasicLit{Kind: token.STRING, Value: `"edited:"`}, Op: token.ADD, Y: field.Value}
		hits++
	}
	return hits
}

// editHelperPrefix appends a marker to the helper's returned request prefix.
//
// The older single-file layout edits Label. Current cases edit Prefix in the transitive
// graphtext package. Both concatenate request on the left.
//
// Takes file (*ast.File) which is the parsed helper source.
//
// Returns error which reports a helper without exactly one returned request prefix.
func editHelperPrefix(file *ast.File) error {
	hits := 0
	for _, decl := range file.Decls {
		function, ok := decl.(*ast.FuncDecl)
		if !ok || (function.Name.Name != "Prefix" && function.Name.Name != "Label") || function.Body == nil {
			continue
		}
		for node := range ast.Preorder(function.Body) {
			returnStatement, ok := node.(*ast.ReturnStmt)
			if ok && len(returnStatement.Results) == 1 && markRequestPrefix(returnStatement.Results[0]) {
				hits++
			}
		}
	}
	if hits != 1 {
		return fmt.Errorf("%w: expected one returned request prefix; found %d", errFixtureShape, hits)
	}
	return nil
}

// markRequestPrefix appends a marker after the request operand of a concatenation.
//
// Takes expression (ast.Expr) which is the returned expression.
//
// Returns bool which reports whether a request operand was marked.
func markRequestPrefix(expression ast.Expr) bool {
	for {
		sum, ok := ast.Unparen(expression).(*ast.BinaryExpr)
		if !ok || sum.Op != token.ADD {
			return false
		}
		if identIs(ast.Unparen(sum.X), "request") {
			sum.X = &ast.BinaryExpr{X: sum.X, Op: token.ADD, Y: &ast.BasicLit{Kind: token.STRING, Value: `"-changed"`}}
			return true
		}
		expression = sum.X
	}
}

// readFileIn reads a file beneath directory through a read-only sandbox.
//
// Takes directory (string) which is the sandbox root.
// Takes name (string) which is the file path relative to directory.
//
// Returns []byte which is the file content.
// Returns error which reports a sandbox, read or close failure.
func readFileIn(directory, name string) ([]byte, error) {
	sandbox, err := safedisk.NewSandbox(directory, safedisk.ModeReadOnly)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", directory, err)
	}
	data, err := sandbox.ReadFile(name)
	if closeErr := sandbox.Close(); closeErr != nil {
		err = errors.Join(err, fmt.Errorf("close %s: %w", directory, closeErr))
	}
	if err != nil {
		return nil, err
	}
	return data, nil
}

// writeFileIn writes a private file beneath directory through a read-write sandbox,
// creating missing parent directories.
//
// Takes directory (string) which is the sandbox root.
// Takes name (string) which is the file path relative to directory.
// Takes data ([]byte) which is the file content.
//
// Returns error which reports a sandbox, directory, write or close failure.
func writeFileIn(directory, name string, data []byte) error {
	sandbox, err := safedisk.NewSandbox(directory, safedisk.ModeReadWrite)
	if err != nil {
		return fmt.Errorf("open %s: %w", directory, err)
	}
	err = sandbox.MkdirAll(filepath.Dir(name), privateDirectoryMode)
	if err == nil {
		err = sandbox.WriteFile(name, data, privateFileMode)
	}
	if closeErr := sandbox.Close(); closeErr != nil {
		err = errors.Join(err, fmt.Errorf("close %s: %w", directory, closeErr))
	}
	return err
}

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

package diagnostic_docs

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/colour"
)

const (
	repositoryRoot = "../.."
)

var (
	codeFiles = []string{
		"internal/ast/ast_domain/diagnostic_codes.go",
		"internal/annotator/annotator_dto/diagnostic_codes.go",
		"internal/generator/generator_dto/diagnostic_codes.go",
	}
	emittingTrees = []string{
		"internal/ast",
		"internal/annotator",
		"internal/cssinliner",
		"internal/generator",
	}
	indirectSeverities = map[string]string{
		"T016": "ast_domain/expression_features.go sets the code after building an error diagnostic",
		"T112": "annotator_domain/transform.go assetCollectionContext.addDiagnostic always reports a warning",
		"T113": "annotator_domain/transform.go assetCollectionContext.addDiagnostic always reports a warning",
		"T114": "annotator_domain/css_processor.go passes the code to cssinliner, which reports errors",
		"T115": "annotator_domain/css_processor.go passes the code to cssinliner, which keeps esbuild's severity",
		"T154": "annotator_domain/transform.go assetCollectionContext.addDiagnostic always reports a warning",
		"R001": "the code emitter writes the error into generated code",
		"R002": "the code emitter writes the error into generated code",
	}
	severityNames = map[string]Severity{
		"Error":   SeverityError,
		"Warning": SeverityWarning,
		"Info":    SeverityInfo,
	}
	severityPackages = map[string]bool{"ast_domain": true, "generator_dto": true}
)

func TestCatalogueMatchesConstants(t *testing.T) {
	constants := codeConstants(t)
	entries := All()
	require.Len(t, entries, len(constants))

	previous := map[Family]string{}
	for _, entry := range entries {
		value, found := constants["Code"+entry.Name]
		require.Truef(t, found, "%s: no constant named Code%s", entry.Code, entry.Name)
		require.Equalf(t, value, entry.Code, "Code%s", entry.Name)
		require.Greaterf(t, entry.Code, previous[entry.Family], "%s is out of order", entry.Code)
		previous[entry.Family] = entry.Code
	}
}

func TestEntriesAreComplete(t *testing.T) {
	for _, entry := range All() {
		require.Containsf(t, Families, entry.Family, "%s family", entry.Code)
		require.NotEmptyf(t, entry.Severities, "%s severities", entry.Code)
		for _, field := range []string{entry.Title, entry.Summary, entry.Detail, entry.Fix} {
			require.NotEmptyf(t, field, "%s is missing text", entry.Code)
			require.NotContainsf(t, field, "\u2014", "%s uses an em dash", entry.Code)
			require.NotContainsf(t, field, "\u2013", "%s uses an en dash", entry.Code)
		}
		if example := entry.Example; example != nil {
			end := example.Column - 1 + example.Width
			require.LessOrEqualf(t, end, len([]rune(example.Source)), "%s example highlight runs past its line", entry.Code)
			require.Positivef(t, example.Width, "%s example width", entry.Code)
		}
	}
}

func TestSeveritiesMatchEmissionSites(t *testing.T) {
	constants := codeConstants(t)
	observed := emittedSeverities(t, constants)

	for _, entry := range All() {
		for _, severity := range observed[entry.Code] {
			require.Containsf(t, entry.Severities, severity, "%s is emitted as %s", entry.Code, severity)
		}
		if entry.Unused {
			continue
		}
		if _, indirect := indirectSeverities[entry.Code]; indirect {
			continue
		}
		for _, severity := range entry.Severities {
			require.Containsf(t, observed[entry.Code], severity, "%s is documented as %s but never emitted as it", entry.Code, severity)
		}
	}
}

func TestUnusedCodesAreNotEmitted(t *testing.T) {
	constants := codeConstants(t)
	references := constantReferences(t, constants)

	for _, entry := range All() {
		if entry.Unused {
			require.Zerof(t, references["Code"+entry.Name], "%s is marked unused but is referenced", entry.Code)
		}
	}
}

func TestLookup(t *testing.T) {
	for _, id := range []string{"T141", "t141", " T141 ", "ComparisonError", "CodeComparisonError", "comparisonerror"} {
		entry, found := Lookup(id)
		require.Truef(t, found, "Lookup(%q)", id)
		require.Equal(t, "T141", entry.Code)
	}
	for _, id := range []string{"", "T999", "Q028", "141"} {
		_, found := Lookup(id)
		require.Falsef(t, found, "Lookup(%q)", id)
	}
}

func TestSuggest(t *testing.T) {
	require.Equal(t, "T141", Suggest("ComparisonEror"))
	require.Equal(t, "T127", Suggest("MissingRequiredPrp"))
	require.Empty(t, Suggest("something else entirely"))
	require.Empty(t, Suggest("T999"))
	require.Empty(t, Suggest("t142x"))
	require.Empty(t, Suggest(""))
}

func TestByFamily(t *testing.T) {
	groups := ByFamily()
	require.Len(t, groups, len(Families))
	total := 0
	for index, group := range groups {
		require.Equal(t, Families[index], group.Family)
		require.NotEmpty(t, group.Entries)
		total += len(group.Entries)
	}
	require.Equal(t, len(All()), total)
	require.Equal(t, "template-analysis", FamilySlug(FamilyTemplateAnalysis))
}

func TestEntryURL(t *testing.T) {
	entry, _ := Lookup("T141")
	require.Equal(t, "https://piko.sh/diagnostics/T141", entry.URL())
}

func codeConstants(t *testing.T) map[string]string {
	t.Helper()
	constants := map[string]string{}
	for _, file := range codeFiles {
		parsed, err := parser.ParseFile(token.NewFileSet(), filepath.Join(repositoryRoot, file), nil, 0)
		require.NoError(t, err)
		for _, declaration := range parsed.Decls {
			general, ok := declaration.(*ast.GenDecl)
			if !ok || general.Tok != token.CONST {
				continue
			}
			for _, spec := range general.Specs {
				value, isValue := spec.(*ast.ValueSpec)
				require.True(t, isValue)
				literal, isLiteral := value.Values[0].(*ast.BasicLit)
				require.True(t, isLiteral)
				unquoted, err := strconv.Unquote(literal.Value)
				require.NoError(t, err)
				constants[value.Names[0].Name] = unquoted
			}
		}
	}
	return constants
}

func sourceFiles(t *testing.T) []*ast.File {
	t.Helper()
	var files []*ast.File
	for _, tree := range emittingTrees {
		root := filepath.Join(repositoryRoot, tree)
		err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() && entry.Name() == "testdata" {
				return filepath.SkipDir
			}
			if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") || entry.Name() == "diagnostic_codes.go" {
				return nil
			}
			source, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			parsed, err := parser.ParseFile(token.NewFileSet(), path, source, parser.SkipObjectResolution)
			if err != nil {
				return err
			}
			files = append(files, parsed)
			return nil
		})
		require.NoError(t, err)
	}
	return files
}

func emittedSeverities(t *testing.T, constants map[string]string) map[string][]Severity {
	t.Helper()
	observed := map[string][]Severity{}
	record := func(code string, severity Severity) {
		if !slices.Contains(observed[code], severity) {
			observed[code] = append(observed[code], severity)
		}
	}
	for _, file := range sourceFiles(t) {
		ownsSeverities := severityPackages[file.Name.Name]
		ast.Inspect(file, func(node ast.Node) bool {
			var parts []ast.Expr
			switch typed := node.(type) {
			case *ast.CallExpr:
				parts = typed.Args
			case *ast.CompositeLit:
				for _, element := range typed.Elts {
					if keyValue, ok := element.(*ast.KeyValueExpr); ok {
						parts = append(parts, keyValue.Value)
					}
				}
			default:
				return true
			}
			var codes []string
			var severities []Severity
			for _, part := range parts {
				if code, ok := codeOf(part, constants); ok {
					codes = append(codes, code)
				}
				if severity, ok := severityOf(part, ownsSeverities); ok {
					severities = append(severities, severity)
				}
			}
			for _, code := range codes {
				for _, severity := range severities {
					record(code, severity)
				}
			}
			return true
		})
	}
	return observed
}

func constantReferences(t *testing.T, constants map[string]string) map[string]int {
	t.Helper()
	references := map[string]int{}
	for _, file := range sourceFiles(t) {
		ast.Inspect(file, func(node ast.Node) bool {
			if identifier, ok := node.(*ast.Ident); ok {
				if _, isCode := constants[identifier.Name]; isCode {
					references[identifier.Name]++
				}
			}
			return true
		})
	}
	return references
}

func codeOf(expression ast.Expr, constants map[string]string) (string, bool) {
	switch typed := expression.(type) {
	case *ast.Ident:
		value, ok := constants[typed.Name]
		return value, ok
	case *ast.SelectorExpr:
		value, ok := constants[typed.Sel.Name]
		return value, ok
	}
	return "", false
}

func severityOf(expression ast.Expr, ownsSeverities bool) (Severity, bool) {
	switch typed := expression.(type) {
	case *ast.SelectorExpr:
		if packageName, ok := typed.X.(*ast.Ident); ok && severityPackages[packageName.Name] {
			severity, found := severityNames[typed.Sel.Name]
			return severity, found
		}
	case *ast.Ident:
		if ownsSeverities {
			severity, found := severityNames[typed.Name]
			return severity, found
		}
	case *ast.BasicLit:
		if typed.Kind == token.STRING {
			name, err := strconv.Unquote(typed.Value)
			if err == nil {
				severity, found := severityNames[name]
				return severity, found
			}
		}
	}
	return "", false
}

func TestGeneratedDocsAreCurrent(t *testing.T) {
	encoded, err := RenderJSON()
	require.NoError(t, err)
	generated := map[string][]byte{MarkdownFile: RenderMarkdown(), JSONFile: encoded}
	for name, content := range generated {
		committed, err := os.ReadFile(filepath.Join(repositoryRoot, "docs", "reference", name))
		require.NoError(t, err)
		require.Equalf(t, string(content), string(committed), "docs/reference/%s is stale; run make generate-diagnostics-docs", name)
	}
}

func TestRenderedExamplesMatchBuildOutput(t *testing.T) {
	previous := colour.Enabled()
	colour.SetEnabled(false)
	t.Cleanup(func() { colour.SetEnabled(previous) })

	for _, entry := range All() {
		if entry.Example == nil {
			continue
		}
		rendered := entry.RenderExample()
		require.Containsf(t, rendered, "["+entry.Code+"]: "+entry.Example.Message, "%s example message", entry.Code)
		require.Containsf(t, rendered, "= docs: "+entry.URL(), "%s example docs line", entry.Code)
	}
}

func TestSeverityLabel(t *testing.T) {
	single, _ := Lookup("T141")
	require.Equal(t, "Error", single.SeverityLabel())
	double, _ := Lookup("T101")
	require.Equal(t, "Error or warning", double.SeverityLabel())
	triple, _ := Lookup("T136")
	require.Equal(t, "Error, info or warning", triple.SeverityLabel())
}

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

package driven_code_emitter_go_literal

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	goast "go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/generator/generator_dto"
)

const (
	lineFidelityPkPath = "/project/pages/main.pk"

	lineFidelityGeneratedPath = "/project/dist/pages/test_hash/generated.go"

	lineFidelityScriptStartLine = 7

	lineFidelityScript = `package main

import "fmt"

type Response struct {
	Title string
}

func helper(values []int) int {
	total := 0

	// Comments and blank lines are dropped when the script is reprinted.
	for _, value := range values {
		if value > 10 {
			total += value
		} else if value < 0 {
			total -= value
		} else {
			total--
		}
	}

	message := fmt.Sprintf(
		"%d",
		total,
	)
	_ = message

	switch total {
	case 1:
		total++

	default:
		total = 2
	}

	select {
	default:
		total++
	}

	return total
}

func (r Response) Describe() string {
	label := r.Title


	return label
}
`
)

func TestUserCodeLineDirectives_EveryStatementMapsToItsPkLine(t *testing.T) {
	t.Parallel()

	generated := emitLineFidelityComponent(t, true)

	expected := collectScriptStatementLines(t)
	require.NotEmpty(t, expected)

	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, lineFidelityGeneratedPath, generated, parser.SkipObjectResolution)
	require.NoError(t, err, "generated code must parse:\n%s", generated)

	var actual []token.Position
	for _, functionName := range []string{"helper", "Describe"} {
		functionDecl := findFunction(file, functionName)
		require.NotNil(t, functionDecl, "user function %s must be emitted", functionName)
		for _, statement := range collectStatements(functionDecl.Body) {
			actual = append(actual, fileSet.PositionFor(statement.Pos(), true))
		}
	}

	require.Len(t, actual, len(expected), "every user statement must be emitted")
	for index, position := range actual {
		assert.Equal(t, lineFidelityPkPath, position.Filename, "statement %d must map to the .pk file", index)
		assert.Equal(t, expected[index], position.Line, "statement %d must map to its .pk line", index)
	}

	helperDecl := findFunction(file, "helper")
	assert.Equal(t, lineFidelityScriptStartLine+8, fileSet.PositionFor(helperDecl.Pos(), true).Line,
		"the function signature maps to its .pk line")
}

func TestUserCodeLineDirectives_GeneratedCodeIsResetToTheGeneratedFile(t *testing.T) {
	t.Parallel()

	generated := emitLineFidelityComponent(t, true)

	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, lineFidelityGeneratedPath, generated, parser.SkipObjectResolution)
	require.NoError(t, err)

	userDeclarations := map[string]bool{"helper": true, "Describe": true, "Response": true}
	checkedGeneratedDeclarations := 0
	sawUserDeclaration := false
	for _, declaration := range file.Decls {
		name, _ := declNameAndSignature(declaration)
		if userDeclarations[name] {
			sawUserDeclaration = true
			continue
		}
		if !sawUserDeclaration {
			continue
		}
		adjusted := fileSet.PositionFor(declaration.Pos(), true)
		physical := fileSet.PositionFor(declaration.Pos(), false)
		assert.Equal(t, lineFidelityGeneratedPath, adjusted.Filename, "generated declaration %q must not be attributed to the .pk file", name)
		assert.Equal(t, physical.Line, adjusted.Line, "generated declaration %q must report its real line", name)
		checkedGeneratedDeclarations++
	}
	assert.Positive(t, checkedGeneratedDeclarations, "generated declarations must follow the user code")
	assert.Contains(t, string(generated), "//line "+lineFidelityPkPath+":", "directives use the absolute .pk path")
}

func TestUserCodeLineDirectives_DisabledEmitsNoCompilerDirectives(t *testing.T) {
	t.Parallel()

	generated := emitLineFidelityComponent(t, false)

	for line := range strings.SplitSeq(string(generated), "\n") {
		assert.False(t, strings.HasPrefix(line, "//line "), "no compiler-honoured directive may be emitted: %q", line)
	}
	assert.Contains(t, string(generated), "// line pages/main.pk:15\nfunc helper(",
		"the informational declaration comment keeps the relative path")
	assert.Equal(t, 2, strings.Count(string(generated), "// line pages/main.pk"),
		"no per-statement comments are emitted when directives are disabled")
}

func TestUserCodeLineDirectives_DoesNotMutateTheSharedScriptAST(t *testing.T) {
	t.Parallel()

	component := buildLineFidelityComponent(t)
	before := printNode(t, component.RewrittenScriptAST)

	em := &emitter{
		ctx:    NewEmitterContext(),
		config: EmitterConfig{BaseDir: "/project", EnableDwarfLineDirectives: true},
	}
	fileAST := &goast.File{Name: goast.NewIdent("main")}
	require.NoError(t, copyUserCode(t.Context(), fileAST, component, em))

	assert.Equal(t, before, printNode(t, component.RewrittenScriptAST))
}

func TestAppendLineDirectiveResets(t *testing.T) {
	t.Parallel()

	t.Run("resets after a mapped declaration", func(t *testing.T) {
		t.Parallel()

		source := []byte("package p\n\n//line /a.pk:3\nfunc A() {\n}\n\nfunc B() {}\n")
		result, err := appendLineDirectiveResets(source, "/gen/generated.go")

		require.NoError(t, err)
		assert.Equal(t, "package p\n\n//line /a.pk:3\nfunc A() {\n}\n//line /gen/generated.go:7\n\nfunc B() {}\n", string(result))
	})

	t.Run("numbers each reset by its physical line", func(t *testing.T) {
		t.Parallel()

		source := []byte("package p\n//line /a.pk:3\nfunc A() {}\n//line /a.pk:9\nfunc B() {}\nfunc C() {}\n")
		result, err := appendLineDirectiveResets(source, "/gen/generated.go")

		require.NoError(t, err)
		assert.Equal(t,
			"package p\n//line /a.pk:3\nfunc A() {}\n//line /gen/generated.go:5\n//line /a.pk:9\nfunc B() {}\n//line /gen/generated.go:8\nfunc C() {}\n",
			string(result))
		assertResetsReportPhysicalLines(t, result)
	})

	t.Run("adds a newline when the last declaration ends the file", func(t *testing.T) {
		t.Parallel()

		source := []byte("package p\n//line /a.pk:3\nfunc A() {}")
		result, err := appendLineDirectiveResets(source, "/gen/generated.go")

		require.NoError(t, err)
		assert.Equal(t, "package p\n//line /a.pk:3\nfunc A() {}\n//line /gen/generated.go:5\n", string(result))
	})

	t.Run("leaves source without directives alone", func(t *testing.T) {
		t.Parallel()

		source := []byte("package p\n\nfunc A() {}\n")
		result, err := appendLineDirectiveResets(source, "/gen/generated.go")

		require.NoError(t, err)
		assert.Equal(t, source, result)
	})

	t.Run("leaves source alone when no declaration carries a directive", func(t *testing.T) {
		t.Parallel()

		source := []byte("package p\n\nfunc A() {}\n\n//line /a.pk:3\n")
		result, err := appendLineDirectiveResets(source, "/gen/generated.go")

		require.NoError(t, err)
		assert.Equal(t, source, result)
	})

	t.Run("needs a generated path", func(t *testing.T) {
		t.Parallel()

		source := []byte("package p\n\n//line /a.pk:3\nfunc A() {}\n")
		result, err := appendLineDirectiveResets(source, "")

		require.NoError(t, err)
		assert.Equal(t, source, result)
	})

	t.Run("reports unparsable source", func(t *testing.T) {
		t.Parallel()

		source := []byte("package p\n\n//line /a.pk:3\nfunc A( {}\n")
		result, err := appendLineDirectiveResets(source, "/gen/generated.go")

		require.ErrorIs(t, err, errLineDirectiveResetParse)
		assert.Nil(t, result)
	})
}

func TestUserCodeStatementMapper_StopsAtTheDepthLimit(t *testing.T) {
	t.Parallel()

	source := "package p\nfunc deep() {\n" + strings.Repeat("{\n", maxStatementMappingDepth+4) +
		"_ = 1\n" + strings.Repeat("}\n", maxStatementMappingDepth+4) + "}\n"
	fileSet := token.NewFileSet()
	original, err := parser.ParseFile(fileSet, "script.go", source, 0)
	require.NoError(t, err)
	rewritten, err := parser.ParseFile(token.NewFileSet(), "", source, 0)
	require.NoError(t, err)

	mapper := newTestStatementMapper(fileSet)
	mapped := mapper.mapDeclaration(findFunction(rewritten, "deep"), findFunction(original, "deep"))

	assert.True(t, mapper.depthLimitReached)
	assert.Zero(t, mapper.mismatchedStatements)
	require.IsType(t, &goast.FuncDecl{}, mapped)
	mapper.reportIncompleteMapping(t.Context())
}

func TestUserCodeStatementMapper_CountsMismatchedStatements(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		original  string
		rewritten string
	}{
		{name: "different statement counts", original: "a := 1\n_ = a", rewritten: "a := 1"},
		{name: "different statement kinds", original: "for {}", rewritten: "if true {}"},
		{name: "else on one side only", original: "if true {} else {}", rewritten: "if true {}"},
		{name: "different clause counts", original: "switch {\ncase true:\ndefault:\n}", rewritten: "switch {\ncase true:\n}"},
		{name: "different block kinds", original: "{}", rewritten: "for {}"},
		{name: "different clause kinds", original: "select {\ndefault:\n}", rewritten: "select {}"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			fileSet := token.NewFileSet()
			original, err := parser.ParseFile(fileSet, "script.go", "package p\nfunc f() {\n"+testCase.original+"\n}\n", 0)
			require.NoError(t, err)
			rewritten, err := parser.ParseFile(token.NewFileSet(), "", "package p\nfunc f() {\n"+testCase.rewritten+"\n}\n", 0)
			require.NoError(t, err)

			mapper := newTestStatementMapper(fileSet)
			mapper.mapDeclaration(findFunction(rewritten, "f"), findFunction(original, "f"))

			assert.Positive(t, mapper.mismatchedStatements)
			mapper.reportIncompleteMapping(t.Context())
		})
	}
}

func TestUserCodeStatementMapper_MapsLabelsTypeSwitchesAndRanges(t *testing.T) {
	t.Parallel()

	source := `package p
func f(values []any) {
outer:
	for range values {
		break outer
	}
	switch values[0].(type) {
	case int:
		_ = 1
	}
	for index := 0; index < 1; index++ {
		_ = index
	}
}
`
	fileSet := token.NewFileSet()
	original, err := parser.ParseFile(fileSet, "script.go", source, 0)
	require.NoError(t, err)
	rewritten, err := parser.ParseFile(token.NewFileSet(), "", source, 0)
	require.NoError(t, err)

	mapper := newTestStatementMapper(fileSet)
	mapped := mapper.mapDeclaration(findFunction(rewritten, "f"), findFunction(original, "f"))

	assert.Zero(t, mapper.mismatchedStatements)
	assert.False(t, mapper.depthLimitReached)
	printed := printNode(t, mapped)
	for _, scriptLine := range []int{3, 5, 7, 9, 11, 12} {
		assert.Contains(t, printed, fmt.Sprintf("//line /project/main.pk:%d", lineFidelityScriptStartLine+scriptLine-1))
	}
}

func TestUserCodeStatementMapper_LeavesUnpairedDeclarationsAlone(t *testing.T) {
	t.Parallel()

	rewritten := &goast.FuncDecl{Name: goast.NewIdent("f"), Body: &goast.BlockStmt{}}
	var nilMapper *userCodeStatementMapper

	assert.Same(t, rewritten, nilMapper.mapDeclaration(rewritten, rewritten))
	assert.Same(t, rewritten, newTestStatementMapper(token.NewFileSet()).mapDeclaration(rewritten, nil))
	nilMapper.reportIncompleteMapping(t.Context())
}

func TestPairUserFunctions_SkipsMisalignedDeclarations(t *testing.T) {
	t.Parallel()

	original, err := parser.ParseFile(token.NewFileSet(), "", "package p\nfunc A() {}\nfunc B() {}\n", 0)
	require.NoError(t, err)
	rewritten, err := parser.ParseFile(token.NewFileSet(), "", "package p\nvar _ = 1\nfunc A() {}\nfunc C() {}\n", 0)
	require.NoError(t, err)

	pairs, err := pairUserFunctions(original, rewritten)

	require.ErrorIs(t, err, errUserDeclarationsMisaligned)
	assert.ErrorContains(t, err, "B")
	require.Len(t, pairs, 1)
	for rewrittenFunc, originalFunc := range pairs {
		assert.Equal(t, "A", rewrittenFunc.Name.Name)
		assert.Equal(t, "A", originalFunc.Name.Name)
	}

	pairs, err = pairUserFunctions(rewritten, original)
	require.ErrorIs(t, err, errUserDeclarationsMisaligned, "more originals than rewritten declarations cannot be aligned")
	assert.Empty(t, pairs)

	pairs, err = pairUserFunctions(nil, rewritten)
	require.NoError(t, err)
	assert.Empty(t, pairs)
}

func TestPairUserFunctions_PairsAlignedFunctions(t *testing.T) {
	t.Parallel()

	original, err := parser.ParseFile(token.NewFileSet(), "", "package p\nimport \"fmt\"\ntype T struct{}\nfunc (T) A() {}\nfunc B() { fmt.Println() }\n", 0)
	require.NoError(t, err)
	rewritten, err := parser.ParseFile(token.NewFileSet(), "", "package p\nimport \"fmt\"\nvar _ = fmt.Sprint\ntype T struct{}\nfunc (T) A() {}\nfunc B() { fmt.Println() }\n", 0)
	require.NoError(t, err)

	pairs, err := pairUserFunctions(original, rewritten)

	require.NoError(t, err)
	assert.Len(t, pairs, 2)
}

func TestSameFunctionShape(t *testing.T) {
	t.Parallel()

	method := &goast.FuncDecl{Name: goast.NewIdent("A"), Recv: &goast.FieldList{}}
	function := &goast.FuncDecl{Name: goast.NewIdent("A")}
	unnamed := &goast.FuncDecl{}

	assert.True(t, sameFunctionShape(function, &goast.FuncDecl{Name: goast.NewIdent("A")}))
	assert.False(t, sameFunctionShape(method, function), "a method and a function differ")
	assert.False(t, sameFunctionShape(unnamed, function))
	assert.Equal(t, "<unnamed>", functionName(unnamed))
	assert.Equal(t, "A", functionName(function))
}

func TestGeneratedFilePathForDirectives(t *testing.T) {
	t.Parallel()

	t.Run("empty output path disables resets", func(t *testing.T) {
		t.Parallel()

		path, err := generatedFilePathForDirectives("", filepath.Abs)
		require.NoError(t, err)
		assert.Empty(t, path)
	})

	t.Run("absolute paths are kept", func(t *testing.T) {
		t.Parallel()

		path, err := generatedFilePathForDirectives("/project/dist/generated.go", filepath.Abs)
		require.NoError(t, err)
		assert.Equal(t, "/project/dist/generated.go", path)
	})

	t.Run("relative paths are made absolute", func(t *testing.T) {
		t.Parallel()

		path, err := generatedFilePathForDirectives("dist/generated.go", filepath.Abs)
		require.NoError(t, err)
		assert.True(t, strings.HasSuffix(path, "/dist/generated.go"))
	})

	t.Run("failure to make the path absolute is reported", func(t *testing.T) {
		t.Parallel()

		failure := errors.New("no working directory")
		path, err := generatedFilePathForDirectives("dist/generated.go", func(string) (string, error) {
			return "", failure
		})

		require.ErrorIs(t, err, errGeneratedFilePath)
		require.ErrorIs(t, err, failure)
		assert.Empty(t, path)
	})
}

func TestResolveGeneratedFilePath(t *testing.T) {
	t.Parallel()

	path, err := resolveGeneratedFilePath(t.Context(), generator_dto.GenerateRequest{OutputPath: "/gen/generated.go"})
	require.NoError(t, err)
	assert.Empty(t, path, "directives off needs no path")

	path, err = resolveGeneratedFilePath(t.Context(), generator_dto.GenerateRequest{EnableDwarfLineDirectives: true})
	require.NoError(t, err)
	assert.Empty(t, path, "no output path leaves resets off")

	path, err = resolveGeneratedFilePath(t.Context(), generator_dto.GenerateRequest{
		EnableDwarfLineDirectives: true,
		OutputPath:                "/gen/generated.go",
	})
	require.NoError(t, err)
	assert.Equal(t, "/gen/generated.go", path)
}

func TestUserCodeLineDirectives_CancelledContextStopsResets(t *testing.T) {
	t.Parallel()

	component := buildLineFidelityComponent(t)
	result := createMinimalAnnotationResult("test_hash")
	result.VirtualModule.ComponentsByHash["test_hash"] = component
	ctx, cancel := context.WithCancelCause(t.Context())
	cause := errors.New("build abandoned")
	cancel(cause)

	_, _, err := NewEmitter().EmitCode(ctx, result, generator_dto.GenerateRequest{
		SourcePath:                lineFidelityPkPath,
		OutputPath:                lineFidelityGeneratedPath,
		PackageName:               "main",
		HashedName:                "test_hash",
		CanonicalGoPackagePath:    "test.com/pkg",
		BaseDir:                   "/project",
		IsPage:                    true,
		EnableDwarfLineDirectives: true,
	})

	require.ErrorIs(t, err, cause)
}

func emitLineFidelityComponent(t *testing.T, enableDirectives bool) []byte {
	t.Helper()

	component := buildLineFidelityComponent(t)
	result := createMinimalAnnotationResult("test_hash")
	result.VirtualModule.ComponentsByHash["test_hash"] = component
	result.VirtualModule.Graph.PathToHashedName = map[string]string{lineFidelityPkPath: "test_hash"}
	sourcePath := lineFidelityPkPath
	result.AnnotatedAST.SourcePath = &sourcePath

	code, _, err := NewEmitter().EmitCode(context.Background(), result, generator_dto.GenerateRequest{
		SourcePath:                lineFidelityPkPath,
		OutputPath:                lineFidelityGeneratedPath,
		PackageName:               "main",
		HashedName:                "test_hash",
		CanonicalGoPackagePath:    "test.com/pkg",
		BaseDir:                   "/project",
		IsPage:                    true,
		EnableDwarfLineDirectives: enableDirectives,
	})
	require.NoError(t, err)
	return code
}

func buildLineFidelityComponent(t *testing.T) *annotator_dto.VirtualComponent {
	t.Helper()

	originalFileSet := token.NewFileSet()
	original, err := parser.ParseFile(originalFileSet, "script.go", lineFidelityScript, parser.ParseComments)
	require.NoError(t, err)

	var reprinted bytes.Buffer
	require.NoError(t, printer.Fprint(&reprinted, token.NewFileSet(), original))
	rewritten, err := parser.ParseFile(token.NewFileSet(), "", reprinted.String(), parser.ParseComments)
	require.NoError(t, err)

	useDeclaration := &goast.GenDecl{
		Tok: token.VAR,
		Specs: []goast.Spec{&goast.ValueSpec{
			Names:  []*goast.Ident{goast.NewIdent("_")},
			Values: []goast.Expr{&goast.SelectorExpr{X: goast.NewIdent("fmt"), Sel: goast.NewIdent("Sprint")}},
		}},
	}
	rewritten.Decls = append([]goast.Decl{rewritten.Decls[0], useDeclaration}, rewritten.Decls[1:]...)

	return &annotator_dto.VirtualComponent{
		HashedName:             "test_hash",
		PartialName:            "TestComponent",
		CanonicalGoPackagePath: "test.com/pkg",
		IsPage:                 true,
		Source: &annotator_dto.ParsedComponent{
			SourcePath: lineFidelityPkPath,
			Script: &annotator_dto.ParsedScript{
				AST:                 original,
				Fset:                originalFileSet,
				ScriptStartLocation: ast_domain.Location{Line: lineFidelityScriptStartLine},
			},
		},
		RewrittenScriptAST: rewritten,
	}
}

func collectScriptStatementLines(t *testing.T) []int {
	t.Helper()

	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, "script.go", lineFidelityScript, 0)
	require.NoError(t, err)

	var lines []int
	for _, functionName := range []string{"helper", "Describe"} {
		functionDecl := findFunction(file, functionName)
		require.NotNil(t, functionDecl)
		for _, statement := range collectStatements(functionDecl.Body) {
			lines = append(lines, lineFidelityScriptStartLine+fileSet.Position(statement.Pos()).Line-1)
		}
	}
	return lines
}

func collectStatements(body *goast.BlockStmt) []goast.Stmt {
	var statements []goast.Stmt
	goast.Inspect(body, func(node goast.Node) bool {
		switch statement := node.(type) {
		case *goast.BlockStmt, *goast.CaseClause, *goast.CommClause:
			return true
		case goast.Stmt:
			if statement != nil {
				statements = append(statements, statement)
			}
		}
		return true
	})
	return statements
}

func findFunction(file *goast.File, name string) *goast.FuncDecl {
	for _, declaration := range file.Decls {
		if functionDecl, ok := declaration.(*goast.FuncDecl); ok && functionDecl.Name.Name == name {
			return functionDecl
		}
	}
	return nil
}

func newTestStatementMapper(fileSet *token.FileSet) *userCodeStatementMapper {
	return &userCodeStatementMapper{
		emitter:         &emitter{config: EmitterConfig{EnableDwarfLineDirectives: true}},
		originalFileSet: fileSet,
		directivePath:   "/project/main.pk",
		scriptStartLine: lineFidelityScriptStartLine,
	}
}

func assertResetsReportPhysicalLines(t *testing.T, source []byte) {
	t.Helper()

	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, "/gen/generated.go", source, parser.SkipObjectResolution)
	require.NoError(t, err)
	for _, declaration := range file.Decls[1:] {
		adjusted := fileSet.PositionFor(declaration.Pos(), true)
		if adjusted.Filename != "/gen/generated.go" {
			continue
		}
		assert.Equal(t, fileSet.PositionFor(declaration.Pos(), false).Line, adjusted.Line)
	}
}

func printNode(t *testing.T, node any) string {
	t.Helper()

	var buffer bytes.Buffer
	require.NoError(t, printer.Fprint(&buffer, token.NewFileSet(), node))
	return buffer.String()
}

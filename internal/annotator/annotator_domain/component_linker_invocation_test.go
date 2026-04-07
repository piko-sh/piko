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

package annotator_domain

import (
	"context"
	goast "go/ast"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/inspector/inspector_domain"
	"piko.sh/piko/internal/inspector/inspector_dto"
	"piko.sh/piko/internal/logger/logger_domain"
)

func TestNewInvocationLinker(t *testing.T) {
	t.Run("ValidInputs", func(t *testing.T) {
		pInfo := createTestPartialInvocationInfo()
		resolver := &TypeResolver{
			inspector: &inspector_domain.MockTypeQuerier{
				GetImportsForFileFunc: func(_, _ string) map[string]string {
					return map[string]string{}
				},
			},
		}
		vm := createInvocationTestVirtualModule()
		ctx := createInvocationTestContext()

		linker, err := newInvocationLinker(pInfo, resolver, vm, ctx)

		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		require.NotNil(t, linker, "Expected non-nil linker")
		if linker.typeResolver != resolver {
			t.Error("Expected resolver to be set")
		}
		if linker.invokerCtx != ctx {
			t.Error("Expected context to be set")
		}
		if linker.canonicalProps == nil {
			t.Error("Expected canonicalProps map to be initialised")
		}
		if linker.providedPropOrigins == nil {
			t.Error("Expected providedPropOrigins map to be initialised")
		}
	})

	t.Run("PartialComponentNotFound", func(t *testing.T) {
		pInfo := &ast_domain.PartialInvocationInfo{
			PartialPackageName: "nonexistent_hash",
			PartialAlias:       "unknown",
		}
		resolver := &TypeResolver{
			inspector: &inspector_domain.MockTypeQuerier{
				GetImportsForFileFunc: func(_, _ string) map[string]string {
					return map[string]string{}
				},
			},
		}
		vm := createInvocationTestVirtualModule()
		ctx := createInvocationTestContext()

		_, err := newInvocationLinker(pInfo, resolver, vm, ctx)

		if err == nil {
			t.Error("Expected error for nonexistent partial component")
		}
	})

	t.Run("DeepCopiesPassedProps", func(t *testing.T) {
		originalExpr := &ast_domain.StringLiteral{Value: "original"}
		pInfo := &ast_domain.PartialInvocationInfo{
			PartialPackageName: "partial_123",
			PartialAlias:       "myPartial",
			PassedProps: map[string]ast_domain.PropValue{
				"title": {
					Expression: originalExpr,
					Location:   ast_domain.Location{Line: 1, Column: 1, Offset: 0},
				},
			},
		}
		resolver := &TypeResolver{
			inspector: &inspector_domain.MockTypeQuerier{
				GetImportsForFileFunc: func(_, _ string) map[string]string {
					return map[string]string{}
				},
			},
		}
		vm := createInvocationTestVirtualModule()
		ctx := createInvocationTestContext()

		linker, _ := newInvocationLinker(pInfo, resolver, vm, ctx)

		if _, exists := linker.invocation.PassedProps["title"]; !exists {
			t.Error("Expected props to be copied to linker")
		}
	})
}

func TestInvocationLinker_ApplyRequestOverrides(t *testing.T) {
	t.Run("UnknownRequestOverride", func(t *testing.T) {
		linker := createTestInvocationLinker()
		linker.validProps = map[string]validPropInfo{
			"Method": {
				GoFieldName:     "Method",
				DestinationType: goast.NewIdent("string"),
			},
		}
		linker.invocation.RequestOverrides = map[string]ast_domain.PropValue{
			"UnknownProp": {
				Expression: &ast_domain.StringLiteral{Value: "value"},
				Location:   ast_domain.Location{Line: 1, Column: 1, Offset: 0},
			},
		}

		linker.applyRequestOverrides(context.Background())

		if len(*linker.invokerCtx.Diagnostics) == 0 {
			t.Error("Expected diagnostic for unknown request override")
		}
	})
}

func TestInvocationLinker_CalculateCanonicalKey(t *testing.T) {
	t.Run("GeneratesKey", func(t *testing.T) {
		linker := createTestInvocationLinker()
		linker.invocation.PartialAlias = "myPartial"
		linker.canonicalProps = map[string]ast_domain.PropValue{
			"title": {
				Expression: &ast_domain.StringLiteral{Value: "Hello"},
			},
		}

		key := linker.calculateCanonicalKey()

		if key == "" {
			t.Error("Expected non-empty canonical key")
		}
	})

	t.Run("DeterministicKey", func(t *testing.T) {
		linker1 := createTestInvocationLinker()
		linker1.invocation.PartialAlias = "myPartial"
		linker1.canonicalProps = map[string]ast_domain.PropValue{
			"a": {Expression: &ast_domain.StringLiteral{Value: "1"}},
			"b": {Expression: &ast_domain.StringLiteral{Value: "2"}},
		}

		linker2 := createTestInvocationLinker()
		linker2.invocation.PartialAlias = "myPartial"
		linker2.canonicalProps = map[string]ast_domain.PropValue{
			"b": {Expression: &ast_domain.StringLiteral{Value: "2"}},
			"a": {Expression: &ast_domain.StringLiteral{Value: "1"}},
		}

		key1 := linker1.calculateCanonicalKey()
		key2 := linker2.calculateCanonicalKey()

		if key1 != key2 {
			t.Errorf("Expected identical keys, got '%s' and '%s'", key1, key2)
		}
	})
}

func TestInvocationLinker_GetFinalExprAfterCoercion(t *testing.T) {
	t.Run("NoCoercionWhenNotNeeded", func(t *testing.T) {
		linker := createTestInvocationLinker()
		sourceExpression := &ast_domain.IntegerLiteral{Value: 42}
		propInfo := validPropInfo{
			GoFieldName:     "Count",
			DestinationType: goast.NewIdent("int"),
			ShouldCoerce:    false,
		}

		result := linker.getFinalExprAfterCoercion("count", sourceExpression, ast_domain.Location{}, propInfo)

		if result != sourceExpression {
			t.Error("Expected no coercion when not needed")
		}
	})

	t.Run("CoercionWhenEnabled", func(t *testing.T) {
		linker := createTestInvocationLinker()
		sourceExpression := &ast_domain.StringLiteral{Value: "42"}
		propInfo := validPropInfo{
			GoFieldName:     "Count",
			DestinationType: goast.NewIdent("int"),
			ShouldCoerce:    true,
		}

		result := linker.getFinalExprAfterCoercion("count", sourceExpression, ast_domain.Location{}, propInfo)

		if _, ok := result.(*ast_domain.IntegerLiteral); !ok {
			t.Errorf("Expected IntegerLiteral after coercion, got %T", result)
		}
	})
}

func TestInvocationLinker_HandleLiteralDefault(t *testing.T) {
	t.Run("ParsesLiteralDefault", func(t *testing.T) {
		linker := createTestInvocationLinker()
		propInfo := validPropInfo{
			GoFieldName:     "Count",
			DestinationType: goast.NewIdent("int"),
			DefaultValue:    new("42"),
		}

		linker.handleLiteralDefault(context.Background(), "count", propInfo)

		if value, ok := linker.canonicalProps["count"]; !ok {
			t.Error("Expected default prop to be stored")
		} else if _, isInt := value.Expression.(*ast_domain.IntegerLiteral); !isInt {
			t.Errorf("Expected IntegerLiteral, got %T", value.Expression)
		}
	})
}

func TestInvocationLinker_TryCoerce(t *testing.T) {
	t.Run("SuccessfulCoercion", func(t *testing.T) {
		linker := createTestInvocationLinker()
		sourceExpression := &ast_domain.StringLiteral{Value: "true"}
		sourceAnn := &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: goast.NewIdent("string"),
			},
		}
		destType := &ast_domain.ResolvedTypeInfo{
			TypeExpression: goast.NewIdent("bool"),
		}

		resultExpr, resultAnn, success := linker.tryCoerce(context.Background(), sourceExpression, sourceAnn, destType, ast_domain.Location{}, "enabled")

		if !success {
			t.Error("Expected successful coercion")
		}
		if _, ok := resultExpr.(*ast_domain.BooleanLiteral); !ok {
			t.Errorf("Expected BooleanLiteral after coercion, got %T", resultExpr)
		}
		require.NotNil(t, resultAnn, "Expected non-nil annotation after coercion")
	})

	t.Run("FailedCoercion", func(t *testing.T) {
		linker := createTestInvocationLinker()
		sourceExpression := &ast_domain.StringLiteral{Value: "not a number"}
		sourceAnn := &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: goast.NewIdent("string"),
			},
		}
		destType := &ast_domain.ResolvedTypeInfo{
			TypeExpression: goast.NewIdent("int"),
		}

		_, _, success := linker.tryCoerce(context.Background(), sourceExpression, sourceAnn, destType, ast_domain.Location{}, "count")

		if success {
			t.Error("Expected failed coercion for invalid conversion")
		}
	})
}

func TestInvocationLinker_StoreProp(t *testing.T) {
	t.Run("StoresPropSuccessfully", func(t *testing.T) {
		linker := createTestInvocationLinker()
		expression := &ast_domain.StringLiteral{Value: "Hello"}
		loc := ast_domain.Location{Line: 1, Column: 1, Offset: 0}
		propInfo := validPropInfo{
			GoFieldName:     "Title",
			DestinationType: goast.NewIdent("string"),
		}
		ann := &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: goast.NewIdent("string"),
			},
		}

		linker.storeProp("title", expression, loc, loc, propInfo, ann, false)

		if value, ok := linker.canonicalProps["title"]; !ok {
			t.Error("Expected prop to be stored")
		} else {
			if value.Expression != expression {
				t.Error("Expected stored expression to match")
			}
			if value.GoFieldName != "Title" {
				t.Errorf("Expected GoFieldName 'Title', got '%s'", value.GoFieldName)
			}
		}
	})
}

func TestInvocationLinker_StoreOptionalProp(t *testing.T) {
	t.Run("StoresOptionalProp", func(t *testing.T) {
		linker := createTestInvocationLinker()
		expression := &ast_domain.StringLiteral{Value: "Optional"}
		loc := ast_domain.Location{Line: 1, Column: 1, Offset: 0}
		propInfo := validPropInfo{
			GoFieldName:     "OptionalTitle",
			DestinationType: goast.NewIdent("*string"),
		}
		sourceAnn := &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: goast.NewIdent("string"),
			},
		}
		destType := &ast_domain.ResolvedTypeInfo{
			TypeExpression: &goast.StarExpr{
				X: goast.NewIdent("string"),
			},
		}

		params := &propAssignmentParams{
			PropName:         "optionaltitle",
			SourceExpression: expression,
			Loc:              loc,
			NameLocation:     loc,
			PropInfo:         propInfo,
			SourceAnnotation: sourceAnn,
			DestTypeInfo:     destType,
			IsLoopDependent:  false,
		}
		linker.storeOptionalProp(params)

		if value, ok := linker.canonicalProps["optionaltitle"]; !ok {
			t.Error("Expected optional prop to be stored")
		} else {
			if unary, ok := value.Expression.(*ast_domain.UnaryExpression); !ok {
				t.Errorf("Expected UnaryExpression for optional prop, got %T", value.Expression)
			} else if unary.Operator != ast_domain.OpAddrOf {
				t.Errorf("Expected unary operator OpAddrOf, got '%s'", unary.Operator)
			}
		}
	})
}

func createTestPartialInvocationInfo() *ast_domain.PartialInvocationInfo {
	return &ast_domain.PartialInvocationInfo{
		PartialPackageName: "partial_123",
		PartialAlias:       "myPartial",
		PassedProps:        make(map[string]ast_domain.PropValue),
		Location:           ast_domain.Location{Line: 1, Column: 1, Offset: 0},
	}
}

func createInvocationTestVirtualModule() *annotator_dto.VirtualModule {
	return &annotator_dto.VirtualModule{
		ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
			"partial_123": {
				HashedName:             "partial_123",
				CanonicalGoPackagePath: "test/partial",
				VirtualGoFilePath:      "/virtual/partial.go",
				Source: &annotator_dto.ParsedComponent{
					SourcePath: "/test/partial.piko",
				},
			},
		},
	}
}

func createInvocationTestContext() *AnalysisContext {
	return &AnalysisContext{
		Symbols:                  NewSymbolTable(nil),
		Diagnostics:              new([]*ast_domain.Diagnostic),
		CurrentGoFullPackagePath: "test/invoker",
		CurrentGoPackageName:     "invoker",
		CurrentGoSourcePath:      "/test/invoker.go",
		SFCSourcePath:            "/test/invoker.piko",
		Logger:                   logger_domain.GetLogger("test"),
	}
}

func createTestInvocationLinker() *invocationLinker {
	ctx := &AnalysisContext{
		Symbols:                  NewSymbolTable(nil),
		Diagnostics:              new([]*ast_domain.Diagnostic),
		CurrentGoFullPackagePath: "test/invoker",
		CurrentGoPackageName:     "invoker",
		CurrentGoSourcePath:      "/test/invoker.go",
		SFCSourcePath:            "/test/invoker.piko",
		Logger:                   logger_domain.GetLogger("test"),
	}

	return &invocationLinker{
		invocation: &annotator_dto.PartialInvocation{
			PartialAlias:     "testPartial",
			PassedProps:      make(map[string]ast_domain.PropValue),
			RequestOverrides: make(map[string]ast_domain.PropValue),
		},
		typeResolver: &TypeResolver{
			inspector: &inspector_domain.MockTypeQuerier{
				GetImportsForFileFunc: func(_, _ string) map[string]string {
					return map[string]string{}
				},
			},
		},
		invokerCtx: ctx,
		partialVirtualComponent: &annotator_dto.VirtualComponent{
			HashedName:             "partial_test",
			CanonicalGoPackagePath: "test/partial",
			VirtualGoFilePath:      "/virtual/partial.go",
			RewrittenScriptAST: &goast.File{
				Name: goast.NewIdent("partial_pkg"),
			},
			Source: &annotator_dto.ParsedComponent{
				SourcePath: "/test/partial.piko",
			},
		},
		validProps:          make(map[string]validPropInfo),
		providedPropOrigins: make(map[string]propOrigin),
		canonicalProps:      make(map[string]ast_domain.PropValue),
	}
}

func TestCategorisePassedProps(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name             string
		provisionalProps map[string]ast_domain.PropValue
		expectedStandard []string
		expectedServer   []string
	}{
		{
			name:             "empty map returns empty maps",
			provisionalProps: map[string]ast_domain.PropValue{},
			expectedStandard: []string{},
			expectedServer:   []string{},
		},
		{
			name: "standard props only",
			provisionalProps: map[string]ast_domain.PropValue{
				"title":   {Expression: &ast_domain.StringLiteral{Value: "Hello"}},
				"count":   {Expression: &ast_domain.IntegerLiteral{Value: 42}},
				"enabled": {Expression: &ast_domain.BooleanLiteral{Value: true}},
			},
			expectedStandard: []string{"title", "count", "enabled"},
			expectedServer:   []string{},
		},
		{
			name: "server props only",
			provisionalProps: map[string]ast_domain.PropValue{
				"server.data":   {Expression: &ast_domain.StringLiteral{Value: "data"}},
				"server.config": {Expression: &ast_domain.StringLiteral{Value: "config"}},
			},
			expectedStandard: []string{},
			expectedServer:   []string{"server.data", "server.config"},
		},
		{
			name: "mixed standard and server props",
			provisionalProps: map[string]ast_domain.PropValue{
				"title":         {Expression: &ast_domain.StringLiteral{Value: "Hello"}},
				"server.data":   {Expression: &ast_domain.StringLiteral{Value: "data"}},
				"count":         {Expression: &ast_domain.IntegerLiteral{Value: 42}},
				"server.config": {Expression: &ast_domain.StringLiteral{Value: "config"}},
			},
			expectedStandard: []string{"title", "count"},
			expectedServer:   []string{"server.data", "server.config"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			standard, server := categorisePassedProps(tc.provisionalProps)

			if len(standard) != len(tc.expectedStandard) {
				t.Errorf("Expected %d standard props, got %d", len(tc.expectedStandard), len(standard))
			}
			for _, key := range tc.expectedStandard {
				if _, ok := standard[key]; !ok {
					t.Errorf("Expected standard prop '%s' to be present", key)
				}
			}

			if len(server) != len(tc.expectedServer) {
				t.Errorf("Expected %d server props, got %d", len(tc.expectedServer), len(server))
			}
			for _, key := range tc.expectedServer {
				if _, ok := server[key]; !ok {
					t.Errorf("Expected server prop '%s' to be present", key)
				}
			}
		})
	}
}

func TestCollectDependenciesFromProps(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		props            map[string]ast_domain.PropValue
		name             string
		expectedDepCount int
	}{
		{
			name:             "empty props returns empty slice",
			props:            map[string]ast_domain.PropValue{},
			expectedDepCount: 0,
		},
		{
			name: "props without dependencies",
			props: map[string]ast_domain.PropValue{
				"title": {Expression: &ast_domain.StringLiteral{Value: "Hello"}},
			},
			expectedDepCount: 0,
		},
		{
			name: "nil expression is skipped",
			props: map[string]ast_domain.PropValue{
				"title": {Expression: nil},
			},
			expectedDepCount: 0,
		},
		{
			name: "prop with source invocation key",
			props: map[string]ast_domain.PropValue{
				"title": {
					Expression: createExprWithInvocationKey("inv_123"),
				},
			},
			expectedDepCount: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := collectDependenciesFromProps(tc.props)

			if len(result) != tc.expectedDepCount {
				t.Errorf("Expected %d dependencies, got %d", tc.expectedDepCount, len(result))
			}
		})
	}
}

func createExprWithInvocationKey(key string) ast_domain.Expression {
	return &ast_domain.Identifier{
		Name: "data",
		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
			SourceInvocationKey: new(key),
		},
	}
}

func TestCreateAnnotatedIdentifier(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		idName       string
		packageAlias string
		packagePath  string
	}{
		{
			name:         "simple identifier",
			idName:       "myVar",
			packageAlias: "",
			packagePath:  "",
		},
		{
			name:         "identifier with package",
			idName:       "fmt",
			packageAlias: "fmt",
			packagePath:  "fmt",
		},
		{
			name:         "identifier with different alias",
			idName:       "strconv",
			packageAlias: "strconv",
			packagePath:  "strconv",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := createAnnotatedIdentifier(tc.idName, tc.packageAlias, tc.packagePath)

			if result.Name != tc.idName {
				t.Errorf("Expected name '%s', got '%s'", tc.idName, result.Name)
			}
			if result.GoAnnotations == nil {
				t.Error("Expected non-nil GoAnnotations")
			}
			if result.GoAnnotations.ResolvedType == nil {
				t.Error("Expected non-nil ResolvedType")
			}
			if result.GoAnnotations.ResolvedType.PackageAlias != tc.packageAlias {
				t.Errorf("Expected PackageAlias '%s', got '%s'", tc.packageAlias, result.GoAnnotations.ResolvedType.PackageAlias)
			}
			if result.GoAnnotations.ResolvedType.CanonicalPackagePath != tc.packagePath {
				t.Errorf("Expected CanonicalPackagePath '%s', got '%s'", tc.packagePath, result.GoAnnotations.ResolvedType.CanonicalPackagePath)
			}
		})
	}
}

func TestCreateIntegerLiteralAST(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		value int
	}{
		{name: "zero", value: 0},
		{name: "positive", value: 42},
		{name: "negative", value: -1},
		{name: "large", value: 9999999},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := createIntegerLiteralAST(tc.value)

			if result.Value != int64(tc.value) {
				t.Errorf("Expected value %d, got %d", tc.value, result.Value)
			}
		})
	}
}

func TestCreateTypeCastCallAST(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		typeName string
	}{
		{name: "int64 cast", typeName: "int64"},
		{name: "uint64 cast", typeName: "uint64"},
		{name: "float64 cast", typeName: "float64"},
		{name: "string cast", typeName: "string"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			argument := &ast_domain.IntegerLiteral{Value: 42}
			result := createTypeCastCallAST(tc.typeName, argument)

			if result.Callee == nil {
				t.Error("Expected non-nil Callee")
			}
			identifier, ok := result.Callee.(*ast_domain.Identifier)
			if !ok {
				t.Errorf("Expected Callee to be Identifier, got %T", result.Callee)
			}
			if identifier.Name != tc.typeName {
				t.Errorf("Expected type name '%s', got '%s'", tc.typeName, identifier.Name)
			}
			if len(result.Args) != 1 {
				t.Errorf("Expected 1 argument, got %d", len(result.Args))
			}
		})
	}
}

func TestCreateStrconvCallAST(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		functionName string
		argCount     int
	}{
		{name: "FormatInt", functionName: "FormatInt", argCount: 2},
		{name: "FormatUint", functionName: "FormatUint", argCount: 2},
		{name: "FormatBool", functionName: "FormatBool", argCount: 1},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			arguments := make([]ast_domain.Expression, tc.argCount)
			for i := range tc.argCount {
				arguments[i] = &ast_domain.IntegerLiteral{Value: int64(i)}
			}

			result := createStrconvCallAST(tc.functionName, arguments...)

			if result.Callee == nil {
				t.Error("Expected non-nil Callee")
			}
			memberExpr, ok := result.Callee.(*ast_domain.MemberExpression)
			if !ok {
				t.Errorf("Expected MemberExpr callee, got %T", result.Callee)
			}
			if memberExpr.Property == nil {
				t.Error("Expected non-nil Property")
			}
			propIdent, ok := memberExpr.Property.(*ast_domain.Identifier)
			if !ok {
				t.Errorf("Expected Property to be Identifier, got %T", memberExpr.Property)
			}
			if propIdent.Name != tc.functionName {
				t.Errorf("Expected func name '%s', got '%s'", tc.functionName, propIdent.Name)
			}
			if len(result.Args) != tc.argCount {
				t.Errorf("Expected %d arguments, got %d", tc.argCount, len(result.Args))
			}
		})
	}
}

func TestConvertIntToString(t *testing.T) {
	t.Parallel()

	sourceExpression := &ast_domain.IntegerLiteral{Value: 42}
	result := convertIntToString(sourceExpression)

	callExpr, ok := result.(*ast_domain.CallExpression)
	if !ok {
		t.Fatalf("Expected CallExpr, got %T", result)
	}

	memberExpr, ok := callExpr.Callee.(*ast_domain.MemberExpression)
	if !ok {
		t.Fatalf("Expected MemberExpr callee, got %T", callExpr.Callee)
	}

	propIdent, ok := memberExpr.Property.(*ast_domain.Identifier)
	if !ok {
		t.Fatalf("Expected Property to be Identifier, got %T", memberExpr.Property)
	}
	if propIdent.Name != "FormatInt" {
		t.Errorf("Expected FormatInt, got %s", propIdent.Name)
	}
}

func TestConvertUintToString(t *testing.T) {
	t.Parallel()

	sourceExpression := &ast_domain.IntegerLiteral{Value: 42}
	result := convertUintToString(sourceExpression)

	callExpr, ok := result.(*ast_domain.CallExpression)
	if !ok {
		t.Fatalf("Expected CallExpr, got %T", result)
	}

	memberExpr, ok := callExpr.Callee.(*ast_domain.MemberExpression)
	if !ok {
		t.Fatalf("Expected MemberExpr callee, got %T", callExpr.Callee)
	}

	propIdent, ok := memberExpr.Property.(*ast_domain.Identifier)
	if !ok {
		t.Fatalf("Expected Property to be Identifier, got %T", memberExpr.Property)
	}
	if propIdent.Name != "FormatUint" {
		t.Errorf("Expected FormatUint, got %s", propIdent.Name)
	}
}

func TestConvertFloatToString(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		typeString string
	}{
		{name: "float32", typeString: "float32"},
		{name: "float64", typeString: "float64"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			sourceExpression := &ast_domain.FloatLiteral{Value: 3.14}
			result := convertFloatToString(sourceExpression, tc.typeString)

			callExpr, ok := result.(*ast_domain.CallExpression)
			if !ok {
				t.Fatalf("Expected CallExpr, got %T", result)
			}

			memberExpr, ok := callExpr.Callee.(*ast_domain.MemberExpression)
			if !ok {
				t.Fatalf("Expected MemberExpr callee, got %T", callExpr.Callee)
			}

			propIdent, ok := memberExpr.Property.(*ast_domain.Identifier)
			if !ok {
				t.Fatalf("Expected Property to be Identifier, got %T", memberExpr.Property)
			}
			if propIdent.Name != "FormatFloat" {
				t.Errorf("Expected FormatFloat, got %s", propIdent.Name)
			}

			if len(callExpr.Args) != 4 {
				t.Errorf("Expected 4 arguments for FormatFloat, got %d", len(callExpr.Args))
			}
		})
	}
}

func TestConvertBoolToString(t *testing.T) {
	t.Parallel()

	sourceExpression := &ast_domain.BooleanLiteral{Value: true}
	result := convertBoolToString(sourceExpression)

	callExpr, ok := result.(*ast_domain.CallExpression)
	if !ok {
		t.Fatalf("Expected CallExpr, got %T", result)
	}

	memberExpr, ok := callExpr.Callee.(*ast_domain.MemberExpression)
	if !ok {
		t.Fatalf("Expected MemberExpr callee, got %T", callExpr.Callee)
	}

	propIdent, ok := memberExpr.Property.(*ast_domain.Identifier)
	if !ok {
		t.Fatalf("Expected Property to be Identifier, got %T", memberExpr.Property)
	}
	if propIdent.Name != "FormatBool" {
		t.Errorf("Expected FormatBool, got %s", propIdent.Name)
	}
}

func TestConvertRuneToString(t *testing.T) {
	t.Parallel()

	sourceExpression := &ast_domain.RuneLiteral{Value: 'A'}
	result := convertRuneToString(sourceExpression)

	callExpr, ok := result.(*ast_domain.CallExpression)
	if !ok {
		t.Fatalf("Expected CallExpr, got %T", result)
	}

	identifier, ok := callExpr.Callee.(*ast_domain.Identifier)
	if !ok {
		t.Fatalf("Expected Identifier callee, got %T", callExpr.Callee)
	}
	if identifier.Name != "string" {
		t.Errorf("Expected string type cast, got %s", identifier.Name)
	}
}

func TestConvertViaStringerMethod(t *testing.T) {
	t.Parallel()

	sourceExpression := &ast_domain.Identifier{Name: "myObj"}
	result := convertViaStringerMethod(sourceExpression)

	callExpr, ok := result.(*ast_domain.CallExpression)
	if !ok {
		t.Fatalf("Expected CallExpr, got %T", result)
	}

	memberExpr, ok := callExpr.Callee.(*ast_domain.MemberExpression)
	if !ok {
		t.Fatalf("Expected MemberExpr callee, got %T", callExpr.Callee)
	}

	propIdent, ok := memberExpr.Property.(*ast_domain.Identifier)
	if !ok {
		t.Fatalf("Expected Property to be Identifier, got %T", memberExpr.Property)
	}
	if propIdent.Name != "String" {
		t.Errorf("Expected String method, got %s", propIdent.Name)
	}
}

func TestConvertViaRuntimeFallback(t *testing.T) {
	t.Parallel()

	sourceExpression := &ast_domain.Identifier{Name: "unknownValue"}
	result := convertViaRuntimeFallback(sourceExpression)

	callExpr, ok := result.(*ast_domain.CallExpression)
	if !ok {
		t.Fatalf("Expected CallExpr, got %T", result)
	}

	memberExpr, ok := callExpr.Callee.(*ast_domain.MemberExpression)
	if !ok {
		t.Fatalf("Expected MemberExpr callee, got %T", callExpr.Callee)
	}

	baseIdent, ok := memberExpr.Base.(*ast_domain.Identifier)
	if !ok {
		t.Fatalf("Expected Base to be Identifier, got %T", memberExpr.Base)
	}
	if baseIdent.Name != "pikoruntime" {
		t.Errorf("Expected pikoruntime, got %s", baseIdent.Name)
	}

	propIdent, ok := memberExpr.Property.(*ast_domain.Identifier)
	if !ok {
		t.Fatalf("Expected Property to be Identifier, got %T", memberExpr.Property)
	}
	if propIdent.Name != "ValueToString" {
		t.Errorf("Expected ValueToString, got %s", propIdent.Name)
	}
}

func TestUpdateExpressionBaseCodeGenVarName(t *testing.T) {
	t.Parallel()

	t.Run("nil baseCodeGenVarName does nothing", func(t *testing.T) {
		t.Parallel()

		expression := &ast_domain.Identifier{Name: "myVar"}
		updateExpressionBaseCodeGenVarName(expression, nil)

		if expression.GoAnnotations != nil {
			t.Error("Expected GoAnnotations to remain nil")
		}
	})

	t.Run("updates identifier", func(t *testing.T) {
		t.Parallel()

		varName := "props_parent"
		expression := &ast_domain.Identifier{Name: "myVar"}

		updateExpressionBaseCodeGenVarName(expression, &varName)

		if expression.GoAnnotations == nil {
			t.Fatal("Expected GoAnnotations to be set")
		}
		if expression.GoAnnotations.BaseCodeGenVarName == nil {
			t.Fatal("Expected BaseCodeGenVarName to be set")
		}
		if *expression.GoAnnotations.BaseCodeGenVarName != varName {
			t.Errorf("Expected BaseCodeGenVarName '%s', got '%s'", varName, *expression.GoAnnotations.BaseCodeGenVarName)
		}
	})

	t.Run("updates member expression base", func(t *testing.T) {
		t.Parallel()

		varName := "props_parent"
		baseIdent := &ast_domain.Identifier{Name: "data"}
		expression := &ast_domain.MemberExpression{
			Base:     baseIdent,
			Property: &ast_domain.Identifier{Name: "value"},
		}

		updateExpressionBaseCodeGenVarName(expression, &varName)

		if baseIdent.GoAnnotations == nil {
			t.Fatal("Expected GoAnnotations to be set on base")
		}
		if baseIdent.GoAnnotations.BaseCodeGenVarName == nil {
			t.Fatal("Expected BaseCodeGenVarName to be set on base")
		}
		if *baseIdent.GoAnnotations.BaseCodeGenVarName != varName {
			t.Errorf("Expected BaseCodeGenVarName '%s', got '%s'", varName, *baseIdent.GoAnnotations.BaseCodeGenVarName)
		}
	})

	t.Run("updates index expression base", func(t *testing.T) {
		t.Parallel()

		varName := "props_parent"
		baseIdent := &ast_domain.Identifier{Name: "items"}
		expression := &ast_domain.IndexExpression{
			Base:  baseIdent,
			Index: &ast_domain.IntegerLiteral{Value: 0},
		}

		updateExpressionBaseCodeGenVarName(expression, &varName)

		if baseIdent.GoAnnotations == nil {
			t.Fatal("Expected GoAnnotations to be set on base")
		}
		if *baseIdent.GoAnnotations.BaseCodeGenVarName != varName {
			t.Errorf("Expected BaseCodeGenVarName '%s', got '%s'", varName, *baseIdent.GoAnnotations.BaseCodeGenVarName)
		}
	})

	t.Run("updates call expression callee", func(t *testing.T) {
		t.Parallel()

		varName := "props_parent"
		calleeIdent := &ast_domain.Identifier{Name: "getData"}
		expression := &ast_domain.CallExpression{
			Callee: calleeIdent,
			Args:   []ast_domain.Expression{},
		}

		updateExpressionBaseCodeGenVarName(expression, &varName)

		if calleeIdent.GoAnnotations == nil {
			t.Fatal("Expected GoAnnotations to be set on callee")
		}
		if *calleeIdent.GoAnnotations.BaseCodeGenVarName != varName {
			t.Errorf("Expected BaseCodeGenVarName '%s', got '%s'", varName, *calleeIdent.GoAnnotations.BaseCodeGenVarName)
		}
	})
}

func TestInvocationLinkerPool(t *testing.T) {

	t.Run("getInvocationLinker returns initialised linker", func(t *testing.T) {
		pInfo := createTestPartialInvocationInfo()
		resolver := &TypeResolver{
			inspector: &inspector_domain.MockTypeQuerier{
				GetImportsForFileFunc: func(_, _ string) map[string]string {
					return map[string]string{}
				},
			},
		}
		vm := createInvocationTestVirtualModule()
		ctx := createInvocationTestContext()

		linker, err := getInvocationLinker(pInfo, resolver, vm, ctx)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		require.NotNil(t, linker, "Expected non-nil linker")
		if linker.canonicalProps == nil {
			t.Error("Expected canonicalProps to be initialised")
		}
		if linker.providedPropOrigins == nil {
			t.Error("Expected providedPropOrigins to be initialised")
		}

		putInvocationLinker(linker)
	})

	t.Run("putInvocationLinker clears fields", func(t *testing.T) {
		pInfo := createTestPartialInvocationInfo()
		resolver := &TypeResolver{
			inspector: &inspector_domain.MockTypeQuerier{
				GetImportsForFileFunc: func(_, _ string) map[string]string {
					return map[string]string{}
				},
			},
		}
		vm := createInvocationTestVirtualModule()
		ctx := createInvocationTestContext()

		linker, _ := getInvocationLinker(pInfo, resolver, vm, ctx)
		putInvocationLinker(linker)

		if linker.invocation != nil {
			t.Error("Expected invocation to be nil after put")
		}
		if linker.typeResolver != nil {
			t.Error("Expected typeResolver to be nil after put")
		}
		if linker.invokerCtx != nil {
			t.Error("Expected invokerCtx to be nil after put")
		}
	})

	t.Run("getInvocationLinker returns error for missing partial", func(t *testing.T) {

		pInfo := &ast_domain.PartialInvocationInfo{
			PartialPackageName: "nonexistent_hash",
		}
		resolver := &TypeResolver{
			inspector: &inspector_domain.MockTypeQuerier{
				GetImportsForFileFunc: func(_, _ string) map[string]string {
					return map[string]string{}
				},
			},
		}
		vm := createInvocationTestVirtualModule()
		ctx := createInvocationTestContext()

		_, err := getInvocationLinker(pInfo, resolver, vm, ctx)

		if err == nil {
			t.Error("Expected error for nonexistent partial")
		}
	})
}

func TestInvocationLinker_BuildDestinationTypeInfo(t *testing.T) {
	t.Parallel()

	t.Run("builds type info from prop info", func(t *testing.T) {
		t.Parallel()

		linker := createTestInvocationLinker()
		linker.partialVirtualComponent = &annotator_dto.VirtualComponent{
			CanonicalGoPackagePath: "example.com/partials/card",
			RewrittenScriptAST: &goast.File{
				Name: goast.NewIdent("card_abc123"),
			},
		}
		propInfo := validPropInfo{
			GoFieldName:     "Title",
			DestinationType: goast.NewIdent("string"),
		}

		result := linker.buildDestinationTypeInfo(propInfo)

		assert.NotNil(t, result)
		assert.Equal(t, "card_abc123", result.PackageAlias)
		assert.Equal(t, "example.com/partials/card", result.CanonicalPackagePath)
		assert.False(t, result.IsSynthetic)
		assert.False(t, result.IsExportedPackageSymbol)

		identifier, ok := result.TypeExpression.(*goast.Ident)
		require.True(t, ok)
		assert.Equal(t, "string", identifier.Name)
	})
}

func TestInvocationLinker_IsLoopVariable(t *testing.T) {
	t.Parallel()

	t.Run("variable not in current scope returns false", func(t *testing.T) {
		t.Parallel()

		linker := createTestInvocationLinker()

		result := linker.isLoopVariable("unknownVar")

		assert.False(t, result)
	})

	t.Run("variable in current scope with no parent returns false", func(t *testing.T) {
		t.Parallel()

		linker := createTestInvocationLinker()
		linker.invokerCtx.Symbols = NewSymbolTable(nil)
		linker.invokerCtx.Symbols.Define(Symbol{Name: "item", CodeGenVarName: "item"})

		result := linker.isLoopVariable("item")

		assert.False(t, result)
	})

	t.Run("variable in current scope and parent returns false", func(t *testing.T) {
		t.Parallel()

		linker := createTestInvocationLinker()
		parentScope := NewSymbolTable(nil)
		parentScope.Define(Symbol{Name: "globalVar", CodeGenVarName: "globalVar"})
		childScope := NewSymbolTable(parentScope)
		childScope.Define(Symbol{Name: "globalVar", CodeGenVarName: "globalVar"})
		linker.invokerCtx.Symbols = childScope

		result := linker.isLoopVariable("globalVar")

		assert.False(t, result)
	})

	t.Run("variable in current scope but not in parent returns true", func(t *testing.T) {
		t.Parallel()

		linker := createTestInvocationLinker()
		parentScope := NewSymbolTable(nil)
		parentScope.Define(Symbol{Name: "globalVar", CodeGenVarName: "globalVar"})
		childScope := NewSymbolTable(parentScope)
		childScope.Define(Symbol{Name: "item", CodeGenVarName: "item"})
		linker.invokerCtx.Symbols = childScope

		result := linker.isLoopVariable("item")

		assert.True(t, result)
	})
}

func TestInvocationLinker_IsExpressionLoopDependent(t *testing.T) {
	t.Parallel()

	t.Run("non-loop-dependent expression returns false", func(t *testing.T) {
		t.Parallel()

		linker := createTestInvocationLinker()
		expression := &ast_domain.Identifier{Name: "globalVar"}

		result := linker.isExpressionLoopDependent(expression, "testProp")

		assert.False(t, result)
	})

	t.Run("loop-dependent identifier returns true", func(t *testing.T) {
		t.Parallel()

		linker := createTestInvocationLinker()
		parentScope := NewSymbolTable(nil)
		parentScope.Define(Symbol{Name: "state", CodeGenVarName: "state"})
		childScope := NewSymbolTable(parentScope)
		childScope.Define(Symbol{Name: "item", CodeGenVarName: "item"})
		linker.invokerCtx.Symbols = childScope

		expression := &ast_domain.Identifier{Name: "item"}

		result := linker.isExpressionLoopDependent(expression, "testProp")

		assert.True(t, result)
	})

	t.Run("nested loop-dependent expression returns true", func(t *testing.T) {
		t.Parallel()

		linker := createTestInvocationLinker()
		parentScope := NewSymbolTable(nil)
		parentScope.Define(Symbol{Name: "state", CodeGenVarName: "state"})
		childScope := NewSymbolTable(parentScope)
		childScope.Define(Symbol{Name: "item", CodeGenVarName: "item"})
		linker.invokerCtx.Symbols = childScope

		expression := &ast_domain.MemberExpression{
			Base:     &ast_domain.Identifier{Name: "item"},
			Property: &ast_domain.Identifier{Name: "Title"},
		}

		result := linker.isExpressionLoopDependent(expression, "testProp")

		assert.True(t, result)
	})

	t.Run("string literal is not loop dependent", func(t *testing.T) {
		t.Parallel()

		linker := createTestInvocationLinker()
		expression := &ast_domain.StringLiteral{Value: "hello"}

		result := linker.isExpressionLoopDependent(expression, "testProp")

		assert.False(t, result)
	})
}

func TestInvocationLinker_ProcessStandardProps(t *testing.T) {
	t.Parallel()

	t.Run("valid prop is stored in canonicalProps via origin tracking", func(t *testing.T) {
		t.Parallel()

		linker := createTestInvocationLinker()
		linker.validProps = map[string]validPropInfo{
			"title": {
				GoFieldName:     "Title",
				DestinationType: goast.NewIdent("string"),
			},
		}

		standardProps := map[string]ast_domain.PropValue{
			"title": {
				Expression: &ast_domain.StringLiteral{Value: "Hello"},
				Location:   ast_domain.Location{Line: 1, Column: 1},
			},
		}

		linker.processStandardProps(context.Background(), standardProps)

		assert.Contains(t, linker.providedPropOrigins, "title")
		assert.Equal(t, "title", linker.providedPropOrigins["title"].fullName)
	})

	t.Run("unknown prop is tracked in origins but not stored", func(t *testing.T) {
		t.Parallel()

		linker := createTestInvocationLinker()
		linker.validProps = map[string]validPropInfo{}

		standardProps := map[string]ast_domain.PropValue{
			"unknownProp": {
				Expression: &ast_domain.StringLiteral{Value: "something"},
				Location:   ast_domain.Location{Line: 2, Column: 3},
			},
		}

		linker.processStandardProps(context.Background(), standardProps)

		assert.Contains(t, linker.providedPropOrigins, "unknownProp")
		assert.NotContains(t, linker.canonicalProps, "unknownProp")
	})

	t.Run("processes multiple props in sorted order", func(t *testing.T) {
		t.Parallel()

		linker := createTestInvocationLinker()
		linker.validProps = map[string]validPropInfo{
			"alpha": {GoFieldName: "Alpha", DestinationType: goast.NewIdent("string")},
			"beta":  {GoFieldName: "Beta", DestinationType: goast.NewIdent("string")},
		}

		standardProps := map[string]ast_domain.PropValue{
			"beta":  {Expression: &ast_domain.StringLiteral{Value: "B"}, Location: ast_domain.Location{Line: 1}},
			"alpha": {Expression: &ast_domain.StringLiteral{Value: "A"}, Location: ast_domain.Location{Line: 2}},
		}

		linker.processStandardProps(context.Background(), standardProps)

		assert.Contains(t, linker.providedPropOrigins, "alpha")
		assert.Contains(t, linker.providedPropOrigins, "beta")
	})
}

func TestInvocationLinker_ProcessServerProps(t *testing.T) {
	t.Parallel()

	t.Run("unknown server prop emits diagnostic", func(t *testing.T) {
		t.Parallel()

		linker := createTestInvocationLinker()
		linker.validProps = map[string]validPropInfo{}

		serverProps := map[string]ast_domain.PropValue{
			"server.unknownProp": {
				Expression:   &ast_domain.StringLiteral{Value: "data"},
				Location:     ast_domain.Location{Line: 1},
				NameLocation: ast_domain.Location{Line: 1, Column: 5},
			},
		}

		linker.processServerProps(context.Background(), serverProps)

		assert.NotEmpty(t, *linker.invokerCtx.Diagnostics)
	})

	t.Run("server prop overrides standard prop with warning", func(t *testing.T) {
		t.Parallel()

		linker := createTestInvocationLinker()
		linker.validProps = map[string]validPropInfo{
			"title": {GoFieldName: "Title", DestinationType: goast.NewIdent("string")},
		}
		linker.providedPropOrigins["title"] = propOrigin{fullName: "title", location: ast_domain.Location{Line: 1}}

		serverProps := map[string]ast_domain.PropValue{
			"server.title": {
				Expression:   &ast_domain.StringLiteral{Value: "Server Title"},
				Location:     ast_domain.Location{Line: 2},
				NameLocation: ast_domain.Location{Line: 2, Column: 1},
			},
		}

		linker.processServerProps(context.Background(), serverProps)

		assert.NotEmpty(t, *linker.invokerCtx.Diagnostics)
		assert.Contains(t, linker.providedPropOrigins, "title")
		assert.Equal(t, "server.title", linker.providedPropOrigins["title"].fullName)
	})

	t.Run("server prop with suggestion in diagnostic", func(t *testing.T) {
		t.Parallel()

		linker := createTestInvocationLinker()
		linker.validProps = map[string]validPropInfo{
			"title": {GoFieldName: "Title", DestinationType: goast.NewIdent("string")},
		}

		serverProps := map[string]ast_domain.PropValue{
			"server.titl": {
				Expression:   &ast_domain.StringLiteral{Value: "data"},
				Location:     ast_domain.Location{Line: 1},
				NameLocation: ast_domain.Location{Line: 1, Column: 1},
			},
		}

		linker.processServerProps(context.Background(), serverProps)

		require.NotEmpty(t, *linker.invokerCtx.Diagnostics)
	})
}

func TestInvocationLinker_ProcessProvidedProps(t *testing.T) {
	t.Parallel()

	t.Run("categorises and processes standard and server props", func(t *testing.T) {
		t.Parallel()

		linker := createTestInvocationLinker()
		linker.validProps = map[string]validPropInfo{
			"title": {GoFieldName: "Title", DestinationType: goast.NewIdent("string")},
		}
		linker.invocation.PassedProps = map[string]ast_domain.PropValue{
			"title": {
				Expression: &ast_domain.StringLiteral{Value: "Hello"},
				Location:   ast_domain.Location{Line: 1},
			},
		}

		linker.processProvidedProps(context.Background())

		assert.Contains(t, linker.providedPropOrigins, "title")
	})
}

func TestInvocationLinker_ProcessOmittedProps(t *testing.T) {
	t.Parallel()

	t.Run("required prop emits error diagnostic", func(t *testing.T) {
		t.Parallel()

		linker := createTestInvocationLinker()
		linker.validProps = map[string]validPropInfo{
			"title": {
				GoFieldName:     "Title",
				DestinationType: goast.NewIdent("string"),
				IsRequired:      true,
			},
		}

		linker.processOmittedProps(context.Background())

		require.NotEmpty(t, *linker.invokerCtx.Diagnostics)
		assert.Contains(t, (*linker.invokerCtx.Diagnostics)[0].Message, "Missing required prop 'title'")
	})

	t.Run("provided prop is skipped", func(t *testing.T) {
		t.Parallel()

		linker := createTestInvocationLinker()
		linker.validProps = map[string]validPropInfo{
			"title": {
				GoFieldName:     "Title",
				DestinationType: goast.NewIdent("string"),
				IsRequired:      true,
			},
		}
		linker.providedPropOrigins["title"] = propOrigin{fullName: "title"}

		linker.processOmittedProps(context.Background())

		assert.Empty(t, *linker.invokerCtx.Diagnostics)
	})

	t.Run("optional prop without default is ignored", func(t *testing.T) {
		t.Parallel()

		linker := createTestInvocationLinker()
		linker.validProps = map[string]validPropInfo{
			"title": {
				GoFieldName:     "Title",
				DestinationType: goast.NewIdent("string"),
				IsRequired:      false,
			},
		}

		linker.processOmittedProps(context.Background())

		assert.Empty(t, *linker.invokerCtx.Diagnostics)
		assert.NotContains(t, linker.canonicalProps, "title")
	})

	t.Run("prop with literal default value is stored", func(t *testing.T) {
		t.Parallel()

		linker := createTestInvocationLinker()
		linker.validProps = map[string]validPropInfo{
			"count": {
				GoFieldName:     "Count",
				DestinationType: goast.NewIdent("int"),
				DefaultValue:    new("42"),
			},
		}

		linker.processOmittedProps(context.Background())

		assert.Contains(t, linker.canonicalProps, "count")
	})
}

func TestInvocationLinker_HandleLiteralDefault_AdditionalCases(t *testing.T) {
	t.Parallel()

	t.Run("string default for string type stores correctly", func(t *testing.T) {
		t.Parallel()

		linker := createTestInvocationLinker()
		propInfo := validPropInfo{
			GoFieldName:     "Label",
			DestinationType: goast.NewIdent("string"),
			DefaultValue:    new("hello world"),
		}

		linker.handleLiteralDefault(context.Background(), "label", propInfo)

		require.Contains(t, linker.canonicalProps, "label")
		strLit, ok := linker.canonicalProps["label"].Expression.(*ast_domain.StringLiteral)
		require.True(t, ok)
		assert.Equal(t, "hello world", strLit.Value)
	})

	t.Run("boolean default stores correctly", func(t *testing.T) {
		t.Parallel()

		linker := createTestInvocationLinker()
		propInfo := validPropInfo{
			GoFieldName:     "Enabled",
			DestinationType: goast.NewIdent("bool"),
			DefaultValue:    new("true"),
		}

		linker.handleLiteralDefault(context.Background(), "enabled", propInfo)

		require.Contains(t, linker.canonicalProps, "enabled")
		boolLit, ok := linker.canonicalProps["enabled"].Expression.(*ast_domain.BooleanLiteral)
		require.True(t, ok)
		assert.True(t, boolLit.Value)
	})

	t.Run("string default that cannot coerce to int emits warning", func(t *testing.T) {
		t.Parallel()

		linker := createTestInvocationLinker()
		propInfo := validPropInfo{
			GoFieldName:     "Count",
			DestinationType: goast.NewIdent("int"),
			DefaultValue:    new("notAnumber"),
		}

		linker.handleLiteralDefault(context.Background(), "count", propInfo)

		require.NotEmpty(t, *linker.invokerCtx.Diagnostics)
		assert.Contains(t, (*linker.invokerCtx.Diagnostics)[0].Message, "Invalid default value")
	})

	t.Run("float default stores correctly", func(t *testing.T) {
		t.Parallel()

		linker := createTestInvocationLinker()
		propInfo := validPropInfo{
			GoFieldName:     "Rate",
			DestinationType: goast.NewIdent("float64"),
			DefaultValue:    new("3.14"),
		}

		linker.handleLiteralDefault(context.Background(), "rate", propInfo)

		require.Contains(t, linker.canonicalProps, "rate")
		floatLit, ok := linker.canonicalProps["rate"].Expression.(*ast_domain.FloatLiteral)
		require.True(t, ok)
		assert.InDelta(t, 3.14, floatLit.Value, 0.001)
	})
}

func TestInvocationLinker_TryDirectAssignment(t *testing.T) {
	t.Parallel()

	t.Run("returns false when source annotation is not type checkable", func(t *testing.T) {
		t.Parallel()

		linker := createTestInvocationLinker()
		params := &propAssignmentParams{
			PropName:         "title",
			SourceExpression: &ast_domain.StringLiteral{Value: "hello"},
			SourceAnnotation: nil,
			DestTypeInfo:     &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("string")},
			PropInfo:         validPropInfo{GoFieldName: "Title"},
		}

		result := linker.tryDirectAssignment(params)

		assert.False(t, result)
	})

	t.Run("assigns when types match directly", func(t *testing.T) {
		t.Parallel()

		linker := createTestInvocationLinker()
		params := &propAssignmentParams{
			PropName:         "title",
			SourceExpression: &ast_domain.StringLiteral{Value: "hello"},
			SourceAnnotation: &ast_domain.GoGeneratorAnnotation{
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression: goast.NewIdent("string"),
				},
			},
			DestTypeInfo: &ast_domain.ResolvedTypeInfo{
				TypeExpression: goast.NewIdent("string"),
			},
			PropInfo: validPropInfo{GoFieldName: "Title"},
		}

		result := linker.tryDirectAssignment(params)

		assert.True(t, result)
		assert.Contains(t, linker.canonicalProps, "title")
	})

	t.Run("assigns pointer-to-type for optional props", func(t *testing.T) {
		t.Parallel()

		linker := createTestInvocationLinker()
		params := &propAssignmentParams{
			PropName:         "title",
			SourceExpression: &ast_domain.StringLiteral{Value: "hello"},
			SourceAnnotation: &ast_domain.GoGeneratorAnnotation{
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression: goast.NewIdent("string"),
				},
			},
			DestTypeInfo: &ast_domain.ResolvedTypeInfo{
				TypeExpression: &goast.StarExpr{X: goast.NewIdent("string")},
			},
			PropInfo: validPropInfo{GoFieldName: "Title"},
		}

		result := linker.tryDirectAssignment(params)

		assert.True(t, result)
		assert.Contains(t, linker.canonicalProps, "title")
	})

	t.Run("returns false when types do not match", func(t *testing.T) {
		t.Parallel()

		linker := createTestInvocationLinker()
		params := &propAssignmentParams{
			PropName:         "count",
			SourceExpression: &ast_domain.StringLiteral{Value: "hello"},
			SourceAnnotation: &ast_domain.GoGeneratorAnnotation{
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression: goast.NewIdent("string"),
				},
			},
			DestTypeInfo: &ast_domain.ResolvedTypeInfo{
				TypeExpression: goast.NewIdent("int"),
			},
			PropInfo: validPropInfo{GoFieldName: "Count"},
		}

		result := linker.tryDirectAssignment(params)

		assert.False(t, result)
	})
}

func TestInvocationLinker_TryCoercionAssignment(t *testing.T) {
	t.Parallel()

	t.Run("returns false when coerce is not enabled", func(t *testing.T) {
		t.Parallel()

		linker := createTestInvocationLinker()
		params := &propAssignmentParams{
			PropName:         "count",
			SourceExpression: &ast_domain.StringLiteral{Value: "42"},
			SourceAnnotation: &ast_domain.GoGeneratorAnnotation{
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression: goast.NewIdent("string"),
				},
			},
			DestTypeInfo: &ast_domain.ResolvedTypeInfo{
				TypeExpression: goast.NewIdent("int"),
			},
			PropInfo: validPropInfo{GoFieldName: "Count", ShouldCoerce: false},
		}

		result := linker.tryCoercionAssignment(context.Background(), params)

		assert.False(t, result)
	})

	t.Run("coerces string literal to int when coerce is enabled", func(t *testing.T) {
		t.Parallel()

		linker := createTestInvocationLinker()
		params := &propAssignmentParams{
			PropName:         "count",
			SourceExpression: &ast_domain.StringLiteral{Value: "42"},
			SourceAnnotation: &ast_domain.GoGeneratorAnnotation{
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression: goast.NewIdent("string"),
				},
			},
			DestTypeInfo: &ast_domain.ResolvedTypeInfo{
				TypeExpression: goast.NewIdent("int"),
			},
			PropInfo: validPropInfo{GoFieldName: "Count", ShouldCoerce: true},
		}

		result := linker.tryCoercionAssignment(context.Background(), params)

		assert.True(t, result)
		assert.Contains(t, linker.canonicalProps, "count")
	})
}

func TestInvocationLinker_TryCoerceToString(t *testing.T) {
	t.Parallel()

	t.Run("returns false when stringability is none", func(t *testing.T) {
		t.Parallel()

		linker := createTestInvocationLinker()
		sourceExpression := &ast_domain.Identifier{Name: "myVal"}
		sourceAnn := &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: goast.NewIdent("CustomType"),
			},
			Stringability: int(inspector_dto.StringableNone),
		}

		_, _, coerced := linker.tryCoerceToString(sourceExpression, sourceAnn, "testProp")

		assert.False(t, coerced)
	})

	t.Run("returns false when source is already a string", func(t *testing.T) {
		t.Parallel()

		linker := createTestInvocationLinker()
		sourceExpression := &ast_domain.Identifier{Name: "myString"}
		sourceAnn := &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: goast.NewIdent("string"),
			},
			Stringability: int(inspector_dto.StringablePrimitive),
		}

		_, _, coerced := linker.tryCoerceToString(sourceExpression, sourceAnn, "testProp")

		assert.False(t, coerced)
	})

	t.Run("coerces int to string", func(t *testing.T) {
		t.Parallel()

		linker := createTestInvocationLinker()
		sourceExpression := &ast_domain.Identifier{Name: "myInt"}
		sourceAnn := &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: goast.NewIdent("int"),
			},
			Stringability: int(inspector_dto.StringablePrimitive),
		}

		resultExpr, resultAnn, coerced := linker.tryCoerceToString(sourceExpression, sourceAnn, "testProp")

		assert.True(t, coerced)
		assert.NotNil(t, resultExpr)
		assert.NotNil(t, resultAnn)

		_, isCall := resultExpr.(*ast_domain.CallExpression)
		assert.True(t, isCall, "expected CallExpr for int-to-string conversion")
	})

	t.Run("coerces via stringer method", func(t *testing.T) {
		t.Parallel()

		linker := createTestInvocationLinker()
		sourceExpression := &ast_domain.Identifier{Name: "myObj"}
		sourceAnn := &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: goast.NewIdent("MyType"),
				PackageAlias:   "pkg",
			},
			Stringability: int(inspector_dto.StringableViaStringer),
		}

		resultExpr, resultAnn, coerced := linker.tryCoerceToString(sourceExpression, sourceAnn, "testProp")

		assert.True(t, coerced)
		assert.NotNil(t, resultExpr)
		assert.NotNil(t, resultAnn)
	})
}

func TestInvocationLinker_BuildStringConversionAST(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		typeExprName  string
		expectFunc    string
		stringability inspector_dto.StringabilityMethod
	}{
		{
			name:          "int type uses FormatInt",
			typeExprName:  "int",
			stringability: inspector_dto.StringablePrimitive,
			expectFunc:    "FormatInt",
		},
		{
			name:          "int64 type uses FormatInt",
			typeExprName:  "int64",
			stringability: inspector_dto.StringablePrimitive,
			expectFunc:    "FormatInt",
		},
		{
			name:          "uint type uses FormatUint",
			typeExprName:  "uint",
			stringability: inspector_dto.StringablePrimitive,
			expectFunc:    "FormatUint",
		},
		{
			name:          "byte type uses FormatUint",
			typeExprName:  "byte",
			stringability: inspector_dto.StringablePrimitive,
			expectFunc:    "FormatUint",
		},
		{
			name:          "float64 type uses FormatFloat",
			typeExprName:  "float64",
			stringability: inspector_dto.StringablePrimitive,
			expectFunc:    "FormatFloat",
		},
		{
			name:          "float32 type uses FormatFloat",
			typeExprName:  "float32",
			stringability: inspector_dto.StringablePrimitive,
			expectFunc:    "FormatFloat",
		},
		{
			name:          "bool type uses FormatBool",
			typeExprName:  "bool",
			stringability: inspector_dto.StringablePrimitive,
			expectFunc:    "FormatBool",
		},
		{
			name:          "rune type uses string cast",
			typeExprName:  "rune",
			stringability: inspector_dto.StringablePrimitive,
			expectFunc:    "string",
		},
		{
			name:          "stringer type uses String method",
			typeExprName:  "MyType",
			stringability: inspector_dto.StringableViaStringer,
			expectFunc:    "String",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			linker := createTestInvocationLinker()
			sourceExpression := &ast_domain.Identifier{Name: "myVal"}
			ann := &ast_domain.GoGeneratorAnnotation{
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression: goast.NewIdent(tc.typeExprName),
				},
				Stringability: int(tc.stringability),
			}

			result := linker.buildStringConversionAST(sourceExpression, ann)

			require.NotNil(t, result)
			callExpr, isCall := result.(*ast_domain.CallExpression)
			require.True(t, isCall, "expected CallExpr, got %T", result)

			switch callee := callExpr.Callee.(type) {
			case *ast_domain.MemberExpression:
				propIdent, ok := callee.Property.(*ast_domain.Identifier)
				require.True(t, ok)
				assert.Equal(t, tc.expectFunc, propIdent.Name)
			case *ast_domain.Identifier:
				assert.Equal(t, tc.expectFunc, callee.Name)
			default:
				t.Fatalf("unexpected callee type: %T", callExpr.Callee)
			}
		})
	}

	t.Run("unknown primitive falls back to runtime", func(t *testing.T) {
		t.Parallel()

		linker := createTestInvocationLinker()
		sourceExpression := &ast_domain.Identifier{Name: "myVal"}
		ann := &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: goast.NewIdent("complex128"),
			},
			Stringability: int(inspector_dto.StringablePrimitive),
		}

		result := linker.buildStringConversionAST(sourceExpression, ann)

		require.NotNil(t, result)
		callExpr, isCall := result.(*ast_domain.CallExpression)
		require.True(t, isCall)
		memberExpr, isMember := callExpr.Callee.(*ast_domain.MemberExpression)
		require.True(t, isMember)
		propIdent, ok := memberExpr.Property.(*ast_domain.Identifier)
		require.True(t, ok)
		assert.Equal(t, "ValueToString", propIdent.Name)
	})

	t.Run("unknown stringability falls back to runtime", func(t *testing.T) {
		t.Parallel()

		linker := createTestInvocationLinker()
		sourceExpression := &ast_domain.Identifier{Name: "myVal"}
		ann := &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: goast.NewIdent("SomeType"),
				PackageAlias:   "pkg",
			},
			Stringability: 999,
		}

		result := linker.buildStringConversionAST(sourceExpression, ann)

		require.NotNil(t, result)
		callExpr, isCall := result.(*ast_domain.CallExpression)
		require.True(t, isCall)
		memberExpr, isMember := callExpr.Callee.(*ast_domain.MemberExpression)
		require.True(t, isMember)
		propIdent, ok := memberExpr.Property.(*ast_domain.Identifier)
		require.True(t, ok)
		assert.Equal(t, "ValueToString", propIdent.Name)
	})
}

func TestInvocationLinker_ReportTypeMismatch(t *testing.T) {
	t.Parallel()

	t.Run("emits error diagnostic for type mismatch", func(t *testing.T) {
		t.Parallel()

		linker := createTestInvocationLinker()
		params := &propAssignmentParams{
			PropName:         "count",
			SourceExpression: &ast_domain.StringLiteral{Value: "hello"},
			SourceAnnotation: &ast_domain.GoGeneratorAnnotation{
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression: goast.NewIdent("string"),
				},
			},
			DestTypeInfo: &ast_domain.ResolvedTypeInfo{
				TypeExpression: goast.NewIdent("int"),
			},
			PropInfo: validPropInfo{GoFieldName: "Count", ShouldCoerce: false},
			Loc:      ast_domain.Location{Line: 5, Column: 10},
		}

		linker.reportTypeMismatch(context.Background(), params)

		require.NotEmpty(t, *linker.invokerCtx.Diagnostics)
		assert.Contains(t, (*linker.invokerCtx.Diagnostics)[0].Message, "Type mismatch")
	})

	t.Run("suggests coerce tag when coercion would succeed", func(t *testing.T) {
		t.Parallel()

		linker := createTestInvocationLinker()
		params := &propAssignmentParams{
			PropName:         "count",
			SourceExpression: &ast_domain.StringLiteral{Value: "42"},
			SourceAnnotation: &ast_domain.GoGeneratorAnnotation{
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression: goast.NewIdent("string"),
				},
			},
			DestTypeInfo: &ast_domain.ResolvedTypeInfo{
				TypeExpression: goast.NewIdent("int"),
			},
			PropInfo: validPropInfo{GoFieldName: "Count", ShouldCoerce: false},
			Loc:      ast_domain.Location{Line: 5, Column: 10},
		}

		linker.reportTypeMismatch(context.Background(), params)

		require.NotEmpty(t, *linker.invokerCtx.Diagnostics)
		assert.Contains(t, (*linker.invokerCtx.Diagnostics)[0].Message, "coerce")
	})
}

func TestInvocationLinker_GetFinalExprAfterCoercion_AdditionalCases(t *testing.T) {
	t.Parallel()

	t.Run("coercion that fails emits warning for string literal", func(t *testing.T) {
		t.Parallel()

		linker := createTestInvocationLinker()
		sourceExpression := &ast_domain.StringLiteral{Value: "not_a_number"}
		propInfo := validPropInfo{
			GoFieldName:     "Count",
			DestinationType: goast.NewIdent("int"),
			ShouldCoerce:    true,
		}

		result := linker.getFinalExprAfterCoercion("count", sourceExpression, ast_domain.Location{}, propInfo)

		assert.Equal(t, sourceExpression, result, "expected original expression when coercion fails")
		require.NotEmpty(t, *linker.invokerCtx.Diagnostics)
		assert.Contains(t, (*linker.invokerCtx.Diagnostics)[0].Message, "Could not coerce")
	})

	t.Run("successful coercion returns new expression", func(t *testing.T) {
		t.Parallel()

		linker := createTestInvocationLinker()
		sourceExpression := &ast_domain.StringLiteral{Value: "42"}
		propInfo := validPropInfo{
			GoFieldName:     "Count",
			DestinationType: goast.NewIdent("int"),
			ShouldCoerce:    true,
		}

		result := linker.getFinalExprAfterCoercion("count", sourceExpression, ast_domain.Location{}, propInfo)

		assert.NotEqual(t, sourceExpression, result, "expected new expression after coercion")
		_, isInt := result.(*ast_domain.IntegerLiteral)
		assert.True(t, isInt, "expected IntegerLiteral after coercion to int")
	})

	t.Run("string literal coercion to string type returns original", func(t *testing.T) {
		t.Parallel()

		linker := createTestInvocationLinker()
		sourceExpression := &ast_domain.StringLiteral{Value: "hello"}
		propInfo := validPropInfo{
			GoFieldName:     "Label",
			DestinationType: goast.NewIdent("string"),
			ShouldCoerce:    true,
		}

		result := linker.getFinalExprAfterCoercion("label", sourceExpression, ast_domain.Location{}, propInfo)

		assert.Equal(t, sourceExpression, result, "expected original expression for string-to-string")
		assert.Empty(t, *linker.invokerCtx.Diagnostics)
	})
}

func TestInvocationLinker_ApplyRequestOverrides_AdditionalCases(t *testing.T) {
	t.Parallel()

	t.Run("unknown override with close match suggests alternative", func(t *testing.T) {
		t.Parallel()

		linker := createTestInvocationLinker()
		linker.validProps = map[string]validPropInfo{
			"title": {GoFieldName: "Title", DestinationType: goast.NewIdent("string")},
		}
		linker.invocation.RequestOverrides = map[string]ast_domain.PropValue{
			"titl": {
				Expression:   &ast_domain.StringLiteral{Value: "value"},
				Location:     ast_domain.Location{Line: 1},
				NameLocation: ast_domain.Location{Line: 1, Column: 1},
			},
		}

		linker.applyRequestOverrides(context.Background())

		require.NotEmpty(t, *linker.invokerCtx.Diagnostics)
		assert.Contains(t, (*linker.invokerCtx.Diagnostics)[0].Message, "Did you mean")
	})
}

func TestCollectDependenciesFromExpression(t *testing.T) {
	t.Parallel()

	t.Run("nil expression does not panic", func(t *testing.T) {
		t.Parallel()

		seen := make(map[string]struct{})
		collectDependenciesFromExpression(nil, seen)
		assert.Empty(t, seen)
	})

	t.Run("expression without annotations collects nothing", func(t *testing.T) {
		t.Parallel()

		seen := make(map[string]struct{})
		expression := &ast_domain.StringLiteral{Value: "hello"}
		collectDependenciesFromExpression(expression, seen)
		assert.Empty(t, seen)
	})

	t.Run("expression with source invocation key collects key", func(t *testing.T) {
		t.Parallel()

		seen := make(map[string]struct{})
		expression := &ast_domain.Identifier{
			Name: "data",
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				SourceInvocationKey: new("inv_abc"),
			},
		}
		collectDependenciesFromExpression(expression, seen)
		assert.Contains(t, seen, "inv_abc")
	})

	t.Run("deduplicates keys", func(t *testing.T) {
		t.Parallel()

		seen := make(map[string]struct{})
		expression := &ast_domain.Identifier{
			Name: "data",
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				SourceInvocationKey: new("inv_abc"),
			},
		}
		collectDependenciesFromExpression(expression, seen)
		collectDependenciesFromExpression(expression, seen)
		assert.Len(t, seen, 1)
	})
}

func TestCollectDependenciesFromProps_Sorting(t *testing.T) {
	t.Parallel()

	t.Run("returns dependencies in sorted order", func(t *testing.T) {
		t.Parallel()

		props := map[string]ast_domain.PropValue{
			"propC": {
				Expression: &ast_domain.Identifier{
					Name: "c",
					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
						SourceInvocationKey: new("inv_c"),
					},
				},
			},
			"propA": {
				Expression: &ast_domain.Identifier{
					Name: "a",
					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
						SourceInvocationKey: new("inv_a"),
					},
				},
			},
			"propB": {
				Expression: &ast_domain.Identifier{
					Name: "b",
					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
						SourceInvocationKey: new("inv_b"),
					},
				},
			},
		}

		result := collectDependenciesFromProps(props)

		require.Len(t, result, 3)
		assert.Equal(t, "inv_a", result[0])
		assert.Equal(t, "inv_b", result[1])
		assert.Equal(t, "inv_c", result[2])
	})
}

func TestInvocationLinker_ResolveAndStoreProp(t *testing.T) {
	t.Parallel()

	t.Run("stores prop when type is directly assignable", func(t *testing.T) {
		t.Parallel()

		linker := createTestInvocationLinker()
		sourceExpression := &ast_domain.StringLiteral{Value: "hello"}
		propInfo := validPropInfo{
			GoFieldName:     "Title",
			DestinationType: goast.NewIdent("string"),
		}

		linker.resolveAndStoreProp(context.Background(), "title", sourceExpression, ast_domain.Location{Line: 1}, ast_domain.Location{Line: 1}, propInfo)

		assert.Contains(t, linker.canonicalProps, "title")
	})

	t.Run("stores prop via coercion when coerce is enabled and types mismatch", func(t *testing.T) {
		t.Parallel()

		linker := createTestInvocationLinker()
		sourceExpression := &ast_domain.StringLiteral{Value: "42"}
		propInfo := validPropInfo{
			GoFieldName:     "Count",
			DestinationType: goast.NewIdent("int"),
			ShouldCoerce:    true,
		}

		linker.resolveAndStoreProp(context.Background(), "count", sourceExpression, ast_domain.Location{Line: 2}, ast_domain.Location{Line: 2}, propInfo)

		require.Contains(t, linker.canonicalProps, "count")
		_, isInt := linker.canonicalProps["count"].Expression.(*ast_domain.IntegerLiteral)
		assert.True(t, isInt, "expected IntegerLiteral after coercion")
	})

	t.Run("reports type mismatch when types are incompatible and no coercion", func(t *testing.T) {
		t.Parallel()

		linker := createTestInvocationLinkerWithResolvedType("string")

		sourceExpression := &ast_domain.Identifier{Name: "myVar"}
		propInfo := validPropInfo{
			GoFieldName:     "Count",
			DestinationType: goast.NewIdent("int"),
			ShouldCoerce:    false,
		}

		linker.resolveAndStoreProp(context.Background(), "count", sourceExpression, ast_domain.Location{Line: 3}, ast_domain.Location{Line: 3}, propInfo)

		require.NotEmpty(t, *linker.invokerCtx.Diagnostics)
		assert.Contains(t, (*linker.invokerCtx.Diagnostics)[0].Message, "Type mismatch")
	})

	t.Run("stores prop with any type when variable is undefined", func(t *testing.T) {
		t.Parallel()

		linker := createTestInvocationLinkerWithNilResolve()
		sourceExpression := &ast_domain.Identifier{Name: "unknownVar"}
		propInfo := validPropInfo{
			GoFieldName:     "Data",
			DestinationType: goast.NewIdent("any"),
			ShouldCoerce:    false,
		}

		linker.resolveAndStoreProp(context.Background(), "data", sourceExpression, ast_domain.Location{Line: 4}, ast_domain.Location{Line: 4}, propInfo)

		require.Contains(t, linker.canonicalProps, "data")
	})

	t.Run("stores prop when source annotation resolves to any", func(t *testing.T) {
		t.Parallel()

		linker := createTestInvocationLinkerWithResolvedType("any")

		sourceExpression := &ast_domain.Identifier{Name: "myVar"}
		propInfo := validPropInfo{
			GoFieldName:     "Title",
			DestinationType: goast.NewIdent("string"),
			ShouldCoerce:    false,
		}

		linker.resolveAndStoreProp(context.Background(), "title", sourceExpression, ast_domain.Location{Line: 5}, ast_domain.Location{Line: 5}, propInfo)

		require.Contains(t, linker.canonicalProps, "title")
	})

	t.Run("assigns optional pointer prop when source is base type", func(t *testing.T) {
		t.Parallel()

		linker := createTestInvocationLinkerWithResolvedType("string")

		sourceExpression := &ast_domain.Identifier{Name: "myVar"}
		propInfo := validPropInfo{
			GoFieldName:     "OptTitle",
			DestinationType: &goast.StarExpr{X: goast.NewIdent("string")},
			ShouldCoerce:    false,
		}

		linker.resolveAndStoreProp(context.Background(), "opttitle", sourceExpression, ast_domain.Location{Line: 6}, ast_domain.Location{Line: 6}, propInfo)

		require.Contains(t, linker.canonicalProps, "opttitle")
		_, isUnary := linker.canonicalProps["opttitle"].Expression.(*ast_domain.UnaryExpression)
		assert.True(t, isUnary, "expected UnaryExpr (address-of) for optional prop")
	})

	t.Run("detects loop-dependent expression and stores with flag", func(t *testing.T) {
		t.Parallel()

		linker := createTestInvocationLinker()
		parentScope := NewSymbolTable(nil)
		parentScope.Define(Symbol{Name: "state", CodeGenVarName: "state"})
		childScope := NewSymbolTable(parentScope)
		childScope.Define(Symbol{Name: "item", CodeGenVarName: "item"})
		linker.invokerCtx.Symbols = childScope

		sourceExpression := &ast_domain.MemberExpression{
			Base:     &ast_domain.Identifier{Name: "item"},
			Property: &ast_domain.Identifier{Name: "Title"},
		}
		propInfo := validPropInfo{
			GoFieldName:     "Title",
			DestinationType: goast.NewIdent("string"),
		}

		linker.resolveAndStoreProp(context.Background(), "title", sourceExpression, ast_domain.Location{Line: 7}, ast_domain.Location{Line: 7}, propInfo)

	})

	t.Run("type mismatch with coercion suggestion when coercion would help", func(t *testing.T) {
		t.Parallel()

		linker := createTestInvocationLinkerWithResolvedType("string")
		sourceExpression := &ast_domain.StringLiteral{Value: "42"}
		propInfo := validPropInfo{
			GoFieldName:     "Count",
			DestinationType: goast.NewIdent("int"),
			ShouldCoerce:    false,
		}

		linker.resolveAndStoreProp(context.Background(), "count", sourceExpression, ast_domain.Location{Line: 8}, ast_domain.Location{Line: 8}, propInfo)

		require.NotEmpty(t, *linker.invokerCtx.Diagnostics)
		assert.Contains(t, (*linker.invokerCtx.Diagnostics)[0].Message, "coerce")
	})
}

func TestInvocationLinker_HandleFactoryDefault(t *testing.T) {
	t.Parallel()

	t.Run("stores factory default when annotation is not type checkable", func(t *testing.T) {
		t.Parallel()

		linker := createTestInvocationLinkerWithVirtualModule()
		propInfo := validPropInfo{
			GoFieldName:     "Items",
			DestinationType: goast.NewIdent("[]string"),
			FactoryFuncName: "NewItems",
		}

		linker.handleFactoryDefault(context.Background(), "items", propInfo)

		require.Contains(t, linker.canonicalProps, "items")
		callExpr, isCall := linker.canonicalProps["items"].Expression.(*ast_domain.CallExpression)
		require.True(t, isCall, "expected CallExpr for factory default")
		memberExpr, isMember := callExpr.Callee.(*ast_domain.MemberExpression)
		require.True(t, isMember, "expected MemberExpr callee")
		propIdent, ok := memberExpr.Property.(*ast_domain.Identifier)
		require.True(t, ok)
		assert.Equal(t, "NewItems", propIdent.Name)
		assert.Equal(t, "Items", linker.canonicalProps["items"].GoFieldName)
		assert.False(t, linker.canonicalProps["items"].IsLoopDependent)
		assert.Nil(t, linker.canonicalProps["items"].InvokerAnnotation)
	})

	t.Run("stores factory default when type is assignable", func(t *testing.T) {
		t.Parallel()

		linker := createTestInvocationLinkerWithResolvedType("string")
		propInfo := validPropInfo{
			GoFieldName:     "Label",
			DestinationType: goast.NewIdent("string"),
			FactoryFuncName: "DefaultLabel",
		}

		linker.handleFactoryDefault(context.Background(), "label", propInfo)

		require.Contains(t, linker.canonicalProps, "label")
		assert.Equal(t, "Label", linker.canonicalProps["label"].GoFieldName)
	})

	t.Run("stores factory default when type resolution fails gracefully", func(t *testing.T) {
		t.Parallel()

		linker := createTestInvocationLinkerWithResolvedType("int")
		propInfo := validPropInfo{
			GoFieldName:     "Title",
			DestinationType: goast.NewIdent("string"),
			FactoryFuncName: "NewTitle",
		}

		linker.handleFactoryDefault(context.Background(), "title", propInfo)

		require.Contains(t, linker.canonicalProps, "title")
		assert.Equal(t, "Title", linker.canonicalProps["title"].GoFieldName)
	})

	t.Run("factory call expression uses partial package name", func(t *testing.T) {
		t.Parallel()

		linker := createTestInvocationLinkerWithVirtualModule()
		linker.partialVirtualComponent.RewrittenScriptAST.Name = goast.NewIdent("my_partial_pkg")
		propInfo := validPropInfo{
			GoFieldName:     "Config",
			DestinationType: goast.NewIdent("Config"),
			FactoryFuncName: "NewConfig",
		}

		linker.handleFactoryDefault(context.Background(), "config", propInfo)

		require.Contains(t, linker.canonicalProps, "config")
		callExpr, ok := linker.canonicalProps["config"].Expression.(*ast_domain.CallExpression)
		require.True(t, ok)
		memberExpr, ok := callExpr.Callee.(*ast_domain.MemberExpression)
		require.True(t, ok)
		baseIdent, ok := memberExpr.Base.(*ast_domain.Identifier)
		require.True(t, ok)
		assert.Equal(t, "my_partial_pkg", baseIdent.Name)
	})
}

func TestInvocationLinker_HandleLiteralDefault_InvalidDefault(t *testing.T) {
	t.Parallel()

	t.Run("emits error diagnostic when default value parse returns nil expression", func(t *testing.T) {
		t.Parallel()

		linker := createTestInvocationLinker()
		propInfo := validPropInfo{
			GoFieldName:     "Label",
			DestinationType: goast.NewIdent("string"),
			DefaultValue:    new("hello world"),
		}

		linker.handleLiteralDefault(context.Background(), "label", propInfo)

		require.Contains(t, linker.canonicalProps, "label")
	})

	t.Run("nil literal as default for non-nil type stores correctly", func(t *testing.T) {
		t.Parallel()

		linker := createTestInvocationLinker()
		propInfo := validPropInfo{
			GoFieldName:     "OptData",
			DestinationType: &goast.StarExpr{X: goast.NewIdent("string")},
			DefaultValue:    new("nil"),
		}

		linker.handleLiteralDefault(context.Background(), "optdata", propInfo)

		require.Contains(t, linker.canonicalProps, "optdata")
		_, isNil := linker.canonicalProps["optdata"].Expression.(*ast_domain.NilLiteral)
		assert.True(t, isNil, "expected NilLiteral for nil default")
	})

	t.Run("integer default for non-int type stores without warning", func(t *testing.T) {
		t.Parallel()

		linker := createTestInvocationLinker()
		propInfo := validPropInfo{
			GoFieldName:     "Size",
			DestinationType: goast.NewIdent("string"),
			DefaultValue:    new("100"),
		}

		linker.handleLiteralDefault(context.Background(), "size", propInfo)

		require.Contains(t, linker.canonicalProps, "size")
	})
}

func TestInvocationLinker_TryCoerce_DestinationString(t *testing.T) {
	t.Parallel()

	t.Run("coerces int to string when destination is string", func(t *testing.T) {
		t.Parallel()

		linker := createTestInvocationLinker()
		sourceExpression := &ast_domain.Identifier{Name: "myInt"}
		sourceAnn := &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: goast.NewIdent("int"),
			},
			Stringability: int(inspector_dto.StringablePrimitive),
		}
		destType := &ast_domain.ResolvedTypeInfo{
			TypeExpression: goast.NewIdent("string"),
		}

		resultExpr, resultAnn, coerced := linker.tryCoerce(context.Background(), sourceExpression, sourceAnn, destType, ast_domain.Location{}, "display")

		assert.True(t, coerced)
		assert.NotNil(t, resultExpr)
		assert.NotNil(t, resultAnn)
		_, isCall := resultExpr.(*ast_domain.CallExpression)
		assert.True(t, isCall, "expected CallExpr for int-to-string coercion")
	})

	t.Run("coerces bool to string when destination is string", func(t *testing.T) {
		t.Parallel()

		linker := createTestInvocationLinker()
		sourceExpression := &ast_domain.Identifier{Name: "myBool"}
		sourceAnn := &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: goast.NewIdent("bool"),
			},
			Stringability: int(inspector_dto.StringablePrimitive),
		}
		destType := &ast_domain.ResolvedTypeInfo{
			TypeExpression: goast.NewIdent("string"),
		}

		resultExpr, resultAnn, coerced := linker.tryCoerce(context.Background(), sourceExpression, sourceAnn, destType, ast_domain.Location{}, "flag")

		assert.True(t, coerced)
		assert.NotNil(t, resultExpr)
		assert.NotNil(t, resultAnn)
	})

	t.Run("coerces stringer type to string", func(t *testing.T) {
		t.Parallel()

		linker := createTestInvocationLinker()
		sourceExpression := &ast_domain.Identifier{Name: "myEnum"}
		sourceAnn := &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: goast.NewIdent("Status"),
				PackageAlias:   "pkg",
			},
			Stringability: int(inspector_dto.StringableViaStringer),
		}
		destType := &ast_domain.ResolvedTypeInfo{
			TypeExpression: goast.NewIdent("string"),
		}

		resultExpr, resultAnn, coerced := linker.tryCoerce(context.Background(), sourceExpression, sourceAnn, destType, ast_domain.Location{}, "status")

		assert.True(t, coerced)
		assert.NotNil(t, resultExpr)
		assert.NotNil(t, resultAnn)
	})

	t.Run("does not coerce when dest is string but source is already string", func(t *testing.T) {
		t.Parallel()

		linker := createTestInvocationLinker()
		sourceExpression := &ast_domain.Identifier{Name: "myString"}
		sourceAnn := &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: goast.NewIdent("string"),
			},
			Stringability: int(inspector_dto.StringablePrimitive),
		}
		destType := &ast_domain.ResolvedTypeInfo{
			TypeExpression: goast.NewIdent("string"),
		}

		_, _, coerced := linker.tryCoerce(context.Background(), sourceExpression, sourceAnn, destType, ast_domain.Location{}, "name")

		assert.False(t, coerced, "should not coerce when source is already string")
	})

	t.Run("does not coerce when dest is string but source has no stringability", func(t *testing.T) {
		t.Parallel()

		linker := createTestInvocationLinker()
		sourceExpression := &ast_domain.Identifier{Name: "myStruct"}
		sourceAnn := &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: goast.NewIdent("MyStruct"),
				PackageAlias:   "pkg",
			},
			Stringability: int(inspector_dto.StringableNone),
		}
		destType := &ast_domain.ResolvedTypeInfo{
			TypeExpression: goast.NewIdent("string"),
		}

		_, _, coerced := linker.tryCoerce(context.Background(), sourceExpression, sourceAnn, destType, ast_domain.Location{}, "data")

		assert.False(t, coerced)
	})

	t.Run("coerces string literal to bool when dest is not string", func(t *testing.T) {
		t.Parallel()

		linker := createTestInvocationLinker()
		sourceExpression := &ast_domain.StringLiteral{Value: "true"}
		sourceAnn := &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: goast.NewIdent("string"),
			},
			Stringability: int(inspector_dto.StringablePrimitive),
		}
		destType := &ast_domain.ResolvedTypeInfo{
			TypeExpression: goast.NewIdent("bool"),
		}

		resultExpr, resultAnn, coerced := linker.tryCoerce(context.Background(), sourceExpression, sourceAnn, destType, ast_domain.Location{}, "enabled")

		assert.True(t, coerced)
		boolLit, ok := resultExpr.(*ast_domain.BooleanLiteral)
		assert.True(t, ok, "expected BooleanLiteral after string-to-bool coercion")
		assert.True(t, boolLit.Value)
		assert.NotNil(t, resultAnn)
	})

	t.Run("coerces string literal to float64 when dest is float64", func(t *testing.T) {
		t.Parallel()

		linker := createTestInvocationLinker()
		sourceExpression := &ast_domain.StringLiteral{Value: "3.14"}
		sourceAnn := &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: goast.NewIdent("string"),
			},
		}
		destType := &ast_domain.ResolvedTypeInfo{
			TypeExpression: goast.NewIdent("float64"),
		}

		resultExpr, _, coerced := linker.tryCoerce(context.Background(), sourceExpression, sourceAnn, destType, ast_domain.Location{}, "rate")

		assert.True(t, coerced)
		floatLit, ok := resultExpr.(*ast_domain.FloatLiteral)
		assert.True(t, ok, "expected FloatLiteral after string-to-float coercion")
		assert.InDelta(t, 3.14, floatLit.Value, 0.001)
	})

	t.Run("string literal coercion to int takes priority over dest-string path", func(t *testing.T) {
		t.Parallel()

		linker := createTestInvocationLinker()
		sourceExpression := &ast_domain.StringLiteral{Value: "42"}
		sourceAnn := &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: goast.NewIdent("string"),
			},
			Stringability: int(inspector_dto.StringablePrimitive),
		}
		destType := &ast_domain.ResolvedTypeInfo{
			TypeExpression: goast.NewIdent("int"),
		}

		resultExpr, _, coerced := linker.tryCoerce(context.Background(), sourceExpression, sourceAnn, destType, ast_domain.Location{}, "count")

		assert.True(t, coerced)
		intLit, ok := resultExpr.(*ast_domain.IntegerLiteral)
		assert.True(t, ok, "expected IntegerLiteral after string-to-int coercion")
		assert.Equal(t, int64(42), intLit.Value)
	})
}

func TestInvocationLinker_TryCoerceStringLiteralToPrimitive(t *testing.T) {
	t.Parallel()

	t.Run("coerces string literal to int", func(t *testing.T) {
		t.Parallel()

		linker := createTestInvocationLinker()
		sourceExpression := &ast_domain.StringLiteral{Value: "99"}
		destType := &ast_domain.ResolvedTypeInfo{
			TypeExpression: goast.NewIdent("int"),
		}

		resultExpr, resultAnn, coerced := linker.tryCoerceStringLiteralToPrimitive(context.Background(), sourceExpression, destType, ast_domain.Location{})

		assert.True(t, coerced)
		intLit, ok := resultExpr.(*ast_domain.IntegerLiteral)
		require.True(t, ok)
		assert.Equal(t, int64(99), intLit.Value)
		assert.NotNil(t, resultAnn)
	})

	t.Run("returns false for non-string-literal expression", func(t *testing.T) {
		t.Parallel()

		linker := createTestInvocationLinker()
		sourceExpression := &ast_domain.IntegerLiteral{Value: 42}
		destType := &ast_domain.ResolvedTypeInfo{
			TypeExpression: goast.NewIdent("string"),
		}

		_, _, coerced := linker.tryCoerceStringLiteralToPrimitive(context.Background(), sourceExpression, destType, ast_domain.Location{})

		assert.False(t, coerced)
	})

	t.Run("returns false when string cannot be parsed as target type", func(t *testing.T) {
		t.Parallel()

		linker := createTestInvocationLinker()
		sourceExpression := &ast_domain.StringLiteral{Value: "not_a_number"}
		destType := &ast_domain.ResolvedTypeInfo{
			TypeExpression: goast.NewIdent("int"),
		}

		_, _, coerced := linker.tryCoerceStringLiteralToPrimitive(context.Background(), sourceExpression, destType, ast_domain.Location{})

		assert.False(t, coerced)
	})
}

func TestInvocationLinker_Process(t *testing.T) {
	t.Parallel()

	t.Run("returns error when invoker context is nil", func(t *testing.T) {
		t.Parallel()

		linker := createTestInvocationLinker()
		linker.invokerCtx = nil

		_, err := linker.process(context.Background())

		require.Error(t, err)
		assert.Contains(t, err.Error(), "nil invoker context")
	})
}

func createTestInvocationLinkerWithResolvedType(typeName string) *invocationLinker {
	ctx := &AnalysisContext{
		Symbols:                  NewSymbolTable(nil),
		Diagnostics:              new([]*ast_domain.Diagnostic),
		CurrentGoFullPackagePath: "test/invoker",
		CurrentGoPackageName:     "invoker",
		CurrentGoSourcePath:      "/test/invoker.go",
		SFCSourcePath:            "/test/invoker.piko",
		Logger:                   logger_domain.GetLogger("test"),
	}

	mockInspector := &inspector_domain.MockTypeQuerier{
		GetImportsForFileFunc: func(_, _ string) map[string]string {
			return map[string]string{}
		},
		GetAllPackagesFunc: func() map[string]*inspector_dto.Package {
			return map[string]*inspector_dto.Package{}
		},
	}

	partialComp := &annotator_dto.VirtualComponent{
		HashedName:             "partial_test",
		CanonicalGoPackagePath: "test/partial",
		VirtualGoFilePath:      "/virtual/partial.go",
		RewrittenScriptAST: &goast.File{
			Name: goast.NewIdent("partial_pkg"),
		},
		Source: &annotator_dto.ParsedComponent{
			SourcePath: "/test/partial.piko",
		},
	}

	invokerComp := &annotator_dto.VirtualComponent{
		HashedName:             "invoker_test",
		CanonicalGoPackagePath: "test/invoker",
		VirtualGoFilePath:      "/test/invoker.go",
		RewrittenScriptAST: &goast.File{
			Name: goast.NewIdent("invoker"),
		},
		Source: &annotator_dto.ParsedComponent{
			SourcePath: "/test/invoker.piko",
		},
	}

	vm := &annotator_dto.VirtualModule{
		ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
			"partial_test": partialComp,
			"invoker_test": invokerComp,
		},
		ComponentsByGoPath: map[string]*annotator_dto.VirtualComponent{
			"test/partial": partialComp,
			"test/invoker": invokerComp,
		},
	}

	resolver := &TypeResolver{inspector: mockInspector, virtualModule: vm}

	ctx.Symbols.Define(Symbol{
		Name:           "myVar",
		CodeGenVarName: "myVar",
		TypeInfo:       newSimpleTypeInfo(goast.NewIdent(typeName)),
	})

	return &invocationLinker{
		invocation: &annotator_dto.PartialInvocation{
			PartialAlias:     "testPartial",
			PassedProps:      make(map[string]ast_domain.PropValue),
			RequestOverrides: make(map[string]ast_domain.PropValue),
			Location:         ast_domain.Location{},
		},
		typeResolver:            resolver,
		virtualModule:           vm,
		invokerCtx:              ctx,
		partialVirtualComponent: partialComp,
		validProps:              make(map[string]validPropInfo),
		providedPropOrigins:     make(map[string]propOrigin),
		canonicalProps:          make(map[string]ast_domain.PropValue),
	}
}

func createTestInvocationLinkerWithVirtualModule() *invocationLinker {
	ctx := &AnalysisContext{
		Symbols:                  NewSymbolTable(nil),
		Diagnostics:              new([]*ast_domain.Diagnostic),
		CurrentGoFullPackagePath: "test/invoker",
		CurrentGoPackageName:     "invoker",
		CurrentGoSourcePath:      "/test/invoker.go",
		SFCSourcePath:            "/test/invoker.piko",
		Logger:                   logger_domain.GetLogger("test"),
	}

	mockInspector := &inspector_domain.MockTypeQuerier{
		GetImportsForFileFunc: func(_, _ string) map[string]string {
			return map[string]string{}
		},
		GetAllPackagesFunc: func() map[string]*inspector_dto.Package {
			return map[string]*inspector_dto.Package{}
		},
	}

	partialComp := &annotator_dto.VirtualComponent{
		HashedName:             "partial_test",
		CanonicalGoPackagePath: "test/partial",
		VirtualGoFilePath:      "/virtual/partial.go",
		RewrittenScriptAST: &goast.File{
			Name: goast.NewIdent("partial_pkg"),
		},
		Source: &annotator_dto.ParsedComponent{
			SourcePath: "/test/partial.piko",
		},
	}

	invokerComp := &annotator_dto.VirtualComponent{
		HashedName:             "invoker_test",
		CanonicalGoPackagePath: "test/invoker",
		VirtualGoFilePath:      "/test/invoker.go",
		RewrittenScriptAST: &goast.File{
			Name: goast.NewIdent("invoker"),
		},
		Source: &annotator_dto.ParsedComponent{
			SourcePath: "/test/invoker.piko",
		},
	}

	vm := &annotator_dto.VirtualModule{
		ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
			"partial_test": partialComp,
			"invoker_test": invokerComp,
		},
		ComponentsByGoPath: map[string]*annotator_dto.VirtualComponent{
			"test/partial": partialComp,
			"test/invoker": invokerComp,
		},
	}

	return &invocationLinker{
		invocation: &annotator_dto.PartialInvocation{
			PartialAlias:     "testPartial",
			PassedProps:      make(map[string]ast_domain.PropValue),
			RequestOverrides: make(map[string]ast_domain.PropValue),
			Location:         ast_domain.Location{},
		},
		typeResolver:            &TypeResolver{inspector: mockInspector, virtualModule: vm},
		virtualModule:           vm,
		invokerCtx:              ctx,
		partialVirtualComponent: partialComp,
		validProps:              make(map[string]validPropInfo),
		providedPropOrigins:     make(map[string]propOrigin),
		canonicalProps:          make(map[string]ast_domain.PropValue),
	}
}

func createTestInvocationLinkerWithNilResolve() *invocationLinker {
	ctx := &AnalysisContext{
		Symbols:                  NewSymbolTable(nil),
		Diagnostics:              new([]*ast_domain.Diagnostic),
		CurrentGoFullPackagePath: "test/invoker",
		CurrentGoPackageName:     "invoker",
		CurrentGoSourcePath:      "/test/invoker.go",
		SFCSourcePath:            "/test/invoker.piko",
		Logger:                   logger_domain.GetLogger("test"),
	}

	mockInspector := &inspector_domain.MockTypeQuerier{
		GetImportsForFileFunc: func(_, _ string) map[string]string {
			return map[string]string{}
		},
		GetAllPackagesFunc: func() map[string]*inspector_dto.Package {
			return map[string]*inspector_dto.Package{}
		},
	}

	partialComp := &annotator_dto.VirtualComponent{
		HashedName:             "partial_test",
		CanonicalGoPackagePath: "test/partial",
		VirtualGoFilePath:      "/virtual/partial.go",
		RewrittenScriptAST: &goast.File{
			Name: goast.NewIdent("partial_pkg"),
		},
		Source: &annotator_dto.ParsedComponent{
			SourcePath: "/test/partial.piko",
		},
	}

	vm := &annotator_dto.VirtualModule{
		ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
			"partial_test": partialComp,
		},
		ComponentsByGoPath: map[string]*annotator_dto.VirtualComponent{
			"test/partial": partialComp,
		},
	}

	return &invocationLinker{
		invocation: &annotator_dto.PartialInvocation{
			PartialAlias:     "testPartial",
			PassedProps:      make(map[string]ast_domain.PropValue),
			RequestOverrides: make(map[string]ast_domain.PropValue),
			Location:         ast_domain.Location{},
		},
		typeResolver:            &TypeResolver{inspector: mockInspector, virtualModule: vm},
		virtualModule:           vm,
		invokerCtx:              ctx,
		partialVirtualComponent: partialComp,
		validProps:              make(map[string]validPropInfo),
		providedPropOrigins:     make(map[string]propOrigin),
		canonicalProps:          make(map[string]ast_domain.PropValue),
	}
}

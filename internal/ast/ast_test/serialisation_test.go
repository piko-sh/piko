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

package ast_test

import (
	"context"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/ast/ast_adapters"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/goastutil"
)

func TestEncodingRoundTrip(t *testing.T) {

	complexAST := createComplexTestAST()
	ultraComplexAST := createUltraComplexTestAST()

	testCases := []struct {
		ast  *ast_domain.TemplateAST
		name string
	}{
		{
			name: "Simple AST with text interpolation",
			ast:  mustParse(t, `<div class="simple">Hello {{ name }}!</div>`),
		},
		{
			name: "Complex AST with most features",
			ast:  complexAST,
		},
		{
			name: "Ultra-Complex AST with deep nesting and all types",
			ast:  ultraComplexAST,
		},
		{
			name: "Nil AST",
			ast:  nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.ast == nil {

				decodedNil, errNil := ast_adapters.DecodeAST(context.Background(), nil)
				require.Error(t, errNil)
				require.Nil(t, decodedNil)

				decodedEmpty, errEmpty := ast_adapters.DecodeAST(context.Background(), []byte{})
				require.Error(t, errEmpty)
				require.Nil(t, decodedEmpty)
				return
			}

			basePath, err := filepath.Abs(".")
			require.NoError(t, err)
			sanitisedOriginal := ast_domain.SanitiseForEncoding(tc.ast, basePath)
			require.NotNil(t, sanitisedOriginal)

			data, err := ast_adapters.EncodeAST(sanitisedOriginal)
			require.NoError(t, err, "Encoding failed")
			require.NotEmpty(t, data, "Encoded data should not be empty")

			roundTrippedAST, err := ast_adapters.DecodeAST(context.Background(), data)
			require.NoError(t, err, "Decoding failed")
			require.NotNil(t, roundTrippedAST, "Decoded AST should not be nil")

			assertASTsAreEqual(t, sanitisedOriginal, roundTrippedAST)
		})
	}
}

func assertASTsAreEqual(t *testing.T, expected, actual *ast_domain.TemplateAST) {
	t.Helper()
	require.NotNil(t, expected)
	require.NotNil(t, actual)

	require.Equal(t, expected.Tidied, actual.Tidied, "Tidied flag mismatch")

	expectedPath := ""
	if expected.SourcePath != nil {
		expectedPath = *expected.SourcePath
	}
	actualPath := ""
	if actual.SourcePath != nil {
		actualPath = *actual.SourcePath
	}
	require.Equal(t, expectedPath, actualPath, "SFCSourcePath mismatch")

	require.Len(t, actual.RootNodes, len(expected.RootNodes), "RootNodes length mismatch")
	for i := range expected.RootNodes {
		path := "RootNodes[" + strconv.Itoa(i) + "]"
		assertNodesAreEqual(t, expected.RootNodes[i], actual.RootNodes[i], path)
	}

	require.Nil(t, actual.Diagnostics, "Decoded AST should have a nil Diagnostics slice")
}

func assertNodesAreEqual(t *testing.T, expected, actual *ast_domain.TemplateNode, path string) {
	t.Helper()
	if expected == nil {
		require.Nil(t, actual, "Expected node was nil, but actual was not at path: %s", path)
		return
	}
	require.NotNil(t, actual, "Actual node was nil, but expected was not at path: %s", path)

	require.Equal(t, expected.NodeType, actual.NodeType, "NodeType mismatch at %s", path)
	require.Equal(t, expected.TagName, actual.TagName, "TagName mismatch at %s", path)
	require.Equal(t, expected.Location, actual.Location, "Location mismatch at %s", path)
	require.Equal(t, expected.TextContent, actual.TextContent, "TextContent mismatch at %s", path)
	require.Equal(t, expected.InnerHTML, actual.InnerHTML, "InnerHTML mismatch at %s", path)
	require.Equal(t, expected.IsContentEditable, actual.IsContentEditable, "IsContentEditable mismatch at %s", path)

	assertExprsAreEqual(t, expected.Key, actual.Key, path+".Key")

	require.Len(t, actual.Attributes, len(expected.Attributes), "Attributes length mismatch at %s", path)
	for i := range expected.Attributes {
		require.Equal(t, expected.Attributes[i], actual.Attributes[i], "Attribute mismatch at %s.Attributes[%d]", path, i)
	}
	require.Len(t, actual.DynamicAttributes, len(expected.DynamicAttributes), "DynamicAttributes length mismatch at %s", path)
	for i := range expected.DynamicAttributes {
		assertDynamicAttributesAreEqual(t, &expected.DynamicAttributes[i], &actual.DynamicAttributes[i], path+".DynamicAttributes["+strconv.Itoa(i)+"]")
	}
	require.Len(t, actual.RichText, len(expected.RichText), "RichText length mismatch at %s", path)
	for i := range expected.RichText {
		assertTextPartsAreEqual(t, &expected.RichText[i], &actual.RichText[i], path+".RichText["+strconv.Itoa(i)+"]")
	}
	require.Len(t, actual.Directives, len(expected.Directives), "Directives length mismatch at %s", path)
	for i := range expected.Directives {
		assertDirectivesAreEqual(t, &expected.Directives[i], &actual.Directives[i], path+".Directives["+strconv.Itoa(i)+"]")
	}

	assertDirectivesAreEqual(t, expected.DirIf, actual.DirIf, path+".DirIf")
	assertDirectivesAreEqual(t, expected.DirElseIf, actual.DirElseIf, path+".DirElseIf")
	assertDirectivesAreEqual(t, expected.DirElse, actual.DirElse, path+".DirElse")
	assertDirectivesAreEqual(t, expected.DirFor, actual.DirFor, path+".DirFor")
	assertDirectivesAreEqual(t, expected.DirShow, actual.DirShow, path+".DirShow")
	assertDirectivesAreEqual(t, expected.DirModel, actual.DirModel, path+".DirModel")
	assertDirectivesAreEqual(t, expected.DirRef, actual.DirRef, path+".DirRef")
	assertDirectivesAreEqual(t, expected.DirClass, actual.DirClass, path+".DirClass")
	assertDirectivesAreEqual(t, expected.DirStyle, actual.DirStyle, path+".DirStyle")
	assertDirectivesAreEqual(t, expected.DirText, actual.DirText, path+".DirText")
	assertDirectivesAreEqual(t, expected.DirHTML, actual.DirHTML, path+".DirHTML")
	assertDirectivesAreEqual(t, expected.DirKey, actual.DirKey, path+".DirKey")
	assertDirectivesAreEqual(t, expected.DirContext, actual.DirContext, path+".DirContext")
	assertDirectivesAreEqual(t, expected.DirScaffold, actual.DirScaffold, path+".DirScaffold")
	assertExprsAreEqual(t, expected.Key, actual.Key, path+".Key")

	require.Equal(t, len(expected.OnEvents), len(actual.OnEvents), "OnEvents map length mismatch at %s", path)
	for k, v := range expected.OnEvents {
		require.Contains(t, actual.OnEvents, k, "Missing key '%s' in OnEvents map at %s", k, path)
		require.Len(t, actual.OnEvents[k], len(v), "Directive slice length mismatch for OnEvents key '%s' at %s", k, path)
		for i := range v {
			assertDirectivesAreEqual(t, &v[i], &actual.OnEvents[k][i], path+".OnEvents["+k+"]["+strconv.Itoa(i)+"]")
		}
	}
	require.Equal(t, len(expected.CustomEvents), len(actual.CustomEvents), "CustomEvents map length mismatch at %s", path)
	for k, v := range expected.CustomEvents {
		require.Contains(t, actual.CustomEvents, k)
		require.Len(t, actual.CustomEvents[k], len(v))
		for i := range v {
			assertDirectivesAreEqual(t, &v[i], &actual.CustomEvents[k][i], path+".CustomEvents["+k+"]["+strconv.Itoa(i)+"]")
		}
	}
	require.Equal(t, len(expected.Binds), len(actual.Binds), "Binds map length mismatch at %s", path)
	for k, v := range expected.Binds {
		require.Contains(t, actual.Binds, k)
		assertDirectivesAreEqual(t, v, actual.Binds[k], path+".Binds["+k+"]")
	}

	assertGoAnnotationsAreEqual(t, expected.GoAnnotations, actual.GoAnnotations, path+".GoAnnotations")
	require.Equal(t, expected.RuntimeAnnotations, actual.RuntimeAnnotations, "RuntimeAnnotations mismatch at %s", path)

	require.Len(t, actual.Children, len(expected.Children), "Children length mismatch at %s", path)
	for i := range expected.Children {
		assertNodesAreEqual(t, expected.Children[i], actual.Children[i], path+".Children["+strconv.Itoa(i)+"]")
	}
}

func assertDirectivesAreEqual(t *testing.T, expected, actual *ast_domain.Directive, path string) {
	t.Helper()
	if expected == nil {
		require.Nil(t, actual, "Expected directive was nil, but actual was not at path: %s", path)
		return
	}
	require.NotNil(t, actual, "Actual directive was nil, but expected was not at path: %s", path)

	require.Equal(t, expected.Type, actual.Type, "Directive.Type mismatch at %s", path)
	require.Equal(t, expected.Arg, actual.Arg, "Directive.Arg mismatch at %s", path)
	require.Equal(t, expected.Modifier, actual.Modifier, "Directive.Modifier mismatch at %s", path)
	require.Equal(t, expected.RawExpression, actual.RawExpression, "Directive.RawExpression mismatch at %s", path)
	require.Equal(t, expected.Location, actual.Location, "Directive.Location mismatch at %s", path)
	require.Equal(t, expected.NameLocation, actual.NameLocation, "Directive.NameLocation mismatch at %s", path)
	assertExprsAreEqual(t, expected.Expression, actual.Expression, path+".Expression")
	assertExprsAreEqual(t, expected.ChainKey, actual.ChainKey, path+".ChainKey")
	assertGoAnnotationsAreEqual(t, expected.GoAnnotations, actual.GoAnnotations, path+".GoAnnotations")
}

func assertDynamicAttributesAreEqual(t *testing.T, expected, actual *ast_domain.DynamicAttribute, path string) {
	t.Helper()

	require.Equal(t, expected.Name, actual.Name, "DynamicAttribute.Name mismatch at %s", path)
	require.Equal(t, expected.RawExpression, actual.RawExpression, "DynamicAttribute.RawExpression mismatch at %s", path)
	require.Equal(t, expected.Location, actual.Location, "DynamicAttribute.Location mismatch at %s", path)
	require.Equal(t, expected.NameLocation, actual.NameLocation, "DynamicAttribute.NameLocation mismatch at %s", path)
	assertExprsAreEqual(t, expected.Expression, actual.Expression, path+".Expression")
	assertGoAnnotationsAreEqual(t, expected.GoAnnotations, actual.GoAnnotations, path+".GoAnnotations")
}

func assertTextPartsAreEqual(t *testing.T, expected, actual *ast_domain.TextPart, path string) {
	t.Helper()
	require.Equal(t, expected.IsLiteral, actual.IsLiteral, "TextPart.IsLiteral mismatch at %s", path)
	require.Equal(t, expected.Literal, actual.Literal, "TextPart.Literal mismatch at %s", path)
	require.Equal(t, expected.RawExpression, actual.RawExpression, "TextPart.RawExpression mismatch at %s", path)
	require.Equal(t, expected.Location, actual.Location, "TextPart.Location mismatch at %s", path)
	assertExprsAreEqual(t, expected.Expression, actual.Expression, path+".Expression")
	assertGoAnnotationsAreEqual(t, expected.GoAnnotations, actual.GoAnnotations, path+".GoAnnotations")
}

func assertExprsAreEqual(t *testing.T, expected, actual ast_domain.Expression, path string) {
	t.Helper()
	if expected == nil {
		require.Nil(t, actual, "Expected expression was nil, but actual was not at path: %s", path)
		return
	}
	require.NotNil(t, actual, "Actual expression was nil, but expected was not at path: %s", path)

	require.IsType(t, expected, actual, "Expression type mismatch at %s", path)
	require.Equal(t, expected.String(), actual.String(), "Expression string mismatch at %s", path)

	switch exp := expected.(type) {
	case *ast_domain.BinaryExpression:
		act, ok := actual.(*ast_domain.BinaryExpression)
		require.True(t, ok, "Expected *ast_domain.BinaryExpression at %s", path)
		assertExprsAreEqual(t, exp.Left, act.Left, path+".Left")
		assertExprsAreEqual(t, exp.Right, act.Right, path+".Right")
	case *ast_domain.CallExpression:
		act, ok := actual.(*ast_domain.CallExpression)
		require.True(t, ok, "Expected *ast_domain.CallExpression at %s", path)
		assertExprsAreEqual(t, exp.Callee, act.Callee, path+".Callee")
		require.Len(t, act.Args, len(exp.Args), "CallExpression.Args length mismatch at %s", path)
		for i := range exp.Args {
			assertExprsAreEqual(t, exp.Args[i], act.Args[i], path+".Args["+strconv.Itoa(i)+"]")
		}

	case *ast_domain.ObjectLiteral:
		act, ok := actual.(*ast_domain.ObjectLiteral)
		require.True(t, ok, "Expected *ast_domain.ObjectLiteral at %s", path)
		require.Len(t, act.Pairs, len(exp.Pairs))
		for k, v := range exp.Pairs {
			require.Contains(t, act.Pairs, k)
			assertExprsAreEqual(t, v, act.Pairs[k], path+".Pairs["+k+"]")
		}
	}

	assertGoAnnotationsAreEqual(t, expected.GetGoAnnotation(), actual.GetGoAnnotation(), path+".GoAnnotation")
}

func assertGoAnnotationsAreEqual(t *testing.T, expected, actual *ast_domain.GoGeneratorAnnotation, path string) {
	t.Helper()
	if expected == nil {
		require.Nil(t, actual, "Expected GoAnnotation was nil, but actual was not at path: %s", path)
		return
	}
	require.NotNil(t, actual, "Actual GoAnnotation was nil, but expected was not at path: %s", path)

	require.Equal(t, expected.NeedsCSRF, actual.NeedsCSRF, "NeedsCSRF mismatch at %s", path)
	require.Equal(t, expected.IsStatic, actual.IsStatic, "IsStatic mismatch at %s", path)
	require.Equal(t, expected.IsStructurallyStatic, actual.IsStructurallyStatic, "IsStructurallyStatic mismatch at %s", path)
	require.Equal(t, expected.Stringability, actual.Stringability, "Stringability mismatch at %s", path)

	require.Equal(t, expected.BaseCodeGenVarName, actual.BaseCodeGenVarName, "BaseCodeGenVarName mismatch at %s", path)
	require.Equal(t, expected.OriginalPackageAlias, actual.OriginalPackageAlias, "OriginalPackageAlias mismatch at %s", path)
	require.Equal(t, expected.OriginalSourcePath, actual.OriginalSourcePath, "OriginalSourcePath mismatch at %s", path)

	require.Equal(t, expected.DynamicAttributeOrigins, actual.DynamicAttributeOrigins, "DynamicAttributeOrigins mismatch at %s", path)

	assertResolvedTypeInfosAreEqual(t, expected.ResolvedType, actual.ResolvedType, path+".ResolvedType")
	require.Equal(t, expected.Symbol, actual.Symbol, "Symbol mismatch at %s", path)
	assertPropDataSourcesAreEqual(t, expected.PropDataSource, actual.PropDataSource, path+".PropDataSource")
	assertPartialInvocationInfosAreEqual(t, expected.PartialInfo, actual.PartialInfo, path+".PartialInfo")
}

func assertResolvedTypeInfosAreEqual(t *testing.T, expected, actual *ast_domain.ResolvedTypeInfo, path string) {
	t.Helper()
	if expected == nil {
		require.Nil(t, actual, "Expected ResolvedTypeInfo was nil, but actual was not at path: %s", path)
		return
	}
	require.NotNil(t, actual, "Actual ResolvedTypeInfo was nil, but expected was not at path: %s", path)

	require.Equal(t, expected.PackageAlias, actual.PackageAlias, "PackageAlias mismatch at %s", path)

	expectedTypeString := goastutil.ASTToTypeString(expected.TypeExpression, expected.PackageAlias)
	actualTypeString := goastutil.ASTToTypeString(actual.TypeExpression, actual.PackageAlias)
	require.Equal(t, expectedTypeString, actualTypeString, "ResolvedType.TypeExpr string mismatch at %s", path)
}

func assertPropDataSourcesAreEqual(t *testing.T, expected, actual *ast_domain.PropDataSource, path string) {
	t.Helper()
	if expected == nil {
		require.Nil(t, actual, "Expected PropDataSource was nil, but actual was not at path: %s", path)
		return
	}
	require.NotNil(t, actual, "Actual PropDataSource was nil, but expected was not at path: %s", path)

	require.Equal(t, expected.BaseCodeGenVarName, actual.BaseCodeGenVarName, "BaseCodeGenVarName mismatch at %s", path)
	assertResolvedTypeInfosAreEqual(t, expected.ResolvedType, actual.ResolvedType, path+".ResolvedType")
	require.Equal(t, expected.Symbol, actual.Symbol, "Symbol mismatch at %s", path)
}

func assertPartialInvocationInfosAreEqual(t *testing.T, expected, actual *ast_domain.PartialInvocationInfo, path string) {
	t.Helper()
	if expected == nil {
		require.Nil(t, actual, "Expected PartialInvocationInfo was nil, but actual was not at path: %s", path)
		return
	}
	require.NotNil(t, actual, "Actual PartialInvocationInfo was nil, but expected was not at path: %s", path)

	require.Equal(t, expected.InvocationKey, actual.InvocationKey, "InvocationKey mismatch at %s", path)
	require.Equal(t, expected.PartialAlias, actual.PartialAlias, "PartialAlias mismatch at %s", path)
	require.Equal(t, expected.PartialPackageName, actual.PartialPackageName, "PartialPackageName mismatch at %s", path)
	require.Equal(t, expected.InvokerPackageAlias, actual.InvokerPackageAlias, "InvokerPackageAlias mismatch at %s", path)
	require.Equal(t, expected.Location, actual.Location, "Location mismatch at %s", path)

	require.Equal(t, len(expected.RequestOverrides), len(actual.RequestOverrides), "RequestOverrides map length mismatch at %s", path)
	for k, expectedVal := range expected.RequestOverrides {
		actualVal, ok := actual.RequestOverrides[k]
		require.True(t, ok, "Missing key '%s' in RequestOverrides map at %s", k, path)
		assertPropValuesAreEqual(t, &expectedVal, &actualVal, path+".RequestOverrides["+k+"]")
	}

	require.Equal(t, len(expected.PassedProps), len(actual.PassedProps), "PassedProps map length mismatch at %s", path)
	for k, expectedVal := range expected.PassedProps {
		actualVal, ok := actual.PassedProps[k]
		require.True(t, ok, "Missing key '%s' in PassedProps map at %s", k, path)
		assertPropValuesAreEqual(t, &expectedVal, &actualVal, path+".PassedProps["+k+"]")
	}
}

func assertPropValuesAreEqual(t *testing.T, expected, actual *ast_domain.PropValue, path string) {
	t.Helper()
	if expected == nil {
		require.Nil(t, actual, "Expected PropValue was nil, but actual was not at path: %s", path)
		return
	}
	require.NotNil(t, actual, "Actual PropValue was nil, but expected was not at path: %s", path)

	require.Equal(t, expected.GoFieldName, actual.GoFieldName, "GoFieldName mismatch at %s", path)
	require.Equal(t, expected.Location, actual.Location, "Location mismatch at %s", path)
	assertExprsAreEqual(t, expected.Expression, actual.Expression, path+".Expression")
	assertGoAnnotationsAreEqual(t, expected.InvokerAnnotation, actual.InvokerAnnotation, path+".InvokerAnnotation")
}

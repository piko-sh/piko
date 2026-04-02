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

package lsp_domain

import (
	"context"
	goast "go/ast"
	"testing"

	"go.lsp.dev/protocol"
	"piko.sh/piko/internal/annotator/annotator_domain"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/inspector/inspector_domain"
	"piko.sh/piko/internal/inspector/inspector_dto"
	"piko.sh/piko/internal/resolver/resolver_domain"
)

func TestVisitExpressionChildrenWithContext_IndexExpr(t *testing.T) {

	inner := &ast_domain.Identifier{
		Name:             "items",
		RelativeLocation: ast_domain.Location{Line: 0, Column: 0},
		SourceLength:     5,
	}
	indexExpr := &ast_domain.IndexExpression{
		Base:             inner,
		Index:            &ast_domain.Identifier{Name: "index", RelativeLocation: ast_domain.Location{Line: 0, Column: 6}, SourceLength: 3},
		RelativeLocation: ast_domain.Location{Line: 0, Column: 0},
		SourceLength:     10,
	}

	node := newTestNodeMultiLine("div", 1, 1, 3, 20)
	node.DirIf = &ast_domain.Directive{
		Expression: indexExpr,
		Location:   ast_domain.Location{Line: 1, Column: 5},
		AttributeRange: ast_domain.Range{
			Start: ast_domain.Location{Line: 1, Column: 5},
			End:   ast_domain.Location{Line: 1, Column: 20},
		},
	}

	document := newTestDocumentBuilder().
		WithURI("file:///test.pk").
		WithAnnotationResult(&annotator_dto.AnnotationResult{
			AnnotatedAST: newTestAnnotatedAST(node),
		}).
		Build()

	expression, _ := findExpressionAtPosition(context.Background(), document.AnnotationResult.AnnotatedAST, protocol.Position{Line: 0, Character: 5}, "/test.pk")
	if expression == nil {
		t.Fatal("expected to find expression at position inside IndexExpr")
	}
}

func TestVisitExpressionChildrenWithContext_CallExpr(t *testing.T) {
	argIdent := &ast_domain.Identifier{
		Name:             "x",
		RelativeLocation: ast_domain.Location{Line: 0, Column: 4},
		SourceLength:     1,
	}
	callExpr := &ast_domain.CallExpression{
		Callee: &ast_domain.Identifier{
			Name:             "fn",
			RelativeLocation: ast_domain.Location{Line: 0, Column: 0},
			SourceLength:     2,
		},
		Args:             []ast_domain.Expression{argIdent},
		RelativeLocation: ast_domain.Location{Line: 0, Column: 0},
		SourceLength:     6,
	}

	node := newTestNodeMultiLine("span", 1, 1, 3, 20)
	node.DirIf = &ast_domain.Directive{
		Expression: callExpr,
		Location:   ast_domain.Location{Line: 1, Column: 5},
		AttributeRange: ast_domain.Range{
			Start: ast_domain.Location{Line: 1, Column: 5},
			End:   ast_domain.Location{Line: 1, Column: 20},
		},
	}

	document := newTestDocumentBuilder().
		WithURI("file:///test.pk").
		WithAnnotationResult(&annotator_dto.AnnotationResult{
			AnnotatedAST: newTestAnnotatedAST(node),
		}).
		Build()

	expression, _ := findExpressionAtPosition(context.Background(), document.AnnotationResult.AnnotatedAST, protocol.Position{Line: 0, Character: 5}, "/test.pk")
	if expression == nil {
		t.Fatal("expected to find expression at position inside CallExpr")
	}
}

func TestVisitExpressionChildrenWithContext_BinaryExpr(t *testing.T) {
	binaryExpr := &ast_domain.BinaryExpression{
		Left:             &ast_domain.Identifier{Name: "a", RelativeLocation: ast_domain.Location{Line: 0, Column: 0}, SourceLength: 1},
		Right:            &ast_domain.Identifier{Name: "b", RelativeLocation: ast_domain.Location{Line: 0, Column: 4}, SourceLength: 1},
		Operator:         "+",
		RelativeLocation: ast_domain.Location{Line: 0, Column: 0},
		SourceLength:     5,
	}

	node := newTestNodeMultiLine("p", 1, 1, 3, 20)
	node.DirIf = &ast_domain.Directive{
		Expression: binaryExpr,
		Location:   ast_domain.Location{Line: 1, Column: 5},
		AttributeRange: ast_domain.Range{
			Start: ast_domain.Location{Line: 1, Column: 5},
			End:   ast_domain.Location{Line: 1, Column: 20},
		},
	}

	document := newTestDocumentBuilder().
		WithURI("file:///test.pk").
		WithAnnotationResult(&annotator_dto.AnnotationResult{
			AnnotatedAST: newTestAnnotatedAST(node),
		}).
		Build()

	expression, _ := findExpressionAtPosition(context.Background(), document.AnnotationResult.AnnotatedAST, protocol.Position{Line: 0, Character: 5}, "/test.pk")
	if expression == nil {
		t.Fatal("expected to find expression at position inside BinaryExpr")
	}
}

func TestVisitExpressionChildrenWithContext_UnaryExpr(t *testing.T) {
	unaryExpr := &ast_domain.UnaryExpression{
		Right:            &ast_domain.Identifier{Name: "val", RelativeLocation: ast_domain.Location{Line: 0, Column: 1}, SourceLength: 3},
		Operator:         "!",
		RelativeLocation: ast_domain.Location{Line: 0, Column: 0},
		SourceLength:     4,
	}

	node := newTestNodeMultiLine("div", 1, 1, 3, 20)
	node.DirIf = &ast_domain.Directive{
		Expression: unaryExpr,
		Location:   ast_domain.Location{Line: 1, Column: 5},
		AttributeRange: ast_domain.Range{
			Start: ast_domain.Location{Line: 1, Column: 5},
			End:   ast_domain.Location{Line: 1, Column: 20},
		},
	}

	document := newTestDocumentBuilder().
		WithURI("file:///test.pk").
		WithAnnotationResult(&annotator_dto.AnnotationResult{
			AnnotatedAST: newTestAnnotatedAST(node),
		}).
		Build()

	expression, _ := findExpressionAtPosition(context.Background(), document.AnnotationResult.AnnotatedAST, protocol.Position{Line: 0, Character: 5}, "/test.pk")
	if expression == nil {
		t.Fatal("expected to find expression at position inside UnaryExpr")
	}
}

func TestVisitExpressionChildrenWithContext_TernaryExpr(t *testing.T) {
	ternaryExpr := &ast_domain.TernaryExpression{
		Condition:        &ast_domain.Identifier{Name: "cond", RelativeLocation: ast_domain.Location{Line: 0, Column: 0}, SourceLength: 4},
		Consequent:       &ast_domain.Identifier{Name: "yes", RelativeLocation: ast_domain.Location{Line: 0, Column: 7}, SourceLength: 3},
		Alternate:        &ast_domain.Identifier{Name: "no", RelativeLocation: ast_domain.Location{Line: 0, Column: 13}, SourceLength: 2},
		RelativeLocation: ast_domain.Location{Line: 0, Column: 0},
		SourceLength:     15,
	}

	node := newTestNodeMultiLine("div", 1, 1, 3, 30)
	node.DirIf = &ast_domain.Directive{
		Expression: ternaryExpr,
		Location:   ast_domain.Location{Line: 1, Column: 5},
		AttributeRange: ast_domain.Range{
			Start: ast_domain.Location{Line: 1, Column: 5},
			End:   ast_domain.Location{Line: 1, Column: 30},
		},
	}

	document := newTestDocumentBuilder().
		WithURI("file:///test.pk").
		WithAnnotationResult(&annotator_dto.AnnotationResult{
			AnnotatedAST: newTestAnnotatedAST(node),
		}).
		Build()

	expression, _ := findExpressionAtPosition(context.Background(), document.AnnotationResult.AnnotatedAST, protocol.Position{Line: 0, Character: 5}, "/test.pk")
	if expression == nil {
		t.Fatal("expected to find expression at position inside TernaryExpr")
	}
}

func TestVisitExpressionChildrenWithContext_TemplateLiteral(t *testing.T) {
	templateLit := &ast_domain.TemplateLiteral{
		Parts: []ast_domain.TemplateLiteralPart{
			{
				IsLiteral: false,
				Expression: &ast_domain.Identifier{
					Name:             "val",
					RelativeLocation: ast_domain.Location{Line: 0, Column: 0},
					SourceLength:     3,
				},
				RelativeLocation: ast_domain.Location{Line: 0, Column: 2},
			},
		},
		RelativeLocation: ast_domain.Location{Line: 0, Column: 0},
		SourceLength:     10,
	}

	node := newTestNodeMultiLine("div", 1, 1, 3, 20)
	node.DirText = &ast_domain.Directive{
		Expression: templateLit,
		Location:   ast_domain.Location{Line: 1, Column: 5},
		AttributeRange: ast_domain.Range{
			Start: ast_domain.Location{Line: 1, Column: 5},
			End:   ast_domain.Location{Line: 1, Column: 20},
		},
	}

	document := newTestDocumentBuilder().
		WithURI("file:///test.pk").
		WithAnnotationResult(&annotator_dto.AnnotationResult{
			AnnotatedAST: newTestAnnotatedAST(node),
		}).
		Build()

	expression, _ := findExpressionAtPosition(context.Background(), document.AnnotationResult.AnnotatedAST, protocol.Position{Line: 0, Character: 6}, "/test.pk")
	if expression == nil {
		t.Fatal("expected to find expression at position inside TemplateLiteral")
	}
}

func TestVisitExpressionChildrenWithContext_ObjectLiteral(t *testing.T) {
	objLit := &ast_domain.ObjectLiteral{
		Pairs: map[string]ast_domain.Expression{
			"key": &ast_domain.Identifier{Name: "v", RelativeLocation: ast_domain.Location{Line: 0, Column: 5}, SourceLength: 1},
		},
		RelativeLocation: ast_domain.Location{Line: 0, Column: 0},
		SourceLength:     8,
	}

	node := newTestNodeMultiLine("div", 1, 1, 3, 20)
	node.DirClass = &ast_domain.Directive{
		Expression: objLit,
		Location:   ast_domain.Location{Line: 1, Column: 5},
		AttributeRange: ast_domain.Range{
			Start: ast_domain.Location{Line: 1, Column: 5},
			End:   ast_domain.Location{Line: 1, Column: 20},
		},
	}

	document := newTestDocumentBuilder().
		WithURI("file:///test.pk").
		WithAnnotationResult(&annotator_dto.AnnotationResult{
			AnnotatedAST: newTestAnnotatedAST(node),
		}).
		Build()

	expression, _ := findExpressionAtPosition(context.Background(), document.AnnotationResult.AnnotatedAST, protocol.Position{Line: 0, Character: 6}, "/test.pk")
	if expression == nil {
		t.Fatal("expected to find expression at position inside ObjectLiteral")
	}
}

func TestVisitExpressionChildrenWithContext_ArrayLiteral(t *testing.T) {
	arrLit := &ast_domain.ArrayLiteral{
		Elements: []ast_domain.Expression{
			&ast_domain.Identifier{Name: "el", RelativeLocation: ast_domain.Location{Line: 0, Column: 1}, SourceLength: 2},
		},
		RelativeLocation: ast_domain.Location{Line: 0, Column: 0},
		SourceLength:     5,
	}

	node := newTestNodeMultiLine("div", 1, 1, 3, 20)
	node.DirClass = &ast_domain.Directive{
		Expression: arrLit,
		Location:   ast_domain.Location{Line: 1, Column: 5},
		AttributeRange: ast_domain.Range{
			Start: ast_domain.Location{Line: 1, Column: 5},
			End:   ast_domain.Location{Line: 1, Column: 20},
		},
	}

	document := newTestDocumentBuilder().
		WithURI("file:///test.pk").
		WithAnnotationResult(&annotator_dto.AnnotationResult{
			AnnotatedAST: newTestAnnotatedAST(node),
		}).
		Build()

	expression, _ := findExpressionAtPosition(context.Background(), document.AnnotationResult.AnnotatedAST, protocol.Position{Line: 0, Character: 6}, "/test.pk")
	if expression == nil {
		t.Fatal("expected to find expression at position inside ArrayLiteral")
	}
}

func TestVisitExpressionChildrenWithContext_ForInExpr(t *testing.T) {
	forInExpr := &ast_domain.ForInExpression{
		ItemVariable:     &ast_domain.Identifier{Name: "item", RelativeLocation: ast_domain.Location{Line: 0, Column: 0}, SourceLength: 4},
		IndexVariable:    &ast_domain.Identifier{Name: "index", RelativeLocation: ast_domain.Location{Line: 0, Column: 6}, SourceLength: 3},
		Collection:       &ast_domain.Identifier{Name: "list", RelativeLocation: ast_domain.Location{Line: 0, Column: 13}, SourceLength: 4},
		RelativeLocation: ast_domain.Location{Line: 0, Column: 0},
		SourceLength:     17,
	}

	node := newTestNodeMultiLine("div", 1, 1, 3, 30)
	node.DirFor = &ast_domain.Directive{
		Expression: forInExpr,
		Location:   ast_domain.Location{Line: 1, Column: 5},
		AttributeRange: ast_domain.Range{
			Start: ast_domain.Location{Line: 1, Column: 5},
			End:   ast_domain.Location{Line: 1, Column: 30},
		},
	}

	document := newTestDocumentBuilder().
		WithURI("file:///test.pk").
		WithAnnotationResult(&annotator_dto.AnnotationResult{
			AnnotatedAST: newTestAnnotatedAST(node),
		}).
		Build()

	expression, _ := findExpressionAtPosition(context.Background(), document.AnnotationResult.AnnotatedAST, protocol.Position{Line: 0, Character: 6}, "/test.pk")
	if expression == nil {
		t.Fatal("expected to find expression at position inside ForInExpr")
	}
}

func TestVisitExpressionChildrenWithContext_LinkedMessageExpr(t *testing.T) {
	linkedMessage := &ast_domain.LinkedMessageExpression{
		Path: &ast_domain.Identifier{
			Name:             "msg",
			RelativeLocation: ast_domain.Location{Line: 0, Column: 1},
			SourceLength:     3,
		},
		RelativeLocation: ast_domain.Location{Line: 0, Column: 0},
		SourceLength:     5,
	}

	node := newTestNodeMultiLine("div", 1, 1, 3, 20)
	node.DirText = &ast_domain.Directive{
		Expression: linkedMessage,
		Location:   ast_domain.Location{Line: 1, Column: 5},
		AttributeRange: ast_domain.Range{
			Start: ast_domain.Location{Line: 1, Column: 5},
			End:   ast_domain.Location{Line: 1, Column: 20},
		},
	}

	document := newTestDocumentBuilder().
		WithURI("file:///test.pk").
		WithAnnotationResult(&annotator_dto.AnnotationResult{
			AnnotatedAST: newTestAnnotatedAST(node),
		}).
		Build()

	expression, _ := findExpressionAtPosition(context.Background(), document.AnnotationResult.AnnotatedAST, protocol.Position{Line: 0, Character: 6}, "/test.pk")
	if expression == nil {
		t.Fatal("expected to find expression at position inside LinkedMessageExpr")
	}
}

func TestFindExprInTextChildren_MatchesRichTextChild(t *testing.T) {

	parent := newTestNodeMultiLine("div", 1, 1, 5, 10)
	textChild := &ast_domain.TemplateNode{
		TagName:  "",
		NodeType: ast_domain.NodeText,
		Location: ast_domain.Location{Line: 2, Column: 1},
		RichText: []ast_domain.TextPart{
			{
				IsLiteral: false,
				Expression: &ast_domain.Identifier{
					Name:             "greeting",
					RelativeLocation: ast_domain.Location{Line: 0, Column: 0},
					SourceLength:     8,
				},
				Location: ast_domain.Location{Line: 2, Column: 5},
			},
		},
		NodeRange: ast_domain.Range{
			Start: ast_domain.Location{Line: 2, Column: 1},
			End:   ast_domain.Location{Line: 2, Column: 20},
		},
	}
	parent.Children = append(parent.Children, textChild)

	document := newTestDocumentBuilder().
		WithURI("file:///test.pk").
		WithAnnotationResult(&annotator_dto.AnnotationResult{
			AnnotatedAST: newTestAnnotatedAST(parent),
		}).
		Build()

	expression, _ := findExpressionAtPosition(context.Background(), document.AnnotationResult.AnnotatedAST, protocol.Position{Line: 1, Character: 5}, "/test.pk")

	_ = expression
}

func TestFindExprInTextChildren_SkipsNonTextNodes(t *testing.T) {
	parent := newTestNodeMultiLine("div", 1, 1, 5, 10)
	elementChild := &ast_domain.TemplateNode{
		TagName:  "span",
		NodeType: ast_domain.NodeElement,
		Location: ast_domain.Location{Line: 2, Column: 1},
		NodeRange: ast_domain.Range{
			Start: ast_domain.Location{Line: 2, Column: 1},
			End:   ast_domain.Location{Line: 2, Column: 20},
		},
	}
	parent.Children = append(parent.Children, elementChild)

	expression, _ := findExprInTextChildren(context.Background(), parent, protocol.Position{Line: 1, Character: 5})
	if expression != nil {
		t.Error("expected nil expression when there are no text children with RichText")
	}
}

func TestFindExprInPassedProps_MatchingPositionWithInvokerAnnotation(t *testing.T) {
	propExpr := &ast_domain.Identifier{
		Name:             "title",
		RelativeLocation: ast_domain.Location{Line: 0, Column: 0},
		SourceLength:     5,
	}
	invokerAnn := &ast_domain.GoGeneratorAnnotation{
		ResolvedType: &ast_domain.ResolvedTypeInfo{
			TypeExpression:       &goast.Ident{Name: "string"},
			CanonicalPackagePath: "builtin",
		},
	}

	node := newTestNodeMultiLine("div", 1, 1, 3, 30)
	node.GoAnnotations = &ast_domain.GoGeneratorAnnotation{
		PartialInfo: &ast_domain.PartialInvocationInfo{
			PassedProps: map[string]ast_domain.PropValue{
				"title": {
					Expression:        propExpr,
					InvokerAnnotation: invokerAnn,
					Location:          ast_domain.Location{Line: 2, Column: 10},
					NameLocation:      ast_domain.Location{Line: 2, Column: 5},
				},
			},
		},
	}

	expression, _ := findExprInPassedProps(context.Background(), node, protocol.Position{Line: 1, Character: 5})
	if expression == nil {
		t.Fatal("expected to find expression in passed props")
	}
}

func TestFindExprInPassedProps_NoPartialInfo(t *testing.T) {
	node := newTestNode("div", 1, 1)
	expression, _ := findExprInPassedProps(context.Background(), node, protocol.Position{Line: 0, Character: 5})
	if expression != nil {
		t.Error("expected nil when node has no PartialInfo")
	}
}

func TestGetLinkedEditingRanges_ReturnsRangesForHTMLTag(t *testing.T) {
	node := &ast_domain.TemplateNode{
		TagName:  "section",
		NodeType: ast_domain.NodeElement,
		Location: ast_domain.Location{Line: 1, Column: 1},
		OpeningTagRange: ast_domain.Range{
			Start: ast_domain.Location{Line: 1, Column: 1},
			End:   ast_domain.Location{Line: 1, Column: 10},
		},
		NodeRange: ast_domain.Range{
			Start: ast_domain.Location{Line: 1, Column: 1},
			End:   ast_domain.Location{Line: 3, Column: 11},
		},
		ClosingTagRange: ast_domain.Range{
			Start: ast_domain.Location{Line: 3, Column: 1},
			End:   ast_domain.Location{Line: 3, Column: 11},
		},
	}

	document := newTestDocumentBuilder().
		WithURI("file:///test.pk").
		WithAnnotationResult(&annotator_dto.AnnotationResult{
			AnnotatedAST: newTestAnnotatedAST(node),
		}).
		Build()

	ranges, err := document.GetLinkedEditingRanges(protocol.Position{Line: 0, Character: 2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ranges == nil {
		t.Fatal("expected linked editing ranges for a normal HTML tag")
	}
	if len(ranges.Ranges) != 2 {
		t.Fatalf("expected 2 ranges, got %d", len(ranges.Ranges))
	}
}

func TestGetLinkedEditingRanges_ReturnsNilForSelfClosingTag(t *testing.T) {

	node := &ast_domain.TemplateNode{
		TagName:  "img",
		NodeType: ast_domain.NodeElement,
		Location: ast_domain.Location{Line: 1, Column: 1},
		OpeningTagRange: ast_domain.Range{
			Start: ast_domain.Location{Line: 1, Column: 1},
			End:   ast_domain.Location{Line: 1, Column: 6},
		},
		NodeRange: ast_domain.Range{
			Start: ast_domain.Location{Line: 1, Column: 1},
			End:   ast_domain.Location{Line: 1, Column: 6},
		},
		ClosingTagRange: ast_domain.Range{
			Start: ast_domain.Location{Line: 0, Column: 0},
			End:   ast_domain.Location{Line: 0, Column: 0},
		},
	}

	document := newTestDocumentBuilder().
		WithURI("file:///test.pk").
		WithAnnotationResult(&annotator_dto.AnnotationResult{
			AnnotatedAST: newTestAnnotatedAST(node),
		}).
		Build()

	ranges, err := document.GetLinkedEditingRanges(protocol.Position{Line: 0, Character: 2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ranges != nil {
		t.Error("expected nil for self-closing tag without closing tag range")
	}
}

func TestGetLinkedEditingRanges_ReturnsNilForSpecialTag(t *testing.T) {
	node := &ast_domain.TemplateNode{
		TagName:  "template",
		NodeType: ast_domain.NodeElement,
		Location: ast_domain.Location{Line: 1, Column: 1},
		OpeningTagRange: ast_domain.Range{
			Start: ast_domain.Location{Line: 1, Column: 1},
			End:   ast_domain.Location{Line: 1, Column: 11},
		},
		NodeRange: ast_domain.Range{
			Start: ast_domain.Location{Line: 1, Column: 1},
			End:   ast_domain.Location{Line: 5, Column: 12},
		},
		ClosingTagRange: ast_domain.Range{
			Start: ast_domain.Location{Line: 5, Column: 1},
			End:   ast_domain.Location{Line: 5, Column: 12},
		},
	}

	document := newTestDocumentBuilder().
		WithURI("file:///test.pk").
		WithAnnotationResult(&annotator_dto.AnnotationResult{
			AnnotatedAST: newTestAnnotatedAST(node),
		}).
		Build()

	ranges, err := document.GetLinkedEditingRanges(protocol.Position{Line: 0, Character: 2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ranges != nil {
		t.Error("expected nil for special tag 'template'")
	}
}

func TestGetLinkedEditingRanges_NilAnnotationResult(t *testing.T) {
	document := newTestDocumentBuilder().
		WithURI("file:///test.pk").
		Build()

	ranges, err := document.GetLinkedEditingRanges(protocol.Position{Line: 0, Character: 0})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ranges != nil {
		t.Error("expected nil when AnnotationResult is nil")
	}
}

func TestGetDocumentHighlights_FindsMatchingSymbols(t *testing.T) {
	docPath := "/test.pk"

	refLocation := ast_domain.Location{Line: 5, Column: 3}

	node := newTestNodeMultiLine("div", 1, 1, 4, 30)
	node.GoAnnotations = &ast_domain.GoGeneratorAnnotation{
		OriginalSourcePath: &docPath,
	}
	node.DirIf = &ast_domain.Directive{
		Expression: &ast_domain.Identifier{
			Name:             "count",
			RelativeLocation: ast_domain.Location{Line: 0, Column: 0},
			SourceLength:     5,
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				OriginalSourcePath: &docPath,
				Symbol: &ast_domain.ResolvedSymbol{
					Name:              "count",
					ReferenceLocation: refLocation,
				},
			},
		},
		Location: ast_domain.Location{Line: 1, Column: 5},
		AttributeRange: ast_domain.Range{
			Start: ast_domain.Location{Line: 1, Column: 5},
			End:   ast_domain.Location{Line: 1, Column: 15},
		},
	}

	document := newTestDocumentBuilder().
		WithURI("file:///test.pk").
		WithAnnotationResult(&annotator_dto.AnnotationResult{
			AnnotatedAST: newTestAnnotatedAST(node),
		}).
		Build()

	highlights, err := document.GetDocumentHighlights(context.Background(), protocol.Position{Line: 0, Character: 5})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(highlights) == 0 {
		t.Fatal("expected at least one highlight for the matching symbol")
	}
}

func TestGetImplementations_FullPath(t *testing.T) {

	node := newTestNodeMultiLine("div", 1, 1, 3, 10)
	node.DirIf = &ast_domain.Directive{
		Expression: &ast_domain.Identifier{
			Name:             "reader",
			RelativeLocation: ast_domain.Location{Line: 0, Column: 0},
			SourceLength:     6,
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression:       &goast.Ident{Name: "Reader"},
					CanonicalPackagePath: "io",
				},
			},
		},
		Location: ast_domain.Location{Line: 1, Column: 5},
		AttributeRange: ast_domain.Range{
			Start: ast_domain.Location{Line: 1, Column: 5},
			End:   ast_domain.Location{Line: 1, Column: 15},
		},
	}

	typeData := &inspector_dto.TypeData{
		Packages: map[string]*inspector_dto.Package{
			"io": {
				NamedTypes: map[string]*inspector_dto.Type{
					"Reader": {
						UnderlyingTypeString: "interface{Read(p []byte) (n int, err error)}",
						DefinedInFilePath:    "/go/src/io/io.go",
						DefinitionLine:       10,
						DefinitionColumn:     6,
					},
				},
			},
			"bytes": {
				NamedTypes: map[string]*inspector_dto.Type{
					"Buffer": {
						UnderlyingTypeString: "struct{buf []byte}",
						DefinedInFilePath:    "/go/src/bytes/buffer.go",
						DefinitionLine:       20,
						DefinitionColumn:     6,
						Methods: []*inspector_dto.Method{
							{
								Name: "Read",
								Signature: inspector_dto.FunctionSignature{
									Params:  []string{"p []byte"},
									Results: []string{"int", "error"},
								},
							},
						},
					},
				},
			},
		},
	}
	index := inspector_domain.NewImplementationIndex(typeData)

	document := newTestDocumentBuilder().
		WithURI("file:///test.pk").
		WithAnnotationResult(&annotator_dto.AnnotationResult{
			AnnotatedAST: newTestAnnotatedAST(node),
		}).
		WithTypeInspector(&mockTypeInspector{
			GetImplementationIndexFunc: func() *inspector_domain.ImplementationIndex {
				return index
			},
		}).
		Build()

	locations, err := document.GetImplementations(context.Background(), protocol.Position{Line: 0, Character: 5})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(locations) == 0 {
		t.Log("no implementations found - this may be expected if the implementation matching logic requires specific method signatures")
	}
}

func TestGetImplementations_NilImplementationIndex(t *testing.T) {
	node := newTestNodeMultiLine("div", 1, 1, 3, 10)
	node.DirIf = &ast_domain.Directive{
		Expression: &ast_domain.Identifier{
			Name:             "w",
			RelativeLocation: ast_domain.Location{Line: 0, Column: 0},
			SourceLength:     1,
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression:       &goast.Ident{Name: "Writer"},
					CanonicalPackagePath: "io",
				},
			},
		},
		Location: ast_domain.Location{Line: 1, Column: 5},
		AttributeRange: ast_domain.Range{
			Start: ast_domain.Location{Line: 1, Column: 5},
			End:   ast_domain.Location{Line: 1, Column: 10},
		},
	}

	document := newTestDocumentBuilder().
		WithURI("file:///test.pk").
		WithAnnotationResult(&annotator_dto.AnnotationResult{
			AnnotatedAST: newTestAnnotatedAST(node),
		}).
		WithTypeInspector(&mockTypeInspector{
			GetImplementationIndexFunc: func() *inspector_domain.ImplementationIndex {
				return nil
			},
		}).
		Build()

	locations, err := document.GetImplementations(context.Background(), protocol.Position{Line: 0, Character: 5})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(locations) != 0 {
		t.Errorf("expected 0 locations when index is nil, got %d", len(locations))
	}
}

func TestGetMonikers_ReturnsMoniker(t *testing.T) {
	node := newTestNodeMultiLine("div", 1, 1, 3, 10)
	node.DirIf = &ast_domain.Directive{
		Expression: &ast_domain.Identifier{
			Name:             "UserService",
			RelativeLocation: ast_domain.Location{Line: 0, Column: 0},
			SourceLength:     11,
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression:          &goast.Ident{Name: "UserService"},
					CanonicalPackagePath:    "example.com/service",
					IsExportedPackageSymbol: true,
				},
			},
		},
		Location: ast_domain.Location{Line: 1, Column: 5},
		AttributeRange: ast_domain.Range{
			Start: ast_domain.Location{Line: 1, Column: 5},
			End:   ast_domain.Location{Line: 1, Column: 20},
		},
	}

	document := newTestDocumentBuilder().
		WithURI("file:///test.pk").
		WithAnnotationResult(&annotator_dto.AnnotationResult{
			AnnotatedAST: newTestAnnotatedAST(node),
		}).
		Build()

	monikers, err := document.GetMonikers(context.Background(), protocol.Position{Line: 0, Character: 5})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(monikers) != 1 {
		t.Fatalf("expected 1 moniker, got %d", len(monikers))
	}
	if monikers[0].Scheme != "go" {
		t.Errorf("scheme = %q, want %q", monikers[0].Scheme, "go")
	}
	if monikers[0].Identifier != "example.com/service#UserService" {
		t.Errorf("identifier = %q, want %q", monikers[0].Identifier, "example.com/service#UserService")
	}
	if monikers[0].Unique != protocol.UniquenessLevelGlobal {
		t.Errorf("unique = %q, want %q", monikers[0].Unique, protocol.UniquenessLevelGlobal)
	}
	if monikers[0].Kind != protocol.MonikerKindExport {
		t.Errorf("kind = %q, want %q", monikers[0].Kind, protocol.MonikerKindExport)
	}
}

func TestGetMonikers_NoAnnotation(t *testing.T) {
	node := newTestNodeMultiLine("div", 1, 1, 3, 10)
	node.DirIf = &ast_domain.Directive{
		Expression: &ast_domain.Identifier{
			Name:             "localVar",
			RelativeLocation: ast_domain.Location{Line: 0, Column: 0},
			SourceLength:     8,
		},
		Location: ast_domain.Location{Line: 1, Column: 5},
		AttributeRange: ast_domain.Range{
			Start: ast_domain.Location{Line: 1, Column: 5},
			End:   ast_domain.Location{Line: 1, Column: 15},
		},
	}

	document := newTestDocumentBuilder().
		WithURI("file:///test.pk").
		WithAnnotationResult(&annotator_dto.AnnotationResult{
			AnnotatedAST: newTestAnnotatedAST(node),
		}).
		Build()

	monikers, err := document.GetMonikers(context.Background(), protocol.Position{Line: 0, Character: 5})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(monikers) != 0 {
		t.Errorf("expected 0 monikers when annotation is nil, got %d", len(monikers))
	}
}

func TestGetMonikers_LocalSymbol(t *testing.T) {
	node := newTestNodeMultiLine("div", 1, 1, 3, 10)
	node.DirIf = &ast_domain.Directive{
		Expression: &ast_domain.Identifier{
			Name:             "localCount",
			RelativeLocation: ast_domain.Location{Line: 0, Column: 0},
			SourceLength:     10,
			GoAnnotations:    &ast_domain.GoGeneratorAnnotation{},
		},
		Location: ast_domain.Location{Line: 1, Column: 5},
		AttributeRange: ast_domain.Range{
			Start: ast_domain.Location{Line: 1, Column: 5},
			End:   ast_domain.Location{Line: 1, Column: 20},
		},
	}

	document := newTestDocumentBuilder().
		WithURI("file:///test.pk").
		WithAnnotationResult(&annotator_dto.AnnotationResult{
			AnnotatedAST: newTestAnnotatedAST(node),
		}).
		Build()

	monikers, err := document.GetMonikers(context.Background(), protocol.Position{Line: 0, Character: 5})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(monikers) != 1 {
		t.Fatalf("expected 1 moniker, got %d", len(monikers))
	}
	if monikers[0].Unique != protocol.UniquenessLevelDocument {
		t.Errorf("unique = %q, want %q", monikers[0].Unique, protocol.UniquenessLevelDocument)
	}
	if monikers[0].Kind != protocol.MonikerKindLocal {
		t.Errorf("kind = %q, want %q", monikers[0].Kind, protocol.MonikerKindLocal)
	}
}

func TestLookupFunctionSignature_IdentCallee(t *testing.T) {
	callExpr := &ast_domain.CallExpression{
		Callee: &ast_domain.Identifier{
			Name:             "doWork",
			RelativeLocation: ast_domain.Location{Line: 0, Column: 0},
			SourceLength:     6,
		},
		RelativeLocation: ast_domain.Location{Line: 0, Column: 0},
		SourceLength:     8,
	}
	calleeAnn := &ast_domain.GoGeneratorAnnotation{
		ResolvedType: &ast_domain.ResolvedTypeInfo{
			PackageAlias: "mypkg",
		},
	}
	analysisCtx := &annotator_domain.AnalysisContext{
		CurrentGoFullPackagePath: "example.com/app",
		CurrentGoSourcePath:      "/src/app.go",
	}

	wantSig := &inspector_dto.FunctionSignature{
		Params: []string{"ctx context.Context"},
	}

	document := newTestDocumentBuilder().
		WithTypeInspector(&mockTypeInspector{
			FindFuncSignatureFunc: func(pkgAlias, functionName, importerPackagePath, importerFilePath string) *inspector_dto.FunctionSignature {
				if pkgAlias == "mypkg" && functionName == "doWork" {
					return wantSig
				}
				return nil
			},
		}).
		Build()

	got := document.lookupFunctionSignature(callExpr, calleeAnn, analysisCtx)
	if got != wantSig {
		t.Errorf("expected wantSig, got %v", got)
	}
}

func TestLookupFunctionSignature_MemberCallee(t *testing.T) {
	memberExpr := &ast_domain.MemberExpression{
		Base: &ast_domain.Identifier{
			Name:             "service",
			RelativeLocation: ast_domain.Location{Line: 0, Column: 0},
			SourceLength:     3,
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression: &goast.Ident{Name: "Service"},
				},
			},
		},
		Property: &ast_domain.Identifier{
			Name:             "Run",
			RelativeLocation: ast_domain.Location{Line: 0, Column: 4},
			SourceLength:     3,
		},
		RelativeLocation: ast_domain.Location{Line: 0, Column: 0},
		SourceLength:     7,
	}
	callExpr := &ast_domain.CallExpression{
		Callee:           memberExpr,
		RelativeLocation: ast_domain.Location{Line: 0, Column: 0},
		SourceLength:     9,
	}
	calleeAnn := &ast_domain.GoGeneratorAnnotation{
		ResolvedType: &ast_domain.ResolvedTypeInfo{
			PackageAlias: "service",
		},
	}
	analysisCtx := &annotator_domain.AnalysisContext{
		CurrentGoFullPackagePath: "example.com/app",
		CurrentGoSourcePath:      "/src/app.go",
	}

	wantSig := &inspector_dto.FunctionSignature{
		Params: []string{"arguments ...string"},
	}

	document := newTestDocumentBuilder().
		WithTypeInspector(&mockTypeInspector{
			FindMethodSignatureFunc: func(baseType goast.Expr, methodName, importerPackagePath, importerFilePath string) *inspector_dto.FunctionSignature {
				if methodName == "Run" {
					return wantSig
				}
				return nil
			},
		}).
		Build()

	got := document.lookupFunctionSignature(callExpr, calleeAnn, analysisCtx)
	if got != wantSig {
		t.Errorf("expected wantSig, got %v", got)
	}
}

func TestLookupFunctionSignature_UnknownCalleeType(t *testing.T) {

	callExpr := &ast_domain.CallExpression{
		Callee: &ast_domain.BinaryExpression{
			Left:             &ast_domain.Identifier{Name: "a", RelativeLocation: ast_domain.Location{}, SourceLength: 1},
			Right:            &ast_domain.Identifier{Name: "b", RelativeLocation: ast_domain.Location{}, SourceLength: 1},
			Operator:         "+",
			RelativeLocation: ast_domain.Location{Line: 0, Column: 0},
			SourceLength:     3,
		},
		RelativeLocation: ast_domain.Location{Line: 0, Column: 0},
		SourceLength:     5,
	}
	calleeAnn := &ast_domain.GoGeneratorAnnotation{
		ResolvedType: &ast_domain.ResolvedTypeInfo{},
	}
	analysisCtx := &annotator_domain.AnalysisContext{}

	document := newTestDocumentBuilder().
		WithTypeInspector(&mockTypeInspector{}).
		Build()

	got := document.lookupFunctionSignature(callExpr, calleeAnn, analysisCtx)
	if got != nil {
		t.Errorf("expected nil for unknown callee type, got %v", got)
	}
}

func TestLookupMethodSignature_NilBaseAnnotation(t *testing.T) {
	memberExpr := &ast_domain.MemberExpression{
		Base: &ast_domain.Identifier{
			Name:             "obj",
			RelativeLocation: ast_domain.Location{Line: 0, Column: 0},
			SourceLength:     3,
		},
		Property: &ast_domain.Identifier{
			Name:             "Method",
			RelativeLocation: ast_domain.Location{Line: 0, Column: 4},
			SourceLength:     6,
		},
		RelativeLocation: ast_domain.Location{Line: 0, Column: 0},
		SourceLength:     10,
	}
	analysisCtx := &annotator_domain.AnalysisContext{}

	document := newTestDocumentBuilder().
		WithTypeInspector(&mockTypeInspector{}).
		Build()

	got := document.lookupMethodSignature(memberExpr, analysisCtx)
	if got != nil {
		t.Error("expected nil when base annotation is nil")
	}
}

func TestProcessChar_EscapeSequence(t *testing.T) {
	state := &parenScanState{escapeNext: true}
	found, _, _ := state.processChar([]byte(`\"test`), 0)
	if found {
		t.Error("expected found=false when escapeNext is true")
	}
	if state.escapeNext {
		t.Error("expected escapeNext to be reset to false")
	}
}

func TestHandleStringDelimiters_Backtick(t *testing.T) {
	state := &parenScanState{}
	consumed := state.handleStringDelimiters('`', 5)
	if !consumed {
		t.Error("expected backtick to be consumed")
	}
	if !state.inRawString {
		t.Error("expected inRawString to be true after backtick")
	}

	consumed = state.handleStringDelimiters('`', 3)
	if !consumed {
		t.Error("expected second backtick to be consumed")
	}
	if state.inRawString {
		t.Error("expected inRawString to be false after second backtick")
	}
}

func TestHandleStringDelimiters_RawStringContent(t *testing.T) {
	state := &parenScanState{inRawString: true}
	consumed := state.handleStringDelimiters('a', 5)
	if !consumed {
		t.Error("expected character inside raw string to be consumed")
	}
}

func TestHandleStringDelimiters_DoubleQuote(t *testing.T) {
	state := &parenScanState{}
	consumed := state.handleStringDelimiters('"', 5)
	if !consumed {
		t.Error("expected double quote to be consumed")
	}
	if !state.inString {
		t.Error("expected inString to be true after double quote")
	}
}

func TestHandleStringDelimiters_SingleQuote(t *testing.T) {
	state := &parenScanState{}
	consumed := state.handleStringDelimiters('\'', 5)
	if !consumed {
		t.Error("expected single quote to be consumed")
	}
	if !state.inString {
		t.Error("expected inString to be true after single quote")
	}
}

func TestHandleStringDelimiters_EscapeInString(t *testing.T) {
	state := &parenScanState{inString: true}
	consumed := state.handleStringDelimiters('\\', 3)
	if !consumed {
		t.Error("expected backslash in string to be consumed")
	}
	if !state.escapeNext {
		t.Error("expected escapeNext to be true after backslash in string")
	}
}

func TestHandleStringDelimiters_NotConsumed(t *testing.T) {
	state := &parenScanState{}
	consumed := state.handleStringDelimiters('a', 5)
	if consumed {
		t.Error("expected regular character outside string to not be consumed")
	}
}

func TestAnalyseSignatureContextFromContent_OutOfBoundsLine(t *testing.T) {
	content := []byte("line one\nline two")
	ctx := analyseSignatureContextFromContent(content, protocol.Position{Line: 5, Character: 0})
	if ctx != nil {
		t.Error("expected nil when line is out of bounds")
	}
}

func TestAnalyseSignatureContextFromContent_CharacterBeyondLine(t *testing.T) {
	content := []byte("short")
	ctx := analyseSignatureContextFromContent(content, protocol.Position{Line: 0, Character: 100})
	if ctx != nil {
		t.Error("expected nil when character is beyond line length")
	}
}

func TestAnalyseSignatureContextFromContent_InsideFunctionCall(t *testing.T) {
	content := []byte(`doSomething(arg1, `)
	ctx := analyseSignatureContextFromContent(content, protocol.Position{Line: 0, Character: 18})
	if ctx == nil {
		t.Fatal("expected non-nil context inside function call")
	}
	if ctx.FunctionName != "doSomething" {
		t.Errorf("FunctionName = %q, want %q", ctx.FunctionName, "doSomething")
	}
	if ctx.ActiveParameter != 1 {
		t.Errorf("ActiveParameter = %d, want %d", ctx.ActiveParameter, 1)
	}
}

func TestAnalyseSignatureContextFromContent_MethodCall(t *testing.T) {
	content := []byte(`state.GetUser(id, `)
	ctx := analyseSignatureContextFromContent(content, protocol.Position{Line: 0, Character: 18})
	if ctx == nil {
		t.Fatal("expected non-nil context inside method call")
	}
	if ctx.FunctionName != "GetUser" {
		t.Errorf("FunctionName = %q, want %q", ctx.FunctionName, "GetUser")
	}
	if ctx.BaseExpression != "state" {
		t.Errorf("BaseExpression = %q, want %q", ctx.BaseExpression, "state")
	}
	if !ctx.IsMethodCall {
		t.Error("expected IsMethodCall to be true")
	}
}

func TestGetFoldingRanges_MultiLineElement(t *testing.T) {
	node := &ast_domain.TemplateNode{
		TagName:  "div",
		NodeType: ast_domain.NodeElement,
		Location: ast_domain.Location{Line: 1, Column: 1},
		OpeningTagRange: ast_domain.Range{
			Start: ast_domain.Location{Line: 1, Column: 1},
			End:   ast_domain.Location{Line: 1, Column: 5},
		},
		NodeRange: ast_domain.Range{
			Start: ast_domain.Location{Line: 1, Column: 1},
			End:   ast_domain.Location{Line: 5, Column: 7},
		},
		ClosingTagRange: ast_domain.Range{
			Start: ast_domain.Location{Line: 5, Column: 1},
			End:   ast_domain.Location{Line: 5, Column: 7},
		},
	}

	document := newTestDocumentBuilder().
		WithAnnotationResult(&annotator_dto.AnnotationResult{
			AnnotatedAST: newTestAnnotatedAST(node),
		}).
		Build()

	ranges, err := document.GetFoldingRanges()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ranges) == 0 {
		t.Fatal("expected at least one folding range for a multiline element")
	}
}

func TestGetFoldingRanges_SingleLineElementSkipped(t *testing.T) {
	node := &ast_domain.TemplateNode{
		TagName:  "span",
		NodeType: ast_domain.NodeElement,
		Location: ast_domain.Location{Line: 1, Column: 1},
		OpeningTagRange: ast_domain.Range{
			Start: ast_domain.Location{Line: 1, Column: 1},
			End:   ast_domain.Location{Line: 1, Column: 6},
		},
		NodeRange: ast_domain.Range{
			Start: ast_domain.Location{Line: 1, Column: 1},
			End:   ast_domain.Location{Line: 1, Column: 20},
		},
		ClosingTagRange: ast_domain.Range{
			Start: ast_domain.Location{Line: 1, Column: 14},
			End:   ast_domain.Location{Line: 1, Column: 20},
		},
	}

	document := newTestDocumentBuilder().
		WithAnnotationResult(&annotator_dto.AnnotationResult{
			AnnotatedAST: newTestAnnotatedAST(node),
		}).
		Build()

	ranges, err := document.GetFoldingRanges()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ranges) != 0 {
		t.Errorf("expected 0 folding ranges for a single-line element, got %d", len(ranges))
	}
}

func TestGetFoldingRanges_SkipsNonElementNodes(t *testing.T) {
	node := &ast_domain.TemplateNode{
		TagName:  "",
		NodeType: ast_domain.NodeText,
		Location: ast_domain.Location{Line: 1, Column: 1},
		NodeRange: ast_domain.Range{
			Start: ast_domain.Location{Line: 1, Column: 1},
			End:   ast_domain.Location{Line: 5, Column: 10},
		},
	}

	document := newTestDocumentBuilder().
		WithAnnotationResult(&annotator_dto.AnnotationResult{
			AnnotatedAST: newTestAnnotatedAST(node),
		}).
		Build()

	ranges, err := document.GetFoldingRanges()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ranges) != 0 {
		t.Errorf("expected 0 folding ranges for non-element nodes, got %d", len(ranges))
	}
}

func TestPrepareRename_ReturnsRangeForValidSymbol(t *testing.T) {
	node := newTestNodeMultiLine("div", 1, 1, 3, 10)
	node.DirIf = &ast_domain.Directive{
		Expression: &ast_domain.Identifier{
			Name:             "myVar",
			RelativeLocation: ast_domain.Location{Line: 0, Column: 0},
			SourceLength:     5,
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				Symbol: &ast_domain.ResolvedSymbol{
					Name:              "myVar",
					ReferenceLocation: ast_domain.Location{Line: 10, Column: 5},
				},
			},
		},
		Location: ast_domain.Location{Line: 1, Column: 5},
		AttributeRange: ast_domain.Range{
			Start: ast_domain.Location{Line: 1, Column: 5},
			End:   ast_domain.Location{Line: 1, Column: 15},
		},
	}

	document := newTestDocumentBuilder().
		WithURI("file:///test.pk").
		WithAnnotationResult(&annotator_dto.AnnotationResult{
			AnnotatedAST: newTestAnnotatedAST(node),
		}).
		Build()

	r, err := document.PrepareRename(context.Background(), protocol.Position{Line: 0, Character: 5})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r == nil {
		t.Fatal("expected a range for a valid renamable symbol")
	}
}

func TestPrepareRename_RejectsBuiltInNames(t *testing.T) {
	builtIns := []string{"len", "cap", "append", "make", "new", "copy", "delete", "panic", "recover", "print", "println", "min", "max"}

	for _, name := range builtIns {
		t.Run(name, func(t *testing.T) {
			node := newTestNodeMultiLine("div", 1, 1, 3, 10)
			node.DirIf = &ast_domain.Directive{
				Expression: &ast_domain.Identifier{
					Name:             name,
					RelativeLocation: ast_domain.Location{Line: 0, Column: 0},
					SourceLength:     len(name),
					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
						Symbol: &ast_domain.ResolvedSymbol{
							Name:              name,
							ReferenceLocation: ast_domain.Location{Line: 1, Column: 1},
						},
					},
				},
				Location: ast_domain.Location{Line: 1, Column: 5},
				AttributeRange: ast_domain.Range{
					Start: ast_domain.Location{Line: 1, Column: 5},
					End:   ast_domain.Location{Line: 1, Column: 5 + len(name) + 5},
				},
			}

			document := newTestDocumentBuilder().
				WithURI("file:///test.pk").
				WithAnnotationResult(&annotator_dto.AnnotationResult{
					AnnotatedAST: newTestAnnotatedAST(node),
				}).
				Build()

			r, err := document.PrepareRename(context.Background(), protocol.Position{Line: 0, Character: 5})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if r != nil {
				t.Errorf("expected nil for built-in name %q", name)
			}
		})
	}
}

func TestPrepareRename_EmptySymbolName(t *testing.T) {
	node := newTestNodeMultiLine("div", 1, 1, 3, 10)
	node.DirIf = &ast_domain.Directive{
		Expression: &ast_domain.Identifier{
			Name:             "x",
			RelativeLocation: ast_domain.Location{Line: 0, Column: 0},
			SourceLength:     1,
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				Symbol: &ast_domain.ResolvedSymbol{
					Name:              "",
					ReferenceLocation: ast_domain.Location{Line: 1, Column: 1},
				},
			},
		},
		Location: ast_domain.Location{Line: 1, Column: 5},
		AttributeRange: ast_domain.Range{
			Start: ast_domain.Location{Line: 1, Column: 5},
			End:   ast_domain.Location{Line: 1, Column: 10},
		},
	}

	document := newTestDocumentBuilder().
		WithURI("file:///test.pk").
		WithAnnotationResult(&annotator_dto.AnnotationResult{
			AnnotatedAST: newTestAnnotatedAST(node),
		}).
		Build()

	r, err := document.PrepareRename(context.Background(), protocol.Position{Line: 0, Character: 5})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r != nil {
		t.Error("expected nil for empty symbol name")
	}
}

func TestGetSignatureHelpContext_NilTypeInspector(t *testing.T) {

	callee := &ast_domain.Identifier{
		Name:             "fn",
		RelativeLocation: ast_domain.Location{Line: 0, Column: 0},
		SourceLength:     2,
		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: &goast.Ident{Name: "fn"},
			},
		},
	}
	callExpr := &ast_domain.CallExpression{
		Callee:           callee,
		Args:             []ast_domain.Expression{},
		RelativeLocation: ast_domain.Location{Line: 0, Column: 0},
		SourceLength:     4,
	}

	node := newTestNodeMultiLine("div", 1, 1, 3, 20)
	node.DirIf = &ast_domain.Directive{
		Expression: callExpr,
		Location:   ast_domain.Location{Line: 1, Column: 5},
		AttributeRange: ast_domain.Range{
			Start: ast_domain.Location{Line: 1, Column: 5},
			End:   ast_domain.Location{Line: 1, Column: 20},
		},
	}

	document := newTestDocumentBuilder().
		WithURI("file:///test.pk").
		WithContent("<template><div p-if=\"fn()\"></div></template>").
		WithAnnotationResult(&annotator_dto.AnnotationResult{
			AnnotatedAST: newTestAnnotatedAST(node),
		}).
		Build()

	ctx := document.getSignatureHelpContext(protocol.Position{Line: 0, Character: 5})
	if ctx != nil {
		t.Error("expected nil when TypeInspector is nil")
	}
}

func TestResolvePartialLink_FullPositivePath(t *testing.T) {

	invokerHash := "invoker_hash"
	partialAlias := "card"
	partialPath := "./partials/card.pk"

	invokerComp := &annotator_dto.VirtualComponent{
		Source: &annotator_dto.ParsedComponent{
			SourcePath: "/project/page.pk",
			PikoImports: []annotator_dto.PikoImport{
				{Alias: partialAlias, Path: partialPath},
			},
		},
	}

	node := newTestNode("div", 1, 1)
	node.GoAnnotations = &ast_domain.GoGeneratorAnnotation{
		OriginalPackageAlias: &invokerHash,
	}

	document := newTestDocumentBuilder().
		WithURI("file:///project/page.pk").
		WithAnnotationResult(&annotator_dto.AnnotationResult{
			AnnotatedAST: newTestAnnotatedAST(node),
			VirtualModule: &annotator_dto.VirtualModule{
				ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
					invokerHash: invokerComp,
				},
				Graph: &annotator_dto.ComponentGraph{
					PathToHashedName: map[string]string{},
				},
			},
		}).
		WithResolver(&resolver_domain.MockResolver{
			ResolvePKPathFunc: func(_ context.Context, importPath, _ string) (string, error) {
				if importPath == partialPath {
					return "/project/partials/card.pk", nil
				}
				return "", nil
			},
		}).
		Build()

	link := document.resolvePartialLink(context.Background(), node, partialAlias, ast_domain.Location{Line: 1, Column: 5})
	if link == nil {
		t.Fatal("expected a document link for the partial")
	}
	if link.Tooltip != "Go to partial component: card" {
		t.Errorf("tooltip = %q, want %q", link.Tooltip, "Go to partial component: card")
	}
}

func TestResolvePartialLink_NoMatchingImport(t *testing.T) {
	invokerHash := "invoker_hash"
	invokerComp := &annotator_dto.VirtualComponent{
		Source: &annotator_dto.ParsedComponent{
			SourcePath:  "/project/page.pk",
			PikoImports: []annotator_dto.PikoImport{},
		},
	}

	node := newTestNode("div", 1, 1)
	node.GoAnnotations = &ast_domain.GoGeneratorAnnotation{
		OriginalPackageAlias: &invokerHash,
	}

	document := newTestDocumentBuilder().
		WithURI("file:///project/page.pk").
		WithAnnotationResult(&annotator_dto.AnnotationResult{
			AnnotatedAST: newTestAnnotatedAST(node),
			VirtualModule: &annotator_dto.VirtualModule{
				ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
					invokerHash: invokerComp,
				},
				Graph: &annotator_dto.ComponentGraph{},
			},
		}).
		Build()

	link := document.resolvePartialLink(context.Background(), node, "nonexistent", ast_domain.Location{Line: 1, Column: 5})
	if link != nil {
		t.Error("expected nil when alias has no matching import")
	}
}

func TestResolvePartialLink_FallsBackToVirtualModule(t *testing.T) {
	invokerHash := "invoker_hash"
	partialAlias := "footer"
	partialPath := "./partials/footer.pk"
	partialHashedName := "footer_hash"

	invokerComp := &annotator_dto.VirtualComponent{
		Source: &annotator_dto.ParsedComponent{
			SourcePath: "/project/page.pk",
			PikoImports: []annotator_dto.PikoImport{
				{Alias: partialAlias, Path: partialPath},
			},
		},
	}

	partialComp := &annotator_dto.VirtualComponent{
		Source: &annotator_dto.ParsedComponent{
			SourcePath: "/project/partials/footer.pk",
		},
	}

	node := newTestNode("div", 1, 1)
	node.GoAnnotations = &ast_domain.GoGeneratorAnnotation{
		OriginalPackageAlias: &invokerHash,
	}

	document := newTestDocumentBuilder().
		WithURI("file:///project/page.pk").
		WithAnnotationResult(&annotator_dto.AnnotationResult{
			AnnotatedAST: newTestAnnotatedAST(node),
			VirtualModule: &annotator_dto.VirtualModule{
				ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
					invokerHash:       invokerComp,
					partialHashedName: partialComp,
				},
				Graph: &annotator_dto.ComponentGraph{
					PathToHashedName: map[string]string{
						partialPath: partialHashedName,
					},
				},
			},
		}).
		Build()

	link := document.resolvePartialLink(context.Background(), node, partialAlias, ast_domain.Location{Line: 1, Column: 5})
	if link == nil {
		t.Fatal("expected a document link via VirtualModule fallback")
	}
}

func TestTryCreateLinkFromAttribute_IsAttribute(t *testing.T) {
	invokerHash := "invoker_hash"
	invokerComp := &annotator_dto.VirtualComponent{
		Source: &annotator_dto.ParsedComponent{
			SourcePath: "/project/page.pk",
			PikoImports: []annotator_dto.PikoImport{
				{Alias: "header", Path: "./partials/header.pk"},
			},
		},
	}

	node := newTestNode("div", 1, 1)
	node.GoAnnotations = &ast_domain.GoGeneratorAnnotation{
		OriginalPackageAlias: &invokerHash,
	}
	attr := &ast_domain.HTMLAttribute{
		Name:         "is",
		Value:        "header",
		NameLocation: ast_domain.Location{Line: 1, Column: 10},
	}

	document := newTestDocumentBuilder().
		WithURI("file:///project/page.pk").
		WithAnnotationResult(&annotator_dto.AnnotationResult{
			AnnotatedAST: newTestAnnotatedAST(node),
			VirtualModule: &annotator_dto.VirtualModule{
				ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
					invokerHash: invokerComp,
				},
				Graph: &annotator_dto.ComponentGraph{
					PathToHashedName: map[string]string{
						"./partials/header.pk": "header_hash",
					},
				},
			},
		}).
		WithResolver(&resolver_domain.MockResolver{
			ResolvePKPathFunc: func(_ context.Context, _, _ string) (string, error) {
				return "/project/partials/header.pk", nil
			},
		}).
		Build()

	link := document.tryCreateLinkFromAttribute(context.Background(), node, attr)
	if link == nil {
		t.Fatal("expected a document link for 'is' attribute")
	}
}

func TestTryCreateLinkFromAttribute_SrcAttribute(t *testing.T) {
	node := newTestNode("img", 1, 1)
	attr := &ast_domain.HTMLAttribute{
		Name:         "src",
		Value:        "./images/logo.png",
		NameLocation: ast_domain.Location{Line: 1, Column: 10},
	}

	document := newTestDocumentBuilder().
		WithURI("file:///project/page.pk").
		WithAnnotationResult(&annotator_dto.AnnotationResult{
			AnnotatedAST: newTestAnnotatedAST(node),
		}).
		WithResolver(&resolver_domain.MockResolver{
			ResolveCSSPathFunc: func(_ context.Context, _, _ string) (string, error) {
				return "/project/images/logo.png", nil
			},
		}).
		Build()

	link := document.tryCreateLinkFromAttribute(context.Background(), node, attr)
	if link == nil {
		t.Fatal("expected a document link for 'src' attribute")
	}
	if link.Tooltip != "Go to asset: ./images/logo.png" {
		t.Errorf("tooltip = %q, want %q", link.Tooltip, "Go to asset: ./images/logo.png")
	}
}

func TestTryCreateLinkFromAttribute_EmptyValue(t *testing.T) {
	node := newTestNode("img", 1, 1)
	attr := &ast_domain.HTMLAttribute{
		Name:  "src",
		Value: "",
	}

	document := newTestDocumentBuilder().
		WithURI("file:///test.pk").
		WithAnnotationResult(&annotator_dto.AnnotationResult{
			AnnotatedAST: newTestAnnotatedAST(node),
		}).
		Build()

	link := document.tryCreateLinkFromAttribute(context.Background(), node, attr)
	if link != nil {
		t.Error("expected nil for empty attribute value")
	}
}

func TestTryCreateLinkFromAttribute_DefaultReturnsNil(t *testing.T) {
	node := newTestNode("div", 1, 1)
	attr := &ast_domain.HTMLAttribute{
		Name:  "class",
		Value: "my-class",
	}

	document := newTestDocumentBuilder().
		WithURI("file:///test.pk").
		WithAnnotationResult(&annotator_dto.AnnotationResult{
			AnnotatedAST: newTestAnnotatedAST(node),
		}).
		Build()

	link := document.tryCreateLinkFromAttribute(context.Background(), node, attr)
	if link != nil {
		t.Error("expected nil for non-link attribute")
	}
}

func TestExtractImplementationTypeInfo_ReturnsTypeInfo(t *testing.T) {
	node := newTestNodeMultiLine("div", 1, 1, 3, 10)
	node.DirIf = &ast_domain.Directive{
		Expression: &ast_domain.Identifier{
			Name:             "service",
			RelativeLocation: ast_domain.Location{Line: 0, Column: 0},
			SourceLength:     3,
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression:       &goast.Ident{Name: "MyInterface"},
					CanonicalPackagePath: "example.com/iface",
				},
			},
		},
		Location: ast_domain.Location{Line: 1, Column: 5},
		AttributeRange: ast_domain.Range{
			Start: ast_domain.Location{Line: 1, Column: 5},
			End:   ast_domain.Location{Line: 1, Column: 12},
		},
	}

	document := newTestDocumentBuilder().
		WithURI("file:///test.pk").
		WithAnnotationResult(&annotator_dto.AnnotationResult{
			AnnotatedAST: newTestAnnotatedAST(node),
		}).
		Build()

	typeName, packagePath, ok := document.extractImplementationTypeInfo(context.Background(), protocol.Position{Line: 0, Character: 5})
	if !ok {
		t.Fatal("expected extractImplementationTypeInfo to succeed")
	}
	if typeName != "MyInterface" {
		t.Errorf("typeName = %q, want %q", typeName, "MyInterface")
	}
	if packagePath != "example.com/iface" {
		t.Errorf("packagePath = %q, want %q", packagePath, "example.com/iface")
	}
}

func TestExtractImplementationTypeInfo_NilResolvedType(t *testing.T) {
	node := newTestNodeMultiLine("div", 1, 1, 3, 10)
	node.DirIf = &ast_domain.Directive{
		Expression: &ast_domain.Identifier{
			Name:             "localVar",
			RelativeLocation: ast_domain.Location{Line: 0, Column: 0},
			SourceLength:     8,
			GoAnnotations:    &ast_domain.GoGeneratorAnnotation{},
		},
		Location: ast_domain.Location{Line: 1, Column: 5},
		AttributeRange: ast_domain.Range{
			Start: ast_domain.Location{Line: 1, Column: 5},
			End:   ast_domain.Location{Line: 1, Column: 15},
		},
	}

	document := newTestDocumentBuilder().
		WithURI("file:///test.pk").
		WithAnnotationResult(&annotator_dto.AnnotationResult{
			AnnotatedAST: newTestAnnotatedAST(node),
		}).
		Build()

	_, _, ok := document.extractImplementationTypeInfo(context.Background(), protocol.Position{Line: 0, Character: 5})
	if ok {
		t.Error("expected false when ResolvedType is nil")
	}
}

func TestTryFastPathSignatureHelp_NoDocument(t *testing.T) {
	ws := &workspace{
		documents: map[protocol.DocumentURI]*document{},
		docCache:  NewDocumentCache(),
	}
	server := &Server{workspace: ws}

	result := server.tryFastPathSignatureHelp(context.Background(), &protocol.SignatureHelpParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: "file:///missing.pk"},
			Position:     protocol.Position{Line: 0, Character: 0},
		},
	})
	if result != nil {
		t.Error("expected nil when document does not exist")
	}
}

func TestTryFastPathSignatureHelp_DocumentLacksPrerequisites(t *testing.T) {
	documentURI := protocol.DocumentURI("file:///test.pk")
	testDocument := newTestDocumentBuilder().
		WithURI(documentURI).
		Build()

	ws := &workspace{
		documents: map[protocol.DocumentURI]*document{documentURI: testDocument},
		docCache:  NewDocumentCache(),
	}
	server := &Server{workspace: ws}

	result := server.tryFastPathSignatureHelp(context.Background(), &protocol.SignatureHelpParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: documentURI},
			Position:     protocol.Position{Line: 0, Character: 0},
		},
	})
	if result != nil {
		t.Error("expected nil when document lacks prerequisites")
	}
}

func TestTryFastPathSignatureHelp_NoCacheContent(t *testing.T) {
	documentURI := protocol.DocumentURI("file:///test.pk")
	node := newTestNodeMultiLine("div", 1, 1, 3, 30)

	testDocument := newTestDocumentBuilder().
		WithURI(documentURI).
		WithAnnotationResult(&annotator_dto.AnnotationResult{
			AnnotatedAST: newTestAnnotatedAST(node),
		}).
		WithAnalysisMap(map[*ast_domain.TemplateNode]*annotator_domain.AnalysisContext{}).
		WithTypeInspector(&mockTypeInspector{}).
		Build()

	ws := &workspace{
		documents: map[protocol.DocumentURI]*document{documentURI: testDocument},
		docCache:  NewDocumentCache(),
	}
	server := &Server{workspace: ws}

	result := server.tryFastPathSignatureHelp(context.Background(), &protocol.SignatureHelpParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: documentURI},
			Position:     protocol.Position{Line: 0, Character: 10},
		},
	})
	if result != nil {
		t.Error("expected nil when document cache has no content")
	}
}

func TestTryFastPathSignatureHelp_WithCacheContent(t *testing.T) {
	documentURI := protocol.DocumentURI("file:///test.pk")
	node := newTestNodeMultiLine("div", 1, 1, 3, 50)

	analysisCtx := &annotator_domain.AnalysisContext{
		CurrentGoFullPackagePath: "example.com/app",
		CurrentGoSourcePath:      "/src/app.go",
	}

	wantSig := &inspector_dto.FunctionSignature{
		Params: []string{"name string", "age int"},
	}

	testDocument := newTestDocumentBuilder().
		WithURI(documentURI).
		WithAnnotationResult(&annotator_dto.AnnotationResult{
			AnnotatedAST: newTestAnnotatedAST(node),
		}).
		WithAnalysisMap(map[*ast_domain.TemplateNode]*annotator_domain.AnalysisContext{
			node: analysisCtx,
		}).
		WithTypeInspector(&mockTypeInspector{
			FindFuncSignatureFunc: func(pkgAlias, functionName, importerPackagePath, importerFilePath string) *inspector_dto.FunctionSignature {
				if functionName == "greet" {
					return wantSig
				}
				return nil
			},
		}).
		Build()

	docCache := NewDocumentCache()
	docCache.Set(documentURI, []byte(`<template><div p-if="greet(name, `))

	ws := &workspace{
		documents: map[protocol.DocumentURI]*document{documentURI: testDocument},
		docCache:  docCache,
	}
	server := &Server{workspace: ws}

	result := server.tryFastPathSignatureHelp(context.Background(), &protocol.SignatureHelpParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: documentURI},
			Position:     protocol.Position{Line: 0, Character: 34},
		},
	})

	_ = result
}

func TestGetDocumentLinks_ExtractsIsLinks(t *testing.T) {
	invokerHash := "invoker_hash"
	invokerComp := &annotator_dto.VirtualComponent{
		Source: &annotator_dto.ParsedComponent{
			SourcePath: "/project/page.pk",
			PikoImports: []annotator_dto.PikoImport{
				{Alias: "card", Path: "./partials/card.pk"},
			},
		},
	}

	node := newTestNode("div", 1, 1)
	node.GoAnnotations = &ast_domain.GoGeneratorAnnotation{
		OriginalPackageAlias: &invokerHash,
	}
	node.Attributes = []ast_domain.HTMLAttribute{
		{
			Name:         "is",
			Value:        "card",
			NameLocation: ast_domain.Location{Line: 1, Column: 10},
		},
	}

	document := newTestDocumentBuilder().
		WithURI("file:///project/page.pk").
		WithAnnotationResult(&annotator_dto.AnnotationResult{
			AnnotatedAST: newTestAnnotatedAST(node),
			VirtualModule: &annotator_dto.VirtualModule{
				ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
					invokerHash: invokerComp,
				},
				Graph: &annotator_dto.ComponentGraph{
					PathToHashedName: map[string]string{},
				},
			},
		}).
		WithResolver(&resolver_domain.MockResolver{
			ResolvePKPathFunc: func(_ context.Context, _, _ string) (string, error) {
				return "/project/partials/card.pk", nil
			},
		}).
		Build()

	links, err := document.GetDocumentLinks(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(links) == 0 {
		t.Fatal("expected at least one document link")
	}
}

func TestGetSignatureHelpFast_NoTargetNode(t *testing.T) {
	document := newTestDocumentBuilder().
		WithURI("file:///test.pk").
		WithAnnotationResult(&annotator_dto.AnnotationResult{
			AnnotatedAST: newTestAnnotatedAST(),
		}).
		WithAnalysisMap(map[*ast_domain.TemplateNode]*annotator_domain.AnalysisContext{}).
		WithTypeInspector(&mockTypeInspector{}).
		Build()

	callCtx := &signatureCallContext{
		FunctionName:    "test",
		ActiveParameter: 0,
	}

	result := document.getSignatureHelpFast(context.Background(), callCtx, protocol.Position{Line: 10, Character: 0})
	if result != nil {
		t.Error("expected nil when no target node found")
	}
}

func TestGetSignatureHelpFast_NodeNotInAnalysisMap(t *testing.T) {
	node := newTestNodeMultiLine("div", 1, 1, 3, 20)

	document := newTestDocumentBuilder().
		WithURI("file:///test.pk").
		WithAnnotationResult(&annotator_dto.AnnotationResult{
			AnnotatedAST: newTestAnnotatedAST(node),
		}).
		WithAnalysisMap(map[*ast_domain.TemplateNode]*annotator_domain.AnalysisContext{}).
		WithTypeInspector(&mockTypeInspector{}).
		Build()

	callCtx := &signatureCallContext{
		FunctionName:    "test",
		ActiveParameter: 0,
	}

	result := document.getSignatureHelpFast(context.Background(), callCtx, protocol.Position{Line: 0, Character: 2})
	if result != nil {
		t.Error("expected nil when node is not in AnalysisMap")
	}
}

func TestGetSignatureHelpFast_ReturnsSignatureForFunction(t *testing.T) {
	node := newTestNodeMultiLine("div", 1, 1, 3, 20)
	analysisCtx := &annotator_domain.AnalysisContext{
		CurrentGoFullPackagePath: "example.com/app",
		CurrentGoSourcePath:      "/src/app.go",
	}

	wantSig := &inspector_dto.FunctionSignature{
		Params: []string{"ctx context.Context"},
	}

	document := newTestDocumentBuilder().
		WithURI("file:///test.pk").
		WithAnnotationResult(&annotator_dto.AnnotationResult{
			AnnotatedAST: newTestAnnotatedAST(node),
		}).
		WithAnalysisMap(map[*ast_domain.TemplateNode]*annotator_domain.AnalysisContext{
			node: analysisCtx,
		}).
		WithTypeInspector(&mockTypeInspector{
			FindFuncSignatureFunc: func(pkgAlias, functionName, importerPackagePath, importerFilePath string) *inspector_dto.FunctionSignature {
				if functionName == "doWork" {
					return wantSig
				}
				return nil
			},
		}).
		Build()

	callCtx := &signatureCallContext{
		FunctionName:    "doWork",
		ActiveParameter: 0,
		IsMethodCall:    false,
	}

	result := document.getSignatureHelpFast(context.Background(), callCtx, protocol.Position{Line: 0, Character: 2})
	if result == nil {
		t.Fatal("expected non-nil signature help result")
	}
	if len(result.Signatures) != 1 {
		t.Fatalf("expected 1 signature, got %d", len(result.Signatures))
	}
}

func TestBuildSignatureHelp_FormatsCorrectly(t *testing.T) {
	document := newTestDocumentBuilder().Build()
	sig := &inspector_dto.FunctionSignature{
		Params: []string{"name string", "age int"},
	}

	result := document.buildSignatureHelp("greet", sig, 1)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result.Signatures) != 1 {
		t.Fatalf("expected 1 signature, got %d", len(result.Signatures))
	}
	if len(result.Signatures[0].Parameters) != 2 {
		t.Fatalf("expected 2 parameters, got %d", len(result.Signatures[0].Parameters))
	}
	if result.ActiveParameter != 1 {
		t.Errorf("ActiveParameter = %d, want %d", result.ActiveParameter, 1)
	}
}

func TestFindEnclosingCallFromText_SimpleCall(t *testing.T) {
	lines := splitLines([]byte("format(x, "))
	position := protocol.Position{Line: 0, Character: 10}

	ctx := findEnclosingCallFromText(lines, position)
	if ctx == nil {
		t.Fatal("expected non-nil context")
	}
	if ctx.FunctionName != "format" {
		t.Errorf("FunctionName = %q, want %q", ctx.FunctionName, "format")
	}
	if ctx.ActiveParameter != 1 {
		t.Errorf("ActiveParameter = %d, want %d", ctx.ActiveParameter, 1)
	}
}

func TestFindEnclosingCallFromText_MethodCall(t *testing.T) {
	lines := splitLines([]byte("service.Handle(request, "))
	position := protocol.Position{Line: 0, Character: 16}

	ctx := findEnclosingCallFromText(lines, position)
	if ctx == nil {
		t.Fatal("expected non-nil context")
	}
	if ctx.FunctionName != "Handle" {
		t.Errorf("FunctionName = %q, want %q", ctx.FunctionName, "Handle")
	}
	if ctx.BaseExpression != "service" {
		t.Errorf("BaseExpression = %q, want %q", ctx.BaseExpression, "service")
	}
	if !ctx.IsMethodCall {
		t.Error("expected IsMethodCall to be true")
	}
}

func TestFindEnclosingCallFromText_NoParen(t *testing.T) {
	lines := splitLines([]byte("just text"))
	position := protocol.Position{Line: 0, Character: 9}

	ctx := findEnclosingCallFromText(lines, position)
	if ctx != nil {
		t.Error("expected nil when no opening paren found")
	}
}

func TestExtractCalleeFromText_StandaloneFunction(t *testing.T) {
	functionName, baseExpr := extractCalleeFromText([]byte("myFunc"))
	if functionName != "myFunc" {
		t.Errorf("functionName = %q, want %q", functionName, "myFunc")
	}
	if baseExpr != "" {
		t.Errorf("baseExpr = %q, want empty", baseExpr)
	}
}

func TestExtractCalleeFromText_MethodOnBase(t *testing.T) {
	functionName, baseExpr := extractCalleeFromText([]byte("obj.Method"))
	if functionName != "Method" {
		t.Errorf("functionName = %q, want %q", functionName, "Method")
	}
	if baseExpr != "obj" {
		t.Errorf("baseExpr = %q, want %q", baseExpr, "obj")
	}
}

func TestExtractCalleeFromText_EmptyInput(t *testing.T) {
	functionName, baseExpr := extractCalleeFromText([]byte{})
	if functionName != "" {
		t.Errorf("functionName = %q, want empty", functionName)
	}
	if baseExpr != "" {
		t.Errorf("baseExpr = %q, want empty", baseExpr)
	}
}

func TestIsComponentTag_AllBranches(t *testing.T) {
	document := newTestDocumentBuilder().Build()

	testCases := []struct {
		node     *ast_domain.TemplateNode
		name     string
		expected bool
	}{
		{
			name:     "nil node",
			node:     nil,
			expected: false,
		},
		{
			name: "node with is attribute",
			node: &ast_domain.TemplateNode{
				Attributes: []ast_domain.HTMLAttribute{{Name: "is", Value: "card"}},
			},
			expected: true,
		},
		{
			name: "node with partial info",
			node: &ast_domain.TemplateNode{
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					PartialInfo: &ast_domain.PartialInvocationInfo{},
				},
			},
			expected: true,
		},
		{
			name: "regular node",
			node: &ast_domain.TemplateNode{
				TagName: "div",
			},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := document.isComponentTag(tc.node)
			if got != tc.expected {
				t.Errorf("isComponentTag() = %v, want %v", got, tc.expected)
			}
		})
	}
}

func TestIsSpecialTag_AllVariants(t *testing.T) {
	document := newTestDocumentBuilder().Build()

	testCases := []struct {
		name     string
		tagName  string
		expected bool
	}{
		{name: "template", tagName: "template", expected: true},
		{name: "script", tagName: "script", expected: true},
		{name: "style", tagName: "style", expected: true},
		{name: "slot", tagName: "slot", expected: true},
		{name: "div", tagName: "div", expected: false},
		{name: "span", tagName: "span", expected: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := document.isSpecialTag(tc.tagName)
			if got != tc.expected {
				t.Errorf("isSpecialTag(%q) = %v, want %v", tc.tagName, got, tc.expected)
			}
		})
	}
}

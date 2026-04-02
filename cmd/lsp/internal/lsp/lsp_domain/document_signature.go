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
	"go.lsp.dev/protocol"
	"piko.sh/piko/internal/annotator/annotator_domain"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/inspector/inspector_dto"
	"piko.sh/piko/wdk/safeconv"
)

// signatureHelpContext holds the checked context needed for signature help.
type signatureHelpContext struct {
	// callExpr is the call expression found at the cursor position.
	callExpr *ast_domain.CallExpression

	// calleeAnn is the annotation for the called function; nil if not found.
	calleeAnn *ast_domain.GoGeneratorAnnotation

	// analysisCtx provides type analysis context for finding the callee signature.
	analysisCtx *annotator_domain.AnalysisContext

	// activeParam is the zero-based index of the active parameter in the
	// function signature.
	activeParam int
}

// GetSignatureHelp provides parameter hints for function calls. It finds the
// enclosing function call and returns signature information with the active
// parameter highlighted.
//
// Takes position (protocol.Position) which specifies the cursor
// position within the
// document.
//
// Returns *protocol.SignatureHelp which contains the function signature with
// the active parameter highlighted.
// Returns error when the signature help cannot be determined.
func (d *document) GetSignatureHelp(position protocol.Position) (*protocol.SignatureHelp, error) {
	ctx := d.getSignatureHelpContext(position)
	if ctx == nil {
		return emptySignatureHelp()
	}

	funcSig := d.lookupFunctionSignature(ctx.callExpr, ctx.calleeAnn, ctx.analysisCtx)
	if funcSig == nil {
		return emptySignatureHelp()
	}

	return d.buildSignatureHelp(ctx.callExpr.Callee.String(), funcSig, ctx.activeParam), nil
}

// getSignatureHelpContext validates preconditions and returns the context
// needed for signature help.
//
// Takes position (protocol.Position) which specifies the cursor position.
//
// Returns *signatureHelpContext which contains the call expression and
// analysis context, or nil if any precondition fails.
func (d *document) getSignatureHelpContext(position protocol.Position) *signatureHelpContext {
	if d.AnnotationResult == nil || d.AnnotationResult.AnnotatedAST == nil {
		return nil
	}

	callExpr, activeParam := findEnclosingCallExpr(d.AnnotationResult.AnnotatedAST, position, d.Content)
	if callExpr == nil {
		return nil
	}

	calleeAnn := callExpr.Callee.GetGoAnnotation()
	if calleeAnn == nil || calleeAnn.ResolvedType == nil {
		return nil
	}

	if d.TypeInspector == nil || d.AnalysisMap == nil {
		return nil
	}

	targetNode := findNodeAtPosition(d.AnnotationResult.AnnotatedAST, position, d.URI.Filename())
	if targetNode == nil {
		return nil
	}

	analysisCtx, exists := d.AnalysisMap[targetNode]
	if !exists || analysisCtx == nil {
		return nil
	}

	return &signatureHelpContext{
		callExpr:    callExpr,
		calleeAnn:   calleeAnn,
		analysisCtx: analysisCtx,
		activeParam: activeParam,
	}
}

// lookupFunctionSignature finds the function signature for a call expression.
// Handles both method calls (baseExpr.MethodName()) and function calls
// (FuncName()).
//
// Takes callExpr (*ast_domain.CallExpression) which is the call expression to look
// up.
// Takes calleeAnn (*ast_domain.GoGeneratorAnnotation) which provides type
// resolution for the callee.
// Takes analysisCtx (*annotator_domain.AnalysisContext) which provides the
// current analysis context.
//
// Returns *inspector_dto.FunctionSignature which is the resolved signature,
// or nil if the callee type is not recognised.
func (d *document) lookupFunctionSignature(
	callExpr *ast_domain.CallExpression,
	calleeAnn *ast_domain.GoGeneratorAnnotation,
	analysisCtx *annotator_domain.AnalysisContext,
) *inspector_dto.FunctionSignature {
	if memberExpr, isMember := callExpr.Callee.(*ast_domain.MemberExpression); isMember {
		return d.lookupMethodSignature(memberExpr, analysisCtx)
	}

	if identifier, isIdent := callExpr.Callee.(*ast_domain.Identifier); isIdent {
		pkgAlias := calleeAnn.ResolvedType.PackageAlias
		return d.TypeInspector.FindFuncSignature(
			pkgAlias,
			identifier.Name,
			analysisCtx.CurrentGoFullPackagePath,
			analysisCtx.CurrentGoSourcePath,
		)
	}

	return nil
}

// lookupMethodSignature finds the method signature for a member expression call.
//
// Takes memberExpr (*ast_domain.MemberExpression) which is the
// member expression to look up.
// Takes analysisCtx (*annotator_domain.AnalysisContext) which provides the
// current package and source path context.
//
// Returns *inspector_dto.FunctionSignature which is the found method signature,
// or nil if the base type cannot be found.
func (d *document) lookupMethodSignature(
	memberExpr *ast_domain.MemberExpression,
	analysisCtx *annotator_domain.AnalysisContext,
) *inspector_dto.FunctionSignature {
	baseAnn := memberExpr.Base.GetGoAnnotation()
	if baseAnn == nil || baseAnn.ResolvedType == nil || baseAnn.ResolvedType.TypeExpression == nil {
		return nil
	}

	return d.TypeInspector.FindMethodSignature(
		baseAnn.ResolvedType.TypeExpression,
		memberExpr.Property.String(),
		analysisCtx.CurrentGoFullPackagePath,
		analysisCtx.CurrentGoSourcePath,
	)
}

// buildSignatureHelp constructs the SignatureHelp response from a function
// signature.
//
// Takes calleeName (string) which is the name of the function being called.
// Takes funcSig (*inspector_dto.FunctionSignature) which provides the function
// signature details.
// Takes activeParam (int) which indicates the current parameter position.
//
// Returns *protocol.SignatureHelp which contains the formatted signature
// information for display.
func (*document) buildSignatureHelp(calleeName string, funcSig *inspector_dto.FunctionSignature, activeParam int) *protocol.SignatureHelp {
	signature := protocol.SignatureInformation{
		Label: calleeName + funcSig.ToSignatureString(),
		Documentation: &protocol.MarkupContent{
			Kind:  protocol.Markdown,
			Value: "Function signature from type information",
		},
		Parameters: make([]protocol.ParameterInformation, 0, len(funcSig.Params)),
	}

	for _, param := range funcSig.Params {
		signature.Parameters = append(signature.Parameters, protocol.ParameterInformation{
			Label: param,
		})
	}

	activeParam = clampActiveParam(activeParam, len(signature.Parameters))

	return &protocol.SignatureHelp{
		Signatures:      []protocol.SignatureInformation{signature},
		ActiveSignature: 0,
		ActiveParameter: safeconv.IntToUint32(activeParam),
	}
}

// emptySignatureHelp returns an empty signature help response.
//
// Returns *protocol.SignatureHelp which contains an empty list of signatures.
// Returns error which is always nil.
func emptySignatureHelp() (*protocol.SignatureHelp, error) {
	return &protocol.SignatureHelp{Signatures: []protocol.SignatureInformation{}}, nil
}

// clampActiveParam keeps the active parameter index within valid bounds.
//
// Takes activeParam (int) which is the parameter index to clamp.
// Takes paramCount (int) which is the total number of parameters.
//
// Returns int which is the clamped index between 0 and paramCount-1.
func clampActiveParam(activeParam, paramCount int) int {
	if activeParam < 0 {
		return 0
	}
	if paramCount > 0 && activeParam >= paramCount {
		return paramCount - 1
	}
	return activeParam
}

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

// Implements the per-component annotation pipeline that processes individual components through expansion, analysis, and linking.
// Executes the annotation phase for each component including partial expansion, semantic analysis, CSS processing, and prop linking.

import (
	"context"
	"fmt"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/logger/logger_domain"
)

// componentAnnotationPipeline coordinates the annotation process for a single
// virtual component through the annotator service.
type componentAnnotationPipeline struct {
	// componentRegistry provides lookup of registered PKC components.
	componentRegistry ComponentRegistryPort

	// service is the parent annotator service that owns this pipeline.
	service *AnnotatorService

	// vc is the virtual component being processed through the pipeline.
	vc *annotator_dto.VirtualComponent

	// componentGraph holds the graph showing how components link to each other.
	componentGraph *annotator_dto.ComponentGraph

	// virtualModule holds the module used during the annotation process.
	virtualModule *annotator_dto.VirtualModule

	// typeResolver provides type information for the linking and annotation stages.
	typeResolver *TypeResolver

	// actions maps action names to their information providers.
	actions map[string]ActionInfoProvider

	// options holds settings that control annotation behaviour, such as fault
	// tolerance for pipeline stages.
	options *annotationOptions

	// diagnostics collects issues found during pipeline processing.
	diagnostics []*ast_domain.Diagnostic
}

// run executes the full annotation pipeline through all stages.
//
// Takes ctx (context.Context) which controls cancellation and timeout for
// pipeline stages.
//
// Returns *annotator_dto.AnnotationResult which contains the processed AST.
// Returns []*ast_domain.Diagnostic which contains any warnings or errors.
// Returns error when a pipeline stage fails.
func (p *componentAnnotationPipeline) run(ctx context.Context) (*annotator_dto.AnnotationResult, []*ast_domain.Diagnostic, error) {
	expansionResult, err := p.runExpansionStage(ctx)
	if err != nil || expansionResult == nil || expansionResult.FlattenedAST == nil {
		return p.earlyResult(expansionResult), p.diagnostics, err
	}
	if !p.options.faultTolerant && ast_domain.HasErrors(p.diagnostics) {
		return p.earlyResult(expansionResult), p.diagnostics, nil
	}

	linkingResult, err := p.runLinkingStage(ctx, expansionResult)
	if err != nil {
		return p.earlyResultFromLink(expansionResult, linkingResult), p.diagnostics, err
	}
	if !p.options.faultTolerant && ast_domain.HasErrors(p.diagnostics) {
		return p.earlyResultFromLink(expansionResult, linkingResult), p.diagnostics, nil
	}

	analysisResult, err := p.runAnnotationStage(ctx, linkingResult)
	if err != nil {
		return p.earlyResultFromAnnotation(linkingResult, analysisResult), p.diagnostics, err
	}
	if !p.options.faultTolerant && ast_domain.HasErrors(p.diagnostics) {
		return p.earlyResultFromAnnotation(linkingResult, analysisResult), p.diagnostics, nil
	}

	p.runPropDataSourceLinking(ctx, analysisResult)
	if !p.options.faultTolerant && ast_domain.HasErrors(p.diagnostics) {
		return analysisResult, p.diagnostics, nil
	}

	p.runFinalTransformations(ctx, analysisResult)
	return analysisResult, p.diagnostics, nil
}

// runExpansionStage runs the expansion stage of the annotation pipeline.
//
// Takes ctx (context.Context) which controls cancellation and timeout.
//
// Returns *annotator_dto.ExpansionResult which contains the expanded component
// tree with a flat AST structure.
// Returns error when the partial expander fails to process the component graph.
func (p *componentAnnotationPipeline) runExpansionStage(ctx context.Context) (*annotator_dto.ExpansionResult, error) {
	effectiveResolver := p.service.getEffectiveResolver(p.options)
	cssProc := p.service.cssProcessor.WithResolver(effectiveResolver)
	partialExpander := NewPartialExpander(effectiveResolver, cssProc, p.service.fsReader)
	result, diagnostics, err := partialExpander.Expand(ctx, p.componentGraph, p.vc.HashedName, p.vc.IsPage, p.vc.IsEmail)
	if err != nil {
		return nil, p.handleStageError(ctx, err, "expansion", nil)
	}
	p.diagnostics = append(p.diagnostics, diagnostics...)
	if result != nil && result.FlattenedAST != nil {
		result.FlattenedAST.Diagnostics = nil
	}
	return result, nil
}

// runLinkingStage resolves symbol references and links the expanded AST.
//
// Takes ctx (context.Context) which controls cancellation and timeout.
// Takes expansionResult (*annotator_dto.ExpansionResult) which contains the
// expanded AST to link.
//
// Returns *annotator_dto.LinkingResult which contains the linked AST.
// Returns error when linking fails.
func (p *componentAnnotationPipeline) runLinkingStage(ctx context.Context, expansionResult *annotator_dto.ExpansionResult) (*annotator_dto.LinkingResult, error) {
	componentLinker := NewComponentLinker(p.typeResolver)
	result, diagnostics, err := componentLinker.Link(ctx, expansionResult, p.virtualModule, p.vc.Source.SourcePath)
	if err != nil {
		return nil, p.handleStageError(ctx, err, "linking", expansionResult.FlattenedAST)
	}
	p.diagnostics = append(p.diagnostics, diagnostics...)
	if result != nil && result.LinkedAST != nil {
		result.LinkedAST.Diagnostics = nil
	}
	return result, nil
}

// runAnnotationStage processes the linked AST to produce annotations.
//
// Takes ctx (context.Context) which controls cancellation and timeout.
// Takes linkingResult (*annotator_dto.LinkingResult) which contains the linked
// AST to annotate.
//
// Returns *annotator_dto.AnnotationResult which contains the annotated AST.
// Returns error when annotation fails.
func (p *componentAnnotationPipeline) runAnnotationStage(ctx context.Context, linkingResult *annotator_dto.LinkingResult) (*annotator_dto.AnnotationResult, error) {
	result, diagnostics, err := Annotate(ctx, linkingResult, p.typeResolver, p.vc.Source.SourcePath, p.actions)
	if err != nil {
		return nil, p.handleStageError(ctx, err, "annotation", linkingResult.LinkedAST)
	}
	p.diagnostics = append(p.diagnostics, diagnostics...)
	if result != nil && result.AnnotatedAST != nil {
		result.AnnotatedAST.Diagnostics = nil
	}
	return result, nil
}

// runPropDataSourceLinking links prop references to their data sources.
//
// Takes ctx (context.Context) which controls cancellation and timeout.
// Takes analysisResult (*annotator_dto.AnnotationResult) which contains the
// annotated AST to process.
func (p *componentAnnotationPipeline) runPropDataSourceLinking(ctx context.Context, analysisResult *annotator_dto.AnnotationResult) {
	diagnostics := LinkAllPropDataSources(ctx, analysisResult.AnnotatedAST, p.virtualModule, p.typeResolver)
	p.diagnostics = append(p.diagnostics, diagnostics...)
	if analysisResult.AnnotatedAST != nil {
		analysisResult.AnnotatedAST.Diagnostics = nil
	}
}

// runFinalTransformations applies final changes to the annotated AST.
//
// Takes ctx (context.Context) which controls cancellation and timeout.
// Takes analysisResult (*annotator_dto.AnnotationResult) which holds the
// annotated AST to transform and receives the computed asset dependencies
// and custom tags.
func (p *componentAnnotationPipeline) runFinalTransformations(ctx context.Context, analysisResult *annotator_dto.AnnotationResult) {
	resolver := p.service.getEffectiveResolver(p.options)
	assetDeps, customTags, usesCaptcha, diagnostics := performFinalTransformations(
		ctx, analysisResult.AnnotatedAST, resolver, p.service.pathsConfig, &p.service.assetsConfig, p.service.fsReader, p.componentRegistry)
	analysisResult.AssetDependencies = assetDeps
	analysisResult.CustomTags = customTags
	analysisResult.UsesCaptcha = usesCaptcha
	p.diagnostics = append(p.diagnostics, diagnostics...)
	if analysisResult.AnnotatedAST != nil {
		p.diagnostics = append(p.diagnostics, analysisResult.AnnotatedAST.Diagnostics...)
		analysisResult.AnnotatedAST.Diagnostics = nil
	}
}

// handleStageError handles an error from a pipeline stage, wrapping
// it in strict mode or logging and returning nil in fault-tolerant
// mode so the pipeline can continue with partial results.
//
// Takes ctx (context.Context) which carries the logger.
// Takes err (error) which is the stage error to handle.
// Takes stage (string) which identifies the pipeline stage that failed.
//
// Returns error which is the wrapped error in strict mode, or nil in
// fault-tolerant mode.
func (p *componentAnnotationPipeline) handleStageError(ctx context.Context, err error, stage string, _ *ast_domain.TemplateAST) error {
	if !p.options.faultTolerant {
		return fmt.Errorf("annotator stage %q failed: %w", stage, err)
	}
	_, l := logger_domain.From(ctx, log)
	l.Error(fmt.Sprintf("Fatal error during %s, continuing with partial results", stage), logger_domain.Error(err))
	diagnostic := ast_domain.NewDiagnosticWithCode(
		ast_domain.Error, fmt.Sprintf("Fatal %s error: %v", stage, err), "",
		annotator_dto.CodeFatalAnnotationError,
		ast_domain.Location{Line: 1, Column: 1, Offset: 0}, p.vc.Source.SourcePath,
	)
	p.diagnostics = append(p.diagnostics, diagnostic)
	return nil
}

// earlyResult builds an annotation result from an expansion result.
//
// Takes expansionResult (*annotator_dto.ExpansionResult) which provides the
// flattened AST, or nil for an empty result.
//
// Returns *annotator_dto.AnnotationResult which contains the AST and virtual
// module.
func (p *componentAnnotationPipeline) earlyResult(expansionResult *annotator_dto.ExpansionResult) *annotator_dto.AnnotationResult {
	var ast *ast_domain.TemplateAST
	if expansionResult != nil {
		ast = expansionResult.FlattenedAST
	}
	return &annotator_dto.AnnotationResult{
		AnalysisMap:           nil,
		EntryPointStyleBlocks: nil,
		AnnotatedAST:          ast,
		VirtualModule:         p.virtualModule,
		StyleBlock:            "",
		ClientScript:          "",
		AssetRefs:             nil,
		CustomTags:            nil,
		UniqueInvocations:     nil,
		AssetDependencies:     nil,
	}
}

// earlyResultFromLink builds an annotation result from early pipeline stages.
//
// Takes expansionResult (*annotator_dto.ExpansionResult) which provides the
// flattened AST.
// Takes linkingResult (*annotator_dto.LinkingResult) which may provide a
// linked AST that is used instead if present.
//
// Returns *annotator_dto.AnnotationResult which contains the best available
// AST and the virtual module.
func (p *componentAnnotationPipeline) earlyResultFromLink(expansionResult *annotator_dto.ExpansionResult, linkingResult *annotator_dto.LinkingResult) *annotator_dto.AnnotationResult {
	ast := expansionResult.FlattenedAST
	if linkingResult != nil && linkingResult.LinkedAST != nil {
		ast = linkingResult.LinkedAST
	}
	return &annotator_dto.AnnotationResult{
		AnalysisMap:           nil,
		EntryPointStyleBlocks: nil,
		AnnotatedAST:          ast,
		VirtualModule:         p.virtualModule,
		StyleBlock:            "",
		ClientScript:          "",
		AssetRefs:             nil,
		CustomTags:            nil,
		UniqueInvocations:     nil,
		AssetDependencies:     nil,
	}
}

// earlyResultFromAnnotation returns the analysis result if present, or builds
// a default result from the linking result.
//
// Takes linkingResult (*annotator_dto.LinkingResult) which provides the linked
// AST used to build a default result.
// Takes analysisResult (*annotator_dto.AnnotationResult) which is returned
// directly if not nil.
//
// Returns *annotator_dto.AnnotationResult which is either the given analysis
// result or a new result built from the linking result.
func (p *componentAnnotationPipeline) earlyResultFromAnnotation(linkingResult *annotator_dto.LinkingResult, analysisResult *annotator_dto.AnnotationResult) *annotator_dto.AnnotationResult {
	if analysisResult != nil {
		return analysisResult
	}
	return &annotator_dto.AnnotationResult{
		AnalysisMap:           nil,
		EntryPointStyleBlocks: nil,
		AnnotatedAST:          linkingResult.LinkedAST,
		VirtualModule:         p.virtualModule,
		StyleBlock:            "",
		ClientScript:          "",
		AssetRefs:             nil,
		CustomTags:            nil,
		UniqueInvocations:     nil,
		AssetDependencies:     nil,
	}
}

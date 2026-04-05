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

// Applies AST transformations to template nodes including attribute processing,
// directive handling, and node restructuring. Transforms special attributes,
// processes asset references, and prepares the AST for final code generation.

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/resolver/resolver_adapters"
	"piko.sh/piko/internal/resolver/resolver_domain"
)

const (
	// actionCallModifier is the modifier value set on directives for
	// server-side action calls using the v2 action.namespace.Name() syntax.
	// The compiler reads this to generate the appropriate handler code.
	actionCallModifier = "action"

	// helperModifierName is the modifier value set on directives for
	// client-side helper functions.
	helperModifierName = "helper"

	// actionMethodAttributeName is the HTML attribute name for the HTTP
	// method used by server-side actions.
	actionMethodAttributeName = "data-pk-action-method"

	// defaultViewportWidth is the fallback width in pixels for responsive layout
	// calculations when no breakpoint is specified.
	defaultViewportWidth = 1280
)

var (
	// allowedEventModifiers defines the set of valid user-facing event modifiers
	// for p-on and p-event directives. Unknown modifiers produce a compile-time
	// error.
	allowedEventModifiers = map[string]bool{
		"prevent": true,
		"stop":    true,
		"once":    true,
		"self":    true,
		"passive": true,
		"capture": true,
	}

	// defaultResponsiveBreakpoints defines standard responsive image widths.
	defaultResponsiveBreakpoints = []int{320, 640, 768, 1024, 1280}

	// staticAssetTags maps tag names that reference static
	// asset files for dependency collection.
	staticAssetTags = map[string]bool{
		"piko:svg":     true,
		"piko:img":     true,
		"piko:picture": true,
		"pml-img":      true,
		"piko:video":   true,
	}

	// runtimeProcessingTags maps tag names that require runtime attribute processing.
	runtimeProcessingTags = map[string]bool{
		"piko:svg":     true,
		"piko:a":       true,
		"piko:img":     true,
		"piko:picture": true,
		"piko:video":   true,
		"piko:element": true,
	}
)

// assetCollectionContext holds the state needed during static asset dependency
// collection.
type assetCollectionContext struct {
	// assetsConfig holds asset profiles and responsive image settings.
	assetsConfig *config.AssetsConfig

	// resolver finds the full path to asset files.
	resolver resolver_domain.ResolverPort

	// fsReader reads asset files from the file system.
	fsReader FSReaderPort

	// baseDir is the path to the root folder for asset files.
	baseDir string

	// assetsDir is the path to the assets folder.
	assetsDir string

	// originComponentPath is the file path of the component that owns this asset.
	originComponentPath string

	// dependencies holds the static asset dependencies found during collection.
	dependencies []*annotator_dto.StaticAssetDependency

	// diagnostics holds warning messages found during processing.
	diagnostics []*ast_domain.Diagnostic
}

// processAssetNode handles a single node during asset dependency collection.
//
// Takes ctx (context.Context) which controls cancellation and timeout.
// Takes node (*ast_domain.TemplateNode) which is the AST node to process.
//
// Returns bool which shows whether to continue traversal.
func (ac *assetCollectionContext) processAssetNode(ctx context.Context, node *ast_domain.TemplateNode) bool {
	if node.NodeType != ast_domain.NodeElement {
		return true
	}
	if _, isStaticAsset := staticAssetTags[node.TagName]; !isStaticAsset {
		return true
	}

	ac.expandProfileIfPresent(ctx, node)

	if ac.hasDynamicSrc(node) {
		return true
	}

	staticSrc, isStatic := node.GetAttribute(attributeSrc)
	if !isStatic || staticSrc == "" {
		return true
	}

	expandedPath, ok := ac.validateAssetExists(ctx, node, staticSrc)
	if !ok {
		return true
	}

	if expandedPath != staticSrc {
		ac.updateNodeSrcAttribute(node, expandedPath)
	}

	dependency := ac.buildDependency(node, expandedPath)
	ac.validateAndEnrichResponsiveImage(ctx, node, dependency, staticSrc)
	ac.dependencies = append(ac.dependencies, dependency)

	if node.TagName == "piko:video" {
		ac.processPosterAttribute(ctx, node)
	}

	return true
}

// expandProfileIfPresent expands a profile attribute into explicit
// transformation parameters.
//
// Takes ctx (context.Context) which controls cancellation and timeout.
// Takes node (*ast_domain.TemplateNode) which is the template node to check
// for a profile attribute.
func (ac *assetCollectionContext) expandProfileIfPresent(ctx context.Context, node *ast_domain.TemplateNode) {
	profileName, hasProfile := node.GetAttribute("profile")
	if !hasProfile {
		return
	}

	_, l := logger_domain.From(ctx, log)
	l.Trace("Expanding profile on asset tag",
		logger_domain.String(logKeyTag, node.TagName),
		logger_domain.String(logKeyProfile, profileName),
	)

	var profileDef []config.AssetTransformationStep
	var profileFound bool
	var assetType string

	switch node.TagName {
	case "piko:video":
		profileDef, profileFound = ac.assetsConfig.Video.Profiles[profileName]
		assetType = "Video"
	default:
		profileDef, profileFound = ac.assetsConfig.Image.Profiles[profileName]
		assetType = "Image"
	}

	if !profileFound {
		ac.addDiagnostic(node, fmt.Sprintf(
			"%s profile '%s' not found in configuration. Tag will be processed with explicit attributes only.",
			assetType, profileName,
		), annotator_dto.CodeAssetProfileError)
		return
	}

	if len(profileDef) == 0 {
		ac.addDiagnostic(node, fmt.Sprintf(
			"%s profile '%s' is defined but has no capabilities. Tag will be processed with explicit attributes only.",
			assetType, profileName,
		), annotator_dto.CodeAssetProfileError)
		return
	}

	ac.mergeProfileAttributes(ctx, node, profileDef, profileName)
}

// mergeProfileAttributes combines profile settings with tag attributes.
//
// Takes ctx (context.Context) which controls cancellation and timeout.
// Takes node (*ast_domain.TemplateNode) which is the template node to update.
// Takes profileDef ([]config.AssetTransformationStep) which provides the
// profile settings to merge.
// Takes profileName (string) which identifies the profile for logging.
func (*assetCollectionContext) mergeProfileAttributes(
	ctx context.Context,
	node *ast_domain.TemplateNode,
	profileDef []config.AssetTransformationStep,
	profileName string,
) {
	_, l := logger_domain.From(ctx, log)
	mergedAttrs := make(map[string]string)

	for key, value := range profileDef[0].Params {
		mergedAttrs[strings.ToLower(key)] = value
	}

	for i := range node.Attributes {
		attr := &node.Attributes[i]
		mergedAttrs[strings.ToLower(attr.Name)] = attr.Value
	}

	delete(mergedAttrs, "profile")

	newAttrs := make([]ast_domain.HTMLAttribute, 0, len(mergedAttrs))
	for name, value := range mergedAttrs {
		newAttrs = append(newAttrs, ast_domain.HTMLAttribute{
			Name:           name,
			Value:          value,
			Location:       ast_domain.Location{Line: 0, Column: 0, Offset: 0},
			NameLocation:   ast_domain.Location{Line: 0, Column: 0, Offset: 0},
			AttributeRange: ast_domain.Range{Start: ast_domain.Location{Line: 0, Column: 0, Offset: 0}, End: ast_domain.Location{Line: 0, Column: 0, Offset: 0}},
		})
	}
	slices.SortFunc(newAttrs, func(a, b ast_domain.HTMLAttribute) int {
		return cmp.Compare(a.Name, b.Name)
	})
	node.Attributes = newAttrs

	l.Trace("Profile expanded successfully",
		logger_domain.String(logKeyTag, node.TagName),
		logger_domain.String(logKeyProfile, profileName),
		logger_domain.Int("final_attr_count", len(newAttrs)),
	)
}

// hasDynamicSrc checks if the node has a dynamic :src binding.
//
// Takes node (*ast_domain.TemplateNode) which is the template node to check.
//
// Returns bool which is true if the node has a dynamic src attribute.
func (*assetCollectionContext) hasDynamicSrc(node *ast_domain.TemplateNode) bool {
	for i := range node.DynamicAttributes {
		if node.DynamicAttributes[i].Name == attributeSrc {
			return true
		}
	}
	return false
}

// validateAssetExists checks if the asset file exists on the filesystem.
// It uses the resolver to convert the module-absolute path (or @ alias) to a
// filesystem path.
//
// Takes ctx (context.Context) which controls cancellation and timeout.
// Takes node (*ast_domain.TemplateNode) which identifies the template location
// for diagnostic reporting.
// Takes staticSrc (string) which is the asset path, possibly with @ alias.
//
// Returns string which is the expanded module-absolute path with @ alias
// resolved, for use as the dependency's SourcePath in registry lookups.
// Returns bool which indicates whether validation succeeded.
func (ac *assetCollectionContext) validateAssetExists(ctx context.Context, node *ast_domain.TemplateNode, staticSrc string) (string, bool) {
	expandedPath, err := resolver_adapters.ExpandModuleAlias(staticSrc, ac.originComponentPath)
	if err != nil {
		ac.addDiagnostic(node, fmt.Sprintf(
			"Cannot expand module alias in asset path: %s (%v)",
			staticSrc, err,
		), annotator_dto.CodeAssetResolutionError)
		return "", false
	}

	assetFullPath, err := ac.resolver.ResolveAssetPath(ctx, expandedPath, ac.originComponentPath)
	if err != nil {
		ac.addDiagnostic(node, fmt.Sprintf(
			"Invalid asset path: %s (%v)",
			staticSrc, err,
		), annotator_dto.CodeAssetResolutionError)
		return "", false
	}

	if _, err := ac.fsReader.ReadFile(ctx, assetFullPath); err != nil {
		ac.addDiagnostic(node, fmt.Sprintf(
			"Static asset not found at path: %s (resolved to: %s)",
			staticSrc, assetFullPath,
		), annotator_dto.CodeAssetResolutionError)
		return "", false
	}
	return expandedPath, true
}

// updateNodeSrcAttribute sets the src attribute value in an AST node.
// This replaces @ aliases with full module paths during the build phase, so
// the built AST contains correct paths for use at runtime.
//
// Takes node (*ast_domain.TemplateNode) which is the node to update.
// Takes newSrc (string) which is the new source path value.
func (*assetCollectionContext) updateNodeSrcAttribute(node *ast_domain.TemplateNode, newSrc string) {
	for i := range node.Attributes {
		if strings.EqualFold(node.Attributes[i].Name, attributeSrc) {
			node.Attributes[i].Value = newSrc
			return
		}
	}
}

// processPosterAttribute handles the poster attribute for piko:video elements,
// collecting it as an image dependency for analysis and processing.
//
// Takes ctx (context.Context) which controls cancellation and timeout.
// Takes node (*ast_domain.TemplateNode) which is the piko:video element to
// check for a poster attribute.
func (ac *assetCollectionContext) processPosterAttribute(ctx context.Context, node *ast_domain.TemplateNode) {
	posterSrc, hasPoster := node.GetAttribute("poster")
	if !hasPoster || posterSrc == "" {
		return
	}

	if isExternalURL(posterSrc) {
		return
	}

	expandedPath, ok := ac.validateAssetExists(ctx, node, posterSrc)
	if !ok {
		return
	}

	if expandedPath != posterSrc {
		ac.updatePosterAttribute(node, expandedPath)
	}

	dependency := ac.buildPosterDependency(node, expandedPath)
	ac.dependencies = append(ac.dependencies, dependency)

	_, l := logger_domain.From(ctx, log)
	l.Trace("Collected poster dependency for piko:video",
		logger_domain.String(logKeySrc, expandedPath),
	)
}

// updatePosterAttribute sets the poster attribute value on a template node.
//
// Takes node (*ast_domain.TemplateNode) which is the node to update.
// Takes newPoster (string) which is the new poster path to set.
func (*assetCollectionContext) updatePosterAttribute(node *ast_domain.TemplateNode, newPoster string) {
	for i := range node.Attributes {
		if strings.EqualFold(node.Attributes[i].Name, "poster") {
			node.Attributes[i].Value = newPoster
			return
		}
	}
}

// buildPosterDependency creates a static asset dependency for a video poster.
// The poster is treated as an image asset with poster-specific settings.
//
// Takes node (*ast_domain.TemplateNode) which is the piko:video element.
// Takes posterSrc (string) which is the path to the poster image.
//
// Returns *annotator_dto.StaticAssetDependency which is the configured poster
// dependency.
func (ac *assetCollectionContext) buildPosterDependency(
	node *ast_domain.TemplateNode,
	posterSrc string,
) *annotator_dto.StaticAssetDependency {
	dependency := &annotator_dto.StaticAssetDependency{
		SourcePath:           posterSrc,
		AssetType:            "img",
		TransformationParams: make(map[string]string),
		OriginComponentPath:  ac.originComponentPath,
		Location:             node.Location,
	}

	posterAttrMap := map[string]string{
		"poster-widths":    "widths",
		"poster-width":     "widths",
		"poster-formats":   "formats",
		"poster-format":    "formats",
		"poster-densities": "densities",
		"poster-density":   "densities",
		"poster-sizes":     "sizes",
	}

	for i := range node.Attributes {
		attr := &node.Attributes[i]
		if targetName, ok := posterAttrMap[strings.ToLower(attr.Name)]; ok {
			dependency.TransformationParams[targetName] = attr.Value
		}
	}

	return dependency
}

// buildDependency creates a static asset dependency from a template node.
//
// Takes node (*ast_domain.TemplateNode) which provides the template node to
// convert.
// Takes staticSrc (string) which specifies the source path for the asset.
//
// Returns *annotator_dto.StaticAssetDependency which contains the dependency
// with settings copied from the node attributes.
func (ac *assetCollectionContext) buildDependency(
	node *ast_domain.TemplateNode,
	staticSrc string,
) *annotator_dto.StaticAssetDependency {
	dependency := &annotator_dto.StaticAssetDependency{
		SourcePath:           staticSrc,
		AssetType:            strings.TrimPrefix(node.TagName, "piko:"),
		TransformationParams: make(map[string]string),
		OriginComponentPath:  ac.originComponentPath,
		Location:             node.Location,
	}

	for i := range node.Attributes {
		attr := &node.Attributes[i]
		if attr.Name != attributeSrc {
			dependency.TransformationParams[attr.Name] = attr.Value
		}
	}

	return dependency
}

// validateAndEnrichResponsiveImage checks responsive image attributes and sets
// default values.
//
// Takes ctx (context.Context) which controls cancellation and timeout.
// Takes node (*ast_domain.TemplateNode) which is the template node to check.
// Takes dependency (*annotator_dto.StaticAssetDependency) which holds the image
// settings.
// Takes staticSrc (string) which is the source path used for logging.
func (ac *assetCollectionContext) validateAndEnrichResponsiveImage(
	ctx context.Context,
	node *ast_domain.TemplateNode,
	dependency *annotator_dto.StaticAssetDependency,
	staticSrc string,
) {
	_, hasWidth := dependency.TransformationParams["width"]
	densitiesString, hasDensities := dependency.TransformationParams[transformKeyDensities]
	_, hasSizes := dependency.TransformationParams["sizes"]

	if !hasDensities && !hasSizes {
		return
	}

	dependency.TransformationParams[transformKeyResponsive] = "true"

	if hasDensities {
		ac.validateDensitiesFormat(node, densitiesString)
	}

	if hasDensities && !hasSizes && !hasWidth {
		ac.addDiagnostic(node,
			"Responsive image with 'densities' attribute should also have 'width' or 'sizes' attribute",
			annotator_dto.CodeAssetResolutionError,
		)
	}

	_, l := logger_domain.From(ctx, log)
	l.Trace("Marked image as responsive",
		logger_domain.String(logKeyTag, node.TagName),
		logger_domain.String(logKeySrc, staticSrc),
		logger_domain.Bool("has_densities", hasDensities),
		logger_domain.Bool("has_sizes", hasSizes),
	)

	ac.applyDefaultDensitiesIfNeeded(ctx, node, dependency, staticSrc, hasSizes, hasDensities)
}

// validateDensitiesFormat checks that each density value has a valid format.
//
// Takes node (*ast_domain.TemplateNode) which is the node to report errors on.
// Takes densitiesString (string) which holds the density values to check.
func (ac *assetCollectionContext) validateDensitiesFormat(node *ast_domain.TemplateNode, densitiesString string) {
	for density := range strings.FieldsSeq(densitiesString) {
		if parseDensity(density) <= 0 {
			ac.addDiagnostic(node, fmt.Sprintf(
				"Invalid density value '%s' in densities attribute. Expected format: 'x1', 'x2', '2x', etc.",
				density,
			), annotator_dto.CodeAssetResolutionError)
		}
	}
}

// applyDefaultDensitiesIfNeeded adds default density values to responsive
// images that have sizes but no densities set.
//
// Takes ctx (context.Context) which controls cancellation and timeout.
// Takes node (*ast_domain.TemplateNode) which is the template node being
// processed.
// Takes dependency (*annotator_dto.StaticAssetDependency) which receives the
// density settings.
// Takes staticSrc (string) which is the source path used for logging.
// Takes hasSizes (bool) which indicates whether the image has sizes set.
// Takes hasDensities (bool) which indicates whether densities are already set.
func (ac *assetCollectionContext) applyDefaultDensitiesIfNeeded(
	ctx context.Context,
	node *ast_domain.TemplateNode,
	dependency *annotator_dto.StaticAssetDependency,
	staticSrc string,
	hasSizes, hasDensities bool,
) {
	if !hasSizes || hasDensities {
		return
	}

	defaultDensities := ac.assetsConfig.Image.DefaultDensities
	if len(defaultDensities) == 0 {
		return
	}

	dependency.TransformationParams[transformKeyDensities] = strings.Join(defaultDensities, " ")
	dependency.TransformationParams[transformKeyResponsive] = "true"

	_, l := logger_domain.From(ctx, log)
	l.Trace("Applied default densities to responsive image",
		logger_domain.String(logKeyTag, node.TagName),
		logger_domain.String(logKeySrc, staticSrc),
		logger_domain.String(logKeyDensities, dependency.TransformationParams[transformKeyDensities]),
	)
}

// addDiagnostic adds a warning diagnostic for the given node.
//
// Takes node (*ast_domain.TemplateNode) which specifies the location and tag
// for the diagnostic.
// Takes message (string) which provides the warning text to display.
// Takes code (string) which identifies the diagnostic type for tooling.
func (ac *assetCollectionContext) addDiagnostic(node *ast_domain.TemplateNode, message, code string) {
	sourcePath := ac.originComponentPath
	if node.GoAnnotations != nil && node.GoAnnotations.OriginalSourcePath != nil {
		sourcePath = *node.GoAnnotations.OriginalSourcePath
	}
	ac.diagnostics = append(ac.diagnostics, ast_domain.NewDiagnosticWithCode(
		ast_domain.Warning,
		message,
		node.TagName,
		code,
		node.Location,
		sourcePath,
	))
}

// pageCapabilities captures which frontend capabilities a page requires,
// derived from the elements and directives present in its template.
type pageCapabilities struct {
	hasNav      bool
	hasActions  bool
	hasPartials bool
	hasForms    bool
}

// allDetected reports whether every capability flag has been set, allowing
// the detection walk to terminate early once nothing more can be learned.
func (c pageCapabilities) allDetected() bool {
	return c.hasNav && c.hasActions && c.hasPartials && c.hasForms
}

// performFinalTransformations runs all transformation passes after semantic
// analysis. It adds optimisation flags and runtime metadata to the AST.
//
// Takes templateAst (*ast_domain.TemplateAST) which is the AST to transform.
// Takes resolver (resolver_domain.ResolverPort) which resolves asset paths.
// Takes pathsConfig (AnnotatorPathsConfig) which provides path settings.
// Takes assetsConfig (*config.AssetsConfig) which provides asset settings.
// Takes fsReader (FSReaderPort) which reads files from the filesystem.
// Takes componentRegistry (ComponentRegistryPort) which provides component
// metadata for custom tag collection.
//
// Returns []*annotator_dto.StaticAssetDependency which contains the expanded
// static asset dependencies.
// Returns []string which contains the custom tags found in the template.
// Returns pageCapabilities which indicates which frontend capabilities the
// page requires.
// Returns []*ast_domain.Diagnostic which contains any diagnostics collected
// during transformation.
func performFinalTransformations(
	ctx context.Context,
	templateAst *ast_domain.TemplateAST,
	resolver resolver_domain.ResolverPort,
	pathsConfig AnnotatorPathsConfig,
	assetsConfig *config.AssetsConfig,
	fsReader FSReaderPort,
	componentRegistry ComponentRegistryPort,
) ([]*annotator_dto.StaticAssetDependency, []string, pageCapabilities, []*ast_domain.Diagnostic) {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "performFinalTransformations")
	defer span.End()

	if templateAst == nil {
		l.Trace("Template AST is nil, nothing to transform")
		return nil, nil, pageCapabilities{}, nil
	}

	l.Internal("Starting Static Analysis pass...")
	performStaticAnalysis(ctx, templateAst)
	l.Internal("Finished Static Analysis pass.")

	l.Internal("Starting Custom Tag Collection pass...")
	customTags := collectCustomTags(templateAst, componentRegistry)
	l.Internal("Finished Custom Tag Collection pass.", logger_domain.Int("tags_found", len(customTags)))

	l.Internal("Starting Page Capability Detection pass...")
	capabilities := detectPageCapabilities(templateAst)
	l.Internal("Finished Page Capability Detection pass.")

	l.Internal("Starting Static Asset Dependency Collection pass...")
	dependencies, diagnostics := collectStaticAssetDependencies(ctx, templateAst, resolver, pathsConfig, assetsConfig, fsReader)
	l.Internal("Finished Static Asset Dependency Collection pass.", logger_domain.Int("dependencies_found", len(dependencies)))

	l.Internal("Starting Responsive Asset Expansion pass...")
	expandedDeps := expandResponsiveAssets(dependencies, assetsConfig)
	l.Internal("Finished Responsive Asset Expansion pass.",
		logger_domain.Int("original_count", len(dependencies)),
		logger_domain.Int("expanded_count", len(expandedDeps)))

	return expandedDeps, customTags, capabilities, diagnostics
}

// performStaticAnalysis walks the AST in post-order to mark which nodes are
// static and which can be fully prerendered.
//
// Takes templateAst (*ast_domain.TemplateAST) which is the AST to analyse.
func performStaticAnalysis(_ context.Context, templateAst *ast_domain.TemplateAST) {
	iterator := templateAst.NewPostOrderIterator()
	for iterator.Next() {
		analyseNodeForStaticity(iterator.Node)
	}

	iterator = templateAst.NewPostOrderIterator()
	for iterator.Next() {
		analyseNodeForPrerenderability(iterator.Node)
	}
}

// analyseNodeForStaticity checks if a node is static and stores the result in
// its annotation.
//
// A node is structurally static when it has no dynamic features. A node is
// fully static when it is structurally static, has no structural or presence
// directives, and all its children are also static.
//
// Takes node (*ast_domain.TemplateNode) which is the node to check.
func analyseNodeForStaticity(node *ast_domain.TemplateNode) {
	if node.GoAnnotations == nil {
		node.GoAnnotations = &ast_domain.GoGeneratorAnnotation{}
	}

	isStructurallyStatic := !hasDynamicFeatures(node)
	node.GoAnnotations.IsStructurallyStatic = isStructurallyStatic

	isSemanticallyStatic := isStructurallyStatic && !hasStructuralOrPresenceDirectives(node)
	if isSemanticallyStatic {
		for _, child := range node.Children {
			if child.GoAnnotations == nil || !child.GoAnnotations.IsStatic {
				isSemanticallyStatic = false
				break
			}
		}
	}
	node.GoAnnotations.IsStatic = isSemanticallyStatic
}

// analyseNodeForPrerenderability checks whether a static node and all its
// children can be converted to HTML bytes at generation time.
//
// A node is fully prerenderable only if:
//   - It is marked IsStatic (no dynamic features or structural directives).
//   - Its TagName is not in runtimeProcessingTags (piko:svg, piko:img, etc.).
//   - It has no partial invocation.
//   - All its children are also fully prerenderable.
//
// This function must be called after analyseNodeForStaticity sets IsStatic.
//
// Takes node (*ast_domain.TemplateNode) which is the node to check.
func analyseNodeForPrerenderability(node *ast_domain.TemplateNode) {
	if node.GoAnnotations == nil || !node.GoAnnotations.IsStatic {
		return
	}

	if runtimeProcessingTags[node.TagName] {
		return
	}
	if nodeHasPartialInvocation(node) {
		return
	}

	for _, child := range node.Children {
		if child.GoAnnotations == nil || !child.GoAnnotations.IsFullyPrerenderable {
			return
		}
	}

	node.GoAnnotations.IsFullyPrerenderable = true
}

// hasStructuralOrPresenceDirectives reports whether the node has any
// structural or presence directives attached.
//
// Takes node (*ast_domain.TemplateNode) which is the template node to check.
//
// Returns bool which is true if the node has a for, if, else-if, or else
// directive.
func hasStructuralOrPresenceDirectives(node *ast_domain.TemplateNode) bool {
	return node.DirFor != nil ||
		node.DirIf != nil ||
		node.DirElseIf != nil ||
		node.DirElse != nil
}

// hasDynamicFeatures checks whether a node has any feature that prevents it
// from being static.
//
// Takes node (*ast_domain.TemplateNode) which is the template node to check.
//
// Returns bool which is true if the node has dynamic keys, runtime-processed
// elements, dynamic text content, rendering directives, or dynamic bindings.
func hasDynamicFeatures(node *ast_domain.TemplateNode) bool {
	return hasDynamicKey(node) ||
		isRuntimeProcessedElement(node) ||
		hasDynamicTextContent(node) ||
		hasRenderingDirectives(node) ||
		hasDynamicBindings(node)
}

// hasDynamicKey reports whether the template node has a dynamic key.
//
// Takes node (*ast_domain.TemplateNode) which is the node to check.
//
// Returns bool which is true if the node has a key that is not static.
func hasDynamicKey(node *ast_domain.TemplateNode) bool {
	return node.Key != nil && !isStaticKey(node.Key)
}

// isRuntimeProcessedElement checks whether a template node needs processing at
// runtime.
//
// Takes node (*ast_domain.TemplateNode) which is the node to check.
//
// Returns bool which is true if the node is an element or fragment that has a
// runtime processing tag or contains a partial invocation.
func isRuntimeProcessedElement(node *ast_domain.TemplateNode) bool {
	if node.NodeType != ast_domain.NodeElement && node.NodeType != ast_domain.NodeFragment {
		return false
	}
	return runtimeProcessingTags[node.TagName] || nodeHasPartialInvocation(node)
}

// nodeHasPartialInvocation checks whether the node has a partial invocation.
//
// Takes node (*ast_domain.TemplateNode) which is the node to check.
//
// Returns bool which is true if the node has partial info annotations.
func nodeHasPartialInvocation(node *ast_domain.TemplateNode) bool {
	return node.GoAnnotations != nil && node.GoAnnotations.PartialInfo != nil
}

// hasDynamicTextContent reports whether the node contains dynamic text.
//
// Takes node (*ast_domain.TemplateNode) which is the template node to check.
//
// Returns bool which is true if the node is a text node with rich text.
func hasDynamicTextContent(node *ast_domain.TemplateNode) bool {
	return node.NodeType == ast_domain.NodeText && len(node.RichText) > 0
}

// hasRenderingDirectives reports whether the node has any rendering directives.
//
// Takes node (*ast_domain.TemplateNode) which is the template node to check.
//
// Returns bool which is true if the node has a text, HTML, class, style, or
// show directive.
func hasRenderingDirectives(node *ast_domain.TemplateNode) bool {
	return node.DirText != nil ||
		node.DirHTML != nil ||
		node.DirClass != nil ||
		node.DirStyle != nil ||
		node.DirShow != nil
}

// hasDynamicBindings checks whether a template node has any dynamic bindings.
//
// Event handlers are only considered dynamic if their expressions reference
// template scope variables (IsStaticEvent == false). Static event handlers
// like p-on:click="handleClick" or p-on:click="doSomething($event)" can be
// hoisted because they do not depend on runtime template values.
//
// Takes node (*ast_domain.TemplateNode) which is the node to check.
//
// Returns bool which is true if the node has dynamic attributes or non-static
// event handlers.
func hasDynamicBindings(node *ast_domain.TemplateNode) bool {
	if len(node.DynamicAttributes) > 0 {
		return true
	}
	if hasNonStaticEvents(node.OnEvents) {
		return true
	}
	if hasNonStaticEvents(node.CustomEvents) {
		return true
	}
	return false
}

// hasNonStaticEvents checks if any event directive in the map is dynamic.
//
// Takes events (map[string][]ast_domain.Directive) which is the event map to
// check.
//
// Returns bool which is true if any event is not a static event.
func hasNonStaticEvents(events map[string][]ast_domain.Directive) bool {
	for _, directives := range events {
		for i := range directives {
			if !directives[i].IsStaticEvent {
				return true
			}
		}
	}
	return false
}

// isStaticKey checks whether a key expression is a static string literal.
//
// Takes key (ast_domain.Expression) which is the expression to check.
//
// Returns bool which is true if the key is a string literal.
func isStaticKey(key ast_domain.Expression) bool {
	_, isStringLit := key.(*ast_domain.StringLiteral)
	return isStringLit
}

// collectStaticAssetDependencies walks the template AST to find all piko:*
// tags with a static src attribute, checks that each asset file exists, and
// returns all valid dependencies found.
//
// Takes templateAST (*ast_domain.TemplateAST) which is the parsed template to
// scan for static asset references.
// Takes resolver (resolver_domain.ResolverPort) which resolves asset paths.
// Takes pathsConfig (AnnotatorPathsConfig) which provides path settings.
// Takes assetsConfig (*config.AssetsConfig) which provides asset profiles and
// responsive image settings.
// Takes fsReader (FSReaderPort) which checks whether asset files exist.
//
// Returns []*annotator_dto.StaticAssetDependency which contains all valid
// static asset dependencies found in the template.
// Returns []*ast_domain.Diagnostic which contains errors for missing or
// invalid asset references.
func collectStaticAssetDependencies(
	ctx context.Context,
	templateAST *ast_domain.TemplateAST,
	resolver resolver_domain.ResolverPort,
	pathsConfig AnnotatorPathsConfig,
	assetsConfig *config.AssetsConfig,
	fsReader FSReaderPort,
) ([]*annotator_dto.StaticAssetDependency, []*ast_domain.Diagnostic) {
	if templateAST == nil {
		return nil, nil
	}

	originComponentPath := ""
	if templateAST.SourcePath != nil {
		originComponentPath = *templateAST.SourcePath
	}

	ac := &assetCollectionContext{
		assetsConfig:        assetsConfig,
		resolver:            resolver,
		fsReader:            fsReader,
		baseDir:             resolver.GetBaseDir(),
		assetsDir:           pathsConfig.AssetsSourceDir,
		originComponentPath: originComponentPath,
		dependencies:        nil,
		diagnostics:         nil,
	}

	templateAST.Walk(func(node *ast_domain.TemplateNode) bool {
		return ac.processAssetNode(ctx, node)
	})

	return ac.dependencies, ac.diagnostics
}

// isExternalURL checks whether a URL points to an external resource.
// External URLs start with http://, https://, //, or data:.
//
// Takes url (string) which is the URL to check.
//
// Returns bool which is true if the URL is external.
func isExternalURL(url string) bool {
	return strings.HasPrefix(url, "http://") ||
		strings.HasPrefix(url, "https://") ||
		strings.HasPrefix(url, "//") ||
		strings.HasPrefix(url, "data:")
}

// collectCustomTags walks the given AST to find custom element tags.
//
// When a component registry is provided, only tags that are registered in the
// registry are collected. This gives reliable custom tag detection based on
// which PKC components are available.
//
// When no registry is provided (nil), the function uses a simple rule instead:
// tags with a hyphen are treated as custom elements. This keeps things working
// during the change to registry-based detection.
//
// Takes templateAST (*ast_domain.TemplateAST) which is the parsed template to
// search.
// Takes registry (ComponentRegistryPort) which provides component lookup, or
// nil to use hyphen-based detection.
//
// Returns []string which contains the sorted list of unique custom tag names,
// or nil if templateAST is nil.
func collectCustomTags(templateAST *ast_domain.TemplateAST, registry ComponentRegistryPort) []string {
	if templateAST == nil {
		return nil
	}

	uniqueTags := make(map[string]struct{})

	templateAST.Walk(func(node *ast_domain.TemplateNode) bool {
		if node.NodeType != ast_domain.NodeElement {
			return true
		}

		tagNameLower := strings.ToLower(node.TagName)

		if strings.HasPrefix(tagNameLower, "piko:") || strings.HasPrefix(tagNameLower, "pml-") {
			return true
		}

		var isCustomTag bool
		if registry != nil {
			isCustomTag = registry.IsRegistered(node.TagName)
		} else {
			isCustomTag = strings.Contains(tagNameLower, "-")
		}

		if isCustomTag {
			uniqueTags[node.TagName] = struct{}{}
		}

		return true
	})

	sortedTags := make([]string, 0, len(uniqueTags))
	for tag := range uniqueTags {
		sortedTags = append(sortedTags, tag)
	}
	slices.Sort(sortedTags)

	return sortedTags
}

// detectPageCapabilities walks the template AST and detects which frontend
// capabilities are required by this page, based on the elements and
// directives present in the template.
func detectPageCapabilities(templateAST *ast_domain.TemplateAST) pageCapabilities {
	var capabilities pageCapabilities
	if templateAST == nil {
		return capabilities
	}

	templateAST.Walk(func(node *ast_domain.TemplateNode) bool {
		if node.NodeType != ast_domain.NodeElement {
			return true
		}
		if capabilities.allDetected() {
			return false
		}
		updatePageCapabilities(&capabilities, node)
		return true
	})

	return capabilities
}

// updatePageCapabilities inspects a single element node and sets the matching
// capability flags on the accumulator. Extracted from detectPageCapabilities
// to keep the walk body small and its cognitive complexity within limits.
func updatePageCapabilities(capabilities *pageCapabilities, node *ast_domain.TemplateNode) {
	if strings.EqualFold(node.TagName, "piko:a") {
		capabilities.hasNav = true
	}
	if len(node.OnEvents) > 0 || len(node.CustomEvents) > 0 {
		capabilities.hasActions = true
	}
	if nodeHasPartialSrc(node) {
		capabilities.hasPartials = true
	}
	if node.DirModel != nil || strings.EqualFold(node.TagName, "form") {
		capabilities.hasForms = true
	}
}

// nodeHasPartialSrc reports whether the element node declares a partial_src
// attribute, which indicates the page requires the partials capability.
func nodeHasPartialSrc(node *ast_domain.TemplateNode) bool {
	for i := range node.Attributes {
		if node.Attributes[i].Name == "partial_src" {
			return true
		}
	}
	return false
}

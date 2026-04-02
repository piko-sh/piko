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

package compiler_domain

import (
	"context"
	"strconv"
	"strings"

	"piko.sh/piko/internal/assetpath"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/esbuild/js_ast"
)

const (
	// tagPikoImg is the tag name for piko:img custom elements.
	tagPikoImg = "piko:img"

	// tagPikoSvg is the tag name for piko:svg custom elements.
	tagPikoSvg = "piko:svg"

	// tagPikoPicture is the tag name for piko:picture custom elements.
	tagPikoPicture = "piko:picture"

	// tagPikoSvgInline is the custom element name that fetches and inlines SVGs at
	// runtime, enabling CSS styling of SVG internals (fill, stroke, etc.).
	tagPikoSvgInline = "piko-svg-inline"

	// attributeSrc is the name of the src attribute for image and SVG elements.
	attributeSrc = "src"

	// attributeSizes is the sizes attribute name for responsive images.
	attributeSizes = "sizes"

	// attributeDensities is the densities attribute name for pixel density support.
	attributeDensities = "densities"

	// attributeFormats is the attribute name for image format options.
	attributeFormats = "formats"

	// attributeWidths is the widths attribute name for explicit width variants.
	attributeWidths = "widths"

	// attributeSrcset is the srcset attribute name.
	attributeSrcset = "srcset"
)

// pikoImgAttrs holds the attributes taken from a piko:img element.
type pikoImgAttrs struct {
	// source is the static source URL for the image; empty if dynamically bound.
	source string

	// dynamicSource holds the AST expression for a dynamic src attribute binding;
	// nil means the source is static or not present.
	dynamicSource ast_domain.Expression

	// sizes is the HTML sizes attribute value for responsive images.
	sizes string

	// densities specifies pixel density descriptors for responsive images.
	densities string

	// formats lists image output formats as comma-separated values; defaults to
	// webp.
	formats string

	// widths specifies available image widths as a comma-separated list.
	widths string
}

// hasProfile returns true if any responsive profile attributes are set.
//
// Returns bool which is true when sizes, densities, formats, or widths is set.
func (a *pikoImgAttrs) hasProfile() bool {
	return a.sizes != "" || a.densities != "" || a.formats != "" || a.widths != ""
}

// pikoSvgAttrs holds the attributes taken from a piko:svg element.
type pikoSvgAttrs struct {
	// dynamicSource holds a binding expression for computed SVG paths.
	dynamicSource ast_domain.Expression

	// source is the static path to the SVG asset file.
	source string
}

var (
	// pikoImgExcludedAttrs contains attribute names to exclude when copying
	// piko:img attributes.
	pikoImgExcludedAttrs = map[string]bool{
		attributeSrc:       true,
		attributeSizes:     true,
		attributeDensities: true,
		attributeFormats:   true,
		attributeWidths:    true,
	}

	// pikoSvgExcludedAttrs contains attribute names to exclude when copying
	// piko:svg attributes.
	pikoSvgExcludedAttrs = map[string]bool{
		attributeSrc: true,
	}
)

// isAssetTag checks if a tag name is a special asset tag (piko:img or
// piko:svg).
//
// Takes tagName (string) which is the tag name to check.
//
// Returns bool which is true if the tag is a special asset tag.
func isAssetTag(tagName string) bool {
	lower := strings.ToLower(tagName)
	return lower == tagPikoImg || lower == tagPikoSvg || lower == tagPikoPicture
}

// isPikoImg reports whether the tag name is a piko:img element.
//
// Takes tagName (string) which is the tag name to check.
//
// Returns bool which is true if the tag is a piko:img element.
func isPikoImg(tagName string) bool {
	return strings.EqualFold(tagName, tagPikoImg)
}

// isPikoPicture reports whether the tag name is a piko:picture element.
//
// Takes tagName (string) which is the tag name to check.
//
// Returns bool which is true if the tag is a piko:picture element.
func isPikoPicture(tagName string) bool {
	return strings.EqualFold(tagName, tagPikoPicture)
}

// isPikoSvg reports whether the tag name is a piko:svg element.
//
// Takes tagName (string) which is the tag name to check.
//
// Returns bool which is true if the tag is a piko:svg element.
func isPikoSvg(tagName string) bool {
	return strings.EqualFold(tagName, tagPikoSvg)
}

// buildAssetElementNodeAST builds the JavaScript AST for a piko:img or piko:svg
// element.
//
// Takes n (*ast_domain.TemplateNode) which is the asset element node to
// process.
// Takes events (*eventBindingCollection) which collects event bindings.
// Takes keyJSExpr (js_ast.Expr) which is the key expression for the element.
// Takes loopVars (map[string]bool) which tracks loop variables in scope.
// Takes booleanProps ([]string) which lists boolean property names.
//
// Returns js_ast.Expr which is the JavaScript AST for the element.
// Returns error when building the element fails.
func buildAssetElementNodeAST(
	ctx context.Context,
	n *ast_domain.TemplateNode,
	events *eventBindingCollection,
	keyJSExpr js_ast.Expr,
	loopVars map[string]bool,
	booleanProps []string,
) (js_ast.Expr, error) {
	if isPikoImg(n.TagName) {
		return buildPikoImgAST(ctx, n, events, keyJSExpr, loopVars, booleanProps)
	}
	if isPikoPicture(n.TagName) {
		return buildPikoPictureAST(ctx, n, events, keyJSExpr, loopVars, booleanProps)
	}
	if isPikoSvg(n.TagName) {
		return buildPikoSvgAST(ctx, n, events, keyJSExpr, loopVars, booleanProps)
	}
	return buildElementNodeAST(ctx, n, events, keyJSExpr, loopVars, booleanProps)
}

// buildPikoImgAST builds the JavaScript AST for a piko:img element.
// Transforms piko:img to an img tag with transformed src and srcset attributes.
//
// Takes n (*ast_domain.TemplateNode) which is the piko:img element to process.
// Takes events (*eventBindingCollection) which collects event bindings.
// Takes keyJSExpr (js_ast.Expr) which is the key expression for the element.
// Takes loopVars (map[string]bool) which tracks loop variables in scope.
// Takes booleanProps ([]string) which lists boolean property names.
//
// Returns js_ast.Expr which is the JavaScript AST for the img element.
// Returns error when building the element fails.
func buildPikoImgAST(
	ctx context.Context,
	n *ast_domain.TemplateNode,
	events *eventBindingCollection,
	keyJSExpr js_ast.Expr,
	loopVars map[string]bool,
	booleanProps []string,
) (js_ast.Expr, error) {
	registry := events.getRegistry()
	properties := make(map[string]js_ast.Expr)
	multiValueProps := make(map[string][]js_ast.Expr)

	imgAttrs := extractPikoImgAttrs(n)
	handlePikoImgSrcAttr(ctx, imgAttrs, properties, registry)

	collectDirectiveProps(n, properties, registry)
	collectPikoImgStaticAttrs(n, properties, imgAttrs)
	collectPikoAssetDynamicAttrs(n, properties, booleanProps, registry)
	collectEventHandlers(ctx, n, events, loopVars, multiValueProps)
	mergeMultiValueProps(properties, multiValueProps)

	childrenExpr, err := buildChildFragmentAST(ctx, n, events, loopVars, booleanProps)
	if err != nil {
		return js_ast.Expr{}, err
	}

	elementCall := buildDOMCall("el",
		newStringLiteral("img"),
		keyJSExpr,
		buildPropsObject(properties),
		childrenExpr,
	)

	return elementCall, nil
}

// handlePikoImgSrcAttr handles the src attribute for piko:img elements. It
// transforms static src paths and builds srcset for responsive images, or wraps
// dynamic src expressions with runtime transformation.
//
// Takes imgAttrs (pikoImgAttrs) which contains the extracted piko:img
// attributes.
// Takes properties (map[string]js_ast.Expr) which receives the src properties.
// Takes registry (*RegistryContext) which provides compilation context.
func handlePikoImgSrcAttr(ctx context.Context, imgAttrs pikoImgAttrs, properties map[string]js_ast.Expr, registry *RegistryContext) {
	if imgAttrs.source != "" {
		transformedSrc := transformAssetSrc(ctx, imgAttrs.source)
		properties[attributeSrc] = newStringLiteral(transformedSrc)

		if imgAttrs.hasProfile() {
			if srcsetValue := buildSrcsetValue(transformedSrc, imgAttrs); srcsetValue != "" {
				properties[attributeSrcset] = newStringLiteral(srcsetValue)
			}
		}

		if imgAttrs.sizes != "" {
			properties[attributeSizes] = newStringLiteral(imgAttrs.sizes)
		}
		return
	}

	if imgAttrs.dynamicSource == nil {
		return
	}

	jsExpr, err := transformOurASTtoJSAST(imgAttrs.dynamicSource, registry)
	if err != nil || jsExpr.Data == nil {
		return
	}

	transformedExpr := buildAssetSrcTransformCall(ctx, jsExpr)
	properties[attributeSrc] = js_ast.Expr{Data: &js_ast.EUnary{Op: js_ast.UnOpPos, Value: transformedExpr}}
}

// buildPikoSvgAST builds the JavaScript AST for a piko:svg element. Transforms
// piko:svg to a piko-svg-inline custom element that fetches and inlines the SVG
// at runtime, allowing CSS styling of SVG internals.
//
// Takes n (*ast_domain.TemplateNode) which is the piko:svg element to process.
// Takes events (*eventBindingCollection) which collects event bindings.
// Takes keyJSExpr (js_ast.Expr) which is the key expression for the element.
// Takes loopVars (map[string]bool) which tracks loop variables in scope.
// Takes booleanProps ([]string) which lists boolean property names.
//
// Returns js_ast.Expr which is the JavaScript AST for the piko-svg-inline
// element.
// Returns error when building the element fails.
func buildPikoSvgAST(
	ctx context.Context,
	n *ast_domain.TemplateNode,
	events *eventBindingCollection,
	keyJSExpr js_ast.Expr,
	loopVars map[string]bool,
	booleanProps []string,
) (js_ast.Expr, error) {
	registry := events.getRegistry()
	properties := make(map[string]js_ast.Expr)
	multiValueProps := make(map[string][]js_ast.Expr)

	svgAttrs := extractPikoSvgAttrs(n)

	if svgAttrs.source != "" {
		transformedSrc := transformAssetSrc(ctx, svgAttrs.source)
		properties[attributeSrc] = newStringLiteral(transformedSrc)
	} else if svgAttrs.dynamicSource != nil {
		jsExpr, err := transformOurASTtoJSAST(svgAttrs.dynamicSource, registry)
		if err == nil && jsExpr.Data != nil {
			transformedExpr := buildAssetSrcTransformCall(ctx, jsExpr)
			properties[attributeSrc] = js_ast.Expr{Data: &js_ast.EUnary{Op: js_ast.UnOpPos, Value: transformedExpr}}
		}
	}

	collectDirectiveProps(n, properties, registry)

	collectPikoSvgStaticAttrs(n, properties, svgAttrs)

	collectPikoAssetDynamicAttrs(n, properties, booleanProps, registry)

	collectEventHandlers(ctx, n, events, loopVars, multiValueProps)

	mergeMultiValueProps(properties, multiValueProps)

	childrenExpr, err := buildChildFragmentAST(ctx, n, events, loopVars, booleanProps)
	if err != nil {
		return js_ast.Expr{}, err
	}

	elementCall := buildDOMCall("el",
		newStringLiteral(tagPikoSvgInline),
		keyJSExpr,
		buildPropsObject(properties),
		childrenExpr,
	)

	return elementCall, nil
}

// extractPikoImgAttrs gets the piko:img attributes from a template node.
//
// Takes n (*ast_domain.TemplateNode) which is the node to read attributes from.
//
// Returns pikoImgAttrs which holds the attribute values found.
func extractPikoImgAttrs(n *ast_domain.TemplateNode) pikoImgAttrs {
	var result pikoImgAttrs

	for i := range n.Attributes {
		attr := &n.Attributes[i]
		switch strings.ToLower(attr.Name) {
		case attributeSrc:
			result.source = attr.Value
		case attributeSizes:
			result.sizes = attr.Value
		case attributeDensities:
			result.densities = attr.Value
		case attributeFormats:
			result.formats = attr.Value
		case attributeWidths:
			result.widths = attr.Value
		}
	}

	if result.source == "" {
		for i := range n.DynamicAttributes {
			dynamicAttribute := &n.DynamicAttributes[i]
			if strings.EqualFold(dynamicAttribute.Name, attributeSrc) {
				result.dynamicSource = dynamicAttribute.Expression
				break
			}
		}
	}

	return result
}

// extractPikoSvgAttrs gets piko:svg attributes from a template node.
//
// Takes n (*ast_domain.TemplateNode) which is the node to get attributes from.
//
// Returns pikoSvgAttrs which holds the src value, either static or dynamic.
func extractPikoSvgAttrs(n *ast_domain.TemplateNode) pikoSvgAttrs {
	var result pikoSvgAttrs

	for i := range n.Attributes {
		attr := &n.Attributes[i]
		if strings.EqualFold(attr.Name, attributeSrc) {
			result.source = attr.Value
			break
		}
	}

	if result.source == "" {
		for i := range n.DynamicAttributes {
			dynamicAttribute := &n.DynamicAttributes[i]
			if strings.EqualFold(dynamicAttribute.Name, attributeSrc) {
				result.dynamicSource = dynamicAttribute.Expression
				break
			}
		}
	}

	return result
}

// transformAssetSrc transforms an asset source path by prepending the asset
// serve path. Delegates to assetpath.Transform with the module name from
// context.
//
// Takes ctx (context.Context) which carries the module name.
// Takes src (string) which is the original source path.
//
// Returns string which is the transformed source path.
func transformAssetSrc(ctx context.Context, src string) string {
	return assetpath.Transform(src, GetModuleName(ctx), assetpath.DefaultServePath)
}

// buildSrcsetValue builds a srcset value from responsive image attributes.
// Generates URLs with profile query parameters for on-demand variant
// generation.
//
// Takes baseSrc (string) which is the transformed base source URL.
// Takes attrs (pikoImgAttrs) which contains the responsive image attributes.
//
// Returns string which is the generated srcset value, or empty if no variants.
func buildSrcsetValue(baseSrc string, attrs pikoImgAttrs) string {
	var parts []string

	formats := []string{"webp"}
	if attrs.formats != "" {
		formats = parseCommaSeparated(attrs.formats)
	}

	var widths []int
	if attrs.widths != "" {
		widths = parseIntList(attrs.widths)
	}

	var densities []string
	if attrs.densities != "" {
		densities = parseCommaSeparated(attrs.densities)
	}

	if len(widths) > 0 {
		for _, format := range formats {
			for _, width := range widths {
				profileKey := "image_w" + strconv.Itoa(width) + "_" + format
				url := baseSrc + "?v=" + profileKey
				descriptor := strconv.Itoa(width) + "w"
				parts = append(parts, url+" "+descriptor)
			}
		}
	} else if len(densities) > 0 {
		for _, format := range formats {
			for _, density := range densities {
				profileKey := format + "@" + density
				url := baseSrc + "?v=" + profileKey
				parts = append(parts, url+" "+density)
			}
		}
	}

	return strings.Join(parts, ", ")
}

// parseCommaSeparated splits a string by commas and trims whitespace from each
// part.
//
// Takes s (string) which is the input to split.
//
// Returns []string which contains the trimmed parts, or nil if s is empty.
func parseCommaSeparated(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// parseIntList parses a comma-separated list of integers.
//
// Takes s (string) which is the input containing integers separated by commas.
//
// Returns []int which contains the parsed positive integers, skipping any
// values that are not valid or are less than one.
func parseIntList(s string) []int {
	parts := parseCommaSeparated(s)
	result := make([]int, 0, len(parts))
	for _, p := range parts {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			result = append(result, v)
		}
	}
	return result
}

// buildAssetSrcTransformCall wraps a dynamic src expression with runtime
// transformation. Generates: piko.assets.resolve(expr, moduleName).
//
// Takes srcExpr (js_ast.Expr) which is the dynamic src expression.
//
// Returns js_ast.Expr which is the wrapped transformation call.
func buildAssetSrcTransformCall(ctx context.Context, srcExpr js_ast.Expr) js_ast.Expr {
	arguments := []js_ast.Expr{srcExpr}

	if moduleName := GetModuleName(ctx); moduleName != "" {
		arguments = append(arguments, newStringLiteral(moduleName))
	}

	return js_ast.Expr{Data: &js_ast.ECall{
		Target: js_ast.Expr{Data: &js_ast.EDot{
			Target: js_ast.Expr{Data: &js_ast.EDot{
				Target: newIdentifier("piko"),
				Name:   "assets",
			}},
			Name: "resolve",
		}},
		Args: arguments,
	}}
}

// collectPikoImgStaticAttrs collects static attributes excluding piko:img
// specific ones.
//
// Takes n (*ast_domain.TemplateNode) which is the template node.
// Takes properties (map[string]js_ast.Expr) which receives collected
// attributes.
// Takes attrs (pikoImgAttrs) which contains the extracted piko:img attributes.
func collectPikoImgStaticAttrs(n *ast_domain.TemplateNode, properties map[string]js_ast.Expr, attrs pikoImgAttrs) {
	for i := range n.Attributes {
		attr := &n.Attributes[i]
		lowerName := strings.ToLower(attr.Name)
		if pikoImgExcludedAttrs[lowerName] {
			continue
		}
		if lowerName == attributeSizes && attrs.sizes != "" {
			continue
		}
		properties[attr.Name] = newStringLiteral(attr.Value)
	}
}

// collectPikoAssetDynamicAttrs collects dynamic attributes excluding src.
//
// Takes n (*ast_domain.TemplateNode) which is the template node.
// Takes properties (map[string]js_ast.Expr) which receives the collected
// attributes.
// Takes booleanProps ([]string) which lists boolean property names.
// Takes registry (*RegistryContext) which provides compilation context.
func collectPikoAssetDynamicAttrs(n *ast_domain.TemplateNode, properties map[string]js_ast.Expr, booleanProps []string, registry *RegistryContext) {
	for i := range n.DynamicAttributes {
		dynamicAttribute := &n.DynamicAttributes[i]
		if strings.EqualFold(dynamicAttribute.Name, attributeSrc) {
			continue
		}
		jsExpr, _ := transformOurASTtoJSAST(dynamicAttribute.Expression, registry)
		if jsExpr.Data == nil {
			continue
		}
		propName := dynamicAttribute.Name
		if isBooleanBound(dynamicAttribute.Expression, booleanProps) {
			propName = "?" + propName
		}
		properties[propName] = js_ast.Expr{Data: &js_ast.EUnary{Op: js_ast.UnOpPos, Value: jsExpr}}
	}
}

// collectPikoSvgStaticAttrs collects static attributes excluding piko:svg
// specific ones.
//
// Takes n (*ast_domain.TemplateNode) which is the template node.
// Takes properties (map[string]js_ast.Expr) which receives the collected
// attributes.
func collectPikoSvgStaticAttrs(n *ast_domain.TemplateNode, properties map[string]js_ast.Expr, _ pikoSvgAttrs) {
	for i := range n.Attributes {
		attr := &n.Attributes[i]
		lowerName := strings.ToLower(attr.Name)
		if pikoSvgExcludedAttrs[lowerName] {
			continue
		}
		properties[attr.Name] = newStringLiteral(attr.Value)
	}
}

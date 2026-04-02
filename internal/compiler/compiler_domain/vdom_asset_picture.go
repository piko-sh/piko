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
	"path"
	"strconv"
	"strings"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/esbuild/js_ast"
)

// buildPikoPictureAST builds the JavaScript AST for a piko:picture element.
// Transforms piko:picture into a picture element containing source elements
// for each format and a fallback img element.
//
// Takes ctx (context.Context) which carries the module name.
// Takes n (*ast_domain.TemplateNode) which is the piko:picture element to
// process.
// Takes events (*eventBindingCollection) which collects event bindings.
// Takes keyJSExpr (js_ast.Expr) which is the key expression for the element.
// Takes loopVars (map[string]bool) which tracks loop variables in scope.
// Takes booleanProps ([]string) which lists boolean property names.
//
// Returns js_ast.Expr which is the JavaScript AST for the picture element.
// Returns error when building the element fails.
func buildPikoPictureAST(
	ctx context.Context,
	n *ast_domain.TemplateNode,
	events *eventBindingCollection,
	keyJSExpr js_ast.Expr,
	loopVars map[string]bool,
	booleanProps []string,
) (js_ast.Expr, error) {
	registry := events.getRegistry()
	imgAttrs := extractPikoImgAttrs(n)

	formats := []string{"webp"}
	if imgAttrs.formats != "" {
		formats = parseCommaSeparated(imgAttrs.formats)
	}
	fallbackFormat := inferFallbackFormatCompiler(imgAttrs.source)

	var transformedSrc string
	if imgAttrs.source != "" {
		transformedSrc = transformAssetSrc(ctx, imgAttrs.source)
	}

	children := buildPictureSourceElements(transformedSrc, imgAttrs, formats)

	imgProperties := make(map[string]js_ast.Expr)
	multiValueProps := make(map[string][]js_ast.Expr)

	populateImgSrcProperties(ctx, imgAttrs, transformedSrc, fallbackFormat, imgProperties, registry)

	collectDirectiveProps(n, imgProperties, registry)
	collectPikoImgStaticAttrs(n, imgProperties, imgAttrs)
	collectPikoAssetDynamicAttrs(n, imgProperties, booleanProps, registry)
	collectEventHandlers(ctx, n, events, loopVars, multiValueProps)
	mergeMultiValueProps(imgProperties, multiValueProps)

	imgEl := buildDOMCall("el",
		newStringLiteral("img"),
		newNullLiteral(),
		buildPropsObject(imgProperties),
		newNullLiteral(),
	)
	children = append(children, imgEl)

	var childrenExpr js_ast.Expr
	if len(children) == 1 {
		childrenExpr = children[0]
	} else {
		childrenExpr = js_ast.Expr{Data: &js_ast.EArray{Items: children}}
	}

	pictureCall := buildDOMCall("el",
		newStringLiteral("picture"),
		keyJSExpr,
		newNullLiteral(),
		childrenExpr,
	)

	return pictureCall, nil
}

// buildPictureSourceElements creates <source> elements for each image format
// when the image has responsive profiles.
//
// Takes transformedSrc (string) which is the transformed source URL.
// Takes imgAttrs (pikoImgAttrs) which provides responsive image attributes.
// Takes formats ([]string) which lists the image formats to generate.
//
// Returns []js_ast.Expr which contains the source element expressions.
func buildPictureSourceElements(transformedSrc string, imgAttrs pikoImgAttrs, formats []string) []js_ast.Expr {
	if transformedSrc == "" || !imgAttrs.hasProfile() {
		return nil
	}

	var children []js_ast.Expr
	for _, format := range formats {
		srcsetValue := buildSrcsetValueForFormat(transformedSrc, imgAttrs, format)
		if srcsetValue == "" {
			continue
		}

		sourceProps := map[string]js_ast.Expr{
			"type":   newStringLiteral(formatToMIMETypeCompiler(format)),
			"srcset": newStringLiteral(srcsetValue),
		}
		if imgAttrs.sizes != "" {
			sourceProps[attributeSizes] = newStringLiteral(imgAttrs.sizes)
		}

		sourceEl := buildDOMCall("el",
			newStringLiteral("source"),
			newNullLiteral(),
			buildPropsObject(sourceProps),
			newNullLiteral(),
		)
		children = append(children, sourceEl)
	}
	return children
}

// populateImgSrcProperties sets the src and srcset properties on the img
// element based on whether the image has a static src with profiles, a plain
// static src, or a dynamic src expression.
//
// Takes ctx (context.Context) which provides the module name for asset
// transforms.
// Takes imgAttrs (pikoImgAttrs) which provides the image attributes.
// Takes transformedSrc (string) which is the transformed source URL.
// Takes fallbackFormat (string) which is the fallback image format.
// Takes imgProperties (map[string]js_ast.Expr) which receives the properties.
// Takes registry (*RegistryContext) which provides event binding state.
func populateImgSrcProperties(
	ctx context.Context,
	imgAttrs pikoImgAttrs,
	transformedSrc, fallbackFormat string,
	imgProperties map[string]js_ast.Expr,
	registry *RegistryContext,
) {
	switch {
	case imgAttrs.source != "" && imgAttrs.hasProfile():
		fallbackSrcset := buildSrcsetValueForFormat(transformedSrc, imgAttrs, fallbackFormat)
		if fallbackSrcset != "" {
			imgProperties[attributeSrcset] = newStringLiteral(fallbackSrcset)
		}
		fallbackSrc := buildFallbackSrc(transformedSrc, imgAttrs, fallbackFormat)
		imgProperties[attributeSrc] = newStringLiteral(fallbackSrc)
		if imgAttrs.sizes != "" {
			imgProperties[attributeSizes] = newStringLiteral(imgAttrs.sizes)
		}
	case imgAttrs.source != "":
		imgProperties[attributeSrc] = newStringLiteral(transformedSrc)
	case imgAttrs.dynamicSource != nil:
		jsExpr, err := transformOurASTtoJSAST(imgAttrs.dynamicSource, registry)
		if err == nil && jsExpr.Data != nil {
			transformedExpr := buildAssetSrcTransformCall(ctx, jsExpr)
			imgProperties[attributeSrc] = js_ast.Expr{Data: &js_ast.EUnary{Op: js_ast.UnOpPos, Value: transformedExpr}}
		}
	}
}

// buildSrcsetValueForFormat builds a srcset value filtered to a single image
// format.
//
// Takes baseSrc (string) which is the transformed base source URL.
// Takes attrs (pikoImgAttrs) which contains the responsive image attributes.
// Takes format (string) which is the image format to filter by.
//
// Returns string which is the generated srcset value, or empty if no variants.
func buildSrcsetValueForFormat(baseSrc string, attrs pikoImgAttrs, format string) string {
	var parts []string

	var widths []int
	if attrs.widths != "" {
		widths = parseIntList(attrs.widths)
	}

	var densities []string
	if attrs.densities != "" {
		densities = parseCommaSeparated(attrs.densities)
	}

	if len(widths) > 0 {
		for _, width := range widths {
			profileKey := "image_w" + strconv.Itoa(width) + "_" + format
			url := baseSrc + "?v=" + profileKey
			descriptor := strconv.Itoa(width) + "w"
			parts = append(parts, url+" "+descriptor)
		}
	} else if len(densities) > 0 {
		for _, density := range densities {
			profileKey := format + "@" + density
			url := baseSrc + "?v=" + profileKey
			parts = append(parts, url+" "+density)
		}
	}

	return strings.Join(parts, ", ")
}

// buildFallbackSrc returns the src URL for the fallback img element. Uses the
// largest width variant for the given format, or the base URL if no widths are
// specified.
//
// Takes baseSrc (string) which is the transformed base source URL.
// Takes attrs (pikoImgAttrs) which contains the responsive image attributes.
// Takes format (string) which is the fallback format.
//
// Returns string which is the fallback src URL.
func buildFallbackSrc(baseSrc string, attrs pikoImgAttrs, format string) string {
	if attrs.widths != "" {
		widths := parseIntList(attrs.widths)
		if len(widths) > 0 {
			maxWidth := widths[0]
			for _, w := range widths[1:] {
				if w > maxWidth {
					maxWidth = w
				}
			}
			profileKey := "image_w" + strconv.Itoa(maxWidth) + "_" + format
			return baseSrc + "?v=" + profileKey
		}
	}
	return baseSrc
}

// formatToMIMETypeCompiler maps an image format name to its MIME type for the
// compiler layer.
//
// Takes format (string) which is the format name.
//
// Returns string which is the corresponding MIME type.
func formatToMIMETypeCompiler(format string) string {
	switch format {
	case "avif":
		return "image/avif"
	case "webp":
		return "image/webp"
	case "jpg", "jpeg":
		return "image/jpeg"
	case "png":
		return "image/png"
	case "gif":
		return "image/gif"
	default:
		return "image/" + format
	}
}

// inferFallbackFormatCompiler determines the fallback image format based on
// the source file extension. Transparent formats fall back to png; all others
// fall back to jpg.
//
// Takes src (string) which is the source image path.
//
// Returns string which is the fallback format name.
func inferFallbackFormatCompiler(src string) string {
	ext := strings.ToLower(path.Ext(src))
	switch ext {
	case ".png", ".gif", ".webp":
		return "png"
	default:
		return "jpg"
	}
}

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

package pml_components

import (
	"math"
	"strconv"
	"strings"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/pml/pml_domain"
	"piko.sh/piko/internal/resolver/resolver_adapters"
)

// Image represents the <pml-img> tag and implements the Component interface.
// It renders a responsive image that works well across different email clients.
type Image struct {
	BaseComponent
}

var _ pml_domain.Component = (*Image)(nil)

// NewImage creates a new Image component. An Image renders a responsive image
// with good cross-client support.
//
// Returns *Image which is the configured component ready for use.
func NewImage() *Image {
	return &Image{
		BaseComponent: BaseComponent{},
	}
}

// TagName returns the tag name for this component.
//
// Returns string which is the PML tag name "pml-img".
func (*Image) TagName() string {
	return "pml-img"
}

// IsEndingTag returns whether this is a void element.
//
// Returns bool which is always true as Image is a void element and cannot
// contain children.
func (*Image) IsEndingTag() bool {
	return true
}

// AllowedParents returns the list of valid parent components for this component.
//
// Returns []string which contains the component names that may contain this
// image.
func (*Image) AllowedParents() []string {
	return []string{"pml-col", "pml-hero"}
}

// AllowedAttributes returns the map of valid attributes for this component.
//
// Returns map[string]pml_domain.AttributeDefinition which maps attribute names
// to their type definitions.
func (*Image) AllowedAttributes() map[string]pml_domain.AttributeDefinition {
	return map[string]pml_domain.AttributeDefinition{
		AttrSrc:                      NewAttributeDefinition(pml_domain.TypeString),
		AttrAlt:                      NewAttributeDefinition(pml_domain.TypeString),
		AttrHref:                     NewAttributeDefinition(pml_domain.TypeString),
		AttrWidth:                    NewAttributeDefinition(pml_domain.TypeUnit),
		AttrHeight:                   NewAttributeDefinition(pml_domain.TypeUnit),
		AttrMaxHeight:                NewAttributeDefinition(pml_domain.TypeUnit),
		AttrAlign:                    NewEnumAttributeDefinition([]string{ValueLeft, ValueCentre, ValueRight}),
		AttrBorder:                   NewAttributeDefinition(pml_domain.TypeString),
		AttrBorderLeft:               NewAttributeDefinition(pml_domain.TypeString),
		AttrBorderRight:              NewAttributeDefinition(pml_domain.TypeString),
		AttrBorderTop:                NewAttributeDefinition(pml_domain.TypeString),
		AttrBorderBottom:             NewAttributeDefinition(pml_domain.TypeString),
		AttrBorderRadius:             NewAttributeDefinition(pml_domain.TypeUnit),
		AttrFluidOnMobile:            NewAttributeDefinition(pml_domain.TypeBoolean),
		AttrFullWidth:                NewAttributeDefinition(pml_domain.TypeBoolean),
		AttrPadding:                  NewAttributeDefinition(pml_domain.TypeUnit),
		AttrPaddingTop:               NewAttributeDefinition(pml_domain.TypeUnit),
		AttrPaddingBottom:            NewAttributeDefinition(pml_domain.TypeUnit),
		AttrPaddingLeft:              NewAttributeDefinition(pml_domain.TypeUnit),
		AttrPaddingRight:             NewAttributeDefinition(pml_domain.TypeUnit),
		AttrContainerBackgroundColor: NewAttributeDefinition(pml_domain.TypeColor),
		CSSFontSize:                  NewAttributeDefinition(pml_domain.TypeUnit),
		AttrTarget:                   NewAttributeDefinition(pml_domain.TypeString),
		AttrTitle:                    NewAttributeDefinition(pml_domain.TypeString),
		AttrRel:                      NewAttributeDefinition(pml_domain.TypeString),
		AttrName:                     NewAttributeDefinition(pml_domain.TypeString),
		AttrSrcset:                   NewAttributeDefinition(pml_domain.TypeString),
		AttrSizes:                    NewAttributeDefinition(pml_domain.TypeString),
		AttrUsemap:                   NewAttributeDefinition(pml_domain.TypeString),
		AttrDensities:                NewAttributeDefinition(pml_domain.TypeString),
		AttrProfile:                  NewAttributeDefinition(pml_domain.TypeString),
	}
}

// DefaultAttributes returns the default attribute values for this component.
//
// Returns map[string]string which contains the default values for all
// supported image attributes such as alt text, alignment, and dimensions.
func (*Image) DefaultAttributes() map[string]string {
	return map[string]string{
		AttrAlt:     defaultImageAlt,
		AttrAlign:   defaultImageAlign,
		AttrBorder:  defaultImageBorder,
		AttrHeight:  defaultImageHeight,
		AttrPadding: defaultImagePadding,
		AttrTarget:  defaultImageTarget,
		CSSFontSize: defaultImageFontSize,
	}
}

// GetStyleTargets returns the list of style targets for this component.
//
// Returns []pml_domain.StyleTarget which maps style properties to their
// target elements, distinguishing between container and image targets.
func (*Image) GetStyleTargets() []pml_domain.StyleTarget {
	return []pml_domain.StyleTarget{
		{Property: AttrWidth, Target: TargetContainer},
		{Property: AttrHeight, Target: TargetImage},
		{Property: AttrMaxHeight, Target: TargetImage},
		{Property: AttrBorder, Target: TargetImage},
		{Property: AttrBorderRadius, Target: TargetImage},
		{Property: AttrPadding, Target: TargetContainer},
		{Property: AttrAlign, Target: TargetContainer},
		{Property: CSSFontSize, Target: TargetImage},
	}
}

// Transform converts a <pml-img> node into its final, email-safe HTML structure.
//
// A simple <img> tag is not sufficient for reliable rendering in all email
// clients. Instead, a table-based structure is generated to control alignment,
// spacing, and responsive behaviour.
//
// The transformation implements the following patterns:
//
//  1. Table-based Structure for Alignment and Padding:
//     The <img> tag is wrapped in a <table><tr><td>...</td></tr></table>
//     structure. The align attribute from <pml-img> is applied to the outer
//     <table>, which reliably centres or aligns the image block within its
//     parent column. The padding attributes are applied to the <td>, creating
//     consistent spacing around the image. The width attribute is applied to
//     the <td> to enforce the container size, especially in Outlook. The
//     container-background-colour attribute applies a background colour to the
//     <td> element.
//
//  2. Responsive Image Behaviour:
//     The <img> tag itself is given style="width: 100%; height: auto;
//     display: block;". This makes the image fluid, causing it to scale to
//     the width of its containing <td>. The width attribute on the <img> tag
//     is set to the calculated pixel width (without the "px" unit), which
//     acts as a necessary fallback for Outlook.
//
//  3. Fluid on Mobile Feature:
//     If fluid-on-mobile="true", a special CSS class (pml-fluid-mobile) is
//     added to the wrapping <table> and <td>. A corresponding media query is
//     generated in the document <head> to override any fixed desktop width on
//     mobile devices.
//
//  4. Full Width Mode:
//     If full-width="true", the image stretches to fill its container
//     completely, ignoring the specified width. In this mode, min-width and
//     max-width of 100% are applied to the <img> tag and <table> element.
//
//  5. Border Styling:
//     Supports both unified border and directional border properties for
//     granular control over image borders.
//
//  6. Linking:
//     If an href attribute is present, the <img> tag is wrapped in an <a>
//     tag with all relevant link attributes applied.
//
//  7. Piko Directive Preservation:
//     All p-* directives from the <pml-img> tag are transferred to the
//     outermost <table> element.
//
// Takes node (*ast_domain.TemplateNode) which is the <pml-img> node to
// transform.
// Takes ctx (*pml_domain.TransformationContext) which provides the
// transformation context including style manager and container width.
//
// Returns *ast_domain.TemplateNode which is the transformed table-based HTML
// structure.
// Returns []*pml_domain.Error which contains any diagnostics collected during
// transformation.
func (c *Image) Transform(node *ast_domain.TemplateNode, ctx *pml_domain.TransformationContext) (*ast_domain.TemplateNode, []*pml_domain.Error) {
	styles := ctx.StyleManager

	contentWidth := c.getContentWidth(styles, ctx.ContainerWidth)
	widthPx := mustParsePixels(contentWidth)
	isFluidOnMobile := mustGetStyle(styles, AttrFluidOnMobile) == ValueTrue
	isFullWidth := mustGetStyle(styles, AttrFullWidth) == ValueTrue

	imgStyles := c.getImgStyles(styles, isFullWidth)
	tdStyles := c.getTdStyles(styles, contentWidth, isFullWidth)
	tableStyles := c.getTableStyles(contentWidth, isFullWidth)

	finalSrc := handleEmailAssetRegistration(styles, widthPx, ctx)

	imgAttrs := buildImageAttributes(styles, finalSrc, widthPx, imgStyles)
	imgNode := NewElementNode(ElementImg, imgAttrs, nil)

	finalContent := wrapImageInAnchor(imgNode, styles)

	tableAttrs, tdAttrs := buildTableAndTdAttributes(styles, tableStyles, tdStyles, isFluidOnMobile, ctx)

	tableNode := createImageTableStructure(tableAttrs, tdAttrs, finalContent)

	transferPikoDirectives(node, tableNode)
	return tableNode, ctx.Diagnostics()
}

// getContentWidth returns the image width, choosing the smaller of the width
// attribute and the container width.
//
// Takes styles (*pml_domain.StyleManager) which provides access to the width
// attribute.
// Takes containerWidth (float64) which specifies the available space.
//
// Returns string which is the width in pixels with "px" suffix.
func (*Image) getContentWidth(styles *pml_domain.StyleManager, containerWidth float64) string {
	if containerWidth == 0 {
		containerWidth = defaultContainerWidth
	}

	widthAttr, hasWidth := styles.Get(AttrWidth)
	var imageWidth float64
	if !hasWidth {
		imageWidth = containerWidth
	} else {
		imageWidth = float64(mustParsePixels(widthAttr))
	}

	finalWidth := math.Min(containerWidth, imageWidth)
	return strconv.Itoa(int(finalWidth)) + "px"
}

// getTableStyles builds the CSS style map for an image table element.
//
// Takes contentWidth (string) which specifies the width value for full-width
// tables.
// Takes isFullWidth (bool) which controls whether full-width styles are
// applied.
//
// Returns map[string]string which contains the CSS property-value pairs.
func (*Image) getTableStyles(contentWidth string, isFullWidth bool) map[string]string {
	tableStyles := map[string]string{
		CSSBorderCollapse: "collapse",
		CSSBorderSpacing:  ValueZeroPx,
	}

	if isFullWidth {
		tableStyles[CSSMinWidth] = Value100
		tableStyles[CSSMaxWidth] = Value100
		tableStyles[CSSWidth] = contentWidth
	}

	return tableStyles
}

// getTdStyles builds the CSS style map for the table cell wrapping the image.
//
// Takes contentWidth (string) which specifies the content width value.
// Takes isFullWidth (bool) which enables full-width mode when true.
//
// Returns map[string]string which contains the CSS property-value pairs.
func (*Image) getTdStyles(_ *pml_domain.StyleManager, contentWidth string, isFullWidth bool) map[string]string {
	tdStyles := map[string]string{}

	if !isFullWidth {
		tdStyles[CSSWidth] = contentWidth
	}

	return tdStyles
}

// getImgStyles builds the CSS style map for rendering the image element.
//
// Takes styles (*pml_domain.StyleManager) which provides the style definitions.
// Takes isFullWidth (bool) which enables full-width mode when true.
//
// Returns map[string]string which contains the CSS property-value pairs.
func (*Image) getImgStyles(styles *pml_domain.StyleManager, isFullWidth bool) map[string]string {
	imgStyles := map[string]string{
		CSSDisplay:        ValueBlock,
		CSSOutline:        ValueNone,
		CSSTextDecoration: ValueNone,
		CSSHeight:         getStyleWithDefault(styles, AttrHeight, defaultImageHeight),
		CSSWidth:          Value100,
		CSSFontSize:       getStyleWithDefault(styles, CSSFontSize, defaultImageFontSize),
	}

	copyStyle(styles, imgStyles, AttrBorder)
	copyStyle(styles, imgStyles, CSSBorderLeft)
	copyStyle(styles, imgStyles, CSSBorderRight)
	copyStyle(styles, imgStyles, CSSBorderTop)
	copyStyle(styles, imgStyles, CSSBorderBottom)
	copyStyle(styles, imgStyles, AttrBorderRadius)
	copyStyle(styles, imgStyles, AttrMaxHeight)

	if isFullWidth {
		imgStyles[CSSMinWidth] = Value100
		imgStyles[CSSMaxWidth] = Value100
	}

	return imgStyles
}

// handleEmailAssetRegistration handles the registration of email assets and
// CID conversion. For email contexts, local assets (non-http/https) are
// registered with the EmailAssetRegistry and converted to CID (Content-ID)
// references for inline embedding.
//
// Takes styles (*pml_domain.StyleManager) which provides access to style
// attributes including source path, profile, and density settings.
// Takes widthPx (int) which specifies the computed display width in pixels.
// Takes ctx (*pml_domain.TransformationContext) which contains the email
// context and asset registry.
//
// Returns string which is the original source path or a CID reference for
// email contexts.
func handleEmailAssetRegistration(styles *pml_domain.StyleManager, widthPx int, ctx *pml_domain.TransformationContext) string {
	originalSrc := mustGetStyle(styles, AttrSrc)
	finalSrc := originalSrc

	if !ctx.IsEmailContext || ctx.EmailAssetRegistry == nil {
		return finalSrc
	}

	if strings.HasPrefix(originalSrc, ValueHTTP) || strings.HasPrefix(originalSrc, ValueHTTPS) || originalSrc == "" {
		return finalSrc
	}

	if strings.HasPrefix(originalSrc, resolver_adapters.ModuleAliasPrefix) && ctx.SourceFilePath != "" {
		if expanded, err := resolver_adapters.ExpandModuleAlias(originalSrc, ctx.SourceFilePath); err == nil {
			finalSrc = expanded
		}
	}

	if ctx.IsPreviewMode && ctx.AssetServePath != "" {
		return ctx.AssetServePath + "/" + finalSrc
	}

	profile := mustGetStyle(styles, AttrProfile)
	if profile == "" {
		profile = ValueEmailDefaultProfile
	}

	requestWidth, requestDensity := resolveAssetDensity(styles, widthPx)

	cid := ctx.EmailAssetRegistry.RegisterAsset(originalSrc, profile, requestWidth, requestDensity)
	return ValueCidPrefix + cid
}

// resolveAssetDensity determines the request width and density descriptor for
// asset registration. If a densities attribute is set, the highest density is
// selected and the width is scaled by its multiplier.
//
// Takes styles (*pml_domain.StyleManager) which provides access to the
// densities attribute.
// Takes widthPx (int) which is the base display width in pixels.
//
// Returns requestWidth (int) which is the width scaled by the density
// multiplier.
// Returns requestDensity (string) which is the selected density descriptor, or
// empty if none is configured.
func resolveAssetDensity(styles *pml_domain.StyleManager, widthPx int) (requestWidth int, requestDensity string) {
	requestWidth = widthPx

	densitiesAttr := mustGetStyle(styles, AttrDensities)
	if densitiesAttr == "" {
		return requestWidth, ""
	}

	densities := strings.Fields(densitiesAttr)
	if len(densities) == 0 {
		return requestWidth, ""
	}

	requestDensity = selectHighestDensity(densities)

	if requestWidth > 0 && requestDensity != "" {
		multiplier := parseDensityMultiplier(requestDensity)
		requestWidth = int(float64(requestWidth) * multiplier)
	}

	return requestWidth, requestDensity
}

// buildImageAttributes constructs all HTML attributes for the <img> element.
// This includes src, alt, width, style, and optional attributes like title,
// srcset, sizes, and usemap.
//
// Takes styles (*pml_domain.StyleManager) which provides style lookups for
// attribute values.
// Takes finalSrc (string) which is the resolved image source URL.
// Takes widthPx (int) which specifies the image width in pixels.
// Takes imgStyles (map[string]string) which contains inline CSS styles.
//
// Returns []ast_domain.HTMLAttribute which contains the sorted attributes
// ready for rendering.
func buildImageAttributes(styles *pml_domain.StyleManager, finalSrc string, widthPx int, imgStyles map[string]string) []ast_domain.HTMLAttribute {
	attrs := []ast_domain.HTMLAttribute{
		NewHTMLAttribute(AttrSrc, finalSrc),
		NewHTMLAttribute(AttrAlt, getStyleWithDefault(styles, AttrAlt, defaultImageAlt)),
		NewHTMLAttribute(AttrWidth, strconv.Itoa(widthPx)),
		NewHTMLAttribute(AttrStyle, mapToStyleString(imgStyles)),
	}

	if title := mustGetStyle(styles, AttrTitle); title != "" {
		attrs = append(attrs, NewHTMLAttribute(AttrTitle, title))
	}
	if srcset := mustGetStyle(styles, AttrSrcset); srcset != "" {
		attrs = append(attrs, NewHTMLAttribute(AttrSrcset, srcset))
	}
	if sizes := mustGetStyle(styles, AttrSizes); sizes != "" {
		attrs = append(attrs, NewHTMLAttribute(AttrSizes, sizes))
	}
	if usemap := mustGetStyle(styles, AttrUsemap); usemap != "" {
		attrs = append(attrs, NewHTMLAttribute(AttrUsemap, usemap))
	}

	return sortHTMLAttributes(attrs)
}

// wrapImageInAnchor wraps an image node in an anchor element if a link is set.
//
// Takes imgNode (*ast_domain.TemplateNode) which is the image node to wrap.
// Takes styles (*pml_domain.StyleManager) which provides the link settings.
//
// Returns *ast_domain.TemplateNode which is the anchor node if a link exists,
// or the original image node if no link is set.
func wrapImageInAnchor(imgNode *ast_domain.TemplateNode, styles *pml_domain.StyleManager) *ast_domain.TemplateNode {
	href, ok := styles.Get(AttrHref)
	if !ok || href == "" {
		return imgNode
	}

	anchorAttrs := []ast_domain.HTMLAttribute{
		NewHTMLAttribute(AttrHref, href),
		NewHTMLAttribute(AttrTarget, getStyleWithDefault(styles, AttrTarget, defaultImageTarget)),
	}

	if rel := mustGetStyle(styles, AttrRel); rel != "" {
		anchorAttrs = append(anchorAttrs, NewHTMLAttribute(AttrRel, rel))
	}
	if name := mustGetStyle(styles, AttrName); name != "" {
		anchorAttrs = append(anchorAttrs, NewHTMLAttribute(AttrName, name))
	}
	if title := mustGetStyle(styles, AttrTitle); title != "" {
		anchorAttrs = append(anchorAttrs, NewHTMLAttribute(AttrTitle, title))
	}

	return NewElementNode(ElementA, sortHTMLAttributes(anchorAttrs), []*ast_domain.TemplateNode{imgNode})
}

// buildTableAndTdAttributes constructs the attributes for the table and td
// elements. This includes data-pml-* attributes for parent components to read,
// and fluid-on-mobile classes.
//
// Takes styles (*pml_domain.StyleManager) which provides style values for
// data attributes.
// Takes tableStyles (map[string]string) which contains inline styles for the
// table element.
// Takes tdStyles (map[string]string) which contains inline styles for the td
// element.
// Takes isFluidOnMobile (bool) which enables fluid width classes for mobile.
// Takes ctx (*pml_domain.TransformationContext) which provides the media query
// collector for registering fluid classes.
//
// Returns tableAttrs ([]ast_domain.HTMLAttribute) which contains the sorted
// attributes for the table element.
// Returns tdAttrs ([]ast_domain.HTMLAttribute) which contains the sorted
// attributes for the td element.
func buildTableAndTdAttributes(
	styles *pml_domain.StyleManager,
	tableStyles, tdStyles map[string]string,
	isFluidOnMobile bool,
	ctx *pml_domain.TransformationContext,
) (tableAttrs []ast_domain.HTMLAttribute, tdAttrs []ast_domain.HTMLAttribute) {
	tableAttrs = []ast_domain.HTMLAttribute{
		NewHTMLAttribute(AttrBorder, ValueZero),
		NewHTMLAttribute(AttrCellPadding, ValueZero),
		NewHTMLAttribute(AttrCellSpacing, ValueZero),
		NewHTMLAttribute(AttrRole, ValuePresentation),
		NewHTMLAttribute(AttrStyle, mapToStyleString(tableStyles)),
		NewHTMLAttribute(DataPmlPadding, mustGetStyle(styles, AttrPadding)),
		NewHTMLAttribute(DataPmlAlign, mustGetStyle(styles, AttrAlign)),
	}

	if paddingTop := mustGetStyle(styles, AttrPaddingTop); paddingTop != "" {
		tableAttrs = append(tableAttrs, NewHTMLAttribute(DataPmlPaddingTop, paddingTop))
	}
	if paddingRight := mustGetStyle(styles, AttrPaddingRight); paddingRight != "" {
		tableAttrs = append(tableAttrs, NewHTMLAttribute(DataPmlPaddingRight, paddingRight))
	}
	if paddingBottom := mustGetStyle(styles, AttrPaddingBottom); paddingBottom != "" {
		tableAttrs = append(tableAttrs, NewHTMLAttribute(DataPmlPaddingBottom, paddingBottom))
	}
	if paddingLeft := mustGetStyle(styles, AttrPaddingLeft); paddingLeft != "" {
		tableAttrs = append(tableAttrs, NewHTMLAttribute(DataPmlPaddingLeft, paddingLeft))
	}

	if containerBg := mustGetStyle(styles, AttrContainerBackgroundColor); containerBg != "" {
		tableAttrs = append(tableAttrs, NewHTMLAttribute(DataPmlContainerBgColor, containerBg))
	}

	tdAttrs = []ast_domain.HTMLAttribute{
		NewHTMLAttribute(AttrStyle, mapToStyleString(tdStyles)),
	}

	if isFluidOnMobile {
		tableAttrs = append(tableAttrs, NewHTMLAttribute(AttrClass, ClassFluidMobile))
		tdAttrs = append(tdAttrs, NewHTMLAttribute(AttrClass, ClassFluidMobile))

		if ctx.MediaQueryCollector != nil {
			ctx.MediaQueryCollector.RegisterFluidClass(SelectorTableFluid, CSSFluidTable)
			ctx.MediaQueryCollector.RegisterFluidClass(SelectorTdFluid, CSSFluidTd)
		}
	}

	return sortHTMLAttributes(tableAttrs), sortHTMLAttributes(tdAttrs)
}

// createImageTableStructure creates the complete table wrapper structure for
// an image. Structure: <table><tbody><tr><td>content</td></tr></tbody></table>.
//
// Takes tableAttrs ([]ast_domain.HTMLAttribute) which specifies attributes for
// the outer table element.
// Takes tdAttrs ([]ast_domain.HTMLAttribute) which specifies attributes for
// the inner td cell element.
// Takes content (*ast_domain.TemplateNode) which is the node to wrap inside
// the table cell.
//
// Returns *ast_domain.TemplateNode which is the constructed table structure.
func createImageTableStructure(
	tableAttrs, tdAttrs []ast_domain.HTMLAttribute,
	content *ast_domain.TemplateNode,
) *ast_domain.TemplateNode {
	tdNode := NewElementNode(ElementTd, tdAttrs, []*ast_domain.TemplateNode{content})
	trNode := NewElementNode(ElementTr, nil, []*ast_domain.TemplateNode{tdNode})
	tbodyNode := NewElementNode(ElementTbody, nil, []*ast_domain.TemplateNode{trNode})
	return NewElementNode(ElementTable, tableAttrs, []*ast_domain.TemplateNode{tbodyNode})
}

// selectHighestDensity picks the highest pixel density from a list of density
// descriptors. For email rendering, the best quality variant is always chosen.
//
// Input examples: ["x1", "x2", "x3"], ["1x", "2x"], ["x2", "x1", "x3"]
// Returns: "x3", "x2", "x3" respectively.
//
// Takes densities ([]string) which contains density descriptors to compare.
//
// Returns string which is the highest density value in normalised form, or an
// empty string if the input slice is empty.
func selectHighestDensity(densities []string) string {
	if len(densities) == 0 {
		return ""
	}

	highestDensity := densities[0]
	highestValue := parseDensityMultiplier(densities[0])

	for _, density := range densities[1:] {
		value := parseDensityMultiplier(density)
		if value > highestValue {
			highestValue = value
			highestDensity = density
		}
	}

	return normaliseDensity(highestDensity)
}

// parseDensityMultiplier converts a density string to a number.
// It handles formats such as "x1", "1x", "x2", "2x", "x3", "3x".
//
// Takes density (string) which is the density string to parse.
//
// Returns float64 which is the parsed value, or 1.0 if the input is invalid.
func parseDensityMultiplier(density string) float64 {
	density = strings.ToLower(strings.TrimSpace(density))
	density = strings.TrimPrefix(density, "x")
	density = strings.TrimSuffix(density, "x")

	if multiplier, err := strconv.ParseFloat(density, 64); err == nil && multiplier > 0 {
		return multiplier
	}

	return 1.0
}

// normaliseDensity converts a density descriptor to the standard "xN" format.
// Input examples: "x1", "1x", "2x", "x3" become "x1", "x1", "x2", "x3".
//
// Takes density (string) which is the density descriptor to convert.
//
// Returns string which is the density in "xN" format.
func normaliseDensity(density string) string {
	multiplier := parseDensityMultiplier(density)

	multiplierString := strconv.FormatFloat(multiplier, 'f', -1, 64)

	return "x" + multiplierString
}

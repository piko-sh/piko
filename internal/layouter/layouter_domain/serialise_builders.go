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

package layouter_domain

import (
	"fmt"
	goast "go/ast"
	"go/token"
	"reflect"
	"strconv"
	"strings"
)

// buildLayoutBoxExpr builds a Go AST composite literal for a
// LayoutBox, including its children recursively.
//
// Takes box (*LayoutBox) which is the layout box to convert.
//
// Returns goast.Expr which is the composite literal expression.
func buildLayoutBoxExpr(box *LayoutBox) goast.Expr {
	elements := []goast.Expr{
		newKeyValueExpr(goast.NewIdent("Type"), buildBoxTypeExpr(box.Type)),
		newKeyValueExpr(goast.NewIdent("Style"), buildStyleExpr(&box.Style)),
	}

	if box.Text != "" {
		elements = append(elements, newKeyValueExpr(goast.NewIdent("Text"), strLit(box.Text)))
	}

	elements = appendEdgesField(elements, "Padding", box.Padding)
	elements = appendEdgesField(elements, "Border", box.Border)
	elements = appendEdgesField(elements, "Margin", box.Margin)
	elements = appendFloatField(elements, "ContentX", box.ContentX)
	elements = appendFloatField(elements, "ContentY", box.ContentY)
	elements = appendFloatField(elements, "ContentWidth", box.ContentWidth)
	elements = appendFloatField(elements, "ContentHeight", box.ContentHeight)
	elements = appendFloatField(elements, "IntrinsicWidth", box.IntrinsicWidth)
	elements = appendFloatField(elements, "IntrinsicHeight", box.IntrinsicHeight)
	elements = appendFloatField(elements, "OffsetX", box.OffsetX)
	elements = appendFloatField(elements, "OffsetY", box.OffsetY)

	if box.PageIndex != 0 {
		elements = append(elements, newKeyValueExpr(goast.NewIdent("PageIndex"), intLit(box.PageIndex)))
	}

	if len(box.Children) > 0 {
		childExprs := make([]goast.Expr, len(box.Children))
		for index, child := range box.Children {
			childExprs[index] = buildLayoutBoxExpr(child)
		}
		elements = append(elements, newKeyValueExpr(
			goast.NewIdent("Children"),
			newCompositeLit(
				&goast.ArrayType{Elt: newStarExpr(layouterType(typeLayoutBox))},
				childExprs,
			),
		))
	}

	return newUnaryExpr(token.AND, newCompositeLit(layouterType(typeLayoutBox), elements))
}

// appendFloatField appends a named float key-value pair to the
// element list when value is non-zero.
//
// Takes elements ([]goast.Expr) which is the list to append to.
// Takes name (string) which is the field name.
// Takes value (float64) which is the float value to check.
//
// Returns []goast.Expr which is the updated element list.
func appendFloatField(elements []goast.Expr, name string, value float64) []goast.Expr {
	if value != 0 {
		return append(elements, newKeyValueExpr(goast.NewIdent(name), floatLit(value)))
	}
	return elements
}

// appendEdgesField appends a named BoxEdges key-value pair to the
// element list when any edge value is non-zero.
//
// Takes elements ([]goast.Expr) which is the list to append to.
// Takes name (string) which is the field name.
// Takes edges (BoxEdges) which is the edge values to check.
//
// Returns []goast.Expr which is the updated element list.
func appendEdgesField(elements []goast.Expr, name string, edges BoxEdges) []goast.Expr {
	if hasNonZeroEdges(edges) {
		return append(elements, newKeyValueExpr(goast.NewIdent(name), buildEdgesExpr(edges)))
	}
	return elements
}

// buildStyleExpr builds a Go AST expression for a ComputedStyle,
// using a default style call or a withStyle override closure.
//
// Takes style (*ComputedStyle) which is the style to convert.
//
// Returns goast.Expr which is the style expression.
func buildStyleExpr(style *ComputedStyle) goast.Expr {
	overrides := buildStyleOverrideStatements(style)
	if len(overrides) == 0 {
		return newCallExpr(layouterType(identDefaultComputedStyle), nil)
	}

	return newCallExpr(
		goast.NewIdent(identWithStyle),
		[]goast.Expr{
			&goast.FuncLit{
				Type: &goast.FuncType{
					Params: &goast.FieldList{
						List: []*goast.Field{
							{
								Names: []*goast.Ident{goast.NewIdent(identS)},
								Type:  newStarExpr(layouterType(typeComputedStyle)),
							},
						},
					},
				},
				Body: &goast.BlockStmt{List: overrides},
			},
		},
	)
}

// buildStyleOverrideStatements generates assignment statements for
// each ComputedStyle field that differs from the default.
//
// Takes style (*ComputedStyle) which is the style to compare.
//
// Returns []goast.Stmt which is the list of override assignments.
func buildStyleOverrideStatements(style *ComputedStyle) []goast.Stmt {
	defaultStyle := DefaultComputedStyle()
	var statements []goast.Stmt

	styleValue := reflect.ValueOf(*style)
	defaultValue := reflect.ValueOf(defaultStyle)
	styleType := styleValue.Type()

	for i := range styleType.NumField() {
		field := styleType.Field(i)
		actual := styleValue.Field(i).Interface()
		expected := defaultValue.Field(i).Interface()

		if reflect.DeepEqual(actual, expected) {
			continue
		}

		statements = append(statements, &goast.AssignStmt{
			Lhs: []goast.Expr{
				&goast.SelectorExpr{X: goast.NewIdent(identS), Sel: goast.NewIdent(field.Name)},
			},
			Tok: token.ASSIGN,
			Rhs: []goast.Expr{buildStyleFieldValueExpr(actual)},
		})
	}

	return statements
}

// buildStyleFieldValueExpr converts a ComputedStyle field value
// to its Go AST expression form via type dispatch.
//
// Takes value (any) which is the field value to convert.
//
// Returns goast.Expr which is the AST expression for the value.
//
//nolint:revive // style field dispatch
func buildStyleFieldValueExpr(value any) goast.Expr {
	switch v := value.(type) {
	case string:
		return strLit(v)
	case float64:
		return floatLit(v)
	case int:
		return intLit(v)
	case bool:
		return boolLit(v)
	case Colour:
		return buildColourExpr(v)
	case Dimension:
		return buildDimensionExpr(v)
	case FontStyle:
		return buildEnumConstExpr("FontStyle", v.String())
	case DisplayType:
		return buildEnumConstExpr("Display", v.String())
	case PositionType:
		return buildEnumConstExpr("Position", v.String())
	case FloatType:
		return buildEnumConstExpr("Float", v.String())
	case ClearType:
		return buildEnumConstExpr("Clear", v.String())
	case TextAlignType:
		return buildEnumConstExpr("TextAlign", v.String())
	case TextDecorationFlag:
		return buildTextDecorationExpr(v)
	case TextTransformType:
		return buildEnumConstExpr("TextTransform", v.String())
	case WhiteSpaceType:
		return buildEnumConstExpr("WhiteSpace", v.String())
	case WordBreakType:
		return buildEnumConstExpr("WordBreak", v.String())
	case OverflowWrapType:
		return buildEnumConstExpr("OverflowWrap", v.String())
	case ObjectFitType:
		return buildEnumConstExpr("ObjectFit", v.String())
	case OverflowType:
		return buildEnumConstExpr("Overflow", v.String())
	case VisibilityType:
		return buildEnumConstExpr("Visibility", v.String())
	case BorderStyleType:
		return buildEnumConstExpr("BorderStyle", v.String())
	case FlexDirectionType:
		return buildEnumConstExpr("FlexDirection", v.String())
	case FlexWrapType:
		return buildEnumConstExpr("FlexWrap", v.String())
	case JustifyContentType:
		return buildEnumConstExpr("Justify", v.String())
	case AlignItemsType:
		return buildEnumConstExpr("AlignItems", v.String())
	case AlignSelfType:
		return buildEnumConstExpr("AlignSelf", v.String())
	case AlignContentType:
		return buildEnumConstExpr("AlignContent", v.String())
	case PageBreakType:
		return buildEnumConstExpr("PageBreak", v.String())
	case TableLayoutType:
		return buildEnumConstExpr("TableLayout", v.String())
	case BorderCollapseType:
		return buildEnumConstExpr("BorderCollapse", v.String())
	case VerticalAlignType:
		return buildEnumConstExpr("VerticalAlign", v.String())
	case CaptionSideType:
		return buildEnumConstExpr("CaptionSide", v.String())
	case ListStyleType:
		return buildEnumConstExpr("ListStyleType", v.String())
	case ListStylePositionType:
		return buildEnumConstExpr("ListStylePosition", v.String())
	case DirectionType:
		return buildEnumConstExpr("Direction", v.String())
	case UnicodeBidiType:
		return buildEnumConstExpr("UnicodeBidi", v.String())
	case HyphensType:
		return buildEnumConstExpr("Hyphens", v.String())
	case BackgroundImageType:
		return buildEnumConstExpr("BackgroundImage", v.String())
	case RadialGradientShape:
		return buildEnumConstExpr("RadialShape", v.String())
	case BorderImageRepeatType:
		return buildEnumConstExpr("BorderImageRepeat", v.String())
	case []BoxShadowValue:
		return buildBoxShadowSliceExpr(v)
	case []TextShadowValue:
		return buildTextShadowSliceExpr(v)
	case BackgroundImage:
		return buildBackgroundImageExpr(v)
	case []GradientStop:
		return buildGradientStopSliceExpr(v)
	case []CounterEntry:
		return buildCounterEntrySliceExpr(v)
	case []GridTrack:
		return buildGridTrackSliceExpr(v)
	case *GridAutoRepeat:
		return buildGridAutoRepeatExpr(v)
	case map[string]string:
		return buildStringMapExpr(v)
	default:
		return goast.NewIdent(fmt.Sprintf("%v", value))
	}
}

// buildEnumConstExpr builds a selector expression for a CSS enum
// constant by combining the prefix with the Go name of the keyword.
//
// Takes prefix (string) which is the enum type prefix.
// Takes cssKeyword (string) which is the CSS keyword value.
//
// Returns goast.Expr which is the selector expression.
func buildEnumConstExpr(prefix, cssKeyword string) goast.Expr {
	return layouterType(prefix + cssKeywordToGoName(cssKeyword))
}

// cssKeywordToGoName converts a hyphenated CSS keyword to a
// PascalCase Go identifier name.
//
// Takes keyword (string) which is the CSS keyword to convert.
//
// Returns string which is the PascalCase Go name.
func cssKeywordToGoName(keyword string) string {
	parts := strings.Split(keyword, "-")
	var builder strings.Builder
	for _, part := range parts {
		if len(part) > 0 {
			builder.WriteString(strings.ToUpper(part[:1]) + part[1:])
		}
	}
	return builder.String()
}

// buildBoxTypeExpr builds a selector expression for a BoxType
// enum constant.
//
// Takes boxType (BoxType) which is the box type to convert.
//
// Returns goast.Expr which is the selector expression.
func buildBoxTypeExpr(boxType BoxType) goast.Expr {
	return layouterType("Box" + boxType.String())
}

// buildDimensionExpr builds a Go AST call expression for a
// Dimension value based on its unit type.
//
// Takes dimension (Dimension) which is the dimension to convert.
//
// Returns goast.Expr which is the constructor call expression.
func buildDimensionExpr(dimension Dimension) goast.Expr {
	switch dimension.Unit {
	case DimensionUnitPoints:
		return newCallExpr(layouterType("DimensionPt"), []goast.Expr{floatLit(dimension.Value)})
	case DimensionUnitPercentage:
		return newCallExpr(layouterType("DimensionPct"), []goast.Expr{floatLit(dimension.Value)})
	case DimensionUnitMinContent:
		return newCallExpr(layouterType("DimensionMinContent"), nil)
	case DimensionUnitMaxContent:
		return newCallExpr(layouterType("DimensionMaxContent"), nil)
	case DimensionUnitFitContent:
		return newCallExpr(layouterType("DimensionFitContent"), []goast.Expr{floatLit(dimension.Value)})
	case DimensionUnitFitContentStretch:
		return newCallExpr(layouterType("DimensionFitContentStretch"), nil)
	default:
		return newCallExpr(layouterType("DimensionAuto"), nil)
	}
}

// buildColourExpr builds a Go AST expression for a Colour value,
// using named constants for black, white, and transparent.
//
// Takes colour (Colour) which is the colour to convert.
//
// Returns goast.Expr which is the colour expression.
func buildColourExpr(colour Colour) goast.Expr {
	if colour == ColourBlack {
		return layouterType("ColourBlack")
	}
	if colour == ColourWhite {
		return layouterType("ColourWhite")
	}
	if colour == ColourTransparent {
		return layouterType("ColourTransparent")
	}
	return newCallExpr(layouterType("NewRGBA"), []goast.Expr{
		floatLit(colour.Red), floatLit(colour.Green),
		floatLit(colour.Blue), floatLit(colour.Alpha),
	})
}

// buildEdgesExpr builds a Go AST composite literal for BoxEdges,
// omitting zero-valued edges.
//
// Takes edges (BoxEdges) which is the edge values to convert.
//
// Returns goast.Expr which is the composite literal expression.
func buildEdgesExpr(edges BoxEdges) goast.Expr {
	var elements []goast.Expr
	if edges.Top != 0 {
		elements = append(elements, newKeyValueExpr(goast.NewIdent("Top"), floatLit(edges.Top)))
	}
	if edges.Right != 0 {
		elements = append(elements, newKeyValueExpr(goast.NewIdent("Right"), floatLit(edges.Right)))
	}
	if edges.Bottom != 0 {
		elements = append(elements, newKeyValueExpr(goast.NewIdent("Bottom"), floatLit(edges.Bottom)))
	}
	if edges.Left != 0 {
		elements = append(elements, newKeyValueExpr(goast.NewIdent("Left"), floatLit(edges.Left)))
	}
	return newCompositeLit(layouterType(typeBoxEdges), elements)
}

// buildTextDecorationExpr builds a Go AST expression for text
// decoration flags, combining multiple flags with bitwise OR.
//
// Takes flags (TextDecorationFlag) which is the flags to convert.
//
// Returns goast.Expr which is the decoration expression.
func buildTextDecorationExpr(flags TextDecorationFlag) goast.Expr {
	if flags == TextDecorationNone {
		return layouterType("TextDecorationNone")
	}

	var parts []goast.Expr
	if flags&TextDecorationUnderline != 0 {
		parts = append(parts, layouterType("TextDecorationUnderline"))
	}
	if flags&TextDecorationOverline != 0 {
		parts = append(parts, layouterType("TextDecorationOverline"))
	}
	if flags&TextDecorationLineThrough != 0 {
		parts = append(parts, layouterType("TextDecorationLineThrough"))
	}

	if len(parts) == 1 {
		return parts[0]
	}

	result := parts[0]
	for _, part := range parts[1:] {
		result = &goast.BinaryExpr{X: result, Op: token.OR, Y: part}
	}
	return result
}

// buildBoxShadowSliceExpr builds a Go AST array literal for a
// slice of BoxShadowValue entries.
//
// Takes shadows ([]BoxShadowValue) which contains the shadows.
//
// Returns goast.Expr which is the array literal expression.
func buildBoxShadowSliceExpr(shadows []BoxShadowValue) goast.Expr {
	if len(shadows) == 0 {
		return goast.NewIdent("nil")
	}

	elements := make([]goast.Expr, len(shadows))
	for index, shadow := range shadows {
		elements[index] = buildBoxShadowValueExpr(shadow)
	}

	return newCompositeLit(
		&goast.ArrayType{Elt: layouterType("BoxShadowValue")},
		elements,
	)
}

// buildBoxShadowValueExpr builds a Go AST composite literal for
// a single BoxShadowValue, omitting zero-valued fields.
//
// Takes shadow (BoxShadowValue) which is the shadow to convert.
//
// Returns goast.Expr which is the composite literal expression.
func buildBoxShadowValueExpr(shadow BoxShadowValue) goast.Expr {
	var fields []goast.Expr

	if shadow.OffsetX != 0 {
		fields = append(fields, newKeyValueExpr(goast.NewIdent("OffsetX"), floatLit(shadow.OffsetX)))
	}
	if shadow.OffsetY != 0 {
		fields = append(fields, newKeyValueExpr(goast.NewIdent("OffsetY"), floatLit(shadow.OffsetY)))
	}
	if shadow.BlurRadius != 0 {
		fields = append(fields, newKeyValueExpr(goast.NewIdent("BlurRadius"), floatLit(shadow.BlurRadius)))
	}
	if shadow.SpreadRadius != 0 {
		fields = append(fields, newKeyValueExpr(goast.NewIdent("SpreadRadius"), floatLit(shadow.SpreadRadius)))
	}

	fields = append(fields, newKeyValueExpr(goast.NewIdent("Colour"), buildColourExpr(shadow.Colour)))

	if shadow.Inset {
		fields = append(fields, newKeyValueExpr(goast.NewIdent("Inset"), boolLit(true)))
	}

	return newCompositeLit(layouterType("BoxShadowValue"), fields)
}

// buildStringMapExpr builds a Go AST map literal for a
// map[string]string value.
//
// Takes entries (map[string]string) which is the map to convert.
//
// Returns goast.Expr which is the map literal expression.
func buildStringMapExpr(entries map[string]string) goast.Expr {
	elements := make([]goast.Expr, 0, len(entries))
	for key, value := range entries {
		elements = append(elements, newKeyValueExpr(strLit(key), strLit(value)))
	}
	return newCompositeLit(
		&goast.MapType{
			Key:   goast.NewIdent("string"),
			Value: goast.NewIdent("string"),
		},
		elements,
	)
}

// gridTrackUnitName maps a GridTrackUnit to its Go constant name.
//
// Takes unit (GridTrackUnit) which is the track unit to map.
//
// Returns string which is the Go constant name.
func gridTrackUnitName(unit GridTrackUnit) string {
	switch unit {
	case GridTrackPoints:
		return "GridTrackPoints"
	case GridTrackPercentage:
		return "GridTrackPercentage"
	case GridTrackFr:
		return "GridTrackFr"
	case GridTrackMinContent:
		return "GridTrackMinContent"
	case GridTrackMaxContent:
		return "GridTrackMaxContent"
	case GridTrackFitContent:
		return "GridTrackFitContent"
	case GridTrackFitContentPct:
		return "GridTrackFitContentPct"
	default:
		return "GridTrackAuto"
	}
}

// gridAutoRepeatTypeName maps a GridAutoRepeatType to its Go
// constant name.
//
// Takes t (GridAutoRepeatType) which is the repeat type.
//
// Returns string which is the Go constant name.
func gridAutoRepeatTypeName(t GridAutoRepeatType) string {
	if t == GridAutoRepeatFit {
		return "GridAutoRepeatFit"
	}
	return "GridAutoRepeatFill"
}

// buildGridTrackExpr builds a Go AST composite literal for a
// single GridTrack value.
//
// Takes track (GridTrack) which is the track to convert.
//
// Returns goast.Expr which is the composite literal expression.
func buildGridTrackExpr(track GridTrack) goast.Expr {
	var fields []goast.Expr
	if track.Value != 0 {
		fields = append(fields, newKeyValueExpr(goast.NewIdent("Value"), floatLit(track.Value)))
	}
	fields = append(fields, newKeyValueExpr(goast.NewIdent("Unit"), layouterType(gridTrackUnitName(track.Unit))))
	return newCompositeLit(layouterType("GridTrack"), fields)
}

// buildGridTrackSliceExpr builds a Go AST slice literal for
// a []GridTrack value.
//
// Takes tracks ([]GridTrack) which is the tracks to convert.
//
// Returns goast.Expr which is the slice literal expression.
func buildGridTrackSliceExpr(tracks []GridTrack) goast.Expr {
	elements := make([]goast.Expr, len(tracks))
	for i, track := range tracks {
		elements[i] = buildGridTrackExpr(track)
	}
	return newCompositeLit(
		&goast.ArrayType{Elt: layouterType("GridTrack")},
		elements,
	)
}

// buildGridAutoRepeatExpr builds a Go AST expression for a
// *GridAutoRepeat value including address-of.
//
// Takes ar (*GridAutoRepeat) which is the auto-repeat to convert.
//
// Returns goast.Expr which is the AST expression.
func buildGridAutoRepeatExpr(ar *GridAutoRepeat) goast.Expr {
	if ar == nil {
		return goast.NewIdent("nil")
	}
	fields := []goast.Expr{
		newKeyValueExpr(goast.NewIdent("Pattern"), buildGridTrackSliceExpr(ar.Pattern)),
	}
	if ar.InsertIndex != 0 {
		fields = append(fields, newKeyValueExpr(goast.NewIdent("InsertIndex"), intLit(ar.InsertIndex)))
	}
	if ar.AfterCount != 0 {
		fields = append(fields, newKeyValueExpr(goast.NewIdent("AfterCount"), intLit(ar.AfterCount)))
	}
	if ar.Type != GridAutoRepeatFill {
		fields = append(fields, newKeyValueExpr(goast.NewIdent("Type"), layouterType(gridAutoRepeatTypeName(ar.Type))))
	}
	return newUnaryExpr(token.AND, newCompositeLit(layouterType("GridAutoRepeat"), fields))
}

// buildCounterEntrySliceExpr builds a Go AST composite
// literal for a []CounterEntry slice.
//
// Takes entries ([]CounterEntry) which is the entries to convert.
//
// Returns goast.Expr which is the slice literal expression.
func buildCounterEntrySliceExpr(entries []CounterEntry) goast.Expr {
	elements := make([]goast.Expr, 0, len(entries))
	for _, entry := range entries {
		elements = append(elements, &goast.CompositeLit{
			Elts: []goast.Expr{
				newKeyValueExpr(goast.NewIdent("Name"), strLit(entry.Name)),
				newKeyValueExpr(goast.NewIdent("Value"), intLit(entry.Value)),
			},
		})
	}
	return newCompositeLit(
		&goast.ArrayType{Elt: layouterType("CounterEntry")},
		elements,
	)
}

// hasNonZeroEdges reports whether any edge in the BoxEdges
// has a non-zero value.
//
// Takes edges (BoxEdges) which is the edges to check.
//
// Returns bool which is true if any edge is non-zero.
func hasNonZeroEdges(edges BoxEdges) bool {
	return edges.Top != 0 || edges.Right != 0 || edges.Bottom != 0 || edges.Left != 0
}

// floatLit creates a float literal AST node from the given value.
//
// Takes value (float64) which is the float value to convert.
//
// Returns *goast.BasicLit which is the float literal node.
func floatLit(value float64) *goast.BasicLit {
	return newBasicLit(token.FLOAT, strconv.FormatFloat(value, 'f', -1, 64))
}

// intLit creates an integer literal AST node from the given value.
//
// Takes value (int) which is the integer value to convert.
//
// Returns *goast.BasicLit which is the integer literal node.
func intLit(value int) *goast.BasicLit {
	return newBasicLit(token.INT, strconv.Itoa(value))
}

// strLit creates a quoted string literal AST node.
//
// Takes value (string) which is the string value to quote.
//
// Returns *goast.BasicLit which is the string literal node.
func strLit(value string) *goast.BasicLit {
	return newBasicLit(token.STRING, fmt.Sprintf("%q", value))
}

// boolLit creates a boolean identifier AST node.
//
// Takes value (bool) which is the boolean value to represent.
//
// Returns goast.Expr which is the boolean identifier.
func boolLit(value bool) goast.Expr {
	if value {
		return goast.NewIdent("true")
	}
	return goast.NewIdent("false")
}

// buildTextShadowSliceExpr builds a Go AST expression for a
// slice of TextShadowValue.
//
// Takes shadows ([]TextShadowValue) which is the shadows.
//
// Returns goast.Expr which is the slice literal expression.
func buildTextShadowSliceExpr(shadows []TextShadowValue) goast.Expr {
	if len(shadows) == 0 {
		return goast.NewIdent("nil")
	}

	elements := make([]goast.Expr, len(shadows))
	for index, shadow := range shadows {
		var fields []goast.Expr
		if shadow.OffsetX != 0 {
			fields = append(fields, newKeyValueExpr(goast.NewIdent("OffsetX"), floatLit(shadow.OffsetX)))
		}
		if shadow.OffsetY != 0 {
			fields = append(fields, newKeyValueExpr(goast.NewIdent("OffsetY"), floatLit(shadow.OffsetY)))
		}
		if shadow.BlurRadius != 0 {
			fields = append(fields, newKeyValueExpr(goast.NewIdent("BlurRadius"), floatLit(shadow.BlurRadius)))
		}
		fields = append(fields, newKeyValueExpr(goast.NewIdent("Colour"), buildColourExpr(shadow.Colour)))
		elements[index] = newCompositeLit(layouterType("TextShadowValue"), fields)
	}

	return newCompositeLit(
		&goast.ArrayType{Elt: layouterType("TextShadowValue")},
		elements,
	)
}

// buildBackgroundImageExpr builds a Go AST expression for a
// BackgroundImage value.
//
// Takes bg (BackgroundImage) which is the background image
// to convert.
//
// Returns goast.Expr which is the composite literal.
func buildBackgroundImageExpr(bg BackgroundImage) goast.Expr {
	var fields []goast.Expr
	fields = append(fields, newKeyValueExpr(
		goast.NewIdent("Type"),
		buildEnumConstExpr("BackgroundImage", bg.Type.String()),
	))
	if bg.URL != "" {
		fields = append(fields, newKeyValueExpr(goast.NewIdent("URL"), strLit(bg.URL)))
	}
	if bg.Angle != 0 {
		fields = append(fields, newKeyValueExpr(goast.NewIdent("Angle"), floatLit(bg.Angle)))
	}
	if bg.Shape != RadialShapeEllipse {
		fields = append(fields, newKeyValueExpr(
			goast.NewIdent("Shape"),
			buildEnumConstExpr("RadialShape", bg.Shape.String()),
		))
	}
	if len(bg.Stops) > 0 {
		fields = append(fields, newKeyValueExpr(
			goast.NewIdent("Stops"),
			buildGradientStopSliceExpr(bg.Stops),
		))
	}
	return newCompositeLit(layouterType("BackgroundImage"), fields)
}

// buildGradientStopSliceExpr builds a Go AST expression for
// a slice of GradientStop values.
//
// Takes stops ([]GradientStop) which is the stops to
// convert.
//
// Returns goast.Expr which is the slice literal.
func buildGradientStopSliceExpr(stops []GradientStop) goast.Expr {
	elements := make([]goast.Expr, len(stops))
	for index, stop := range stops {
		var fields []goast.Expr
		fields = append(fields, newKeyValueExpr(goast.NewIdent("Colour"), buildColourExpr(stop.Colour)))
		if stop.Position != 0 {
			fields = append(fields, newKeyValueExpr(goast.NewIdent("Position"), floatLit(stop.Position)))
		}
		elements[index] = newCompositeLit(layouterType("GradientStop"), fields)
	}
	return newCompositeLit(
		&goast.ArrayType{Elt: layouterType("GradientStop")},
		elements,
	)
}

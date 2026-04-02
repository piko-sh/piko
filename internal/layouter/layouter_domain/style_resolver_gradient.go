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
	"math"
	"strconv"
	"strings"
)

const (
	// urlPrefixLength is the byte length of the "url(" prefix.
	urlPrefixLength = len("url(")

	// gradientPrefixLength is the byte length of the
	// "linear-gradient(" and "radial-gradient(" prefixes,
	// since both are 16 characters long.
	gradientPrefixLength = len("linear-gradient(")

	// gradientAngleRight is the angle in degrees for a rightward gradient.
	gradientAngleRight = 90.0

	// gradientAngleBottom is the angle in degrees for a downward gradient.
	gradientAngleBottom = 180.0

	// gradientAngleLeft is the angle in degrees for a leftward gradient.
	gradientAngleLeft = 270.0

	// shadowBlurTokenIndex is the token position of the blur
	// radius in a box-shadow value.
	shadowBlurTokenIndex = 3

	// shadowSpreadTokenIndex is the token position of the
	// spread radius in a box-shadow value.
	shadowSpreadTokenIndex = 4

	// textShadowBlurTokenCount is the minimum number of length
	// tokens required for a text-shadow to include a blur radius.
	textShadowBlurTokenCount = 3

	// degreesPerHalfTurn is the number of degrees in half a turn,
	// used when converting radians to degrees.
	degreesPerHalfTurn = 180.0
)

// parseBackgroundImages splits a CSS background-image value on
// commas that are not inside parentheses, then parses each layer
// individually. CSS layers are ordered front-to-back (first
// declared = topmost), but painting must happen back-to-front,
// so the caller (painter) should iterate in reverse.
//
// Takes value (string) which is the raw CSS background-image value.
// Takes context (ResolutionContext) which provides unit resolution values.
//
// Returns []BackgroundImage which holds the parsed background image layers.
func parseBackgroundImages(value string, context ResolutionContext) []BackgroundImage {
	value = strings.TrimSpace(value)
	if value == cssKeywordNone || value == "" {
		return nil
	}

	layers := splitOutsideParens(value, ',')
	var images []BackgroundImage
	for _, layer := range layers {
		img := parseBackgroundImage(strings.TrimSpace(layer), context)
		if img.Type != BackgroundImageNone {
			images = append(images, img)
		}
	}
	return images
}

// splitOutsideParens splits a string on the given separator, but
// only when the separator is not nested inside parentheses.
//
// Takes s (string) which is the input string to split.
// Takes sep (byte) which is the separator character.
//
// Returns []string which holds the split parts.
func splitOutsideParens(s string, sep byte) []string {
	var parts []string
	depth := 0
	start := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		case sep:
			if depth == 0 {
				parts = append(parts, s[start:i])
				start = i + 1
			}
		}
	}
	parts = append(parts, s[start:])
	return parts
}

// ParseMaskBackgroundImage parses a CSS mask-image value (which uses
// the same syntax as background-image) into a BackgroundImage struct.
//
// Takes value (string) which is the CSS mask-image value string.
// Takes context (ResolutionContext) which provides unit resolution values.
//
// Returns BackgroundImage which holds the parsed mask image.
func ParseMaskBackgroundImage(value string, context ResolutionContext) BackgroundImage {
	return parseBackgroundImage(value, context)
}

// DefaultResolutionContext returns a resolution context with standard
// defaults (16px base font size, zero container/viewport dimensions).
//
// Returns ResolutionContext which holds the default resolution values.
func DefaultResolutionContext() ResolutionContext {
	return ResolutionContext{
		ParentFontSize: defaultFontSizePt,
		RootFontSize:   defaultFontSizePt,
	}
}

// parseBackgroundImage parses a single CSS background-image layer value
// into a BackgroundImage, handling url(), linear-gradient(), radial-gradient(),
// and their repeating variants.
//
// Takes value (string) which is the single layer CSS value.
// Takes context (ResolutionContext) which provides unit resolution values.
//
// Returns BackgroundImage which holds the parsed background image.
func parseBackgroundImage(value string, context ResolutionContext) BackgroundImage {
	value = strings.TrimSpace(value)
	if value == cssKeywordNone || value == "" {
		return BackgroundImage{Type: BackgroundImageNone}
	}

	if strings.HasPrefix(value, "url(") {
		url := value[urlPrefixLength : len(value)-1]
		url = strings.Trim(url, "\"'")
		return BackgroundImage{
			Type: BackgroundImageURL,
			URL:  url,
		}
	}

	if strings.HasPrefix(value, "linear-gradient(") {
		inner := value[gradientPrefixLength : len(value)-1]
		return parseLinearGradient(inner, context)
	}

	if strings.HasPrefix(value, "radial-gradient(") {
		inner := value[gradientPrefixLength : len(value)-1]
		return parseRadialGradient(inner, context)
	}

	if strings.HasPrefix(value, "repeating-linear-gradient(") {
		inner := value[len("repeating-linear-gradient(") : len(value)-1]
		result := parseLinearGradient(inner, context)
		result.Type = BackgroundImageRepeatingLinearGradient
		return result
	}

	if strings.HasPrefix(value, "repeating-radial-gradient(") {
		inner := value[len("repeating-radial-gradient(") : len(value)-1]
		result := parseRadialGradient(inner, context)
		result.Type = BackgroundImageRepeatingRadialGradient
		return result
	}

	return BackgroundImage{Type: BackgroundImageNone}
}

// parseGradientDirection parses the optional direction or
// angle prefix from the first comma-separated segment of a
// gradient function body. If the segment specifies a
// direction, the returned angle is set and consumed is true,
// meaning the caller should skip this segment when parsing
// colour stops.
//
// Takes first (string) which is the first comma-separated
// segment, already trimmed.
//
// Returns angle (float64) which is the parsed angle in
// degrees.
// Returns consumed (bool) which is true when the segment
// was a direction and should be skipped.
func parseGradientDirection(first string) (angle float64, consumed bool) {
	if degStr, ok := strings.CutSuffix(first, "deg"); ok {
		return parseFloatValue(degStr), true
	}
	if strings.HasPrefix(first, "to ") {
		switch strings.TrimSpace(first[3:]) {
		case cssKeywordTop:
			return 0, true
		case "right":
			return gradientAngleRight, true
		case cssKeywordBottom:
			return gradientAngleBottom, true
		case "left":
			return gradientAngleLeft, true
		}
		return 0, true
	}
	return 0, false
}

// parseGradientStop parses a single colour-stop token group
// (e.g. "red 50%" or "rgba(255,0,0,0.5) 50%") into a
// GradientStop.
//
// Splits on the last space outside parentheses so that functional
// colour notation like rgba() with internal spaces is kept intact.
//
// Takes stop (string) which is the trimmed colour-stop
// string.
//
// Returns GradientStop which is the parsed stop value.
func parseGradientStop(stop string) GradientStop {
	stop = strings.TrimSpace(stop)
	if stop == "" {
		return GradientStop{Position: -1}
	}

	colourPart, positionPart := splitGradientStopParts(stop)

	gs := GradientStop{Position: -1}
	if colour, ok := ParseColour(colourPart); ok {
		gs.Colour = colour
	}
	if positionPart != "" {
		if pctStr, ok := strings.CutSuffix(positionPart, percentSuffix); ok {
			gs.Position = parseFloatValue(pctStr) / percentageDivisor
		}
	}
	return gs
}

// splitGradientStopParts splits a gradient stop string into colour and
// position parts by finding the last space outside parentheses.
//
// Takes stop (string) which is the gradient stop string to split.
//
// Returns colour (string) which is the colour portion of the stop.
// Returns position (string) which is the position portion, or empty
// if no position is present.
func splitGradientStopParts(stop string) (colour string, position string) {
	depth := 0
	lastSpace := -1
	for i := 0; i < len(stop); i++ {
		switch stop[i] {
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		case ' ', '\t':
			if depth == 0 {
				lastSpace = i
			}
		}
	}
	if lastSpace < 0 {
		return stop, ""
	}
	return strings.TrimSpace(stop[:lastSpace]), strings.TrimSpace(stop[lastSpace+1:])
}

// parseLinearGradient parses the inner content of a
// linear-gradient() function.
//
// Takes inner (string) which is the content between the
// parentheses.
//
// Returns BackgroundImage which is the parsed gradient.
func parseLinearGradient(inner string, _ ResolutionContext) BackgroundImage {
	bg := BackgroundImage{Type: BackgroundImageLinearGradient}
	parts := splitOutsideParens(inner, ',')
	startIndex := 0

	if len(parts) > 0 {
		first := strings.TrimSpace(parts[0])
		if angle, consumed := parseGradientDirection(first); consumed {
			bg.Angle = angle
			startIndex = 1
		}
	}

	for index := startIndex; index < len(parts); index++ {
		stop := strings.TrimSpace(parts[index])
		if stop == "" {
			continue
		}
		bg.Stops = append(bg.Stops, parseGradientStop(stop))
	}

	return bg
}

// parseRadialGradient parses the inner content of a
// radial-gradient() function.
//
// The first comma-separated segment may contain shape keywords
// (circle, ellipse) and size keywords (closest-side,
// farthest-corner, etc.) which are consumed before parsing
// colour stops. If no shape is specified, the default is ellipse.
//
// Takes inner (string) which is the content between the
// parentheses.
//
// Returns BackgroundImage which is the parsed gradient.
func parseRadialGradient(inner string, _ ResolutionContext) BackgroundImage {
	bg := BackgroundImage{
		Type:  BackgroundImageRadialGradient,
		Shape: RadialShapeEllipse,
	}
	parts := splitOutsideParens(inner, ',')
	startIndex := 0

	if len(parts) > 0 {
		first := strings.TrimSpace(parts[0])
		if shape, consumed := parseRadialShapePrefix(first); consumed {
			bg.Shape = shape
			startIndex = 1
		}
	}

	for index := startIndex; index < len(parts); index++ {
		stop := strings.TrimSpace(parts[index])
		if stop == "" {
			continue
		}
		bg.Stops = append(bg.Stops, parseGradientStop(stop))
	}

	return bg
}

// parseRadialShapePrefix checks whether the first comma-separated
// segment of a radial-gradient contains shape or size keywords
// rather than a colour stop. Returns the parsed shape and true
// if the segment was consumed as a shape/size descriptor.
//
// Takes first (string) which is the first comma-separated
// segment, already trimmed.
//
// Returns shape (RadialGradientShape) which is the parsed shape.
// Returns consumed (bool) which is true when the segment was a
// shape descriptor and should be skipped.
func parseRadialShapePrefix(first string) (shape RadialGradientShape, consumed bool) {
	lower := strings.ToLower(first)

	if atIndex := strings.Index(lower, " at "); atIndex >= 0 {
		lower = strings.TrimSpace(lower[:atIndex])
	}

	tokens := strings.Fields(lower)
	foundKeyword := false
	shape = RadialShapeEllipse

	for _, token := range tokens {
		switch token {
		case "circle":
			shape = RadialShapeCircle
			foundKeyword = true
		case "ellipse":
			shape = RadialShapeEllipse
			foundKeyword = true
		case "closest-side", "closest-corner", "farthest-side", "farthest-corner":
			foundKeyword = true
		}
	}

	return shape, foundKeyword
}

// parseBoxShadow parses a CSS box-shadow value into a slice
// of BoxShadowValue layers.
//
// Takes value (string) which is the CSS box-shadow
// value.
// Takes context (ResolutionContext) which provides the
// values needed for unit resolution.
//
// Returns []BoxShadowValue which is the parsed shadow
// layers, or nil if none.
func parseBoxShadow(value string, context ResolutionContext) []BoxShadowValue {
	if value == cssKeywordNone {
		return nil
	}

	layers := splitBoxShadowLayers(value)
	shadows := make([]BoxShadowValue, 0, len(layers))
	for _, layer := range layers {
		shadow, ok := parseBoxShadowLayer(strings.TrimSpace(layer), context)
		if ok {
			shadows = append(shadows, shadow)
		}
	}

	if len(shadows) == 0 {
		return nil
	}
	return shadows
}

// splitBoxShadowLayers splits a box-shadow value into
// individual layers by commas, respecting parentheses.
//
// Takes value (string) which is the full box-shadow
// value string.
//
// Returns []string which is the individual layer
// strings.
func splitBoxShadowLayers(value string) []string {
	var layers []string
	parenDepth := 0
	segmentStart := 0

	for index, character := range value {
		switch character {
		case '(':
			parenDepth++
		case ')':
			parenDepth--
		case ',':
			if parenDepth == 0 {
				layers = append(layers, value[segmentStart:index])
				segmentStart = index + 1
			}
		}
	}

	if segmentStart < len(value) {
		layers = append(layers, value[segmentStart:])
	}
	return layers
}

// parseBoxShadowLayer parses a single box-shadow layer
// string into a BoxShadowValue.
//
// Takes layer (string) which is the single shadow layer
// string to parse.
// Takes context (ResolutionContext) which provides the
// values needed for unit resolution.
//
// Returns BoxShadowValue which is the parsed shadow.
// Returns bool which is false if parsing failed.
func parseBoxShadowLayer(layer string, context ResolutionContext) (BoxShadowValue, bool) {
	tokens := tokeniseBoxShadow(layer)
	if len(tokens) < 2 {
		return BoxShadowValue{}, false
	}

	shadow := BoxShadowValue{Colour: ColourBlack}

	var lengthTokens []string
	for _, token := range tokens {
		if token == "inset" {
			shadow.Inset = true
			continue
		}
		if isLengthToken(token) {
			lengthTokens = append(lengthTokens, token)
			continue
		}
		if colour, ok := ParseColour(token); ok {
			shadow.Colour = colour
			continue
		}
	}

	if len(lengthTokens) < 2 {
		return BoxShadowValue{}, false
	}

	shadow.OffsetX = resolveLength(lengthTokens[0], context)
	shadow.OffsetY = resolveLength(lengthTokens[1], context)
	if len(lengthTokens) >= shadowBlurTokenIndex {
		shadow.BlurRadius = resolveLength(lengthTokens[2], context)
	}
	if len(lengthTokens) >= shadowSpreadTokenIndex {
		shadow.SpreadRadius = resolveLength(lengthTokens[3], context)
	}

	return shadow, true
}

// tokeniseBoxShadow splits a box-shadow layer string into
// whitespace-separated tokens, respecting parentheses.
//
// Takes layer (string) which is the shadow layer string
// to tokenise.
//
// Returns []string which is the extracted tokens.
func tokeniseBoxShadow(layer string) []string {
	var tokens []string
	parenDepth := 0
	tokenStart := -1

	for index, character := range layer {
		if character == ' ' && parenDepth == 0 {
			if tokenStart != -1 {
				tokens = append(tokens, layer[tokenStart:index])
				tokenStart = -1
			}
			continue
		}
		switch character {
		case '(':
			parenDepth++
		case ')':
			parenDepth--
		}
		if tokenStart == -1 {
			tokenStart = index
		}
	}

	if tokenStart != -1 {
		tokens = append(tokens, layer[tokenStart:])
	}
	return tokens
}

// parseListStyleType maps a CSS list-style-type value string
// to a ListStyleType enum.
//
// Takes value (string) which is the CSS list-style-type
// value.
//
// Returns ListStyleType which is the corresponding
// enum.
func parseListStyleType(value string) ListStyleType {
	switch value {
	case "circle":
		return ListStyleTypeCircle
	case "square":
		return ListStyleTypeSquare
	case "decimal":
		return ListStyleTypeDecimal
	case cssKeywordNone:
		return ListStyleTypeNone
	default:
		return ListStyleTypeDisc
	}
}

// parseListStylePosition maps a CSS list-style-position
// value string to a ListStylePositionType enum.
//
// Takes value (string) which is the CSS
// list-style-position value.
//
// Returns ListStylePositionType which is the
// corresponding enum.
func parseListStylePosition(value string) ListStylePositionType {
	switch value {
	case "inside":
		return ListStylePositionInside
	default:
		return ListStylePositionOutside
	}
}

// parseListStyleShorthand parses the CSS list-style
// shorthand property, setting type and position on the
// style.
//
// Takes style (*ComputedStyle) which is the style to
// modify.
// Takes value (string) which is the CSS list-style
// shorthand value.
func parseListStyleShorthand(style *ComputedStyle, value string) {
	for token := range strings.FieldsSeq(value) {
		switch token {
		case "disc", "circle", "square", "decimal":
			style.ListStyleType = parseListStyleType(token)
		case cssKeywordNone:
			style.ListStyleType = ListStyleTypeNone
		case "inside":
			style.ListStylePosition = ListStylePositionInside
		case "outside":
			style.ListStylePosition = ListStylePositionOutside
		}
	}
}

// isLengthToken reports whether a token is a valid CSS
// length value or bare number.
//
// Takes token (string) which is the token to check.
//
// Returns bool which is true if the token is a valid
// length or number.
func isLengthToken(token string) bool {
	if token == "0" {
		return true
	}
	for _, suffix := range []string{"px", "pt", "rem", "em", "cm", "mm", "vmin", "vmax", "in", "pc", "vw", "vh"} {
		if numberPart, found := strings.CutSuffix(token, suffix); found {
			_, err := strconv.ParseFloat(strings.TrimSpace(numberPart), 64)
			return err == nil
		}
	}
	_, err := strconv.ParseFloat(token, 64)
	return err == nil
}

// parseAspectRatio parses the CSS aspect-ratio value.
//
// Takes value (string) which is the CSS aspect-ratio value string.
//
// Returns float64 which is the ratio (width/height).
// Returns bool which indicates whether the auto keyword is present.
func parseAspectRatio(value string) (float64, bool) {
	normalised := strings.TrimSpace(value)
	if normalised == cssKeywordAuto {
		return 0, true
	}

	autoPrefix := false
	if strings.HasPrefix(normalised, "auto ") {
		autoPrefix = true
		normalised = strings.TrimSpace(normalised[5:])
	}

	if strings.Contains(normalised, "/") {
		parts := strings.SplitN(normalised, "/", 2)
		numerator, numeratorError := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
		denominator, denominatorError := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
		if numeratorError == nil && denominatorError == nil && denominator != 0 {
			return numerator / denominator, autoPrefix
		}
		return 0, autoPrefix
	}

	ratio, parseError := strconv.ParseFloat(normalised, 64)
	if parseError == nil {
		return ratio, autoPrefix
	}
	return 0, false
}

// parseTextOverflow parses the CSS text-overflow value.
//
// Takes value (string) which is the CSS text-overflow value string.
//
// Returns TextOverflowType which is the parsed text overflow type.
func parseTextOverflow(value string) TextOverflowType {
	if strings.TrimSpace(value) == "ellipsis" {
		return TextOverflowEllipsis
	}
	return TextOverflowClip
}

// parseColumnCount parses the CSS column-count value.
//
// Takes value (string) which is the CSS column-count value string.
//
// Returns int which is the parsed column count, or zero for auto.
func parseColumnCount(value string) int {
	if strings.TrimSpace(value) == cssKeywordAuto {
		return 0
	}
	count, parseError := strconv.Atoi(strings.TrimSpace(value))
	if parseError != nil || count < 1 {
		return 0
	}
	return count
}

// parseColumnsShorthand parses the CSS columns shorthand,
// which accepts column-count and column-width in any order.
//
// Takes style (*ComputedStyle) which receives the parsed values.
// Takes value (string) which is the CSS columns shorthand value.
// Takes context (ResolutionContext) which provides unit resolution values.
func parseColumnsShorthand(style *ComputedStyle, value string, context ResolutionContext) {
	for part := range strings.FieldsSeq(value) {
		if part == cssKeywordAuto {
			continue
		}
		if count, parseError := strconv.Atoi(part); parseError == nil && count >= 1 {
			style.ColumnCount = count
			continue
		}
		style.ColumnWidth = parseDimension(part, context)
	}
}

// parseColumnFill parses the CSS column-fill value.
//
// Takes value (string) which is the CSS column-fill value string.
//
// Returns ColumnFillType which is the parsed column fill type.
func parseColumnFill(value string) ColumnFillType {
	if strings.TrimSpace(value) == cssKeywordAuto {
		return ColumnFillAuto
	}
	return ColumnFillBalance
}

// parseColumnSpan parses the CSS column-span value.
//
// Takes value (string) which is the CSS column-span value string.
//
// Returns ColumnSpanType which is the parsed column span type.
func parseColumnSpan(value string) ColumnSpanType {
	if strings.TrimSpace(value) == "all" {
		return ColumnSpanAll
	}
	return ColumnSpanNone
}

// parseContent parses the CSS content property value,
// stripping quotes from string literals.
//
// Takes value (string) which is the CSS content value string.
//
// Returns string which is the parsed content, or empty string
// for "none" and "normal".
func parseContent(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == cssKeywordNone || trimmed == "normal" || trimmed == "" {
		return ""
	}
	if (strings.HasPrefix(trimmed, "\"") && strings.HasSuffix(trimmed, "\"")) ||
		(strings.HasPrefix(trimmed, "'") && strings.HasSuffix(trimmed, "'")) {
		return trimmed[1 : len(trimmed)-1]
	}
	return trimmed
}

// parseTextShadow parses a CSS text-shadow value into a
// slice of TextShadowValue layers. Follows the same pattern
// as box-shadow but without spread radius or inset keyword.
//
// Takes value (string) which is the CSS text-shadow value.
// Takes context (ResolutionContext) which provides unit
// resolution values.
//
// Returns []TextShadowValue which is the parsed shadow
// layers, or nil if none.
func parseTextShadow(value string, context ResolutionContext) []TextShadowValue {
	if value == cssKeywordNone {
		return nil
	}

	layers := splitBoxShadowLayers(value)
	shadows := make([]TextShadowValue, 0, len(layers))
	for _, layer := range layers {
		shadow, ok := parseTextShadowLayer(strings.TrimSpace(layer), context)
		if ok {
			shadows = append(shadows, shadow)
		}
	}

	if len(shadows) == 0 {
		return nil
	}
	return shadows
}

// parseTextShadowLayer parses a single text-shadow layer.
// Format: <offsetX> <offsetY> [blur] [colour].
//
// Takes layer (string) which is the single shadow layer
// string to parse.
// Takes context (ResolutionContext) which provides unit
// resolution values.
//
// Returns TextShadowValue which is the parsed shadow.
// Returns bool which is false if parsing failed.
func parseTextShadowLayer(layer string, context ResolutionContext) (TextShadowValue, bool) {
	tokens := tokeniseBoxShadow(layer)
	if len(tokens) < 2 {
		return TextShadowValue{}, false
	}

	shadow := TextShadowValue{Colour: ColourBlack}

	var lengthTokens []string
	for _, token := range tokens {
		if isLengthToken(token) {
			lengthTokens = append(lengthTokens, token)
			continue
		}
		if colour, ok := ParseColour(token); ok {
			shadow.Colour = colour
			continue
		}
	}

	if len(lengthTokens) < 2 {
		return TextShadowValue{}, false
	}

	shadow.OffsetX = resolveLength(lengthTokens[0], context)
	shadow.OffsetY = resolveLength(lengthTokens[1], context)
	if len(lengthTokens) >= textShadowBlurTokenCount {
		shadow.BlurRadius = resolveLength(lengthTokens[2], context)
	}

	return shadow, true
}

// parseOutlineShorthand parses the CSS outline shorthand
// property. Format: [width] [style] [colour].
//
// Takes style (*ComputedStyle) which receives the parsed
// values.
// Takes value (string) which is the outline shorthand
// string.
// Takes context (ResolutionContext) which provides unit
// resolution values.
func parseOutlineShorthand(style *ComputedStyle, value string, context ResolutionContext) {
	if value == cssKeywordNone {
		style.OutlineStyle = BorderStyleNone
		style.OutlineWidth = 0
		return
	}

	for token := range strings.FieldsSeq(value) {
		if bs := parseBorderStyle(token); bs != BorderStyleNone || token == cssKeywordNone {
			style.OutlineStyle = bs
		} else if isLengthToken(token) {
			style.OutlineWidth = resolveLength(token, context)
		} else if colour, ok := ParseColour(token); ok {
			style.OutlineColour = colour
		}
	}
}

// parseBorderImageRepeat parses a CSS border-image-repeat
// value.
//
// Takes value (string) which is the CSS value.
//
// Returns BorderImageRepeatType which is the parsed type.
func parseBorderImageRepeat(value string) BorderImageRepeatType {
	switch strings.TrimSpace(value) {
	case "repeat":
		return BorderImageRepeatRepeat
	case "round":
		return BorderImageRepeatRound
	case "space":
		return BorderImageRepeatSpace
	default:
		return BorderImageRepeatStretch
	}
}

// parseDirection parses a CSS direction value.
//
// Takes value (string) which is the CSS value.
//
// Returns DirectionType which is the parsed direction.
func parseDirection(value string) DirectionType {
	if strings.TrimSpace(value) == "rtl" {
		return DirectionRTL
	}
	return DirectionLTR
}

// parseUnicodeBidi parses a CSS unicode-bidi value.
//
// Takes value (string) which is the CSS value.
//
// Returns UnicodeBidiType which is the parsed type.
func parseUnicodeBidi(value string) UnicodeBidiType {
	switch strings.TrimSpace(value) {
	case "embed":
		return UnicodeBidiEmbed
	case "isolate":
		return UnicodeBidiIsolate
	case "bidi-override":
		return UnicodeBidiBidiOverride
	case "isolate-override":
		return UnicodeBidiIsolateOverride
	case "plaintext":
		return UnicodeBidiPlaintext
	default:
		return UnicodeBidiNormal
	}
}

// parseHyphens parses a CSS hyphens value.
//
// Takes value (string) which is the CSS value.
//
// Returns HyphensType which is the parsed type.
func parseHyphens(value string) HyphensType {
	switch strings.TrimSpace(value) {
	case cssKeywordNone:
		return HyphensNone
	case cssKeywordAuto:
		return HyphensAuto
	default:
		return HyphensManual
	}
}

// parseDimension parses a CSS dimension value into a
// Dimension, handling auto, calc(), percentages, and lengths.
//
// Takes value (string) which is the CSS dimension string
// to parse.
// Takes context (ResolutionContext) which provides the
// values needed for unit resolution.
//
// Returns Dimension which is the parsed dimension value.
func parseDimension(value string, context ResolutionContext) Dimension {
	if value == cssKeywordAuto {
		return DimensionAuto()
	}
	if value == "min-content" {
		return DimensionMinContent()
	}
	if value == "max-content" {
		return DimensionMaxContent()
	}
	if value == "fit-content" {
		return DimensionFitContentStretch()
	}
	if strings.HasPrefix(value, "fit-content(") && strings.HasSuffix(value, ")") {
		return parseFitContentArgument(value, context)
	}
	if strings.HasPrefix(value, "calc(") && strings.HasSuffix(value, ")") {
		inner := value[calcPrefixLength : len(value)-1]
		expression := parseCalc(inner)
		if expression != nil {
			resolved := expression.resolveCalc(context, context.ContainingBlockWidth)
			return DimensionPt(resolved)
		}
		return DimensionAuto()
	}
	if number, found := strings.CutSuffix(value, percentSuffix); found {
		parsed, err := strconv.ParseFloat(number, 64)
		if err != nil {
			return DimensionAuto()
		}
		return DimensionPct(parsed)
	}
	return DimensionPt(resolveLength(value, context))
}

// parseFitContentArgument extracts and resolves the argument
// from a fit-content(<length-percentage>) function value.
//
// Takes value (string) which is the full fit-content() function string.
// Takes context (ResolutionContext) which provides unit resolution values.
//
// Returns Dimension which is the resolved fit-content dimension.
func parseFitContentArgument(value string, context ResolutionContext) Dimension {
	inner := strings.TrimSpace(value[len("fit-content(") : len(value)-1])
	if number, found := strings.CutSuffix(inner, percentSuffix); found {
		parsed, err := strconv.ParseFloat(number, 64)
		if err != nil {
			return DimensionAuto()
		}
		return DimensionFitContent(parsed / percentageDivisor * context.ContainingBlockWidth)
	}
	return DimensionFitContent(resolveLength(inner, context))
}

// resolveFontSize resolves a CSS font-size value, handling
// absolute keywords, relative keywords, and length values.
//
// Takes value (string) which is the CSS font-size
// string to resolve.
// Takes context (ResolutionContext) which provides the
// values needed for unit resolution.
//
// Returns float64 which is the resolved font size in
// points.
func resolveFontSize(value string, context ResolutionContext) float64 {
	switch value {
	case "xx-small":
		return context.RootFontSize * fontSizeScaleXXSmall
	case "x-small":
		return context.RootFontSize * fontSizeScaleXSmall
	case "small":
		return context.RootFontSize * fontSizeScaleSmall
	case "medium":
		return context.RootFontSize
	case "large":
		return context.RootFontSize * fontSizeScaleLarge
	case "x-large":
		return context.RootFontSize * fontSizeScaleXLarge
	case "xx-large":
		return context.RootFontSize * fontSizeScaleXXLarge
	case "smaller":
		return context.ParentFontSize * fontSizeScaleSmaller
	case "larger":
		return context.ParentFontSize * fontSizeScaleLarger
	default:
		return resolveLength(value, context)
	}
}

// resolveLineHeight resolves a CSS line-height value,
// treating unitless numbers as font size multipliers.
//
// Takes value (string) which is the CSS line-height
// string to resolve.
// Takes fontSize (float64) which is the computed font
// size for unitless multiplier calculation.
// Takes context (ResolutionContext) which provides the
// values needed for unit resolution.
//
// Returns float64 which is the resolved line height in
// points.
func resolveLineHeight(value string, fontSize float64, context ResolutionContext) float64 {
	number, err := strconv.ParseFloat(value, 64)
	if err == nil {
		return number * fontSize
	}
	return resolveLength(value, context)
}

// parseFilterList parses a CSS filter property value into a slice of
// FilterValue structs.
//
// Takes value (string) which is the CSS filter property value string.
//
// Returns []FilterValue which holds the parsed filter functions.
func parseFilterList(value string) []FilterValue {
	value = strings.TrimSpace(value)
	if value == cssKeywordNone || value == "" {
		return nil
	}

	var filters []FilterValue
	remaining := value

	for remaining != "" {
		remaining = strings.TrimSpace(remaining)
		if remaining == "" {
			break
		}

		parenStart := strings.Index(remaining, "(")
		if parenStart < 0 {
			break
		}

		funcName := strings.TrimSpace(remaining[:parenStart])
		parenEnd := strings.Index(remaining[parenStart:], ")")
		if parenEnd < 0 {
			break
		}
		parenEnd += parenStart

		arg := strings.TrimSpace(remaining[parenStart+1 : parenEnd])
		remaining = remaining[parenEnd+1:]

		fv := parseFilterFunction(funcName, arg)
		if fv.Function != FilterNone {
			filters = append(filters, fv)
		}
	}

	return filters
}

// parseFilterFunction maps a CSS filter function name and its argument
// string to a FilterValue.
//
// Takes name (string) which is the filter function name (e.g. "blur").
// Takes arg (string) which is the argument inside the parentheses.
//
// Returns FilterValue which holds the parsed filter type and amount.
func parseFilterFunction(name string, arg string) FilterValue {
	switch name {
	case "blur":
		return FilterValue{Function: FilterBlur, Amount: parseFilterLength(arg)}
	case "brightness":
		return FilterValue{Function: FilterBrightness, Amount: parseFilterAmount(arg, 1.0)}
	case "contrast":
		return FilterValue{Function: FilterContrast, Amount: parseFilterAmount(arg, 1.0)}
	case "grayscale":
		return FilterValue{Function: FilterGrayscale, Amount: parseFilterAmount(arg, 0.0)}
	case "sepia":
		return FilterValue{Function: FilterSepia, Amount: parseFilterAmount(arg, 0.0)}
	case "saturate":
		return FilterValue{Function: FilterSaturate, Amount: parseFilterAmount(arg, 1.0)}
	case "hue-rotate":
		return FilterValue{Function: FilterHueRotate, Amount: parseFilterAngle(arg)}
	case "invert":
		return FilterValue{Function: FilterInvert, Amount: parseFilterAmount(arg, 0.0)}
	case "opacity":
		return FilterValue{Function: FilterOpacity, Amount: parseFilterAmount(arg, 1.0)}
	default:
		return FilterValue{Function: FilterNone}
	}
}

// parseFilterLength parses a CSS length value for filter functions,
// supporting px and pt units.
//
// Takes s (string) which is the length value string to parse.
//
// Returns float64 which is the resolved length in points.
func parseFilterLength(s string) float64 {
	s = strings.TrimSpace(s)
	if num, ok := strings.CutSuffix(s, "px"); ok {
		v, err := strconv.ParseFloat(num, 64)
		if err == nil {
			return v * PixelsToPoints
		}
	}
	if num, ok := strings.CutSuffix(s, "pt"); ok {
		v, err := strconv.ParseFloat(num, 64)
		if err == nil {
			return v
		}
	}
	v, err := strconv.ParseFloat(s, 64)
	if err == nil {
		return v * PixelsToPoints
	}
	return 0
}

// parseFilterAmount parses a CSS filter amount value, handling
// percentages and bare numbers.
//
// Takes s (string) which is the amount value string to parse.
// Takes defaultVal (float64) which is the fallback value when
// the string is empty.
//
// Returns float64 which is the resolved amount as a ratio.
func parseFilterAmount(s string, defaultVal float64) float64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return defaultVal
	}
	if num, ok := strings.CutSuffix(s, "%"); ok {
		v, err := strconv.ParseFloat(num, 64)
		if err == nil {
			return v / percentageDivisor
		}
	}
	v, err := strconv.ParseFloat(s, 64)
	if err == nil {
		return v
	}
	return defaultVal
}

// parseFilterAngle parses a CSS angle value for filter functions,
// supporting deg and rad units.
//
// Takes s (string) which is the angle value string to parse.
//
// Returns float64 which is the resolved angle in degrees.
func parseFilterAngle(s string) float64 {
	s = strings.TrimSpace(s)
	if num, ok := strings.CutSuffix(s, "deg"); ok {
		v, err := strconv.ParseFloat(num, 64)
		if err == nil {
			return v
		}
	}
	if num, ok := strings.CutSuffix(s, "rad"); ok {
		v, err := strconv.ParseFloat(num, 64)
		if err == nil {
			return v * degreesPerHalfTurn / math.Pi
		}
	}
	v, err := strconv.ParseFloat(s, 64)
	if err == nil {
		return v
	}
	return 0
}

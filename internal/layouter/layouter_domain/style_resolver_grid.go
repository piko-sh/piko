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
	"strconv"
	"strings"

	"piko.sh/piko/internal/layouter/layouter_dto"
)

const (
	// gridAreaRowEndIndex is the index of the row-end part in a grid-area shorthand.
	gridAreaRowEndIndex = 3

	// gridAreaColumnEndIndex is the index of the column-end part in a grid-area shorthand.
	gridAreaColumnEndIndex = 4
)

// gridTrackListResult holds the parsed tracks and any deferred auto-repeat pattern from a
// grid-template-columns/rows value.
type gridTrackListResult struct {
	// autoRepeat holds the deferred auto-repeat pattern, if present.
	autoRepeat *GridAutoRepeat

	// tracks holds the explicit grid track definitions.
	tracks []GridTrack
}

// parseGridTrackList parses a CSS grid-template-columns or grid-template-rows value into
// explicit tracks and an optional auto-repeat pattern.
//
// Takes value (string) which is the CSS track list value.
// Takes context (ResolutionContext) which provides unit resolution values and limits.
//
// Returns gridTrackListResult which holds the parsed tracks and auto-repeat.
func parseGridTrackList(value string, context ResolutionContext) gridTrackListResult {
	value = strings.TrimSpace(value)
	if value == "" || value == cssKeywordNone {
		return gridTrackListResult{}
	}

	maxTracks := context.Limits.Limits().MaxGridTracks
	var tracks []GridTrack
	var autoRepeat *GridAutoRepeat
	tokens := strings.Fields(value)

	for index := 0; index < len(tokens); index++ {
		token := tokens[index]

		if strings.HasPrefix(token, "repeat(") {
			repeatTracks, ar, consumed, ok := parseRepeat(tokens, index, context, len(tracks))
			if !ok {
				return gridTrackListResult{}
			}
			if ar != nil {
				autoRepeat = ar
			} else {
				tracks = append(tracks, repeatTracks...)
			}
			index += consumed
		} else {
			tracks = append(tracks, parseGridTrackToken(token, context))
		}

		if len(tracks) > maxTracks {
			context.Limits.fail(fmt.Errorf("grid track list has more than %d tracks: %w",
				maxTracks, layouter_dto.ErrTooManyGridTracks))
			return gridTrackListResult{}
		}
	}

	if autoRepeat != nil {
		autoRepeat.AfterCount = len(tracks) - autoRepeat.InsertIndex
	}
	return gridTrackListResult{tracks: tracks, autoRepeat: autoRepeat}
}

// parseRepeat parses a CSS repeat() function from a token list, returning either expanded
// tracks for integer repetitions or a GridAutoRepeat for auto-fill/auto-fit.
//
// Only the tokens up to the one holding the closing parenthesis are joined, so a track
// list with many repeat() functions parses in linear time. An integer count above
// MaxRepeatCount, or an expansion that would exceed MaxGridTracks, records a breach on
// the context's limit tracker and reports failure without allocating the expansion.
//
// Takes tokens ([]string) which is the full token list.
// Takes startIndex (int) which is the index of the repeat() token.
// Takes context (ResolutionContext) which provides unit resolution values and limits.
// Takes insertIndex (int) which is the position in the track list for auto-repeat
// insertion.
//
// Returns []GridTrack which is the expanded tracks for integer repeat.
// Returns *GridAutoRepeat which is non-nil for auto-fill or auto-fit.
// Returns int which is the number of tokens consumed.
// Returns bool which is false when a limit was breached and the track list must be
// dropped.
func parseRepeat(
	tokens []string, startIndex int, context ResolutionContext, insertIndex int,
) ([]GridTrack, *GridAutoRepeat, int, bool) {
	endIndex := findClosingParenthesisToken(tokens, startIndex)
	if endIndex < 0 {
		return nil, nil, 0, true
	}
	combined := strings.Join(tokens[startIndex:endIndex+1], " ")
	openParenthesis := strings.Index(combined, "(")
	closeParenthesis := strings.Index(combined, ")")
	if openParenthesis == -1 || closeParenthesis < openParenthesis {
		return nil, nil, 0, true
	}

	inner := combined[openParenthesis+1 : closeParenthesis]
	parts := strings.SplitN(inner, commaDelimiter, 2)
	if len(parts) != 2 {
		return nil, nil, 0, true
	}

	consumed := endIndex - startIndex
	countStr := strings.TrimSpace(parts[0])
	trackTokens := strings.Fields(strings.TrimSpace(parts[1]))

	var pattern []GridTrack
	for _, trackToken := range trackTokens {
		pattern = append(pattern, parseGridTrackToken(trackToken, context))
	}

	if countStr == "auto-fill" || countStr == "auto-fit" {
		repeatType := GridAutoRepeatFill
		if countStr == "auto-fit" {
			repeatType = GridAutoRepeatFit
		}
		return nil, &GridAutoRepeat{
			Type:        repeatType,
			Pattern:     pattern,
			InsertIndex: insertIndex,
			AfterCount:  0,
		}, consumed, true
	}

	count, countError := strconv.Atoi(countStr)
	if countError != nil || count < 1 {
		return nil, nil, 0, true
	}
	if !repeatWithinLimits(count, len(pattern), context.Limits) {
		return nil, nil, 0, false
	}

	result := make([]GridTrack, 0, count*len(pattern))
	for range count {
		result = append(result, pattern...)
	}
	return result, nil, consumed, true
}

// parseGridTrackToken parses a single grid track size token into a GridTrack.
//
// Takes token (string) which is the CSS track size token.
// Takes context (ResolutionContext) which provides unit resolution values.
//
// Returns GridTrack which is the parsed track definition.
func parseGridTrackToken(token string, context ResolutionContext) GridTrack {
	switch {
	case token == cssKeywordAuto:
		return GridTrack{}
	case token == "min-content":
		return GridTrack{Unit: GridTrackMinContent, Value: 0}
	case token == "max-content":
		return GridTrack{Unit: GridTrackMaxContent, Value: 0}
	case strings.HasPrefix(token, "fit-content(") && strings.HasSuffix(token, ")"):
		inner := strings.TrimSpace(token[len("fit-content(") : len(token)-1])
		if number, found := strings.CutSuffix(inner, percentSuffix); found {
			pct, err := strconv.ParseFloat(number, 64)
			if err != nil {
				return GridTrack{}
			}
			return GridTrack{Value: pct, Unit: GridTrackFitContentPct}
		}
		return GridTrack{Value: resolveLength(inner, context), Unit: GridTrackFitContent}
	case strings.HasSuffix(token, "fr"):
		numberPart := strings.TrimSuffix(token, "fr")
		fractionalValue, parseError := strconv.ParseFloat(numberPart, 64)
		if parseError != nil {
			return GridTrack{}
		}
		return GridTrack{Value: fractionalValue, Unit: GridTrackFr}
	case strings.HasSuffix(token, percentSuffix):
		numberPart := strings.TrimSuffix(token, percentSuffix)
		percentageValue, parseError := strconv.ParseFloat(numberPart, 64)
		if parseError != nil {
			return GridTrack{}
		}
		return GridTrack{Value: percentageValue, Unit: GridTrackPercentage}
	default:
		return GridTrack{Value: resolveLength(token, context), Unit: GridTrackPoints}
	}
}

// parseGridLine parses a CSS grid line value into a GridLine.
//
// Takes value (string) which is the CSS grid line value.
//
// Returns GridLine which is the parsed grid line.
func parseGridLine(value string) GridLine {
	value = strings.TrimSpace(value)
	if value == "" || value == cssKeywordAuto {
		return DefaultGridLine()
	}

	if spanValue, ok := strings.CutPrefix(value, "span"); ok {
		spanCount, spanError := strconv.Atoi(strings.TrimSpace(spanValue))
		if spanError != nil || spanCount < 1 {
			return DefaultGridLine()
		}
		return GridLine{Span: spanCount, Line: 0, IsAuto: false}
	}

	lineNumber, lineError := strconv.Atoi(value)
	if lineError != nil {
		return DefaultGridLine()
	}
	return GridLine{Line: lineNumber, Span: 0, IsAuto: false}
}

// parseGridShorthand parses a CSS grid-column or grid-row shorthand value into start and
// end grid lines.
//
// Takes value (string) which is the CSS shorthand value.
//
// Returns the start and end GridLine values.
func parseGridShorthand(value string) (startLine, endLine GridLine) {
	parts := strings.SplitN(value, "/", 2)
	if len(parts) == 1 {
		start := parseGridLine(strings.TrimSpace(parts[0]))
		return start, DefaultGridLine()
	}
	start := parseGridLine(strings.TrimSpace(parts[0]))
	end := parseGridLine(strings.TrimSpace(parts[1]))
	return start, end
}

// parseGridTemplateAreas parses a CSS grid-template-areas value into a 2D grid of area
// names.
//
// Each quoted string defines one row; tokens within a quoted row define cell names. A "."
// token represents an unnamed cell.
//
// Takes value (string) which is the CSS grid-template-areas value.
//
// Returns [][]string which is the parsed 2D grid of area names.
func parseGridTemplateAreas(value string) [][]string {
	var areas [][]string
	inQuote := false
	var quoteChar byte
	start := 0
	for i := 0; i < len(value); i++ {
		ch := value[i]
		if inQuote {
			if ch == quoteChar {
				row := strings.Fields(value[start:i])
				areas = append(areas, row)
				inQuote = false
			}
		} else if ch == '"' || ch == '\'' {
			quoteChar = ch
			start = i + 1
			inQuote = true
		}
	}
	return areas
}

// parseGridAreaShorthand parses the CSS grid-area shorthand, which can be a named area
// reference or up to four slash-separated grid line values (row-start / column-start /
// row-end / column-end).
//
// Takes style (*ComputedStyle) which is the style to modify.
// Takes value (string) which is the CSS grid-area value.
func parseGridAreaShorthand(style *ComputedStyle, value string) {
	if !strings.Contains(value, "/") {
		style.GridArea = strings.TrimSpace(value)
		return
	}
	parts := strings.Split(value, "/")
	if len(parts) >= 1 {
		style.GridRowStart = parseGridLine(strings.TrimSpace(parts[0]))
	}
	if len(parts) >= 2 {
		style.GridColumnStart = parseGridLine(strings.TrimSpace(parts[1]))
	}
	if len(parts) >= gridAreaRowEndIndex {
		style.GridRowEnd = parseGridLine(strings.TrimSpace(parts[2]))
	}
	if len(parts) >= gridAreaColumnEndIndex {
		style.GridColumnEnd = parseGridLine(strings.TrimSpace(parts[3]))
	}
}

// parseGridAutoFlow parses the CSS grid-auto-flow value.
//
// Takes value (string) which is the CSS grid-auto-flow value.
//
// Returns GridAutoFlowType which is the parsed flow type.
func parseGridAutoFlow(value string) GridAutoFlowType {
	normalised := strings.TrimSpace(strings.ToLower(value))
	switch normalised {
	case "column":
		return GridAutoFlowColumn
	case "row dense", "dense row", "dense":
		return GridAutoFlowRowDense
	case "column dense", "dense column":
		return GridAutoFlowColumnDense
	default:
		return GridAutoFlowRow
	}
}

// findClosingParenthesisToken returns the index of the first token at or after startIndex
// that contains a closing parenthesis.
//
// Takes tokens ([]string) which is the full token list.
// Takes startIndex (int) which is the index of the first token to search.
//
// Returns int which is the token index, or -1 when no token closes the parenthesis.
func findClosingParenthesisToken(tokens []string, startIndex int) int {
	for index := startIndex; index < len(tokens); index++ {
		if strings.Contains(tokens[index], ")") {
			return index
		}
	}
	return -1
}

// repeatWithinLimits reports whether an integer repeat() count and its expansion fit the
// MaxRepeatCount and MaxGridTracks limits, recording a breach on the tracker when they do
// not.
//
// Takes count (int) which is the declared repetition count.
// Takes patternLength (int) which is the number of tracks in one repetition.
// Takes limits (*LimitTracker) which records a breach, or nil for the defaults.
//
// Returns bool which is false when a limit is breached.
func repeatWithinLimits(count, patternLength int, limits *LimitTracker) bool {
	resolved := limits.Limits()
	if count > resolved.MaxRepeatCount {
		limits.fail(fmt.Errorf("repeat() count %d exceeds the limit of %d: %w",
			count, resolved.MaxRepeatCount, layouter_dto.ErrRepeatCountTooLarge))
		return false
	}
	if patternLength > 0 && count > resolved.MaxGridTracks/patternLength {
		limits.fail(fmt.Errorf("repeat() expands to more than %d tracks: %w",
			resolved.MaxGridTracks, layouter_dto.ErrTooManyGridTracks))
		return false
	}
	return true
}

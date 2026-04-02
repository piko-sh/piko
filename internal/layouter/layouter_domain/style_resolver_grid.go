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
	"strconv"
	"strings"
)

const (
	// gridAreaRowEndIndex is the index of the row-end part in a
	// grid-area shorthand.
	gridAreaRowEndIndex = 3

	// gridAreaColumnEndIndex is the index of the column-end part in a
	// grid-area shorthand.
	gridAreaColumnEndIndex = 4
)

// gridTrackListResult holds the parsed tracks and any deferred
// auto-repeat pattern from a grid-template-columns/rows value.
type gridTrackListResult struct {
	// autoRepeat holds the deferred auto-repeat pattern, if present.
	autoRepeat *GridAutoRepeat

	// tracks holds the explicit grid track definitions.
	tracks []GridTrack
}

// parseGridTrackList parses a CSS grid-template-columns or grid-template-rows
// value into explicit tracks and an optional auto-repeat pattern.
//
// Takes value (string) which is the CSS track list value.
// Takes context (ResolutionContext) which provides unit resolution values.
//
// Returns gridTrackListResult which holds the parsed tracks and auto-repeat.
func parseGridTrackList(value string, context ResolutionContext) gridTrackListResult {
	value = strings.TrimSpace(value)
	if value == "" || value == cssKeywordNone {
		return gridTrackListResult{}
	}

	var tracks []GridTrack
	var autoRepeat *GridAutoRepeat
	tokens := strings.Fields(value)

	for index := 0; index < len(tokens); index++ {
		token := tokens[index]

		if strings.HasPrefix(token, "repeat(") {
			repeatTracks, ar, consumed := parseRepeat(tokens, index, context, len(tracks))
			if ar != nil {
				autoRepeat = ar
			} else {
				tracks = append(tracks, repeatTracks...)
			}
			index += consumed
			continue
		}

		tracks = append(tracks, parseGridTrackToken(token, context))
	}

	if autoRepeat != nil {
		autoRepeat.AfterCount = len(tracks) - autoRepeat.InsertIndex
	}
	return gridTrackListResult{tracks: tracks, autoRepeat: autoRepeat}
}

// parseRepeat parses a CSS repeat() function from a token list, returning
// either expanded tracks for integer repetitions or a GridAutoRepeat for
// auto-fill/auto-fit.
//
// Takes tokens ([]string) which is the full token list.
// Takes startIndex (int) which is the index of the repeat() token.
// Takes context (ResolutionContext) which provides unit resolution values.
// Takes insertIndex (int) which is the position in the track list for
// auto-repeat insertion.
//
// Returns []GridTrack which is the expanded tracks for integer repeat.
// Returns *GridAutoRepeat which is non-nil for auto-fill or auto-fit.
// Returns int which is the number of tokens consumed.
func parseRepeat(
	tokens []string, startIndex int, context ResolutionContext, insertIndex int,
) ([]GridTrack, *GridAutoRepeat, int) {
	combined := strings.Join(tokens[startIndex:], " ")
	openParenthesis := strings.Index(combined, "(")
	closeParenthesis := strings.Index(combined, ")")
	if openParenthesis == -1 || closeParenthesis == -1 {
		return nil, nil, 0
	}

	inner := combined[openParenthesis+1 : closeParenthesis]
	parts := strings.SplitN(inner, commaDelimiter, 2)
	if len(parts) != 2 {
		return nil, nil, 0
	}

	consumed := countConsumedTokens(tokens, startIndex, combined, closeParenthesis)
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
		}, consumed
	}

	count, countError := strconv.Atoi(countStr)
	if countError != nil || count < 1 {
		return nil, nil, 0
	}

	var result []GridTrack
	for range count {
		result = append(result, pattern...)
	}
	return result, nil, consumed
}

// countConsumedTokens calculates how many tokens from startIndex
// were consumed by the repeat() expression ending at
// closeParenthesis in the combined string.
//
// Takes tokens ([]string) which is the full token list.
// Takes startIndex (int) which is the first token index.
// Takes combined (string) which is the joined token string.
// Takes closeParenthesis (int) which is the index of the closing parenthesis.
//
// Returns int which is the number of extra tokens consumed beyond the first.
func countConsumedTokens(tokens []string, startIndex int, combined string, closeParenthesis int) int {
	afterClose := combined[closeParenthesis+1:]
	consumedLength := len(combined) - len(afterClose)
	consumed := 0
	length := 0
	for index := startIndex; index < len(tokens); index++ {
		length += len(tokens[index])
		if index > startIndex {
			length++
		}
		consumed++
		if length >= consumedLength {
			break
		}
	}
	return consumed - 1
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
		return GridTrack{Unit: GridTrackAuto}
	case token == "min-content":
		return GridTrack{Unit: GridTrackMinContent}
	case token == "max-content":
		return GridTrack{Unit: GridTrackMaxContent}
	case strings.HasPrefix(token, "fit-content(") && strings.HasSuffix(token, ")"):
		inner := strings.TrimSpace(token[len("fit-content(") : len(token)-1])
		if number, found := strings.CutSuffix(inner, percentSuffix); found {
			pct, err := strconv.ParseFloat(number, 64)
			if err != nil {
				return GridTrack{Unit: GridTrackAuto}
			}
			return GridTrack{Value: pct, Unit: GridTrackFitContentPct}
		}
		return GridTrack{Value: resolveLength(inner, context), Unit: GridTrackFitContent}
	case strings.HasSuffix(token, "fr"):
		numberPart := strings.TrimSuffix(token, "fr")
		fractionalValue, parseError := strconv.ParseFloat(numberPart, 64)
		if parseError != nil {
			return GridTrack{Unit: GridTrackAuto}
		}
		return GridTrack{Value: fractionalValue, Unit: GridTrackFr}
	case strings.HasSuffix(token, percentSuffix):
		numberPart := strings.TrimSuffix(token, percentSuffix)
		percentageValue, parseError := strconv.ParseFloat(numberPart, 64)
		if parseError != nil {
			return GridTrack{Unit: GridTrackAuto}
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
		return GridLine{Span: spanCount}
	}

	lineNumber, lineError := strconv.Atoi(value)
	if lineError != nil {
		return DefaultGridLine()
	}
	return GridLine{Line: lineNumber}
}

// parseGridShorthand parses a CSS grid-column or grid-row shorthand value
// into start and end grid lines.
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

// parseGridTemplateAreas parses a CSS grid-template-areas value
// into a 2D grid of area names.
//
// Each quoted string defines one row; tokens within a quoted row
// define cell names. A "." token represents an unnamed cell.
//
// Takes value (string) which is the CSS grid-template-areas
// value.
//
// Returns [][]string which is the parsed 2D grid of area
// names.
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

// parseGridAreaShorthand parses the CSS grid-area shorthand,
// which can be a named area reference or up to four slash-separated
// grid line values (row-start / column-start / row-end / column-end).
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

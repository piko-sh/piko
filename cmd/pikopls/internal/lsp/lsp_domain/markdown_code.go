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
	"strings"
)

const (
	// maxMarkdownCodeRunes caps how many runes of text a code span shows, so a long default
	// value or expression cannot flood a hover.
	maxMarkdownCodeRunes = 120

	// markdownCodeEllipsis marks a code span whose text was cut short.
	markdownCodeEllipsis = "..."
)

// markdownCode wraps text in a Markdown inline code span.
//
// Line breaks are collapsed to single spaces first, because a blank line would end the
// paragraph and break the span, and text longer than maxMarkdownCodeRunes runes is cut
// short with an ellipsis so the hover stays readable.
//
// The fence is one backquote longer than the longest backquote run in the text, so text
// that itself contains backquotes (a JavaScript template literal, for example) cannot
// close the span early. A space pads the text when it starts or ends with a backquote, as
// CommonMark requires.
//
// Takes text (string) which is the content to show as code.
//
// Returns string which is the code span.
func markdownCode(text string) string {
	text = truncateRunes(collapseLineBreaks(text), maxMarkdownCodeRunes)

	longestRun, run := 0, 0
	for _, character := range text {
		if character == '`' {
			run++
			longestRun = max(longestRun, run)
		} else {
			run = 0
		}
	}

	fence := strings.Repeat("`", longestRun+1)
	if strings.HasPrefix(text, "`") || strings.HasSuffix(text, "`") {
		return fence + " " + text + " " + fence
	}
	return fence + text + fence
}

// collapseLineBreaks replaces each run of carriage returns and line feeds with a single
// space, leaving every other byte untouched.
//
// Takes text (string) which may span several lines.
//
// Returns string which is the text on one line.
func collapseLineBreaks(text string) string {
	if !strings.ContainsAny(text, "\r\n") {
		return text
	}

	var builder strings.Builder
	builder.Grow(len(text))
	inBreak := false
	for index := range len(text) {
		character := text[index]
		if character == '\r' || character == '\n' {
			if !inBreak {
				builder.WriteByte(' ')
			}
			inBreak = true
			continue
		}
		inBreak = false
		builder.WriteByte(character)
	}
	return builder.String()
}

// truncateRunes cuts text to at most limit runes, appending markdownCodeEllipsis when
// anything was removed so the reader can tell the text is incomplete.
//
// Takes text (string) which is the text to shorten.
// Takes limit (int) which is the maximum number of runes kept.
//
// Returns string which is text unchanged when it fits, or its first limit runes followed
// by the ellipsis.
func truncateRunes(text string, limit int) string {
	count := 0
	for index := range text {
		if count == limit {
			return text[:index] + markdownCodeEllipsis
		}
		count++
	}
	return text
}

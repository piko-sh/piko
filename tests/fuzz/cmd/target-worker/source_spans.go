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

package main

import (
	"cmp"
	"maps"
	"slices"
	"strings"
	"unicode/utf8"

	"piko.sh/piko/internal/sfcparser"
	"piko.sh/piko/tests/fuzz/internal/casefile"
)

// parsedSection holds a parsed source section and its coordinates.
type parsedSection struct {
	// attributes holds parsed section attributes.
	attributes map[string]string

	// tag is the section's element name.
	tag string

	// text is the section's content.
	text string

	// at is the location of the section's opening tag.
	at sfcparser.Location

	// content is the location of the section's first content character.
	content sfcparser.Location
}

// checkSectionSpans checks section order and independently calculated source coordinates.
//
// Takes source (string) which holds the saved source text.
// Takes parsed (*sfcparser.ParseResult) which is the production parser's result.
// Takes want ([]casefile.SourceSection) which holds the independently expected result.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func checkSectionSpans(source string, parsed *sfcparser.ParseResult, want []casefile.SourceSection) error {
	got := parsedSections(parsed)
	if len(got) != len(want) {
		return property("source-section/count", "got %d blocks, want %d", len(got), len(want))
	}

	for index := range got {
		section := &got[index]
		for _, next := range got[index+1:] {
			if section.tag == next.tag && compareLocation(section.at, next.at) >= 0 {
				return property("source-section/order/"+section.tag, "non-monotone %v and %v", section.at, next.at)
			}
		}
	}
	slices.SortFunc(got, func(first, second parsedSection) int { return compareLocation(first.at, second.at) })
	for index := range want {
		if err := checkSectionSpan(source, &got[index], &want[index], index); err != nil {
			return err
		}
	}
	return nil
}

// parsedSections lists the parser's sections, template first, then each repeated kind.
//
// Takes parsed (*sfcparser.ParseResult) which is the production parser's result.
//
// Returns []parsedSection which lists every parsed section.
func parsedSections(parsed *sfcparser.ParseResult) []parsedSection {
	got := make([]parsedSection, 0, 1+len(parsed.Scripts)+len(parsed.Styles)+len(parsed.I18nBlocks)+len(parsed.Timelines))
	got = append(got, parsedSection{
		tag:        "template",
		text:       parsed.Template,
		attributes: parsed.TemplateAttributes,
		at:         parsed.TemplateLocation,
		content:    parsed.TemplateContentLocation,
	})
	for _, block := range parsed.Scripts {
		got = append(got, parsedSection{tag: "script", text: block.Content, attributes: block.Attributes, at: block.Location, content: block.ContentLocation})
	}
	for _, block := range parsed.Styles {
		got = append(got, parsedSection{tag: "style", text: block.Content, attributes: block.Attributes, at: block.Location, content: block.ContentLocation})
	}
	for _, block := range parsed.I18nBlocks {
		got = append(got, parsedSection{tag: "i18n", text: block.Content, attributes: block.Attributes, at: block.Location, content: block.ContentLocation})
	}
	for _, block := range parsed.Timelines {
		got = append(got, parsedSection{tag: "piko:timeline", text: block.Content, attributes: block.Attributes, at: block.Location, content: block.ContentLocation})
	}
	return got
}

// checkSectionSpan compares one parsed section with its saved byte offsets.
//
// Takes source (string) which holds the saved source text.
// Takes observed (*parsedSection) which is the parsed section in document order.
// Takes expected (*casefile.SourceSection) which holds the saved section offsets.
// Takes index (int) which is the section's position in document order.
//
// Returns error which reports invalid saved offsets or a mismatch, or nil.
func checkSectionSpan(source string, observed *parsedSection, expected *casefile.SourceSection, index int) error {
	if expected.Start < 0 || expected.Start >= expected.ContentStart || expected.ContentStart > expected.ContentEnd || expected.ContentEnd > len(source) {
		return property("source-section/model", "invalid saved offsets %+v", *expected)
	}
	if observed.tag != expected.Tag ||
		!maps.Equal(observed.attributes, expected.Attributes) ||
		observed.at != sourcePosition(source, expected.Start) ||
		observed.content != sourcePosition(source, expected.ContentStart) ||
		observed.text != source[expected.ContentStart:expected.ContentEnd] ||
		!strings.HasPrefix(source[expected.ContentEnd:], "</"+expected.Tag+">") {
		return property("source-section/"+expected.Tag, "block %d got tag %s %v at %v content %v %q, want %+v",
			index, observed.tag, observed.attributes, observed.at, observed.content, observed.text, *expected)
	}
	return nil
}

// compareLocation orders source locations by line and column.
//
// Takes first (sfcparser.Location) which is the first source location.
// Takes second (sfcparser.Location) which is the second source location.
//
// Returns int which is negative, zero or positive according to source order.
func compareLocation(first, second sfcparser.Location) int {
	return cmp.Or(cmp.Compare(first.Line, second.Line), cmp.Compare(first.Column, second.Column))
}

// sourcePosition calculates a one-based Unicode location from a byte offset.
//
// Takes source (string) which holds the saved source text.
// Takes offset (int) which is the absolute source byte offset.
//
// Returns sfcparser.Location which is the one-based Unicode source location.
func sourcePosition(source string, offset int) sfcparser.Location {
	prefix := source[:offset]
	return sfcparser.Location{Line: 1 + strings.Count(prefix, "\n"), Column: 1 + utf8.RuneCountInString(prefix[strings.LastIndexByte(prefix, '\n')+1:])}
}

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

package component_domain

import (
	"errors"
	"fmt"
	"strings"
)

var (
	// errTagNameEmpty is returned when a component tag name is empty.
	errTagNameEmpty = errors.New("component tag name cannot be empty")

	// htmlElements contains all standard HTML element names that cannot be used
	// as custom element tag names. This list includes all elements from the
	// HTML Living Standard.
	//
	// Custom elements must not shadow these names to avoid conflicts with
	// browser behaviour.
	htmlElements = map[string]bool{
		"html":       true,
		"base":       true,
		"head":       true,
		"link":       true,
		"meta":       true,
		"style":      true,
		"title":      true,
		"body":       true,
		"address":    true,
		"article":    true,
		"aside":      true,
		"footer":     true,
		"header":     true,
		"h1":         true,
		"h2":         true,
		"h3":         true,
		"h4":         true,
		"h5":         true,
		"h6":         true,
		"hgroup":     true,
		"main":       true,
		"nav":        true,
		"section":    true,
		"search":     true,
		"blockquote": true,
		"dd":         true,
		"div":        true,
		"dl":         true,
		"dt":         true,
		"figcaption": true,
		"figure":     true,
		"hr":         true,
		"li":         true,
		"menu":       true,
		"ol":         true,
		"p":          true,
		"pre":        true,
		"ul":         true,
		"a":          true,
		"abbr":       true,
		"b":          true,
		"bdi":        true,
		"bdo":        true,
		"br":         true,
		"cite":       true,
		"code":       true,
		"data":       true,
		"dfn":        true,
		"em":         true,
		"i":          true,
		"kbd":        true,
		"mark":       true,
		"q":          true,
		"rp":         true,
		"rt":         true,
		"ruby":       true,
		"s":          true,
		"samp":       true,
		"small":      true,
		"span":       true,
		"strong":     true,
		"sub":        true,
		"sup":        true,
		"time":       true,
		"u":          true,
		"var":        true,
		"wbr":        true,
		"area":       true,
		"audio":      true,
		"img":        true,
		"map":        true,
		"track":      true,
		"video":      true,
		"source":     true,
		"embed":      true,
		"iframe":     true,
		"object":     true,
		"param":      true,
		"picture":    true,
		"portal":     true,
		"svg":        true,
		"math":       true,
		"canvas":     true,
		"noscript":   true,
		"script":     true,
		"del":        true,
		"ins":        true,
		"caption":    true,
		"col":        true,
		"colgroup":   true,
		"table":      true,
		"tbody":      true,
		"td":         true,
		"tfoot":      true,
		"th":         true,
		"thead":      true,
		"tr":         true,
		"button":     true,
		"datalist":   true,
		"fieldset":   true,
		"form":       true,
		"input":      true,
		"label":      true,
		"legend":     true,
		"meter":      true,
		"optgroup":   true,
		"option":     true,
		"output":     true,
		"progress":   true,
		"select":     true,
		"textarea":   true,
		"details":    true,
		"dialog":     true,
		"summary":    true,
		"slot":       true,
		"template":   true,
		"acronym":    true,
		"applet":     true,
		"basefont":   true,
		"bgsound":    true,
		"big":        true,
		"blink":      true,
		"center":     true,
		"dir":        true,
		"font":       true,
		"frame":      true,
		"frameset":   true,
		"isindex":    true,
		"keygen":     true,
		"listing":    true,
		"marquee":    true,
		"menuitem":   true,
		"multicol":   true,
		"nextid":     true,
		"nobr":       true,
		"noembed":    true,
		"noframes":   true,
		"plaintext":  true,
		"rb":         true,
		"rtc":        true,
		"spacer":     true,
		"strike":     true,
		"tt":         true,
		"xmp":        true,
	}
)

// IsHTMLElement checks if the given tag name is a standard HTML element.
// The check is case-insensitive.
//
// Takes tagName (string) which is the HTML tag name to check.
//
// Returns bool which is true if the tag name is a standard HTML element.
func IsHTMLElement(tagName string) bool {
	return htmlElements[strings.ToLower(tagName)]
}

// ValidateTagName checks if a tag name is valid for use as a custom element.
//
// A valid custom element tag name must:
//   - Not use reserved prefixes (piko:, pml-)
//   - Contain at least one hyphen (Web Components specification requirement)
//   - Not shadow any standard HTML element name
//
// Takes tagName (string) which is the proposed custom element tag name.
//
// Returns error when the tag name is empty, uses a reserved prefix, lacks a
// hyphen, or shadows a standard HTML element.
func ValidateTagName(tagName string) error {
	if tagName == "" {
		return errTagNameEmpty
	}

	lower := strings.ToLower(tagName)

	if strings.HasPrefix(lower, "piko:") {
		return fmt.Errorf("component tag name %q uses reserved prefix 'piko:'", tagName)
	}

	if strings.HasPrefix(lower, "pml-") {
		return fmt.Errorf("component tag name %q uses reserved prefix 'pml-' (PikoML components)", tagName)
	}

	if !strings.Contains(lower, "-") {
		return fmt.Errorf("component tag name %q must contain a hyphen (Web Components requirement)", tagName)
	}

	if htmlElements[lower] {
		return fmt.Errorf("component tag name %q shadows a standard HTML element", tagName)
	}

	return nil
}

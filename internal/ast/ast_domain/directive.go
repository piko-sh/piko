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

package ast_domain

// Defines directive types and structures for Piko template directives like
// p-if, p-for, p-bind, and p-on. Provides type constants, parsing utilities,
// and metadata for all supported directives including conditionals, loops,
// bindings, and event handlers.

import (
	"fmt"
	"strings"
	"unicode"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// DirectiveType represents a kind of Piko directive and implements
// fmt.Stringer.
type DirectiveType int

const (
	// DirectiveIf is a conditional directive that shows or hides an element.
	DirectiveIf DirectiveType = iota

	// DirectiveElseIf represents a p-else-if conditional directive.
	DirectiveElseIf

	// DirectiveElse represents a p-else conditional directive.
	DirectiveElse

	// DirectiveFor represents a p-for loop directive.
	DirectiveFor

	// DirectiveShow represents a p-show visibility directive.
	DirectiveShow

	// DirectiveBind represents a p-bind attribute binding directive.
	DirectiveBind

	// DirectiveModel represents a p-model two-way binding directive.
	DirectiveModel

	// DirectiveOn represents a p-on event handler directive.
	DirectiveOn

	// DirectiveEvent represents a custom event directive (p-event).
	DirectiveEvent

	// DirectiveClass represents a p-class conditional class directive.
	DirectiveClass

	// DirectiveStyle represents a p-style conditional style directive.
	DirectiveStyle

	// DirectiveText represents a p-text text content directive.
	DirectiveText

	// DirectiveHTML represents a p-html raw HTML directive.
	DirectiveHTML

	// DirectiveRef represents a p-ref element reference directive.
	DirectiveRef

	// DirectiveSlot represents a p-slot directive for assigning an element to a
	// named slot.
	DirectiveSlot

	// DirectiveKey represents a p-key keying directive for reconciliation.
	DirectiveKey

	// DirectiveContext represents a p-context scoping directive.
	DirectiveContext

	// DirectiveScaffold represents a p-scaffold directive.
	DirectiveScaffold

	// DirectiveTimeline represents a p-timeline directive for animation timeline
	// control (e.g. p-timeline:hidden).
	DirectiveTimeline
)

// Directive represents a parsed directive from a template attribute.
type Directive struct {
	// Expression holds the parsed form of this directive's value.
	Expression Expression

	// ChainKey links p-else-if and p-else directives back to the originating p-if
	// node. Contains the p-if node's Key expression for chain formation; nil for
	// other directive types.
	ChainKey Expression

	// GoAnnotations holds Go-specific metadata for this directive; nil if not set.
	GoAnnotations *GoGeneratorAnnotation

	// Arg stores the primary argument from the directive name. This is the event
	// name for p-on/p-event (e.g., "click") or attribute name for p-bind (e.g.,
	// "class").
	Arg string

	// Modifier stores an internal modifier set during semantic analysis.
	//
	// For p-on and p-event directives, this is "action" or "helper" (set
	// programmatically, not by the user).
	Modifier string

	// RawExpression is the original directive value before parsing.
	RawExpression string

	// EventModifiers stores user-facing event modifiers parsed from
	// the template attribute name (e.g., ["prevent", "stop"]), only
	// meaningful for DirectiveOn/DirectiveEvent.
	//
	// Supported modifiers: prevent, stop, once, self.
	EventModifiers []string

	// Location is where the directive value starts in the source file.
	Location Location

	// NameLocation is where the directive name starts in the source.
	NameLocation Location

	// AttributeRange is the full span of the directive attribute in the source.
	AttributeRange Range

	// Type specifies the kind of directive.
	Type DirectiveType

	// IsStaticEvent indicates whether this event directive's
	// expression uses only client-side values (like $event,
	// function names, static literals) and no dynamic template
	// scope variables, only meaningful for
	// DirectiveOn/DirectiveEvent.
	//
	// When true, nodes with this event can still be considered
	// for static hoisting. Set by the annotator during semantic
	// analysis.
	IsStaticEvent bool
}

var (
	// StructuralDirectives is a set of all directive names that are not
	// event-related. This is used by the parser to quickly identify directive
	// attributes.
	StructuralDirectives = map[string]bool{
		"p-if":       true,
		"p-else-if":  true,
		"p-else":     true,
		"p-for":      true,
		"p-show":     true,
		"p-class":    true,
		"p-style":    true,
		"p-text":     true,
		"p-html":     true,
		"p-bind":     true,
		"p-model":    true,
		"p-ref":      true,
		"p-slot":     true,
		"p-key":      true,
		"p-context":  true,
		"p-scaffold": true,
	}

	// EventDirectives is a set of all event-related directive prefixes.
	EventDirectives = map[string]bool{
		"p-on":       true,
		"p-event":    true,
		"p-timeline": true,
	}

	// DirectiveNameToType maps static directive names to their corresponding
	// DirectiveType.
	DirectiveNameToType = map[string]DirectiveType{
		"p-if":       DirectiveIf,
		"p-else-if":  DirectiveElseIf,
		"p-else":     DirectiveElse,
		"p-for":      DirectiveFor,
		"p-show":     DirectiveShow,
		"p-bind":     DirectiveBind,
		"p-on":       DirectiveOn,
		"p-event":    DirectiveEvent,
		"p-model":    DirectiveModel,
		"p-ref":      DirectiveRef,
		"p-slot":     DirectiveSlot,
		"p-class":    DirectiveClass,
		"p-style":    DirectiveStyle,
		"p-text":     DirectiveText,
		"p-html":     DirectiveHTML,
		"p-key":      DirectiveKey,
		"p-context":  DirectiveContext,
		"p-scaffold": DirectiveScaffold,
		"p-timeline": DirectiveTimeline,
	}

	// DirectiveTypeToName maps DirectiveType constants to their string directive
	// names.
	DirectiveTypeToName = map[DirectiveType]string{
		DirectiveIf:       "p-if",
		DirectiveElseIf:   "p-else-if",
		DirectiveElse:     "p-else",
		DirectiveFor:      "p-for",
		DirectiveShow:     "p-show",
		DirectiveBind:     "p-bind",
		DirectiveOn:       "p-on",
		DirectiveEvent:    "p-event",
		DirectiveModel:    "p-model",
		DirectiveRef:      "p-ref",
		DirectiveSlot:     "p-slot",
		DirectiveClass:    "p-class",
		DirectiveStyle:    "p-style",
		DirectiveText:     "p-text",
		DirectiveHTML:     "p-html",
		DirectiveKey:      "p-key",
		DirectiveContext:  "p-context",
		DirectiveScaffold: "p-scaffold",
		DirectiveTimeline: "p-timeline",
	}
)

// String returns the readable name of the directive type.
//
// Returns string which is the name in title case, or "Unknown(n)" for values
// that are not known.
func (d DirectiveType) String() string {
	titleCaser := cases.Title(language.English)
	for name, typ := range DirectiveNameToType {
		if typ == d {
			parts := strings.Split(name[2:], "-")
			for i, part := range parts {
				parts[i] = titleCaser.String(part)
			}
			return strings.Join(parts, "")
		}
	}

	return fmt.Sprintf("Unknown(%d)", d)
}

// IsValidJSIdentifier reports whether name is a valid JavaScript identifier.
// A valid identifier starts with a letter, underscore, or dollar sign, and
// contains only letters, digits, underscores, or dollar signs.
//
// Takes name (string) which is the identifier to check.
//
// Returns bool which is true if name is a valid JavaScript identifier.
func IsValidJSIdentifier(name string) bool {
	if name == "" {
		return false
	}
	for i, r := range name {
		if i == 0 {
			if !unicode.IsLetter(r) && r != '_' && r != '$' {
				return false
			}
		} else {
			if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' && r != '$' {
				return false
			}
		}
	}
	return true
}

// resolveDirectiveType looks up a directive type by its key name.
//
// Takes key ([]byte) which is the directive name to look up.
//
// Returns DirectiveType which is the matching directive type.
// Returns bool which is true if the directive was found, false otherwise.
func resolveDirectiveType(key []byte) (DirectiveType, bool) {
	dirType, found := DirectiveNameToType[string(key)]
	return dirType, found
}

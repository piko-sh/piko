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
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"piko.sh/piko/internal/htmllexer"
	"piko.sh/piko/internal/json"
)

const (
	// millisecondsPerSecond is the conversion factor from
	// milliseconds to seconds.
	millisecondsPerSecond = 1000

	// attrRef is the HTML attribute name used to identify the
	// target element by p-ref.
	attrRef = "ref"

	// pikoTagPrefix is the namespace prefix for all piko timeline
	// action tags (e.g. piko:show, piko:type).
	pikoTagPrefix = "piko:"

	// errFormatOutsideAt is the error format string used when an
	// action element appears outside a piko:at block.
	errFormatOutsideAt = "%s must be inside a piko:at element"
)

// timelineAction represents a single timed action in a piko:timeline block.
type timelineAction struct {
	// Action is the type of action: "show", "hide", "type", "typehtml",
	// "addclass", "removeclass", "tooltip", or a custom user-defined name.
	Action string `json:"action"`

	// Ref is the p-ref name of the target element.
	Ref string `json:"ref"`

	// Params holds additional attributes for custom (user-defined) actions.
	// Omitted from JSON when empty.
	Params map[string]string `json:"params,omitempty"`

	// Class is the CSS class name to add or remove (addclass/removeclass only).
	// Omitted from JSON when empty.
	Class string `json:"class,omitempty"`

	// Value is the tooltip text to set, where an empty value clears
	// the tooltip (tooltip only, omitted from JSON when empty).
	Value string `json:"value,omitempty"`

	// Time is the absolute time in seconds when this action triggers.
	Time float64 `json:"time"`

	// Speed is the typing speed in milliseconds per character (type/typehtml only).
	// Omitted from JSON when zero.
	Speed float64 `json:"speed,omitempty"`
}

// ParseTimeline parses the content of a piko:timeline block into a
// JSON string suitable for compiled JavaScript injection.
//
// Takes content (string) which is the raw markup inside the
// <piko:timeline> tag, containing <piko:at> elements with nested
// action elements like <piko:show>, <piko:hide>, and <piko:type>.
//
// Returns a JSON array string of timeline actions.
// Returns an error when parsing fails or time values are invalid.
func ParseTimeline(content string) (string, error) {
	if strings.TrimSpace(content) == "" {
		return "[]", nil
	}

	var lexer htmllexer.Lexer
	lexer.Init([]byte(content))

	var (
		actions     []timelineAction
		currentTime float64
		insideAt    bool
	)

	for {
		tt := lexer.Next()

		switch tt {
		case htmllexer.ErrorToken:
			return serialiseActions(actions)

		case htmllexer.StartTagToken, htmllexer.StartTagVoidToken:
			tagName := string(bytes.ToLower(lexer.Text()))
			attrs := readAttributes(&lexer)
			a, err := parseTimelineTag(tagName, attrs, &currentTime, &insideAt)
			if err != nil {
				return "", err
			}
			if a != nil {
				actions = append(actions, *a)
			}

		case htmllexer.EndTagToken:
			if string(bytes.ToLower(lexer.Text())) == "piko:at" {
				insideAt = false
			}
		}
	}
}

// parseTimelineTag dispatches a start tag to the appropriate
// handler based on the tag name.
//
// Takes tagName (string) which is the lowercased HTML tag name.
// Takes attrs (map[string]string) which contains the tag's
// attribute key-value pairs.
// Takes currentTime (*float64) which tracks the current timeline
// position in seconds.
// Takes insideAt (*bool) which tracks whether parsing is inside
// a piko:at block.
//
// Returns a timeline action if the tag produces one, or nil when
// the tag only updates state.
func parseTimelineTag(
	tagName string,
	attrs map[string]string,
	currentTime *float64,
	insideAt *bool,
) (*timelineAction, error) {
	switch tagName {
	case "piko:at":
		return parseAtDirective(attrs, currentTime, insideAt)
	case "piko:show", "piko:hide":
		return parseRefAction(tagName, attrs, *currentTime, *insideAt)
	case "piko:type", "piko:typehtml":
		return parseTypingAction(tagName, attrs, *currentTime, *insideAt)
	case "piko:addclass", "piko:removeclass":
		return parseClassAction(tagName, attrs, *currentTime, *insideAt)
	case "piko:tooltip":
		return parseTooltipAction(attrs, *currentTime, *insideAt)
	default:
		if !strings.HasPrefix(tagName, pikoTagPrefix) {
			return nil, nil
		}
		return parseGenericAction(tagName, attrs, *currentTime, *insideAt)
	}
}

// parseGenericAction handles unknown piko:* elements by passing
// through all attributes as key-value params, enabling user-defined
// custom timeline actions.
//
// Takes tagName (string) which is the lowercased HTML tag name.
// Takes attrs (map[string]string) which contains the tag's
// attribute key-value pairs.
// Takes currentTime (float64) which is the current timeline
// position in seconds.
// Takes insideAt (bool) which indicates whether parsing is
// inside a piko:at block.
//
// Returns a timeline action with the action name derived from
// the tag, an optional ref, and all other attributes in Params.
func parseGenericAction(tagName string, attrs map[string]string, currentTime float64, insideAt bool) (*timelineAction, error) {
	if !insideAt {
		return nil, fmt.Errorf(errFormatOutsideAt, tagName)
	}
	action := strings.TrimPrefix(tagName, pikoTagPrefix)
	ref := attrs[attrRef]
	params := make(map[string]string, len(attrs))
	for k, v := range attrs {
		if k != attrRef {
			params[k] = v
		}
	}
	return &timelineAction{Time: currentTime, Action: action, Ref: ref, Params: params}, nil
}

// parseAtDirective handles a <piko:at time="..."> element by
// updating the current time and entering a piko:at block.
//
// Takes attrs (map[string]string) which contains the tag's
// attribute key-value pairs.
// Takes currentTime (*float64) which is updated to the parsed
// time value.
// Takes insideAt (*bool) which is set to true.
//
// Returns nil action on success since piko:at only updates state.
func parseAtDirective(attrs map[string]string, currentTime *float64, insideAt *bool) (*timelineAction, error) {
	timeStr, ok := attrs["time"]
	if !ok {
		return nil, errors.New("piko:at element missing required 'time' attribute")
	}
	t, err := parseTimeValue(timeStr)
	if err != nil {
		return nil, fmt.Errorf("invalid time value %q: %w", timeStr, err)
	}
	*currentTime = t
	*insideAt = true
	return nil, nil
}

// parseRefAction handles <piko:show> and <piko:hide> elements
// that require only a ref attribute.
//
// Takes tagName (string) which is the lowercased HTML tag name.
// Takes attrs (map[string]string) which contains the tag's
// attribute key-value pairs.
// Takes currentTime (float64) which is the current timeline
// position in seconds.
// Takes insideAt (bool) which indicates whether parsing is
// inside a piko:at block.
//
// Returns a timeline action with the ref and action type set.
func parseRefAction(tagName string, attrs map[string]string, currentTime float64, insideAt bool) (*timelineAction, error) {
	if !insideAt {
		return nil, fmt.Errorf(errFormatOutsideAt, tagName)
	}
	ref, ok := attrs[attrRef]
	if !ok {
		return nil, fmt.Errorf("%s element missing required 'ref' attribute", tagName)
	}
	action := strings.TrimPrefix(tagName, pikoTagPrefix)
	return &timelineAction{Time: currentTime, Action: action, Ref: ref}, nil
}

// parseTypingAction handles <piko:type> and <piko:typehtml>
// elements that have a ref and an optional speed attribute.
//
// Takes tagName (string) which is the lowercased HTML tag name.
// Takes attrs (map[string]string) which contains the tag's
// attribute key-value pairs.
// Takes currentTime (float64) which is the current timeline
// position in seconds.
// Takes insideAt (bool) which indicates whether parsing is
// inside a piko:at block.
//
// Returns a timeline action with the ref, action type, and
// optional speed set.
func parseTypingAction(tagName string, attrs map[string]string, currentTime float64, insideAt bool) (*timelineAction, error) {
	if !insideAt {
		return nil, fmt.Errorf(errFormatOutsideAt, tagName)
	}
	ref, ok := attrs[attrRef]
	if !ok {
		return nil, fmt.Errorf("%s element missing required 'ref' attribute", tagName)
	}
	action := strings.TrimPrefix(tagName, pikoTagPrefix)
	a := &timelineAction{Time: currentTime, Action: action, Ref: ref}
	if speedStr, hasSpeed := attrs["speed"]; hasSpeed {
		speed, err := strconv.ParseFloat(speedStr, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid speed value %q: %w", speedStr, err)
		}
		a.Speed = speed
	}
	return a, nil
}

// parseClassAction handles <piko:addclass> and <piko:removeclass>
// elements that require both a ref and a class attribute.
//
// Takes tagName (string) which is the lowercased HTML tag name.
// Takes attrs (map[string]string) which contains the tag's
// attribute key-value pairs.
// Takes currentTime (float64) which is the current timeline
// position in seconds.
// Takes insideAt (bool) which indicates whether parsing is
// inside a piko:at block.
//
// Returns a timeline action with the ref, action type, and class
// name set.
func parseClassAction(tagName string, attrs map[string]string, currentTime float64, insideAt bool) (*timelineAction, error) {
	if !insideAt {
		return nil, fmt.Errorf(errFormatOutsideAt, tagName)
	}
	ref, ok := attrs[attrRef]
	if !ok {
		return nil, fmt.Errorf("%s element missing required 'ref' attribute", tagName)
	}
	class, ok := attrs["class"]
	if !ok {
		return nil, fmt.Errorf("%s element missing required 'class' attribute", tagName)
	}
	action := strings.TrimPrefix(tagName, pikoTagPrefix)
	return &timelineAction{Time: currentTime, Action: action, Ref: ref, Class: class}, nil
}

// parseTooltipAction handles <piko:tooltip> elements that require
// a ref and have an optional value attribute.
//
// Takes attrs (map[string]string) which contains the tag's
// attribute key-value pairs.
// Takes currentTime (float64) which is the current timeline
// position in seconds.
// Takes insideAt (bool) which indicates whether parsing is
// inside a piko:at block.
//
// Returns a timeline action with the ref, tooltip action, and
// optional value set.
func parseTooltipAction(attrs map[string]string, currentTime float64, insideAt bool) (*timelineAction, error) {
	if !insideAt {
		return nil, fmt.Errorf(errFormatOutsideAt, "piko:tooltip")
	}
	ref, ok := attrs[attrRef]
	if !ok {
		return nil, errors.New("piko:tooltip element missing required 'ref' attribute")
	}
	return &timelineAction{Time: currentTime, Action: "tooltip", Ref: ref, Value: attrs["value"]}, nil
}

// parseTimeValue converts a time string to seconds, supporting
// "s", "ms", or bare number formats.
//
// Takes s (string) which is the time value to parse (e.g.
// "1.5s", "1500ms", "0.252434").
//
// Returns the time in seconds.
// Returns an error when the value cannot be parsed.
func parseTimeValue(s string) (float64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, errors.New("empty time value")
	}

	if rest, found := strings.CutSuffix(s, "ms"); found {
		ms, err := strconv.ParseFloat(rest, 64)
		if err != nil {
			return 0, err
		}
		return ms / millisecondsPerSecond, nil
	}

	if rest, found := strings.CutSuffix(s, "s"); found {
		return strconv.ParseFloat(rest, 64)
	}

	return strconv.ParseFloat(s, 64)
}

// readAttributes reads all attribute tokens from the lexer until
// the start tag closes.
//
// Takes lexer (*htmllexer.Lexer) which is the HTML lexer positioned
// after a start tag token.
//
// Returns a map of attribute name to value pairs.
func readAttributes(lexer *htmllexer.Lexer) map[string]string {
	attrs := make(map[string]string)
	for {
		tt := lexer.Next()
		if tt != htmllexer.AttributeToken {
			return attrs
		}
		key := string(lexer.Text())
		val := lexer.AttrVal()
		if len(val) >= 2 && (val[0] == '"' || val[0] == '\'') {
			val = val[1 : len(val)-1]
		}
		attrs[key] = string(val)
	}
}

// serialiseActions converts a slice of timeline actions to a JSON
// string.
//
// Takes actions ([]timelineAction) which is the slice of actions
// to serialise.
//
// Returns an empty array "[]" when actions is nil or empty.
// Returns an error when JSON marshalling fails.
func serialiseActions(actions []timelineAction) (string, error) {
	if len(actions) == 0 {
		return "[]", nil
	}
	data, err := json.Marshal(actions)
	if err != nil {
		return "", fmt.Errorf("serialising timeline actions: %w", err)
	}
	return string(data), nil
}

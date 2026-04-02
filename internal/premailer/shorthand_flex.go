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

package premailer

import "strings"

const (
	// cssFlexGrow is the CSS flex-grow property name.
	cssFlexGrow = "flex-grow"

	// cssFlexShrink is the CSS flex-shrink property name.
	cssFlexShrink = "flex-shrink"

	// cssFlexBasis is the CSS flex-basis property name.
	cssFlexBasis = "flex-basis"

	// cssZero is the CSS zero value.
	cssZero = "0"

	// cssAuto is the CSS auto keyword.
	cssAuto = "auto"

	// flexThreeValues is the number of values in a three-value flex shorthand.
	flexThreeValues = 3
)

// expandFlexShorthand expands a CSS flex shorthand value into its longhand
// properties: flex-grow, flex-shrink, and flex-basis.
//
// The flex shorthand accepts one, two, or three values:
//   - Single number: sets flex-grow, with flex-shrink=1, flex-basis=0.
//   - Single length/percentage/auto: sets flex-basis only.
//   - Two values: flex-grow and flex-shrink, with flex-basis=0.
//   - Three values: flex-grow, flex-shrink, flex-basis.
//   - Keywords "none" and "auto" have special meanings.
//
// Takes value (string) which is the flex shorthand value to expand.
//
// Returns map[string]string which maps longhand property names to values,
// or nil when the value is empty.
func expandFlexShorthand(value string) map[string]string {
	switch value {
	case "none":
		return map[string]string{
			cssFlexGrow:   cssZero,
			cssFlexShrink: cssZero,
			cssFlexBasis:  cssAuto,
		}
	case cssAuto:
		return map[string]string{
			cssFlexGrow:   "1",
			cssFlexShrink: "1",
			cssFlexBasis:  cssAuto,
		}
	}

	parts := strings.Fields(value)
	switch len(parts) {
	case 1:
		return expandFlexSingleValue(parts[0])
	case 2:
		return map[string]string{
			cssFlexGrow:   parts[0],
			cssFlexShrink: parts[1],
			cssFlexBasis:  cssZero,
		}
	case flexThreeValues:
		return map[string]string{
			cssFlexGrow:   parts[0],
			cssFlexShrink: parts[1],
			cssFlexBasis:  parts[2],
		}
	default:
		return nil
	}
}

// expandFlexSingleValue expands a single-value flex shorthand. A numeric value
// is treated as flex-grow; a length, percentage, or "auto" is treated as
// flex-basis.
//
// Takes value (string) which is the single flex value.
//
// Returns map[string]string which contains the expanded longhand properties.
func expandFlexSingleValue(value string) map[string]string {
	if isFlexBasisValue(value) {
		return map[string]string{
			cssFlexBasis: value,
		}
	}
	return map[string]string{
		cssFlexGrow:   value,
		cssFlexShrink: "1",
		cssFlexBasis:  cssZero,
	}
}

// isFlexBasisValue checks whether a value looks like a flex-basis value
// (a CSS length, percentage, or "auto") rather than a unitless flex-grow
// number.
//
// Takes value (string) which is the CSS value to check.
//
// Returns bool which is true if the value is auto or ends with a length/
// percentage unit.
func isFlexBasisValue(value string) bool {
	if value == cssAuto || value == "content" {
		return true
	}
	return strings.HasSuffix(value, "px") ||
		strings.HasSuffix(value, "em") ||
		strings.HasSuffix(value, "rem") ||
		strings.HasSuffix(value, "%") ||
		strings.HasSuffix(value, "vw") ||
		strings.HasSuffix(value, "vh")
}

// expandFlexFlowShorthand expands a CSS flex-flow shorthand into flex-direction
// and flex-wrap longhand properties. The two values can appear in either order
// and both are optional.
//
// Takes value (string) which is the flex-flow shorthand value.
//
// Returns map[string]string which maps longhand property names to values,
// or nil when no valid properties are found.
func expandFlexFlowShorthand(value string) map[string]string {
	result := make(map[string]string)
	for part := range strings.FieldsSeq(value) {
		switch part {
		case "row", "row-reverse", "column", "column-reverse":
			result["flex-direction"] = part
		case "nowrap", "wrap", "wrap-reverse":
			result["flex-wrap"] = part
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

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

import (
	"fmt"
	"strings"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/esbuild/css_ast"
	"piko.sh/piko/internal/esbuild/css_lexer"
)

// problematicProperties defines CSS properties that have poor or inconsistent
// support in major email clients, especially Outlook. This map is used to
// generate warnings.
//
// NOTE: MSO-specific properties (mso-*) are intentionally NOT included in
// this map. These are Outlook-specific CSS extensions that SHOULD be
// preserved and used for fixing Outlook rendering issues. Common MSO
// properties include:
//   - mso-table-lspace: 0pt (removes default table left spacing in Outlook)
//   - mso-table-rspace: 0pt (removes default table right spacing in Outlook)
//   - mso-line-height-rule: exactly (fixes line-height issues in Outlook)
//   - mso-margin-top-alt, mso-padding-alt (Outlook margin/padding fixes)
//
// These properties are safe to use and will not trigger validation warnings.
var problematicProperties = map[string]string{
	"flex":                  "Flexbox is not supported in Outlook and has limited support in many email clients.",
	"flex-direction":        "Flexbox is not supported in Outlook and has limited support in many email clients.",
	"flex-wrap":             "Flexbox is not supported in Outlook and has limited support in many email clients.",
	"flex-flow":             "Flexbox is not supported in Outlook and has limited support in many email clients.",
	"flex-grow":             "Flexbox is not supported in Outlook and has limited support in many email clients.",
	"flex-shrink":           "Flexbox is not supported in Outlook and has limited support in many email clients.",
	"flex-basis":            "Flexbox is not supported in Outlook and has limited support in many email clients.",
	"justify-content":       "Flexbox is not supported in Outlook and has limited support in many email clients.",
	"align-items":           "Flexbox is not supported in Outlook and has limited support in many email clients.",
	"align-content":         "Flexbox is not supported in Outlook and has limited support in many email clients.",
	"align-self":            "Flexbox is not supported in Outlook and has limited support in many email clients.",
	"order":                 "Flexbox is not supported in Outlook and has limited support in many email clients.",
	"grid":                  "CSS Grid is not supported in most email clients.",
	"grid-template":         "CSS Grid is not supported in most email clients.",
	"grid-template-rows":    "CSS Grid is not supported in most email clients.",
	"grid-template-columns": "CSS Grid is not supported in most email clients.",
	"grid-template-areas":   "CSS Grid is not supported in most email clients.",
	"grid-area":             "CSS Grid is not supported in most email clients.",
	"grid-row":              "CSS Grid is not supported in most email clients.",
	"grid-column":           "CSS Grid is not supported in most email clients.",
	"grid-gap":              "CSS Grid is not supported in most email clients.",
	"grid-row-gap":          "CSS Grid is not supported in most email clients.",
	"grid-column-gap":       "CSS Grid is not supported in most email clients.",
	"gap":                   "The 'gap' property is not supported in most email clients.",

	"float":        "The 'float' property is not reliably supported, especially in Outlook, and can break layouts.",
	"clear":        "The 'clear' property is not reliably supported as it depends on 'float'.",
	"position":     "The 'position' property is unreliable. 'absolute' and 'relative' have buggy support, while 'fixed' and 'sticky' do not work at all.",
	"columns":      "Multi-column layout ('columns', 'column-count', etc.) is not supported in most email clients.",
	"column-count": "Multi-column layout is not supported in most email clients.",
	"column-width": "Multi-column layout is not supported in most email clients.",
	"column-gap":   "Multi-column layout is not supported in most email clients.",
	"column-rule":  "Multi-column layout is not supported in most email clients.",

	"box-shadow":      "The 'box-shadow' property is not supported in Outlook, Gmail, and other major clients.",
	"border-radius":   "The 'border-radius' property is not supported in many clients, including most versions of Outlook.",
	"opacity":         "The 'opacity' property is not supported in most major email clients.",
	"box-sizing":      "'box-sizing: border-box' is not reliably supported, which can cause layout issues in Outlook.",
	"outline":         "The 'outline' property has inconsistent support and should be avoided in favour of borders.",
	"filter":          "CSS filters are not supported in email clients.",
	"backdrop-filter": "CSS backdrop-filter is not supported in email clients.",
	"clip-path":       "The 'clip-path' property is not supported in most email clients.",
	"object-fit":      "The 'object-fit' property is not supported in Outlook and many email clients.",
	"object-position": "The 'object-position' property is not supported in Outlook and many email clients.",

	"transform":                  "CSS transforms are not supported in Outlook and have limited support in email clients.",
	"transform-origin":           "CSS transforms are not supported in Outlook and have limited support in email clients.",
	"animation":                  "CSS animations are stripped by most email clients.",
	"animation-name":             "CSS animations are stripped by most email clients.",
	"animation-duration":         "CSS animations are stripped by most email clients.",
	"animation-timing-function":  "CSS animations are stripped by most email clients.",
	"animation-delay":            "CSS animations are stripped by most email clients.",
	"animation-iteration-count":  "CSS animations are stripped by most email clients.",
	"animation-direction":        "CSS animations are stripped by most email clients.",
	"animation-fill-mode":        "CSS animations are stripped by most email clients.",
	"animation-play-state":       "CSS animations are stripped by most email clients.",
	"transition":                 "CSS transitions are stripped by most email clients.",
	"transition-property":        "CSS transitions are stripped by most email clients.",
	"transition-duration":        "CSS transitions are stripped by most email clients.",
	"transition-timing-function": "CSS transitions are stripped by most email clients.",
	"transition-delay":           "CSS transitions are stripped by most email clients.",

	"background-size":       "The 'background-size' property is not supported in Outlook or Gmail and requires VML workarounds.",
	"background-blend-mode": "CSS blend modes are not supported in email.",
	"background-clip":       "The 'background-clip' property has inconsistent support.",

	"text-shadow":       "The 'text-shadow' property is not supported in most clients except Apple Mail/iOS.",
	"word-wrap":         "The 'word-wrap' or 'overflow-wrap' properties for breaking long words are not consistently supported.",
	"overflow-wrap":     "The 'word-wrap' or 'overflow-wrap' properties for breaking long words are not consistently supported.",
	"word-break":        "The 'word-break' property is not consistently supported.",
	"text-overflow":     "The 'text-overflow' property has limited support for handling truncated text.",
	"counter-reset":     "CSS counters are not supported in email.",
	"counter-increment": "CSS counters are not supported in email.",

	"fill":         "SVG properties like 'fill' only work in clients that support inline SVG (e.g., Apple Mail).",
	"stroke":       "SVG properties like 'stroke' only work in clients that support inline SVG (e.g., Apple Mail).",
	"stroke-width": "SVG properties like 'stroke-width' only work in clients that support inline SVG (e.g., Apple Mail).",
}

// validateEmailCompatibility checks CSS rules for properties and values that
// may not work well in email clients.
//
// Takes ruleSet (*RuleSet) which contains the CSS rules to check.
// Takes sourcePath (string) which identifies the source file for diagnostics.
//
// Returns []*ast_domain.Diagnostic which contains warnings for any problems
// found.
func validateEmailCompatibility(ruleSet *RuleSet, sourcePath string) []*ast_domain.Diagnostic {
	diagnostics := make([]*ast_domain.Diagnostic, 0, len(ruleSet.InlineableRules))

	for _, rule := range ruleSet.InlineableRules {
		diagnostics = append(diagnostics, validateRuleProperties(rule, sourcePath)...)
	}

	diagnostics = append(diagnostics, checkLeftoverRules(ruleSet.LeftoverRules, sourcePath)...)

	return diagnostics
}

// validateRuleProperties checks all properties in a style rule for email
// client support problems.
//
// Takes rule (styleRule) which contains the CSS properties to check.
// Takes sourcePath (string) which identifies the source file for diagnostics.
//
// Returns []*ast_domain.Diagnostic which contains any warnings found for
// properties that may not work in email clients.
func validateRuleProperties(rule styleRule, sourcePath string) []*ast_domain.Diagnostic {
	var diagnostics []*ast_domain.Diagnostic

	for propName, propValue := range rule.properties {
		specificDiags := validatePropertyValue(propName, propValue.value, sourcePath)
		if len(specificDiags) > 0 {
			diagnostics = append(diagnostics, specificDiags...)
			continue
		}

		if message, isProblematic := problematicProperties[propName]; isProblematic {
			diagnostic := ast_domain.NewDiagnostic(
				ast_domain.Warning,
				fmt.Sprintf("%s. Property: '%s: %s' in selector '%s'", message, propName, propValue.value, rule.selector),
				propName,
				ast_domain.Location{},
				sourcePath,
			)
			diagnostics = append(diagnostics, diagnostic)
		}
	}

	return diagnostics
}

// validatePropertyValue checks CSS values for problems in email clients.
//
// Takes propName (string) which is the CSS property name to check.
// Takes propValue (string) which is the CSS property value to check.
// Takes sourcePath (string) which identifies the source file for diagnostics.
//
// Returns []*ast_domain.Diagnostic which contains warnings for values that do
// not work well in email clients, or nil if the value needs no special checks.
func validatePropertyValue(propName, propValue, sourcePath string) []*ast_domain.Diagnostic {
	value := strings.ToLower(strings.TrimSpace(propValue))

	if strings.Contains(value, "calc(") {
		return []*ast_domain.Diagnostic{
			ast_domain.NewDiagnostic(
				ast_domain.Warning,
				"The CSS calc() function is not supported in most major email clients, including Outlook and Gmail. Use percentage- or pixel-based tables for fluid layouts.",
				propName, ast_domain.Location{}, sourcePath,
			),
		}
	}

	switch propName {
	case propDisplay:
		return validateDisplayValue(value, sourcePath)
	case propPosition:
		return validatePositionValue(value, sourcePath)
	case "background", "background-image":
		return validateBackgroundValue(value, propName, sourcePath)
	}

	return nil
}

// validateDisplayValue checks for CSS display values that do not work well in
// email clients.
//
// Takes value (string) which is the CSS display value to check.
// Takes sourcePath (string) which is the file path for error reporting.
//
// Returns []*ast_domain.Diagnostic which holds warnings for display values
// like flex or grid that email clients do not support, or nil when the value
// is safe to use.
func validateDisplayValue(value, sourcePath string) []*ast_domain.Diagnostic {
	if value == "flex" || value == "inline-flex" {
		return []*ast_domain.Diagnostic{
			ast_domain.NewDiagnostic(
				ast_domain.Warning,
				fmt.Sprintf("The 'display: %s' value is not supported in Outlook and has limited support in other email clients. Use table-based layouts for compatibility.", value),
				propDisplay, ast_domain.Location{}, sourcePath,
			),
		}
	}

	if value == "grid" || value == "inline-grid" {
		return []*ast_domain.Diagnostic{
			ast_domain.NewDiagnostic(
				ast_domain.Warning,
				fmt.Sprintf("The 'display: %s' value is not supported in most email clients. Use table-based layouts for compatibility.", value),
				propDisplay, ast_domain.Location{}, sourcePath,
			),
		}
	}

	return nil
}

// validatePositionValue checks if a position property value works in email
// clients.
//
// Takes value (string) which is the position property value to check.
// Takes sourcePath (string) which identifies the source file for diagnostics.
//
// Returns []*ast_domain.Diagnostic which contains warnings for position values
// like "fixed" or "sticky" that email clients do not support, or nil if the
// value is valid.
func validatePositionValue(value, sourcePath string) []*ast_domain.Diagnostic {
	if value == "fixed" || value == "sticky" {
		return []*ast_domain.Diagnostic{
			ast_domain.NewDiagnostic(
				ast_domain.Warning,
				fmt.Sprintf("The 'position: %s' value is not supported in any major email client.", value),
				propPosition, ast_domain.Location{}, sourcePath,
			),
		}
	}

	return nil
}

// validateBackgroundValue checks a background property value for problems.
//
// Takes value (string) which is the CSS background value to check.
// Takes propName (string) which is the property name for error messages.
// Takes sourcePath (string) which is the file path for error messages.
//
// Returns []*ast_domain.Diagnostic which holds warnings for background values
// that email clients do not support, such as CSS gradients. Returns nil when
// the value is acceptable.
func validateBackgroundValue(value, propName, sourcePath string) []*ast_domain.Diagnostic {
	if strings.Contains(value, "gradient") {
		return []*ast_domain.Diagnostic{
			ast_domain.NewDiagnostic(
				ast_domain.Warning,
				"CSS gradients (linear-gradient, radial-gradient) are not supported in Outlook, Gmail, and other major clients. Use a solid 'background-color' as a fallback.",
				propName, ast_domain.Location{}, sourcePath,
			),
		}
	}

	return nil
}

// checkLeftoverRules checks CSS rules that were not inlined for problems.
// It processes rules one by one, including nested rules in @media queries.
//
// Takes rules ([]css_ast.Rule) which contains the CSS rules to check.
// Takes sourcePath (string) which identifies the source file for error reports.
//
// Returns []*ast_domain.Diagnostic which contains any problems found.
func checkLeftoverRules(rules []css_ast.Rule, sourcePath string) []*ast_domain.Diagnostic {
	diagnostics := make([]*ast_domain.Diagnostic, 0, len(rules))

	for _, rule := range rules {
		diagnostics = append(diagnostics, checkLeftoverRule(rule, sourcePath)...)
	}

	return diagnostics
}

// checkLeftoverRule checks a single leftover rule for problems.
//
// Takes rule (css_ast.Rule) which is the CSS rule to check.
// Takes sourcePath (string) which identifies the source file for diagnostics.
//
// Returns []*ast_domain.Diagnostic which contains any problems found, or nil if
// the rule type is not checked.
func checkLeftoverRule(rule css_ast.Rule, sourcePath string) []*ast_domain.Diagnostic {
	switch r := rule.Data.(type) {
	case *css_ast.RSelector:
		return checkSelectorDeclarations(r, sourcePath)
	case *css_ast.RAtMedia:
		return checkLeftoverRules(r.Rules, sourcePath)
	case *css_ast.RAtLayer:
		return checkLeftoverRules(r.Rules, sourcePath)
	default:
		return nil
	}
}

// checkSelectorDeclarations checks all declarations within a selector rule.
//
// Takes selector (*css_ast.RSelector) which contains the CSS selector and its
// rules to check.
// Takes sourcePath (string) which identifies the source file for error
// messages.
//
// Returns []*ast_domain.Diagnostic which contains any problems found in the
// declarations.
func checkSelectorDeclarations(selector *css_ast.RSelector, sourcePath string) []*ast_domain.Diagnostic {
	var diagnostics []*ast_domain.Diagnostic

	for _, subRule := range selector.Rules {
		declaration, ok := subRule.Data.(*css_ast.RDeclaration)
		if !ok {
			continue
		}

		diagnostics = append(diagnostics, checkLeftoverDeclaration(declaration, sourcePath)...)
	}

	return diagnostics
}

// checkLeftoverDeclaration checks a single CSS declaration for compatibility
// issues.
//
// Takes declaration (*css_ast.RDeclaration) which is the CSS declaration to check.
// Takes sourcePath (string) which identifies the source file for diagnostics.
//
// Returns []*ast_domain.Diagnostic which contains any compatibility warnings
// found.
func checkLeftoverDeclaration(declaration *css_ast.RDeclaration, sourcePath string) []*ast_domain.Diagnostic {
	var diagnostics []*ast_domain.Diagnostic
	propName := declaration.KeyText

	if message, isProblematic := problematicProperties[propName]; isProblematic {
		diagnostics = append(diagnostics, ast_domain.NewDiagnostic(
			ast_domain.Warning,
			fmt.Sprintf("%s. Property: '%s' in leftover rule (e.g., a :hover pseudo-class or @media query).", message, propName),
			propName, ast_domain.Location{}, sourcePath,
		))
	}

	diagnostics = append(diagnostics, checkLeftoverDeclarationValues(declaration, propName, sourcePath)...)

	return diagnostics
}

// checkLeftoverDeclarationValues checks the token values within a CSS
// declaration.
//
// Takes declaration (*css_ast.RDeclaration) which is the CSS declaration to check.
// Takes propName (string) which is the property name being checked.
// Takes sourcePath (string) which is the file path for error reporting.
//
// Returns []*ast_domain.Diagnostic which contains any issues found.
func checkLeftoverDeclarationValues(declaration *css_ast.RDeclaration, propName, sourcePath string) []*ast_domain.Diagnostic {
	var diagnostics []*ast_domain.Diagnostic

	for _, token := range declaration.Value {
		if token.Kind != css_lexer.TIdent && token.Kind != css_lexer.TFunction {
			continue
		}

		value := strings.ToLower(token.Text)
		diagnostic := validateLeftoverTokenValue(propName, value, sourcePath)
		if diagnostic != nil {
			diagnostics = append(diagnostics, diagnostic)
		}
	}

	return diagnostics
}

// validateLeftoverTokenValue checks a single token value for compatibility
// issues in email clients.
//
// Takes propName (string) which specifies the CSS property name to check.
// Takes value (string) which specifies the property value to validate.
// Takes sourcePath (string) which identifies the source file for diagnostics.
//
// Returns *ast_domain.Diagnostic which contains a warning when the value is
// not well-supported in email clients, or nil when no issues are found.
func validateLeftoverTokenValue(propName, value, sourcePath string) *ast_domain.Diagnostic {
	switch propName {
	case propDisplay:
		if value == "flex" || value == "inline-flex" || value == "grid" || value == "inline-grid" {
			return ast_domain.NewDiagnostic(
				ast_domain.Warning,
				fmt.Sprintf("The 'display: %s' value is not well-supported in email clients, even within @media queries.", value),
				propDisplay, ast_domain.Location{}, sourcePath,
			)
		}
	case propPosition:
		if value == "fixed" || value == "sticky" {
			return ast_domain.NewDiagnostic(
				ast_domain.Warning,
				fmt.Sprintf("The 'position: %s' value is not supported in any major email client.", value),
				propPosition, ast_domain.Location{}, sourcePath,
			)
		}
	case "background", "background-image":
		if strings.Contains(value, "gradient") {
			return ast_domain.NewDiagnostic(
				ast_domain.Warning,
				"CSS gradients are not supported in many email clients. Ensure a fallback 'background-color' is provided.",
				propName, ast_domain.Location{}, sourcePath,
			)
		}
	}

	return nil
}

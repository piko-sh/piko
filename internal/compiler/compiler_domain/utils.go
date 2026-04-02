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
	"fmt"
	"regexp"
	"strings"

	"piko.sh/piko/internal/esbuild/js_ast"
)

const (
	// base36Radix is the base value used for base-36 number conversion.
	base36Radix = 36

	// singleSpace is the replacement character used when normalising whitespace.
	singleSpace = " "
)

// whitespacePattern matches one or more consecutive whitespace characters.
var whitespacePattern = regexp.MustCompile(`\s+`)

// nodeKeyGenerator generates unique keys for VDOM nodes in base36 format.
type nodeKeyGenerator struct {
	// counter is the next sequence number for key generation.
	counter int
}

// nextKeyBase36 returns the next unique key in base36 format.
//
// Returns string which is the base36-encoded key value.
func (generator *nodeKeyGenerator) nextKeyBase36() string {
	v := generator.counter
	generator.counter++
	return base36(v)
}

// eventRecord holds the data for a VDOM event binding.
type eventRecord struct {
	// EventID is the unique identifier for this event.
	EventID string

	// IsNative indicates whether the event originated from native code.
	IsNative bool
}

// base36 converts an integer to its base-36 string representation.
//
// Takes n (int) which is the number to convert.
//
// Returns string which is the base-36 encoded value.
func base36(n int) string {
	const chars = "0123456789abcdefghijklmnopqrstuvwxyz"
	if n == 0 {
		return "0"
	}
	sign := ""
	if n < 0 {
		sign = "-"
		n = -n
	}
	result := ""
	for n > 0 {
		r := n % base36Radix
		n /= base36Radix
		result = string(chars[r]) + result
	}
	return sign + result
}

// escapeBackticks replaces backtick characters with escaped backticks for use
// in template literals.
//
// Takes inputText (string) which is the text to process.
//
// Returns string which is the text with all backticks replaced by \`.
func escapeBackticks(inputText string) string {
	return strings.ReplaceAll(inputText, "`", "\\`")
}

// normaliseWhitespace replaces tabs, newlines, and multiple spaces with single
// spaces.
//
// Takes inputText (string) which is the text to process.
//
// Returns string which is the text with all whitespace changed to single
// spaces.
func normaliseWhitespace(inputText string) string {
	noTabs := strings.ReplaceAll(inputText, "\t", singleSpace)
	noNewlines := strings.ReplaceAll(noTabs, "\n", singleSpace)
	noCarriage := strings.ReplaceAll(noNewlines, "\r", singleSpace)
	return whitespacePattern.ReplaceAllString(noCarriage, singleSpace)
}

// createStaticGetterFunction builds a static getter property for a JavaScript
// class by parsing a temporary class snippet.
//
// Takes getterName (string) which specifies the name of the static getter.
// Takes content (string) which provides the string value the getter returns.
//
// Returns *js_ast.Property which is the parsed static getter property node.
// Returns error when parsing fails or no getter property can be extracted.
func createStaticGetterFunction(getterName, content string) (*js_ast.Property, error) {
	escapedContent := escapeBackticks(content)
	classSnippet := fmt.Sprintf(
		"class TemporaryClass { static get %s() { return `%s`; } }",
		getterName,
		escapedContent,
	)

	parser := NewTypeScriptParser()
	tempAST, parseErr := parser.ParseTypeScript(classSnippet, "snippet.ts")
	if parseErr != nil {
		return nil, parseErr
	}

	for _, statement := range getStmtsFromAST(tempAST) {
		if classNode, isClass := statement.Data.(*js_ast.SClass); isClass {
			for i := range classNode.Class.Properties {
				return &classNode.Class.Properties[i], nil
			}
		}
	}
	return nil, fmt.Errorf("failed to create static getter %q", getterName)
}

// findClassDeclarationByName finds the first named class declaration
// in a JavaScript AST, checking both export default and regular class
// statements.
//
// Takes syntaxTree (*js_ast.AST) which is the parsed JavaScript AST
// to search.
//
// Returns *js_ast.Class which is the first named class found, or nil
// if no named class declaration exists.
func findClassDeclarationByName(syntaxTree *js_ast.AST, _ string) *js_ast.Class {
	for _, statement := range getStmtsFromAST(syntaxTree) {
		switch node := statement.Data.(type) {
		case *js_ast.SExportDefault:
			if classDecl, isClass := node.Value.Data.(*js_ast.SClass); isClass {
				if classDecl.Class.Name != nil {
					return &classDecl.Class
				}
			}
		case *js_ast.SClass:
			if node.Class.Name != nil {
				return &node.Class
			}
		}
	}
	return nil
}

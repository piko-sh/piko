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

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEvaluateExpression_IndexExpr(t *testing.T) {
	t.Parallel()

	arrayScope := map[string]any{
		"numbers": []any{10.0, 20.0, 30.0, 40.0, 50.0},
		"empty":   []any{},
		"strings": []any{"apple", "banana", "cherry"},
		"mixed":   []any{10.0, "text", true, nil, map[string]any{"key": "value"}},
		"nested":  []any{[]any{1.0, 2.0}, []any{3.0, 4.0}},
	}

	mapScope := map[string]any{
		"user": map[string]any{
			"name":  "Alice",
			"age":   30.0,
			"roles": []any{"admin", "editor"},
		},
		"config": map[string]any{
			"theme":    "dark",
			"fontSize": 16.0,
			"features": map[string]any{
				"darkMode": true,
				"autoSave": false,
			},
		},
		"emptyMap": map[string]any{},
	}

	tests := []struct {
		expected         any
		scope            map[string]any
		name             string
		expressionString string
	}{

		{name: "Array index - first element", expressionString: "numbers[0]", scope: arrayScope, expected: 10.0},
		{name: "Array index - middle element", expressionString: "numbers[2]", scope: arrayScope, expected: 30.0},
		{name: "Array index - last element", expressionString: "numbers[4]", scope: arrayScope, expected: 50.0},
		{name: "Array index - string element", expressionString: "strings[1]", scope: arrayScope, expected: "banana"},
		{name: "Array index - mixed element (number)", expressionString: "mixed[0]", scope: arrayScope, expected: 10.0},
		{name: "Array index - mixed element (string)", expressionString: "mixed[1]", scope: arrayScope, expected: "text"},
		{name: "Array index - mixed element (boolean)", expressionString: "mixed[2]", scope: arrayScope, expected: true},
		{name: "Array index - mixed element (nil)", expressionString: "mixed[3]", scope: arrayScope, expected: nil},

		{name: "Array index - expression (1+1)", expressionString: "numbers[1+1]", scope: arrayScope, expected: 30.0},
		{name: "Array index - expression (5-1)", expressionString: "numbers[5-1]", scope: arrayScope, expected: 50.0},
		{name: "Array index - expression (2*2)", expressionString: "numbers[2*2]", scope: arrayScope, expected: 50.0},

		{name: "Nested array - first element of first array", expressionString: "nested[0][0]", scope: arrayScope, expected: 1.0},
		{name: "Nested array - second element of second array", expressionString: "nested[1][1]", scope: arrayScope, expected: 4.0},

		{name: "Array index - negative index", expressionString: "numbers[-1]", scope: arrayScope, expected: nil},
		{name: "Array index - out of bounds", expressionString: "numbers[10]", scope: arrayScope, expected: nil},
		{name: "Array index - empty array", expressionString: "empty[0]", scope: arrayScope, expected: nil},

		{name: "Map access - string key", expressionString: `user["name"]`, scope: mapScope, expected: "Alice"},
		{name: "Map access - number key", expressionString: `user["age"]`, scope: mapScope, expected: 30.0},
		{name: "Map access - nested map", expressionString: `config["features"]["darkMode"]`, scope: mapScope, expected: true},

		{name: "Map access - expression as key", expressionString: `config["font" + "Size"]`, scope: mapScope, expected: 16.0},

		{name: "Map access - missing key", expressionString: `user["address"]`, scope: mapScope, expected: nil},
		{name: "Map access - empty map", expressionString: `emptyMap["key"]`, scope: mapScope, expected: nil},

		{name: "Mixed access - array in map", expressionString: `user["roles"][0]`, scope: mapScope, expected: "admin"},
		{name: "Mixed access - map in array", expressionString: `mixed[4]["key"]`, scope: arrayScope, expected: "value"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			expression := mustParseExpr(t, tt.expressionString)
			result := EvaluateExpression(expression, tt.scope)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEvaluateExpression_CallExpr(t *testing.T) {
	t.Parallel()

	add := func(a, b any) any {
		return testToFloat(a) + testToFloat(b)
	}

	concat := func(a, b any) any {
		return testToString(a) + testToString(b)
	}

	getItem := func(arr, index any) any {
		array, ok := arr.([]any)
		if !ok {
			return nil
		}
		itemIndex := int(testToFloat(index))
		if itemIndex < 0 || itemIndex >= len(array) {
			return nil
		}
		return array[itemIndex]
	}

	getProperty := func(obj, key any) any {
		m, ok := obj.(map[string]any)
		if !ok {
			return nil
		}
		k, ok := key.(string)
		if !ok {
			return nil
		}
		return m[k]
	}

	noArgs := func() any {
		return "no arguments function called"
	}

	identity := func(x any) any {
		return x
	}

	funcScope := map[string]any{
		"add":         add,
		"concat":      concat,
		"getItem":     getItem,
		"getProperty": getProperty,
		"noArgs":      noArgs,
		"identity":    identity,
		"numbers":     []any{10.0, 20.0, 30.0},
		"user": map[string]any{
			"name": "Alice",
			"getInfo": func() any {
				return "Alice's info"
			},
			"greet": func(greeting any) any {
				return testToString(greeting) + " Alice"
			},
		},
		"math": map[string]any{
			"max": func(a, b any) any {
				if testToFloat(a) > testToFloat(b) {
					return a
				}
				return b
			},
		},
	}

	tests := []struct {
		expected         any
		scope            map[string]any
		name             string
		expressionString string
	}{

		{name: "Function call - no arguments", expressionString: "noArgs()", scope: funcScope, expected: "no arguments function called"},
		{name: "Function call - identity", expressionString: "identity(42)", scope: funcScope, expected: 42.0},
		{name: "Function call - add", expressionString: "add(5, 3)", scope: funcScope, expected: 8.0},
		{name: "Function call - concat", expressionString: "concat('hello', 'world')", scope: funcScope, expected: "helloworld"},

		{name: "Function call - add with expressions", expressionString: "add(2+3, 4*2)", scope: funcScope, expected: 13.0},
		{name: "Function call - concat with expressions", expressionString: "concat('hello ' + 'there', '!')", scope: funcScope, expected: "hello there!"},

		{name: "Function call - getItem", expressionString: "getItem(numbers, 1)", scope: funcScope, expected: 20.0},
		{name: "Function call - getProperty", expressionString: "getProperty(user, 'name')", scope: funcScope, expected: "Alice"},

		{name: "Method call - no arguments", expressionString: "user.getInfo()", scope: funcScope, expected: "Alice's info"},
		{name: "Method call - with arguments", expressionString: "user.greet('Hello')", scope: funcScope, expected: "Hello Alice"},
		{name: "Method call - nested", expressionString: "math.max(10, 20)", scope: funcScope, expected: 20.0},

		{name: "Nested function calls - identity(add(...))", expressionString: "identity(add(5, 3))", scope: funcScope, expected: 8.0},
		{name: "Nested function calls - add(getItem(...), ...)", expressionString: "add(getItem(numbers, 0), getItem(numbers, 1))", scope: funcScope, expected: 30.0},

		{name: "Function call - non-existent function", expressionString: "nonExistentFunc()", scope: funcScope, expected: nil},
		{name: "Function call - property is not a function", expressionString: "user.name()", scope: funcScope, expected: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			expression := mustParseExpr(t, tt.expressionString)
			result := EvaluateExpression(expression, tt.scope)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEvaluateExpression_ComplexNested(t *testing.T) {
	t.Parallel()

	users := []any{
		map[string]any{
			"id":      1.0,
			"name":    "Alice",
			"active":  true,
			"roles":   []any{"admin", "editor"},
			"profile": map[string]any{"theme": "dark"},
		},
		map[string]any{
			"id":      2.0,
			"name":    "Bob",
			"active":  false,
			"roles":   []any{"user"},
			"profile": map[string]any{"theme": "light"},
		},
		map[string]any{
			"id":      3.0,
			"name":    "Charlie",
			"active":  true,
			"roles":   []any{"user", "moderator"},
			"profile": map[string]any{"theme": "auto"},
		},
	}

	getUser := func(id any) any {
		idVal := testToFloat(id)
		for _, u := range users {
			user, ok := u.(map[string]any)
			if !ok {
				continue
			}
			userID, ok := user["id"].(float64)
			if !ok {
				continue
			}
			if userID == idVal {
				return user
			}
		}
		return nil
	}

	complexScope := map[string]any{
		"users":   users,
		"getUser": getUser,
		"hasRole": func(user, role any) any {
			u, ok := user.(map[string]any)
			if !ok {
				return false
			}
			r, ok := role.(string)
			if !ok {
				return false
			}
			roles, ok := u["roles"].([]any)
			if !ok {
				return false
			}
			for _, roleItem := range roles {
				if roleItem == r {
					return true
				}
			}
			return false
		},
		"config": map[string]any{
			"features": map[string]any{
				"darkMode":   true,
				"newEditor":  false,
				"betaAccess": map[string]any{"enabled": true, "roles": []any{"admin"}},
			},
		},
	}

	tests := []struct {
		expected         any
		scope            map[string]any
		name             string
		expressionString string
	}{

		{
			name:             "Complex - array access with object property",
			expressionString: "users[0].name",
			scope:            complexScope,
			expected:         "Alice",
		},
		{
			name:             "Complex - deeply nested property access",
			expressionString: "users[0].profile.theme",
			scope:            complexScope,
			expected:         "dark",
		},
		{
			name:             "Complex - function call with array index",
			expressionString: "getUser(2).name",
			scope:            complexScope,
			expected:         "Bob",
		},
		{
			name:             "Complex - function call with function call argument",
			expressionString: "hasRole(getUser(1), 'admin')",
			scope:            complexScope,
			expected:         true,
		},
		{
			name:             "Complex - conditional with nested access",
			expressionString: "users[1].active ? users[1].name : 'Inactive user'",
			scope:            complexScope,
			expected:         "Inactive user",
		},
		{
			name:             "Complex - conditional with function calls",
			expressionString: "hasRole(getUser(3), 'admin') ? 'Admin user' : 'Regular user'",
			scope:            complexScope,
			expected:         "Regular user",
		},
		{
			name:             "Complex - multiple nested conditions",
			expressionString: "users[0].active && users[0].profile.theme == 'dark' ? 'Dark active user' : 'Other user'",
			scope:            complexScope,
			expected:         "Dark active user",
		},
		{
			name:             "Complex - deeply nested feature check",
			expressionString: "config.features.betaAccess.enabled && hasRole(getUser(1), config.features.betaAccess.roles[0])",
			scope:            complexScope,
			expected:         true,
		},
		{
			name:             "Complex - array access with expression index",
			expressionString: "users[1 + 1].name",
			scope:            complexScope,
			expected:         "Charlie",
		},
		{
			name:             "Complex - nested array access with expressions",
			expressionString: "users[users[0].id - 1].name",
			scope:            complexScope,
			expected:         "Alice",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			expression := mustParseExpr(t, tt.expressionString)
			result := EvaluateExpression(expression, tt.scope)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEvaluateExpression_EdgeCases(t *testing.T) {
	t.Parallel()

	edgeCaseScope := map[string]any{
		"nil":         nil,
		"emptyString": "",
		"emptyArr":    []any{},
		"emptyObj":    map[string]any{},
		"zero":        0.0,
		"nested":      map[string]any{"a": map[string]any{"b": map[string]any{"c": nil}}},
		"mixedArr":    []any{nil, 0.0, "", false, []any{}, map[string]any{}},
		"truthyArr":   []any{1.0, "text", true, []any{1.0}, map[string]any{"key": "value"}},
	}

	tests := []struct {
		expected         any
		scope            map[string]any
		name             string
		expressionString string
	}{

		{name: "Edge - nil value", expressionString: "nil", scope: edgeCaseScope, expected: nil},
		{name: "Edge - empty string", expressionString: "emptyString", scope: edgeCaseScope, expected: ""},
		{name: "Edge - empty array", expressionString: "emptyArr", scope: edgeCaseScope, expected: []any{}},
		{name: "Edge - empty object", expressionString: "emptyObj", scope: edgeCaseScope, expected: map[string]any{}},
		{name: "Edge - zero", expressionString: "zero", scope: edgeCaseScope, expected: 0.0},

		{name: "Edge - nil + number", expressionString: "nil + 5", scope: edgeCaseScope, expected: 5.0},
		{name: "Edge - nil + string", expressionString: "nil + 'text'", scope: edgeCaseScope, expected: "text"},
		{name: "Edge - string + nil", expressionString: "'text' + nil", scope: edgeCaseScope, expected: "text"},
		{name: "Edge - nil == nil", expressionString: "nil == nil", scope: edgeCaseScope, expected: true},
		{name: "Edge - nil != 0", expressionString: "nil != 0", scope: edgeCaseScope, expected: true},

		{name: "Edge - string to number", expressionString: "'5' + 3", scope: edgeCaseScope, expected: "53"},
		{name: "Edge - number to string", expressionString: "5 + '3'", scope: edgeCaseScope, expected: "53"},
		{name: "Edge - boolean to number", expressionString: "true + 1", scope: edgeCaseScope, expected: 2.0},
		{name: "Edge - number to boolean", expressionString: "!0", scope: edgeCaseScope, expected: true},
		{name: "Edge - string to boolean", expressionString: "!'hello'", scope: edgeCaseScope, expected: false},
		{name: "Edge - empty string to boolean", expressionString: "!''", scope: edgeCaseScope, expected: true},

		{name: "Edge - deeply nested nil", expressionString: "nested.a.b.c", scope: edgeCaseScope, expected: nil},
		{name: "Edge - access beyond nil", expressionString: "nested.a.b.c.d", scope: edgeCaseScope, expected: nil},
		{name: "Edge - nil index access", expressionString: "nil[0]", scope: edgeCaseScope, expected: nil},
		{name: "Edge - nil property access", expressionString: "nil.property", scope: edgeCaseScope, expected: nil},

		{name: "Edge - nil truthiness", expressionString: "nil ? 'truthy' : 'falsy'", scope: edgeCaseScope, expected: "falsy"},
		{name: "Edge - empty string truthiness", expressionString: "emptyString ? 'truthy' : 'falsy'", scope: edgeCaseScope, expected: "falsy"},
		{name: "Edge - zero truthiness", expressionString: "zero ? 'truthy' : 'falsy'", scope: edgeCaseScope, expected: "falsy"},
		{name: "Edge - empty array truthiness", expressionString: "emptyArr ? 'truthy' : 'falsy'", scope: edgeCaseScope, expected: "truthy"},
		{name: "Edge - empty object truthiness", expressionString: "emptyObj ? 'truthy' : 'falsy'", scope: edgeCaseScope, expected: "truthy"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			expression := mustParseExpr(t, tt.expressionString)
			result := EvaluateExpression(expression, tt.scope)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func testToFloat(value any) float64 {
	if value == nil {
		return 0
	}
	switch v := value.(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case bool:
		if v {
			return 1
		}
		return 0
	case string:
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return 0
		}
		return f
	default:
		return 0
	}
}

func testToString(value any) string {
	if value == nil {
		return "null"
	}
	switch v := value.(type) {
	case string:
		return v
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case int:
		return strconv.Itoa(v)
	case bool:
		return strconv.FormatBool(v)
	default:
		return "unknown"
	}
}

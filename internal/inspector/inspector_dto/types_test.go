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

package inspector_dto

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFunctionSignature_ToSignatureString(t *testing.T) {
	testCases := []struct {
		name      string
		want      string
		signature FunctionSignature
	}{
		{
			name:      "no params or returns",
			want:      "func() ",
			signature: FunctionSignature{},
		},
		{
			name:      "single return",
			want:      "func(int, string) error",
			signature: FunctionSignature{Params: []string{"int", "string"}, Results: []string{"error"}},
		},
		{
			name:      "multiple returns",
			want:      "func(string) (int, error)",
			signature: FunctionSignature{Params: []string{"string"}, Results: []string{"int", "error"}},
		},
		{
			name: "generic signature renders its type parameters",
			want: "func[K comparable, V any](K, V) map[K]V",
			signature: FunctionSignature{
				Params:               []string{"K", "V"},
				Results:              []string{"map[K]V"},
				TypeParamNames:       []string{"K", "V"},
				TypeParamConstraints: []string{"comparable", "any"},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.want, testCase.signature.ToSignatureString())
		})
	}
}

func TestFunctionSignature_TypeParamList(t *testing.T) {
	testCases := []struct {
		name      string
		want      string
		signature FunctionSignature
	}{
		{
			name:      "no type parameters",
			want:      "",
			signature: FunctionSignature{Params: []string{"string"}},
		},
		{
			name: "every parameter constrained",
			want: "[T any, U comparable]",
			signature: FunctionSignature{
				TypeParamNames:       []string{"T", "U"},
				TypeParamConstraints: []string{"any", "comparable"},
			},
		},
		{
			name: "fewer constraints than parameters",
			want: "[T fmt.Stringer, U]",
			signature: FunctionSignature{
				TypeParamNames:       []string{"T", "U"},
				TypeParamConstraints: []string{"fmt.Stringer"},
			},
		},
		{
			name: "empty constraint is omitted",
			want: "[T, U ~int | ~string]",
			signature: FunctionSignature{
				TypeParamNames:       []string{"T", "U"},
				TypeParamConstraints: []string{"", "~int | ~string"},
			},
		},
		{
			name:      "no constraints recorded",
			want:      "[T]",
			signature: FunctionSignature{TypeParamNames: []string{"T"}},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.want, testCase.signature.TypeParamList())
		})
	}
}

func TestParseStructTag(t *testing.T) {
	testCases := []struct {
		expectedMap map[string]string
		name        string
		inputTag    string
	}{
		{name: "Known tag: prop", inputTag: `prop:"userID"`, expectedMap: map[string]string{"prop": "userID"}},
		{name: "Known tag: validate", inputTag: `validate:"required,uuid"`, expectedMap: map[string]string{"validate": "required,uuid"}},
		{name: "Known tag: default", inputTag: `default:"guest"`, expectedMap: map[string]string{"default": "guest"}},
		{name: "Known tag: factory", inputTag: `factory:"NewUser"`, expectedMap: map[string]string{"factory": "NewUser"}},
		{name: "Known tag: coerce with value", inputTag: `coerce:"true"`, expectedMap: map[string]string{"coerce": "true"}},
		{name: "Known tag: coerce with empty string value", inputTag: `coerce:""`, expectedMap: map[string]string{"coerce": ""}},
		{name: "All known tags present", inputTag: `prop:"id" validate:"required" default:"0" factory:"New" coerce:""`,
			expectedMap: map[string]string{
				"prop":     "id",
				"validate": "required",
				"default":  "0",
				"factory":  "New",
				"coerce":   "",
			}},
		{name: "Mix of known and unknown tags", inputTag: `prop:"name" json:"userName" validate:"required"`,
			expectedMap: map[string]string{
				"prop":     "name",
				"validate": "required",
			}},
		{name: "Backticked tag drops unknown json key", inputTag: "`prop:\"title\" validate:\"required\" json:\"title\"`",
			expectedMap: map[string]string{
				"prop":     "title",
				"validate": "required",
			}},
		{name: "Only unknown tags", inputTag: `json:"name" xml:"id"`, expectedMap: map[string]string{}},
		{name: "Empty tag string", inputTag: ``, expectedMap: map[string]string{}},
		{name: "Tag string with backticks", inputTag: "`prop:\"name\"`", expectedMap: map[string]string{"prop": "name"}},
		{name: "Duplicate known tag", inputTag: `prop:"a" prop:"b"`, expectedMap: map[string]string{"prop": "a"}},
		{name: "Malformed tag (ignored by reflect)", inputTag: `prop:name`, expectedMap: map[string]string{}},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.expectedMap, ParseStructTag(testCase.inputTag))
		})
	}
}

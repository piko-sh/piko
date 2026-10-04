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

package inspector_domain

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"piko.sh/piko/internal/inspector/inspector_dto"
)

func TestSanitiseNamedTypePaths(t *testing.T) {
	replacer := func(path string) string { return strings.Replace(path, "/machine/", "$ROOT/", 1) }

	testCases := []struct {
		namedType *inspector_dto.Type
		want      *inspector_dto.Type
		name      string
	}{
		{
			name: "rewrites the type, field and method paths",
			namedType: &inspector_dto.Type{
				DefinedInFilePath: "/machine/a.go",
				Fields:            []*inspector_dto.Field{{DefinitionFilePath: "/machine/b.go"}},
				Methods:           []*inspector_dto.Method{{DefinitionFilePath: "/machine/c.go"}},
			},
			want: &inspector_dto.Type{
				DefinedInFilePath: "$ROOT/a.go",
				Fields:            []*inspector_dto.Field{{DefinitionFilePath: "$ROOT/b.go"}},
				Methods:           []*inspector_dto.Method{{DefinitionFilePath: "$ROOT/c.go"}},
			},
		},
		{
			name: "skips nil fields and methods",
			namedType: &inspector_dto.Type{
				DefinedInFilePath: "/machine/a.go",
				Fields:            []*inspector_dto.Field{nil, {DefinitionFilePath: "/machine/b.go"}},
				Methods:           []*inspector_dto.Method{nil},
			},
			want: &inspector_dto.Type{
				DefinedInFilePath: "$ROOT/a.go",
				Fields:            []*inspector_dto.Field{nil, {DefinitionFilePath: "$ROOT/b.go"}},
				Methods:           []*inspector_dto.Method{nil},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.NotPanics(t, func() { sanitiseNamedTypePaths(testCase.namedType, replacer) })
			assert.Equal(t, testCase.want, testCase.namedType)
		})
	}
}

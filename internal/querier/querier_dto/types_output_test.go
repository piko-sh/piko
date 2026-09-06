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

package querier_dto

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQueryParameterIsPaginationBound(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		context ParameterContext
		want    bool
	}{
		{name: "limit clause", context: ParameterContextLimit, want: true},
		{name: "offset clause", context: ParameterContextOffset, want: true},
		{name: "comparison", context: ParameterContextComparison, want: false},
		{name: "in list", context: ParameterContextInList, want: false},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			parameter := QueryParameter{}
			parameter.Context = testCase.context
			assert.Equal(t, testCase.want, parameter.IsPaginationBound())
		})
	}
}

func TestNewOutputColumn(t *testing.T) {
	t.Parallel()

	sqlType := NewSQLType(TypeCategoryText, "text")

	column := NewOutputColumn("email", sqlType, true)

	want := OutputColumn{}
	want.Name = "email"
	want.SQLType = sqlType
	want.Nullable = true
	assert.Equal(t, want, column)
}

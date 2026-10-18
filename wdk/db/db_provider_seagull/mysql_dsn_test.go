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

package db_provider_seagull

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDSNEnablesMultiStatements(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		dsn      string
		expected bool
	}{
		{name: "empty", dsn: "", expected: false},
		{name: "no parameters", dsn: "user:pass@tcp(localhost:3306)/app", expected: false},
		{name: "enabled", dsn: "user:pass@tcp(localhost:3306)/app?multiStatements=true", expected: true},
		{name: "enabled among other parameters", dsn: "user:pass@tcp(localhost:3306)/app?parseTime=true&multiStatements=1", expected: true},
		{name: "disabled", dsn: "user:pass@tcp(localhost:3306)/app?multiStatements=false", expected: false},
		{name: "not a boolean", dsn: "user:pass@tcp(localhost:3306)/app?multiStatements=yes", expected: false},
		{name: "parameter name is case sensitive", dsn: "user:pass@tcp(localhost:3306)/app?multistatements=true", expected: false},
		{name: "slash in the password", dsn: "user:pa/ss@tcp(localhost:3306)/app?multiStatements=true", expected: true},
		{name: "question mark in the password", dsn: "user:pa?multiStatements=true@tcp(localhost:3306)/app", expected: false},
		{name: "no database separator", dsn: "multiStatements=true", expected: false},
		{name: "malformed query", dsn: "user:pass@tcp(localhost:3306)/app?multiStatements=%zz", expected: false},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, testCase.expected, dsnEnablesMultiStatements(testCase.dsn))
		})
	}
}

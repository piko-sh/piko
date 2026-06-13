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

package worker_domain

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFatalAndIsFatal(t *testing.T) {
	sentinel := errors.New("declined")
	testCases := []struct {
		err       error
		name      string
		wantFatal bool
	}{
		{name: "plain error is not fatal", err: sentinel, wantFatal: false},
		{name: "Fatal of a sentinel is fatal", err: Fatal(sentinel), wantFatal: true},
		{name: "Fatal of nil is non-nil and fatal", err: Fatal(nil), wantFatal: true},
		{name: "wrapped Fatal stays fatal", err: fmt.Errorf("ctx: %w", Fatal(sentinel)), wantFatal: true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			require.Equal(t, testCase.wantFatal, IsFatal(testCase.err))
		})
	}
	require.ErrorIs(t, Fatal(sentinel), sentinel, "Fatal must keep the original error inspectable")
}

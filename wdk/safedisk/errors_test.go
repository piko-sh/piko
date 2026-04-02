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

package safedisk

import (
	"errors"
	"io/fs"
	"testing"
)

func TestIsNotExist(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		err      error
		name     string
		expected bool
	}{
		{name: "fs.ErrNotExist", err: fs.ErrNotExist, expected: true},
		{name: "other error", err: errors.New("some error"), expected: false},
		{name: "nil", err: nil, expected: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := IsNotExist(tc.err); got != tc.expected {
				t.Errorf("IsNotExist(%v) = %v, want %v", tc.err, got, tc.expected)
			}
		})
	}
}

func TestIsExist(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		err      error
		name     string
		expected bool
	}{
		{name: "fs.ErrExist", err: fs.ErrExist, expected: true},
		{name: "other error", err: errors.New("some error"), expected: false},
		{name: "nil", err: nil, expected: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := isExist(tc.err); got != tc.expected {
				t.Errorf("isExist(%v) = %v, want %v", tc.err, got, tc.expected)
			}
		})
	}
}

func TestIsPermission(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		err      error
		name     string
		expected bool
	}{
		{name: "fs.ErrPermission", err: fs.ErrPermission, expected: true},
		{name: "other error", err: errors.New("some error"), expected: false},
		{name: "nil", err: nil, expected: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := isPermission(tc.err); got != tc.expected {
				t.Errorf("isPermission(%v) = %v, want %v", tc.err, got, tc.expected)
			}
		})
	}
}

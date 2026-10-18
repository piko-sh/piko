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

// Package goimportname names an imported Go package from its import path alone, by the
package goimportname

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAssumed(t *testing.T) {
	t.Parallel()

	testCases := map[string]string{
		"fmt":                         "fmt",
		"encoding/json":               "json",
		"math/rand/v2":                "rand",
		"gopkg.in/yaml.v3":            "yaml",
		"gopkg.in/check.v1":           "check",
		"github.com/mattn/go-sqlite3": "sqlite3",
		"github.com/example/go-foo":   "foo",
		"github.com/example/mod/v2":   "mod",
		"github.com/jackc/pgx/v5":     "pgx",
		"github.com/example/v2":       "example",
		"example.com/v3":              "example",
		"github.com/example/v2beta":   "v2beta",
		"github.com/example/foo-bar":  "foo",
		"example.com/über":            "über",
		"k8s.io/api/core/v1":          "core",
		"v2":                          "v2",
		"example.com/go-":             "",
		"net/http":                    "http",
		"piko.sh/piko/wdk/runtime":    "runtime",
		"github.com/x/go-foo":         "foo",
		"github.com/x/go-foo-bar":     "foo",
		"github.com/x/foo-bar":        "foo",
		"github.com/x/go-foo/v2":      "foo",
		"example.com/v2":              "example",
		"example.com/x/v":             "v",
		"example.com/x/vtools":        "vtools",
		"example.com/x/foo_bar":       "foo_bar",
	}
	for importPath, want := range testCases {
		assert.Equal(t, want, Assumed(importPath), importPath)
	}
}

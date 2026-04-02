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
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/packages"
)

func TestAggregatePackageErrors_NoErrors(t *testing.T) {
	t.Parallel()
	root := &packages.Package{PkgPath: "example.com/app", Imports: map[string]*packages.Package{}}
	err := aggregatePackageErrors(context.Background(), []*packages.Package{root})
	require.NoError(t, err)
}

func TestAggregatePackageErrors_RootTypeErrorIsFatal(t *testing.T) {
	t.Parallel()
	root := &packages.Package{
		PkgPath: "example.com/app",
		Imports: map[string]*packages.Package{},
		Errors: []packages.Error{
			{Msg: "undefined: Foo", Kind: packages.TypeError},
		},
	}
	err := aggregatePackageErrors(context.Background(), []*packages.Package{root})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "undefined: Foo")
}

func TestAggregatePackageErrors_DependencyTypeErrorIsSkipped(t *testing.T) {
	t.Parallel()
	dep := &packages.Package{
		PkgPath: "github.com/mattn/go-sqlite3",
		Imports: map[string]*packages.Package{},
		Errors: []packages.Error{
			{Msg: "undefined: SQLiteConn", Kind: packages.TypeError},
			{Msg: "undefined: SQLiteStmt", Kind: packages.TypeError},
		},
	}
	root := &packages.Package{
		PkgPath: "example.com/app",
		Imports: map[string]*packages.Package{"github.com/mattn/go-sqlite3": dep},
	}
	err := aggregatePackageErrors(context.Background(), []*packages.Package{root})
	require.NoError(t, err)
}

func TestAggregatePackageErrors_DependencyListErrorIsFatal(t *testing.T) {
	t.Parallel()
	dep := &packages.Package{
		PkgPath: "github.com/missing/module",
		Imports: map[string]*packages.Package{},
		Errors: []packages.Error{
			{Msg: "no required module provides package github.com/missing/module", Kind: packages.ListError},
		},
	}
	root := &packages.Package{
		PkgPath: "example.com/app",
		Imports: map[string]*packages.Package{"github.com/missing/module": dep},
	}
	err := aggregatePackageErrors(context.Background(), []*packages.Package{root})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no required module provides package")
}

func TestAggregatePackageErrors_DependencyParseErrorIsSkipped(t *testing.T) {
	t.Parallel()
	dep := &packages.Package{
		PkgPath: "github.com/example/dep",
		Imports: map[string]*packages.Package{},
		Errors: []packages.Error{
			{Msg: "parsing some_cgo_file.go: expected declaration", Kind: packages.ParseError},
		},
	}
	root := &packages.Package{
		PkgPath: "example.com/app",
		Imports: map[string]*packages.Package{"github.com/example/dep": dep},
	}
	err := aggregatePackageErrors(context.Background(), []*packages.Package{root})
	require.NoError(t, err)
}

func TestAggregatePackageErrors_IgnorableErrorSkippedRegardless(t *testing.T) {
	t.Parallel()

	root := &packages.Package{
		PkgPath: "example.com/app",
		Imports: map[string]*packages.Package{},
		Errors: []packages.Error{
			{Msg: "undefined: _Ctype_int", Kind: packages.TypeError},
		},
	}
	err := aggregatePackageErrors(context.Background(), []*packages.Package{root})
	require.NoError(t, err)
}

func TestAggregatePackageErrors_TransitiveDependencyTypeErrorIsSkipped(t *testing.T) {
	t.Parallel()
	transitiveDep := &packages.Package{
		PkgPath: "github.com/example/transitive",
		Imports: map[string]*packages.Package{},
		Errors: []packages.Error{
			{Msg: "undefined: SomeCgoType", Kind: packages.TypeError},
		},
	}
	dep := &packages.Package{
		PkgPath: "github.com/example/dep",
		Imports: map[string]*packages.Package{"github.com/example/transitive": transitiveDep},
	}
	root := &packages.Package{
		PkgPath: "example.com/app",
		Imports: map[string]*packages.Package{"github.com/example/dep": dep},
	}
	err := aggregatePackageErrors(context.Background(), []*packages.Package{root})
	require.NoError(t, err)
}

func TestIsIgnorablePackageError(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		message  string
		expected bool
	}{
		{"imported and not used", `"fmt" imported and not used`, true},
		{"imported as X and not used", `"fmt" imported as f and not used`, true},
		{"no Go files", "no Go files in /some/dir", true},
		{"build constraints", "build constraints exclude all Go files in /some/dir", true},
		{"CGo _C type", "undefined: _C_int", true},
		{"CGo _Ctype", "undefined: _Ctype_int", true},
		{"real undefined", "undefined: Foo", false},
		{"real type error", "cannot use x (type int) as type string", false},
		{"empty message", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, isIgnorablePackageError(tt.message))
		})
	}
}

func TestErrorKindName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "ListError", errorKindName(packages.ListError))
	assert.Equal(t, "ParseError", errorKindName(packages.ParseError))
	assert.Equal(t, "TypeError", errorKindName(packages.TypeError))
	assert.Equal(t, "UnknownError", errorKindName(packages.ErrorKind(99)))
}

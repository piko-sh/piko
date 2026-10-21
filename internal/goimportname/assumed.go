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

package goimportname

import (
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Assumed returns the package name goimports assumes for an import path.
//
// It takes the last path element that is not a major-version suffix such as "v2", drops a
// leading "go-", then keeps the leading run of identifier characters. So
// "gopkg.in/yaml.v3" is "yaml", "github.com/mattn/go-sqlite3" is "sqlite3" and
// "github.com/jackc/pgx/v5" is "pgx". A package that declares another name, such as
// "k8s.io/api/core/v1" (which declares "v1"), is only recognised under that name when its
// import carries an explicit alias.
//
// Takes importPath (string) which is the import path to name.
//
// Returns string which is the assumed name; it is empty for a degenerate path such as
// "example.com/go-".
func Assumed(importPath string) string {
	base := lastElement(importPath)
	if parent, _, hasParent := strings.CutLast(importPath, "/"); hasParent && isMajorVersion(base) {
		base = lastElement(parent)
	}
	base = strings.TrimPrefix(base, "go-")
	if index := strings.IndexFunc(base, notIdentifierRune); index >= 0 {
		base = base[:index]
	}
	return base
}

// lastElement returns the final element of an import path.
//
// Takes importPath (string) which is the import path.
//
// Returns string which is the text after the last slash, or the whole path.
func lastElement(importPath string) string {
	if _, after, found := strings.CutLast(importPath, "/"); found {
		return after
	}
	return importPath
}

// isMajorVersion reports whether a path element is a major-version suffix such as "v2",
// which names a module version rather than the package.
//
// Takes element (string) which is the path element to classify.
//
// Returns bool which is true when element is "v" followed by a number.
func isMajorVersion(element string) bool {
	digits, found := strings.CutPrefix(element, "v")
	if !found {
		return false
	}
	_, err := strconv.Atoi(digits)
	return err == nil
}

// notIdentifierRune reports whether a rune cannot appear in a Go identifier.
//
// Takes r (rune) which is the rune to classify.
//
// Returns bool which is true when r is not a letter, digit or underscore.
func notIdentifierRune(r rune) bool {
	switch {
	case 'a' <= r && r <= 'z', 'A' <= r && r <= 'Z', '0' <= r && r <= '9', r == '_':
		return false
	case r >= utf8.RuneSelf:
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	default:
		return true
	}
}

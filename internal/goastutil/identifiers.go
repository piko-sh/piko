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

package goastutil

import (
	"fmt"
	"go/token"
	"go/types"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/cespare/xxhash/v2"
)

const (
	// DefaultGoIdentifier is the Go identifier used when a name sanitises to nothing usable,
	// such as an empty string or a name made only of punctuation.
	DefaultGoIdentifier = "piko"

	// DefaultGoPackageName is the package name used when a name sanitises to nothing usable.
	// It matches the annotator's long-standing fallback so the two agree.
	DefaultGoPackageName = "p_default_pkg_name"

	// GoPackageNamePrefix is prepended to a package name that would otherwise start with a
	// digit, which no Go identifier may do.
	GoPackageNamePrefix = "p"

	// ExportedGoIdentifierPrefix is prepended to an exported name whose first rune has no
	// upper-case form, such as a digit or a Han character.
	ExportedGoIdentifierPrefix = "X"

	// DefaultJSONTagName is the JSON struct tag name used when a column name sanitises to
	// nothing usable.
	DefaultJSONTagName = "field"

	// jsonOmittedTagName is the tag encoding/json reads as an instruction to leave a field
	// out. A sanitised column name must never land on it, or the column disappears from the
	// payload with nothing reported.
	jsonOmittedTagName = "-"

	// ShortHashLength is the number of hexadecimal characters kept in a short hash.
	ShortHashLength = 8

	// goMainPackageName is the one package name that is a legal identifier but cannot be
	// imported, because a package called main is a command.
	goMainPackageName = "main"

	// underscore is the repair character: it replaces illegal runes, prefixes leading digits
	// and suffixes reserved words. On its own it is also Go's blank identifier.
	underscore = "_"

	// underscoreRune is underscore as a rune, for writing into identifier builders.
	underscoreRune = '_'

	// disambiguationStart is the first numeric suffix tried when a name is already taken.
	disambiguationStart = 2

	// disambiguationSuffixRoom is the space reserved for the numeric suffix, so the probe
	// buffer is allocated once rather than once per candidate.
	disambiguationSuffixRoom = 8

	// decimalBase is the base numeric suffixes are rendered in.
	decimalBase = 10

	// jsonTagPunctuation lists the punctuation encoding/json tolerates inside a tag name.
	// Any other punctuation invalidates the whole tag.
	jsonTagPunctuation = "!#$%&()*+-./:<=>?@[]^_{|}~ "

	// shortHashFormat renders a 64-bit hash as fixed-width hexadecimal.
	shortHashFormat = "%016x"
)

var (
	// goPredeclaredIdentifiers holds Go's universe block names.
	//
	// Shadowing one of these is legal Go but breaks every other emitted line that relied on
	// the original meaning, so no sanitised name is ever allowed to equal one. The set is
	// read from go/types rather than listed by hand, so a new Go release cannot leave it
	// stale.
	goPredeclaredIdentifiers = buildGoPredeclaredIdentifiers()
)

// IsValidGoIdentifier reports whether a name can be emitted as a Go identifier verbatim.
//
// Takes name (string) which is the candidate identifier.
//
// Returns bool which is true when the name is a legal, referenceable Go identifier.
func IsValidGoIdentifier(name string) bool {
	return name != underscore && token.IsIdentifier(name)
}

// IsGoPredeclared reports whether a name is one of Go's predeclared identifiers.
//
// Takes name (string) which is the candidate identifier.
//
// Returns bool which is true when the name is a universe block type, constant or
// function.
func IsGoPredeclared(name string) bool {
	_, exists := goPredeclaredIdentifiers[name]
	return exists
}

// IsGoPackageNameReserved reports whether a name is a legal identifier that still cannot
// name a generated package.
//
// Takes name (string) which is the candidate package name.
//
// Returns bool which is true when the name cannot be used as a package name.
func IsGoPackageNameReserved(name string) bool {
	return name == goMainPackageName
}

// SanitiseGoIdentifier turns a user-controlled name into a legal Go identifier.
//
// Takes name (string) which is the raw user-controlled name.
//
// Returns string which is a legal Go identifier.
func SanitiseGoIdentifier(name string) string {
	return guardGoReserved(sanitiseGoIdentifierRunes(name))
}

// SanitiseGoExportedIdentifier turns a user-controlled name into an exported Go
// identifier.
//
// Takes name (string) which is the raw user-controlled name.
//
// Returns string which is a legal Go identifier whose first rune is upper-case.
func SanitiseGoExportedIdentifier(name string) string {
	trimmed := strings.TrimLeft(sanitiseGoIdentifierRunes(name), underscore)
	if trimmed == "" {
		trimmed = DefaultGoIdentifier
	}

	first, width := utf8.DecodeRuneInString(trimmed)
	upper := unicode.ToUpper(first)
	if !unicode.IsUpper(upper) {
		return ExportedGoIdentifierPrefix + trimmed
	}
	return string(upper) + trimmed[width:]
}

// SanitiseGoPackageName turns a user-controlled name into a legal Go package name.
//
// Takes name (string) which is the raw user-controlled name.
//
// Returns string which is a legal Go package name.
func SanitiseGoPackageName(name string) string {
	var builder strings.Builder
	builder.Grow(len(name))
	lastWasSeparator := true

	for _, character := range strings.ToLower(name) {
		if unicode.IsLetter(character) || unicode.IsDigit(character) {
			_, _ = builder.WriteRune(character)
			lastWasSeparator = false
			continue
		}
		if !lastWasSeparator {
			_, _ = builder.WriteRune(underscoreRune)
			lastWasSeparator = true
		}
	}

	result := strings.Trim(builder.String(), underscore)
	if result == "" {
		return DefaultGoPackageName
	}
	if first, _ := utf8.DecodeRuneInString(result); !unicode.IsLetter(first) {
		result = GoPackageNamePrefix + result
	}
	return guardGoReserved(result)
}

// GoPackageAlias builds a deterministic, collision-free import alias for an import path.
//
// Takes importPath (string) which is the import path or relative file path to alias.
//
// Returns string which is a legal Go identifier unique to that path.
func GoPackageAlias(importPath string) string {
	stem := strings.ReplaceAll(filepath.ToSlash(importPath), "/", underscore)
	return GoPackageAliasWithStem(stem, importPath)
}

// GoPackageAliasWithStem builds an import alias from a stem the caller chooses.
//
// Takes stem (string) which is the human-readable part of the alias.
// Takes importPath (string) which is the import path the alias must be unique for.
//
// Returns string which is a legal Go identifier unique to that path.
func GoPackageAliasWithStem(stem, importPath string) string {
	return SanitiseGoPackageName(stem) + underscore + ShortHash(importPath)
}

// ShortHash renders a fixed-length hexadecimal digest of a string.
//
// It uses xxhash for speed and to make plain that this is not for cryptographic use: it
// exists only to keep generated names apart.
//
// Takes text (string) which is the input to hash.
//
// Returns string which is ShortHashLength hexadecimal characters.
func ShortHash(text string) string {
	return fmt.Sprintf(shortHashFormat, xxhash.Sum64String(text))[:ShortHashLength]
}

// DisambiguateIdentifier finds the first free variant of a name given the names taken.
//
// Takes name (string) which is the desired identifier.
// Takes used (map[string]V) which holds the identifiers already taken as its keys.
//
// Returns string which is a variant of the name that is not in the set.
func DisambiguateIdentifier[V any](name string, used map[string]V) string {
	if _, exists := used[name]; !exists {
		return name
	}

	candidate := make([]byte, 0, len(name)+disambiguationSuffixRoom)
	candidate = append(candidate, name...)

	for suffix := disambiguationStart; ; suffix++ {
		candidate = strconv.AppendInt(candidate[:len(name)], int64(suffix), decimalBase)
		if _, exists := used[string(candidate)]; !exists {
			return string(candidate)
		}
	}
}

// ReserveIdentifier claims a unique variant of a name given the names already taken.
//
// Takes name (string) which is the desired identifier.
// Takes used (map[string]struct{}) which holds the identifiers already taken and which
// gains the returned name.
//
// Returns string which is the claimed identifier.
func ReserveIdentifier(name string, used map[string]struct{}) string {
	claimed := DisambiguateIdentifier(name, used)
	used[claimed] = struct{}{}

	return claimed
}

// DisambiguateIdentifiers makes an ordered list of identifiers unique, preserving order.
//
// Takes names ([]string) which are the identifiers in declaration order.
//
// Returns []string which are the unique identifiers in the same order.
func DisambiguateIdentifiers(names []string) []string {
	result := make([]string, len(names))
	used := make(map[string]struct{}, len(names))
	for index, name := range names {
		result[index] = ReserveIdentifier(name, used)
	}
	return result
}

// SanitiseJSONTagName turns a raw column name into a usable JSON struct tag name.
//
// Takes name (string) which is the raw column or field name.
//
// Returns string which is safe to place inside a json struct tag.
func SanitiseJSONTagName(name string) string {
	var builder strings.Builder
	builder.Grow(len(name))

	for _, character := range name {
		if unicode.IsLetter(character) || unicode.IsDigit(character) ||
			strings.ContainsRune(jsonTagPunctuation, character) {
			_, _ = builder.WriteRune(character)
			continue
		}
		_, _ = builder.WriteRune(underscoreRune)
	}

	if builder.Len() == 0 {
		return DefaultJSONTagName
	}

	sanitised := builder.String()
	if sanitised == jsonOmittedTagName {
		return DefaultJSONTagName
	}

	return sanitised
}

// sanitiseGoIdentifierRunes rewrites a name so every rune is legal in a Go identifier.
//
// Takes name (string) which is the raw user-controlled name.
//
// Returns string which is a legal Go identifier stem.
func sanitiseGoIdentifierRunes(name string) string {
	var builder strings.Builder
	builder.Grow(len(name) + 1)

	for index, character := range name {
		switch {
		case unicode.IsLetter(character) || character == underscoreRune:
			_, _ = builder.WriteRune(character)
		case unicode.IsDigit(character):
			if index == 0 {
				_, _ = builder.WriteRune(underscoreRune)
			}
			_, _ = builder.WriteRune(character)
		default:
			_, _ = builder.WriteRune(underscoreRune)
		}
	}

	cleaned := builder.String()
	if cleaned == "" || cleaned == underscore {
		return DefaultGoIdentifier
	}
	return cleaned
}

// guardGoReserved suffixes a name that collides with a word Go has already claimed.
//
// A generated identifier equal to a keyword does not compile, and one equal to a
// predeclared identifier shadows the original for the rest of the scope, breaking every
// later line that relied on it. Both gain a trailing underscore, the conventional escape.
//
// Takes name (string) which is an otherwise legal Go identifier.
//
// Returns string which is the name, suffixed when it was reserved.
func guardGoReserved(name string) string {
	if token.IsKeyword(name) || IsGoPredeclared(name) {
		return name + underscore
	}
	return name
}

// buildGoPredeclaredIdentifiers reads the predeclared identifiers from the universe
// scope.
//
// Returns map[string]struct{} which is the predeclared identifier set.
func buildGoPredeclaredIdentifiers() map[string]struct{} {
	names := types.Universe.Names()
	identifiers := make(map[string]struct{}, len(names))
	for _, name := range names {
		identifiers[name] = struct{}{}
	}

	return identifiers
}

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

package modules_domain

import (
	"errors"
	"fmt"
	"strings"
)

// ModuleRef identifies a module dependency. It is the value compiled code embeds when it
// imports a third-party module, and the lookup key a ModuleProvider resolves to produce a
// ModuleBundle.
type ModuleRef struct {
	// Path is the module's canonical identifier, mandatory.
	Path string

	// Version is the human-readable selector or empty for latest.
	Version string

	// Pin is the integrity hash a host can cross-check.
	Pin string
}

// String formats a ModuleRef in canonical "path@version[#pin]" form.
//
// Empty Version is rendered without the @ separator; empty Pin is rendered without the #
// separator. Used by error messages, lockfile output, and CLI listings - not by wire
// serialisation.
//
// Returns string which is the formatted reference.
func (r ModuleRef) String() string {
	var builder strings.Builder
	builder.WriteString(r.Path)
	if r.Version != "" {
		builder.WriteByte('@')
		builder.WriteString(r.Version)
	}
	if r.Pin != "" {
		builder.WriteByte('#')
		builder.WriteString(r.Pin)
	}
	return builder.String()
}

// IsZero reports whether the ref is the zero value. A zero ref is not a valid lookup key;
// providers return ErrModuleNotFound when asked to resolve it.
//
// Returns true when Path, Version, and Pin are all empty.
func (r ModuleRef) IsZero() bool {
	return r.Path == "" && r.Version == "" && r.Pin == ""
}

// ParseModuleRef parses a "path[@version][#pin]" string into a ref.
//
// Whitespace around the input is trimmed. Empty input returns an error; otherwise parsing
// is total - invalid pin shapes are passed through unchanged for the caller to validate
// when checking against actual bundle bytes.
//
// Takes input (string) which is the reference in canonical form.
//
// Returns ModuleRef which is the parsed reference.
// Returns error when input is empty or lacks a path component.
func ParseModuleRef(input string) (ModuleRef, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return ModuleRef{}, errors.New("modules_domain: empty module reference")
	}
	ref := ModuleRef{}
	if pinIndex := strings.IndexByte(trimmed, '#'); pinIndex >= 0 {
		ref.Pin = trimmed[pinIndex+1:]
		trimmed = trimmed[:pinIndex]
	}
	if versionIndex := strings.IndexByte(trimmed, '@'); versionIndex >= 0 {
		ref.Version = trimmed[versionIndex+1:]
		trimmed = trimmed[:versionIndex]
	}
	ref.Path = trimmed
	if ref.Path == "" {
		return ModuleRef{}, fmt.Errorf("modules_domain: module reference %q has empty path", input)
	}
	return ref, nil
}

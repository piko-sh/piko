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
	"strings"
)

// Capability is a single capability claim that a module declares it needs at runtime. The
// host's CapabilityHook interprets these claims - piko itself does not enforce semantics
// on Axis or Scope values.
//
// Axis names a category of operation, in dotted form:
//
//   - "filesystem.read", "filesystem.write": file I/O
//   - "exec": process spawn
//   - "network", "network.dial", "network.listen": socket I/O
//   - "env.read", "env.write": environment variables
//   - "time.wall": wall-clock time access (deterministic time is always available)
//
// Scope refines Axis with a host-defined pattern. For path-shaped axes (filesystem.*)
// Scope is a path prefix or glob. For network axes Scope is a host pattern. For exec it
// is a command pattern. Empty Scope means "all" within the axis.
//
// Hosts are free to invent additional axes; the only contract piko enforces is round-trip
// stability through serialisation.
type Capability struct {
	// Axis names the category of operation for this claim.
	Axis string

	// Scope refines Axis with a host-defined pattern or empty for all.
	Scope string
}

// String formats the capability as "axis" or "axis(scope)" depending on whether Scope is
// set. Used by error messages and lockfile output.
//
// Returns the human-readable form.
func (c Capability) String() string {
	if c.Scope == "" {
		return c.Axis
	}
	var builder strings.Builder
	builder.WriteString(c.Axis)
	builder.WriteByte('(')
	builder.WriteString(c.Scope)
	builder.WriteByte(')')
	return builder.String()
}

// IsZero reports whether the capability is the zero value. The zero value is not a
// meaningful claim; serialisation drops it.
//
// Returns true when both Axis and Scope are empty.
func (c Capability) IsZero() bool {
	return c.Axis == "" && c.Scope == ""
}

// ParseCapability parses a string in "axis" or "axis(scope)" form.
//
// Whitespace around the input is trimmed. Input shape is permissive: a trailing or
// unterminated paren is treated as an empty Scope.
//
// Takes input (string) which is the capability in canonical form.
//
// Returns Capability which is the parsed claim.
func ParseCapability(input string) Capability {
	trimmed := strings.TrimSpace(input)
	axis, scope, hasScope := strings.Cut(trimmed, "(")
	if !hasScope {
		return Capability{Axis: trimmed}
	}
	if closeParen := strings.LastIndexByte(scope, ')'); closeParen >= 0 {
		scope = scope[:closeParen]
	}
	return Capability{Axis: strings.TrimSpace(axis), Scope: strings.TrimSpace(scope)}
}

// CapabilitySet is an ordered collection of Capability claims. Order is preserved through
// serialisation so descriptor hashes are stable, but membership semantics are unordered:
// equal sets must produce equal hashes after Normalise.
type CapabilitySet []Capability

// Normalise returns a copy of the set with zero entries removed and the remaining entries
// sorted by (Axis, Scope). Used before hashing a descriptor so equivalent declarations
// compare equal regardless of declaration order.
//
// Returns a freshly-allocated normalised copy; the receiver is left unchanged.
func (s CapabilitySet) Normalise() CapabilitySet {
	if len(s) == 0 {
		return nil
	}
	out := make(CapabilitySet, 0, len(s))
	for _, capability := range s {
		if capability.IsZero() {
			continue
		}
		out = append(out, capability)
	}
	sortCapabilitiesInPlace(out)
	return out
}

// Contains reports whether the set has an exact (Axis, Scope) match for target. Use this
// for membership checks before hashing; for policy decisions, consult the CapabilityHook
// instead.
//
// Takes target (Capability) which is the claim to look up.
//
// Returns true when an entry matches both Axis and Scope exactly.
func (s CapabilitySet) Contains(target Capability) bool {
	for _, capability := range s {
		if capability.Axis == target.Axis && capability.Scope == target.Scope {
			return true
		}
	}
	return false
}

// IsSubsetOf reports whether every entry in the receiver appears in other. Used by
// validators to check that a module's declared capabilities are a subset of the policy's
// grants.
//
// Takes other (CapabilitySet) which is the superset under test.
//
// Returns true when every receiver entry is contained in other.
func (s CapabilitySet) IsSubsetOf(other CapabilitySet) bool {
	for _, capability := range s {
		if !other.Contains(capability) {
			return false
		}
	}
	return true
}

// sortCapabilitiesInPlace sorts a CapabilitySet by (Axis, Scope).
//
// Tiny insertion sort suits the typical N <= 20 case and avoids the sort package's
// per-call closure overhead.
//
// Takes s (CapabilitySet) which is sorted in place.
func sortCapabilitiesInPlace(s CapabilitySet) {
	for i := 1; i < len(s); i++ {
		current := s[i]
		j := i - 1
		for j >= 0 && capabilityLess(current, s[j]) {
			s[j+1] = s[j]
			j--
		}
		s[j+1] = current
	}
}

// capabilityLess reports whether a sorts before b, breaking ties by Scope.
//
// Takes a (Capability) which is the left-hand entry.
// Takes b (Capability) which is the right-hand entry.
//
// Returns bool which is true when a precedes b.
func capabilityLess(a, b Capability) bool {
	if a.Axis != b.Axis {
		return a.Axis < b.Axis
	}
	return a.Scope < b.Scope
}

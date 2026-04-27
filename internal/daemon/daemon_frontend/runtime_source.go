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

package daemon_frontend

import (
	"crypto/sha256"
	"crypto/sha512"
	_ "embed"
	"encoding/base64"
	"errors"
	"fmt"
	"hash"
	"strings"
)

// frameworkCoreJS is the source of the core Piko frontend module
// (ppframework.core.es.min.js). Available in every build flavour
// including WebAssembly, where the heavier embed.FS in
// embedded_frontend_templates.go is excluded behind `//go:build !js` to
// keep the WASM binary lean (no source maps, no pre-compressed
// variants; just the runtime source the playground iframe needs).
//
//go:embed built/ppframework.core.es.min.js
var frameworkCoreJS string

// frameworkCoreJSSRI is the build-time-fixed SHA-256 SRI hash of the
// embedded core runtime. ValidateRuntimeSourceIntegrity asserts that the
// embedded bytes match this hash on first call so a tampered or
// out-of-date bundle fails-closed before a consumer ever serves it.
//
//go:embed built/ppframework.core.es.min.js.sri
var frameworkCoreJSSRI string

// frameworkComponentsJS is the source of the components extension module
// (ppframework.components.es.js). Compiled component classes import
// PPElement, dom, makeReactive from this module.
//
//go:embed built/ppframework.components.es.js
var frameworkComponentsJS string

// frameworkComponentsJSSRI is the build-time-fixed SHA-256 SRI hash of
// the embedded components runtime. See frameworkCoreJSSRI.
//
//go:embed built/ppframework.components.es.js.sri
var frameworkComponentsJSSRI string

// FrameworkRuntimeSource exposes an embedded framework module source.
//
// Callers within the binary (e.g. the WASM playground iframe assembler)
// receive the source paired with its SRI attribute so consumers can
// fail-closed if the embedded bytes do not match the build-time SRI
// hash, ensuring tampered or stale bundles never reach a browser.
type FrameworkRuntimeSource struct {
	// Source is the JavaScript source as embedded.
	Source string

	// SRI is the SubResource Integrity attribute value the consumer
	// should put on the <script integrity="..."> tag.
	SRI string
}

// errFrameworkRuntimeIntegrity is returned when an embedded runtime
// source does not match its build-time SRI hash. The error chains the
// resource name so the operator can locate the tampered file.
var errFrameworkRuntimeIntegrity = errors.New("framework runtime integrity check failed")

// FrameworkCore returns the core runtime source paired with its SRI
// attribute. The returned source has been verified against its embedded
// SRI hash on the first call.
//
// Returns FrameworkRuntimeSource which is the verified core runtime.
// Returns error when the embedded bytes do not match the build-time SRI.
func FrameworkCore() (FrameworkRuntimeSource, error) {
	return verifyFrameworkSource("ppframework.core.es.min.js", frameworkCoreJS, frameworkCoreJSSRI)
}

// FrameworkComponents returns the components runtime source paired with
// its SRI attribute. See FrameworkCore.
//
// Returns FrameworkRuntimeSource which is the verified components
// runtime.
// Returns error when the embedded bytes do not match the build-time SRI.
func FrameworkComponents() (FrameworkRuntimeSource, error) {
	return verifyFrameworkSource("ppframework.components.es.js", frameworkComponentsJS, frameworkComponentsJSSRI)
}

// verifyFrameworkSource computes a SHA-256 SRI for source and compares
// against the embedded value (stripped of any "sha384-" / "sha256-"
// prefix and surrounding whitespace). The SRI files in built/ may use
// either sha256 or sha384; this verifier accepts either prefix and
// recomputes accordingly.
//
// Takes resource (string) which names the resource for error reporting.
// Takes source (string) which is the embedded JavaScript content.
// Takes sriEntry (string) which is the build-time fixed SRI hash.
//
// Returns FrameworkRuntimeSource which exposes source plus the verified
// SRI attribute value.
// Returns error when the source does not match the embedded hash.
func verifyFrameworkSource(resource, source, sriEntry string) (FrameworkRuntimeSource, error) {
	expected := strings.TrimSpace(sriEntry)
	if expected == "" {
		return FrameworkRuntimeSource{}, fmt.Errorf("%w: %s missing SRI", errFrameworkRuntimeIntegrity, resource)
	}
	if source == "" {
		return FrameworkRuntimeSource{}, fmt.Errorf("%w: %s missing source", errFrameworkRuntimeIntegrity, resource)
	}

	switch {
	case strings.HasPrefix(expected, "sha256-"):
		if !verifySRI(source, expected[len("sha256-"):], sha256.New) {
			return FrameworkRuntimeSource{}, fmt.Errorf("%w: %s mismatched sha256", errFrameworkRuntimeIntegrity, resource)
		}
	case strings.HasPrefix(expected, "sha384-"):
		if !verifySRI(source, expected[len("sha384-"):], sha512.New384) {
			return FrameworkRuntimeSource{}, fmt.Errorf("%w: %s mismatched sha384", errFrameworkRuntimeIntegrity, resource)
		}
	case strings.HasPrefix(expected, "sha512-"):
		if !verifySRI(source, expected[len("sha512-"):], sha512.New) {
			return FrameworkRuntimeSource{}, fmt.Errorf("%w: %s mismatched sha512", errFrameworkRuntimeIntegrity, resource)
		}
	default:
		return FrameworkRuntimeSource{}, fmt.Errorf("%w: %s unsupported SRI prefix", errFrameworkRuntimeIntegrity, resource)
	}

	return FrameworkRuntimeSource{Source: source, SRI: expected}, nil
}

// verifySRI computes the digest of source using factory and compares it
// against the base64-encoded expected hash.
//
// Takes source (string) which is the content to hash.
// Takes expectedBase64 (string) which is the base64-encoded expected
// digest (no algorithm prefix).
// Takes factory which constructs a fresh hash.Hash.
//
// Returns bool which is true when the digest matches.
func verifySRI(source, expectedBase64 string, factory func() hash.Hash) bool {
	h := factory()
	if _, err := h.Write([]byte(source)); err != nil {
		return false
	}
	got := base64.StdEncoding.EncodeToString(h.Sum(nil))
	return got == expectedBase64
}

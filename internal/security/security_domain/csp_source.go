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

package security_domain

import "fmt"

// Source represents a Content-Security-Policy source value. Sources can be
// keywords (like 'self'), schemes (like 'data:'), hosts (like
// 'cdn.example.com'), or special values (like hashes and request tokens).
type Source string

const (
	// Self allows resources from the same origin (scheme, host, and port).
	Self Source = "self"

	// None blocks all resources for this directive.
	None Source = "none"

	// UnsafeInline allows inline scripts and styles.
	// Warning: This significantly reduces CSP protection against XSS.
	UnsafeInline Source = "unsafe-inline"

	// UnsafeEval allows use of eval() and similar dynamic code execution.
	// Warning: This significantly reduces CSP protection against XSS.
	UnsafeEval Source = "unsafe-eval"

	// UnsafeHashes allows specific inline event handlers based on their hash.
	UnsafeHashes Source = "unsafe-hashes"

	// StrictDynamic allows scripts loaded by trusted scripts to execute.
	// When present, 'self' and URL-based allowlists are ignored for script-src.
	StrictDynamic Source = "strict-dynamic"

	// ReportSample tells the browser to include a sample of the code that broke
	// the rules in violation reports.
	ReportSample Source = "report-sample"

	// WasmUnsafeEval allows WebAssembly execution.
	WasmUnsafeEval Source = "wasm-unsafe-eval"

	// Data is a CSP source value that allows data: URIs for inline content
	// such as images and fonts.
	Data Source = "data:"

	// Blob allows blob: URIs for binary data created with URL.createObjectURL.
	Blob Source = "blob:"

	// HTTPS allows any resource loaded over a secure HTTPS connection.
	HTTPS Source = "https:"

	// HTTP allows any resource loaded over HTTP.
	// Warning: this reduces security; prefer HTTPS where possible.
	HTTP Source = "http:"

	// FileSystem is the source prefix for filesystem URIs.
	FileSystem Source = "filesystem:"

	// MediaStream is the source value that allows mediastream: URIs.
	MediaStream Source = "mediastream:"

	// RequestTokenPlaceholder is a placeholder that the security middleware
	// replaces with a unique token for each request, enabling strict CSP
	// policies that permit specific inline scripts and styles.
	RequestTokenPlaceholder Source = "{{REQUEST_TOKEN}}"

	// Script is the keyword for require-trusted-types-for directive.
	// It is the only valid value for that directive and will be quoted.
	Script Source = "script"

	// AllowDuplicates allows more than one Trusted Types policy with the same
	// name. Used with the trusted-types directive.
	AllowDuplicates Source = "allow-duplicates"

	// Wildcard allows Trusted Types policies to be created with any unique name.
	// Unlike keywords, it is not quoted in the output.
	Wildcard Source = "*"
)

// keywordSources is the set of sources that require single-quoting.
var keywordSources = map[Source]bool{
	Self:            true,
	None:            true,
	UnsafeInline:    true,
	UnsafeEval:      true,
	UnsafeHashes:    true,
	StrictDynamic:   true,
	ReportSample:    true,
	WasmUnsafeEval:  true,
	Script:          true,
	AllowDuplicates: true,
}

// Host creates a Source from a host specification string.
//
// Takes h (string) which specifies the host in formats such as
// "cdn.example.com", "*.example.com", or "https://cdn.example.com".
//
// Returns Source which represents the configured host origin.
func Host(h string) Source {
	return Source(h)
}

// Scheme creates a Source from a URL scheme.
// The scheme should include the trailing colon (e.g., "wss:").
//
// Takes s (string) which is the URL scheme including the trailing colon.
//
// Returns Source which is the scheme converted to a Source type.
func Scheme(s string) Source {
	return Source(s)
}

// Hash creates a Source from a hash algorithm and base64-encoded hash value.
// The result is automatically single-quoted.
//
// Takes algorithm (string) which specifies the hash algorithm (e.g. "sha256").
// Takes hash (string) which is the base64-encoded hash value.
//
// Returns Source which is the formatted hash directive (e.g. 'sha256-abc123').
func Hash(algorithm, hash string) Source {
	return Source(fmt.Sprintf("'%s-%s'", algorithm, hash))
}

// SHA256 creates a Source from a base64-encoded SHA-256 hash.
//
// Use this for allowing specific inline scripts or styles by their content
// hash.
//
// Takes hash (string) which is the base64-encoded SHA-256 hash of the content.
//
// Returns Source which represents the hash-based content source.
func SHA256(hash string) Source {
	return Hash("sha256", hash)
}

// SHA384 creates a Source from a base64-encoded SHA-384 hash.
//
// Takes hash (string) which is the base64-encoded SHA-384 hash value.
//
// Returns Source which represents the hash as a content source.
func SHA384(hash string) Source {
	return Hash("sha384", hash)
}

// SHA512 creates a Source from a base64-encoded SHA-512 hash.
//
// Takes hash (string) which is the base64-encoded SHA-512 hash value.
//
// Returns Source which represents the hash as a verification source.
func SHA512(hash string) Source {
	return Hash("sha512", hash)
}

// RequestToken creates a Source from a specific token value.
//
// This produces the CSP-spec required format for inline script/style
// authorisation. For dynamic per-request tokens, use RequestTokenPlaceholder
// instead.
//
// Takes token (string) which is the per-request token value to embed in the
// source.
//
// Returns Source which is the formatted CSP per-request token directive.
func RequestToken(token string) Source {
	return Source(fmt.Sprintf("'nonce-%s'", token))
}

// PolicyName creates a Source from a Trusted Types policy name.
//
// Policy names are not quoted in the CSP header output. Valid characters are
// alphanumeric, dash, hash, equals, underscore, slash, at-sign, period, and
// percent.
//
// Takes name (string) which specifies the policy name.
//
// Returns Source which contains the unquoted policy name for CSP output.
func PolicyName(name string) Source {
	return Source(name)
}

// isKeyword reports whether the source is a CSP keyword that requires quoting.
//
// Takes s (Source) which is the CSP source value to check.
//
// Returns bool which is true if the source is a keyword, false otherwise.
func isKeyword(s Source) bool {
	return keywordSources[s]
}

// formatSourceForHeader formats a source value for inclusion in a CSP header.
// Keywords are automatically single-quoted; other values are returned as-is.
//
// Takes s (Source) which is the source value to format.
//
// Returns string which is the formatted source ready for header inclusion.
func formatSourceForHeader(s Source) string {
	if isKeyword(s) {
		return fmt.Sprintf("'%s'", s)
	}
	return string(s)
}

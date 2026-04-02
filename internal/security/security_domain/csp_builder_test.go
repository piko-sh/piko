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

import (
	"strings"
	"testing"
)

func TestCSPBuilder_Build(t *testing.T) {
	testCases := []struct {
		name     string
		builder  func() *CSPBuilder
		expected string
	}{
		{
			name: "empty builder produces empty string",
			builder: func() *CSPBuilder {
				return NewCSPBuilder()
			},
			expected: "",
		},
		{
			name: "single directive with self",
			builder: func() *CSPBuilder {
				return NewCSPBuilder().DefaultSrc(Self)
			},
			expected: "default-src 'self'",
		},
		{
			name: "single directive with multiple sources",
			builder: func() *CSPBuilder {
				return NewCSPBuilder().ScriptSrc(Self, Host("cdn.example.com"))
			},
			expected: "script-src 'self' cdn.example.com",
		},
		{
			name: "multiple directives maintain order",
			builder: func() *CSPBuilder {
				return NewCSPBuilder().
					DefaultSrc(Self).
					ScriptSrc(Self).
					StyleSrc(Self, UnsafeInline)
			},
			expected: "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'",
		},
		{
			name: "keywords are properly quoted",
			builder: func() *CSPBuilder {
				return NewCSPBuilder().
					ScriptSrc(Self, UnsafeInline, UnsafeEval, StrictDynamic)
			},
			expected: "script-src 'self' 'unsafe-inline' 'unsafe-eval' 'strict-dynamic'",
		},
		{
			name: "schemes are not quoted",
			builder: func() *CSPBuilder {
				return NewCSPBuilder().
					ImgSrc(Self, Data, Blob, HTTPS)
			},
			expected: "img-src 'self' data: blob: https:",
		},
		{
			name: "hosts are not quoted",
			builder: func() *CSPBuilder {
				return NewCSPBuilder().
					StyleSrc(Self, Host("fonts.googleapis.com"), Host("*.example.com"))
			},
			expected: "style-src 'self' fonts.googleapis.com *.example.com",
		},
		{
			name: "hashes are pre-quoted",
			builder: func() *CSPBuilder {
				return NewCSPBuilder().
					ScriptSrc(Self, SHA256("abc123"))
			},
			expected: "script-src 'self' 'sha256-abc123'",
		},
		{
			name: "boolean directive has no sources",
			builder: func() *CSPBuilder {
				return NewCSPBuilder().
					DefaultSrc(Self).
					UpgradeInsecureRequests()
			},
			expected: "default-src 'self'; upgrade-insecure-requests",
		},
		{
			name: "multiple boolean directives",
			builder: func() *CSPBuilder {
				return NewCSPBuilder().
					DefaultSrc(Self).
					UpgradeInsecureRequests().
					BlockAllMixedContent()
			},
			expected: "default-src 'self'; upgrade-insecure-requests; block-all-mixed-content",
		},
		{
			name: "request token placeholder is preserved",
			builder: func() *CSPBuilder {
				return NewCSPBuilder().
					ScriptSrc(Self, RequestTokenPlaceholder)
			},
			expected: "script-src 'self' {{REQUEST_TOKEN}}",
		},
		{
			name: "piko defaults produce expected output",
			builder: func() *CSPBuilder {
				return NewCSPBuilder().WithPikoDefaults()
			},
			expected: "default-src 'self'; " +
				"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; " +
				"script-src-attr 'unsafe-hashes' 'sha256-1jAmyYXcRq6zFldLe/GCgIDJBiOONdXjTLgEFMDnDSM='; " +
				"font-src 'self' https://fonts.gstatic.com data:; " +
				"img-src 'self' data: blob: https:; " +
				"connect-src 'self'",
		},
		{
			name: "report-to directive",
			builder: func() *CSPBuilder {
				return NewCSPBuilder().
					DefaultSrc(Self).
					ReportToDirective("csp-endpoint")
			},
			expected: "default-src 'self'; report-to csp-endpoint",
		},
		{
			name: "none source",
			builder: func() *CSPBuilder {
				return NewCSPBuilder().ObjectSrc(None)
			},
			expected: "object-src 'none'",
		},
		{
			name: "all fetch directives",
			builder: func() *CSPBuilder {
				return NewCSPBuilder().
					DefaultSrc(Self).
					ScriptSrc(Self).
					StyleSrc(Self).
					ImgSrc(Self).
					FontSrc(Self).
					ConnectSrc(Self).
					MediaSrc(Self).
					ObjectSrc(None).
					FrameSrc(Self).
					ChildSrc(Self).
					WorkerSrc(Self).
					ManifestSrc(Self)
			},
			expected: "default-src 'self'; script-src 'self'; style-src 'self'; " +
				"img-src 'self'; font-src 'self'; connect-src 'self'; media-src 'self'; " +
				"object-src 'none'; frame-src 'self'; child-src 'self'; worker-src 'self'; " +
				"manifest-src 'self'",
		},
		{
			name: "document directives",
			builder: func() *CSPBuilder {
				return NewCSPBuilder().
					BaseURI(Self).
					FormAction(Self)
			},
			expected: "base-uri 'self'; form-action 'self'",
		},
		{
			name: "frame-ancestors directive",
			builder: func() *CSPBuilder {
				return NewCSPBuilder().
					FrameAncestors(Self, Host("https://trusted.example.com"))
			},
			expected: "frame-ancestors 'self' https://trusted.example.com",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.builder().Build()
			if result != tc.expected {
				t.Errorf("Build() = %q, want %q", result, tc.expected)
			}
		})
	}
}

func TestCSPBuilder_HeaderName(t *testing.T) {
	testCases := []struct {
		name       string
		expected   string
		reportOnly bool
	}{
		{
			name:       "enforcing mode",
			expected:   "Content-Security-Policy",
			reportOnly: false,
		},
		{
			name:       "report-only mode",
			expected:   "Content-Security-Policy-Report-Only",
			reportOnly: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			builder := NewCSPBuilder().DefaultSrc(Self)
			if tc.reportOnly {
				builder.ReportOnly()
			}
			if got := builder.HeaderName(); got != tc.expected {
				t.Errorf("HeaderName() = %q, want %q", got, tc.expected)
			}
		})
	}
}

func TestCSPBuilder_UsesRequestTokens(t *testing.T) {
	testCases := []struct {
		builder  func() *CSPBuilder
		name     string
		expected bool
	}{
		{
			name: "no tokens",
			builder: func() *CSPBuilder {
				return NewCSPBuilder().DefaultSrc(Self)
			},
			expected: false,
		},
		{
			name: "with request token placeholder",
			builder: func() *CSPBuilder {
				return NewCSPBuilder().ScriptSrc(Self, RequestTokenPlaceholder)
			},
			expected: true,
		},
		{
			name: "with static token (not placeholder)",
			builder: func() *CSPBuilder {
				return NewCSPBuilder().ScriptSrc(Self, RequestToken("static-token"))
			},
			expected: false,
		},
		{
			name: "token in multiple directives",
			builder: func() *CSPBuilder {
				return NewCSPBuilder().
					ScriptSrc(Self, RequestTokenPlaceholder).
					StyleSrc(Self, RequestTokenPlaceholder)
			},
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.builder().UsesRequestTokens(); got != tc.expected {
				t.Errorf("UsesRequestTokens() = %v, want %v", got, tc.expected)
			}
		})
	}
}

func TestCSPBuilder_Clone(t *testing.T) {
	original := NewCSPBuilder().
		DefaultSrc(Self).
		ScriptSrc(Self, Host("cdn.example.com")).
		ReportOnly()

	clone := original.Clone()

	clone.StyleSrc(Self, UnsafeInline)

	originalBuild := original.Build()
	cloneBuild := clone.Build()

	if strings.Contains(originalBuild, "style-src") {
		t.Error("Original was modified when clone was changed")
	}

	if !strings.Contains(cloneBuild, "style-src") {
		t.Error("Clone should have style-src directive")
	}

	if !strings.Contains(originalBuild, "default-src 'self'") {
		t.Error("Original missing default-src")
	}
	if !strings.Contains(cloneBuild, "default-src 'self'") {
		t.Error("Clone missing default-src")
	}

	if !clone.IsReportOnly() {
		t.Error("Clone should preserve report-only setting")
	}
}

func TestCSPBuilder_Add_Generic(t *testing.T) {

	builder := NewCSPBuilder().
		Add(DefaultSrc, Self).
		Add(ScriptSrc, Self, Host("cdn.example.com")).
		Add(UpgradeInsecureRequests)

	result := builder.Build()

	expected := "default-src 'self'; script-src 'self' cdn.example.com; upgrade-insecure-requests"
	if result != expected {
		t.Errorf("Add() produced %q, want %q", result, expected)
	}
}

func TestSource_Helpers(t *testing.T) {
	testCases := []struct {
		name     string
		source   Source
		expected string
	}{
		{
			name:     "Host helper",
			source:   Host("cdn.example.com"),
			expected: "cdn.example.com",
		},
		{
			name:     "Scheme helper",
			source:   Scheme("wss:"),
			expected: "wss:",
		},
		{
			name:     "SHA256 helper",
			source:   SHA256("abc123"),
			expected: "'sha256-abc123'",
		},
		{
			name:     "SHA384 helper",
			source:   SHA384("def456"),
			expected: "'sha384-def456'",
		},
		{
			name:     "SHA512 helper",
			source:   SHA512("ghi789"),
			expected: "'sha512-ghi789'",
		},
		{
			name:     "Hash helper with custom algorithm",
			source:   Hash("sha256", "custom"),
			expected: "'sha256-custom'",
		},
		{
			name:     "RequestToken helper",
			source:   RequestToken("my-token-123"),
			expected: "'nonce-my-token-123'",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if string(tc.source) != tc.expected {
				t.Errorf("Source = %q, want %q", tc.source, tc.expected)
			}
		})
	}
}

func TestFormatSourceForHeader(t *testing.T) {
	testCases := []struct {
		name     string
		source   Source
		expected string
	}{
		{
			name:     "self is quoted",
			source:   Self,
			expected: "'self'",
		},
		{
			name:     "none is quoted",
			source:   None,
			expected: "'none'",
		},
		{
			name:     "unsafe-inline is quoted",
			source:   UnsafeInline,
			expected: "'unsafe-inline'",
		},
		{
			name:     "strict-dynamic is quoted",
			source:   StrictDynamic,
			expected: "'strict-dynamic'",
		},
		{
			name:     "data scheme is not quoted",
			source:   Data,
			expected: "data:",
		},
		{
			name:     "host is not quoted",
			source:   Host("example.com"),
			expected: "example.com",
		},
		{
			name:     "hash is already quoted",
			source:   SHA256("abc"),
			expected: "'sha256-abc'",
		},
		{
			name:     "request token placeholder is not quoted",
			source:   RequestTokenPlaceholder,
			expected: "{{REQUEST_TOKEN}}",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := formatSourceForHeader(tc.source); got != tc.expected {
				t.Errorf("formatSourceForHeader(%q) = %q, want %q", tc.source, got, tc.expected)
			}
		})
	}
}

func TestIsBooleanDirective(t *testing.T) {
	testCases := []struct {
		directive Directive
		expected  bool
	}{
		{directive: UpgradeInsecureRequests, expected: true},
		{directive: BlockAllMixedContent, expected: true},
		{directive: DefaultSrc, expected: false},
		{directive: ScriptSrc, expected: false},
		{directive: StyleSrc, expected: false},
	}

	for _, tc := range testCases {
		t.Run(string(tc.directive), func(t *testing.T) {
			if got := isBooleanDirective(tc.directive); got != tc.expected {
				t.Errorf("isBooleanDirective(%q) = %v, want %v", tc.directive, got, tc.expected)
			}
		})
	}
}

func TestIsKeyword(t *testing.T) {
	testCases := []struct {
		source   Source
		expected bool
	}{
		{source: Self, expected: true},
		{source: None, expected: true},
		{source: UnsafeInline, expected: true},
		{source: UnsafeEval, expected: true},
		{source: UnsafeHashes, expected: true},
		{source: StrictDynamic, expected: true},
		{source: ReportSample, expected: true},
		{source: WasmUnsafeEval, expected: true},
		{source: Data, expected: false},
		{source: Blob, expected: false},
		{source: HTTPS, expected: false},
		{source: Host("example.com"), expected: false},
		{source: RequestTokenPlaceholder, expected: false},
	}

	for _, tc := range testCases {
		t.Run(string(tc.source), func(t *testing.T) {
			if got := isKeyword(tc.source); got != tc.expected {
				t.Errorf("isKeyword(%q) = %v, want %v", tc.source, got, tc.expected)
			}
		})
	}
}

func TestCSPBuilder_Chaining(t *testing.T) {

	builder := NewCSPBuilder().
		DefaultSrc(Self).
		ScriptSrc(Self).
		ScriptSrcElem(Self).
		ScriptSrcAttr(Self).
		StyleSrc(Self).
		StyleSrcElem(Self).
		StyleSrcAttr(Self).
		ImgSrc(Self).
		FontSrc(Self).
		ConnectSrc(Self).
		MediaSrc(Self).
		ObjectSrc(None).
		FrameSrc(Self).
		ChildSrc(Self).
		WorkerSrc(Self).
		ManifestSrc(Self).
		PrefetchSrc(Self).
		BaseURI(Self).
		FormAction(Self).
		FrameAncestors(Self).
		ReportToDirective("csp-group").
		UpgradeInsecureRequests().
		BlockAllMixedContent().
		Sandbox(SandboxAllowScripts).
		ReportOnly()

	result := builder.Build()
	if result == "" {
		t.Error("Chained builder produced empty result")
	}

	if !builder.IsReportOnly() {
		t.Error("ReportOnly() did not set report-only mode")
	}
}

func TestCSPBuilder_Sandbox(t *testing.T) {
	testCases := []struct {
		name     string
		builder  func() *CSPBuilder
		expected string
	}{
		{
			name: "empty sandbox applies maximum restrictions",
			builder: func() *CSPBuilder {
				return NewCSPBuilder().Sandbox()
			},
			expected: "sandbox",
		},
		{
			name: "sandbox with single token",
			builder: func() *CSPBuilder {
				return NewCSPBuilder().Sandbox(SandboxAllowScripts)
			},
			expected: "sandbox allow-scripts",
		},
		{
			name: "sandbox with multiple tokens",
			builder: func() *CSPBuilder {
				return NewCSPBuilder().Sandbox(SandboxAllowScripts, SandboxAllowForms)
			},
			expected: "sandbox allow-scripts allow-forms",
		},
		{
			name: "sandbox tokens are not quoted",
			builder: func() *CSPBuilder {
				return NewCSPBuilder().Sandbox(SandboxAllowSameOrigin, SandboxAllowPopups)
			},
			expected: "sandbox allow-same-origin allow-popups",
		},
		{
			name: "sandbox with other directives",
			builder: func() *CSPBuilder {
				return NewCSPBuilder().
					DefaultSrc(Self).
					Sandbox(SandboxAllowScripts).
					UpgradeInsecureRequests()
			},
			expected: "default-src 'self'; sandbox allow-scripts; upgrade-insecure-requests",
		},
		{
			name: "sandbox all common tokens",
			builder: func() *CSPBuilder {
				return NewCSPBuilder().Sandbox(
					SandboxAllowDownloads,
					SandboxAllowForms,
					SandboxAllowModals,
					SandboxAllowOrientationLock,
					SandboxAllowPointerLock,
					SandboxAllowPopups,
					SandboxAllowPopupsToEscapeSandbox,
					SandboxAllowPresentation,
					SandboxAllowSameOrigin,
					SandboxAllowScripts,
					SandboxAllowStorageAccessByUserActivation,
					SandboxAllowTopNavigation,
					SandboxAllowTopNavigationByUserActivation,
					SandboxAllowTopNavigationToCustomProtocols,
				)
			},
			expected: "sandbox allow-downloads allow-forms allow-modals allow-orientation-lock allow-pointer-lock allow-popups allow-popups-to-escape-sandbox allow-presentation allow-same-origin allow-scripts allow-storage-access-by-user-activation allow-top-navigation allow-top-navigation-by-user-activation allow-top-navigation-to-custom-protocols",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.builder().Build()
			if result != tc.expected {
				t.Errorf("Build() = %q, want %q", result, tc.expected)
			}
		})
	}
}

func TestIsValidSandboxToken(t *testing.T) {
	testCases := []struct {
		tok      SandboxToken
		expected bool
	}{
		{tok: SandboxAllowScripts, expected: true},
		{tok: SandboxAllowForms, expected: true},
		{tok: SandboxAllowModals, expected: true},
		{tok: SandboxAllowSameOrigin, expected: true},
		{tok: SandboxAllowPopups, expected: true},
		{tok: SandboxToken("invalid-token"), expected: false},
		{tok: SandboxToken(""), expected: false},
	}

	for _, tc := range testCases {
		t.Run(string(tc.tok), func(t *testing.T) {
			if got := isValidSandboxToken(tc.tok); got != tc.expected {
				t.Errorf("isValidSandboxToken(%q) = %v, want %v", tc.tok, got, tc.expected)
			}
		})
	}
}

func TestCSPBuilder_TrustedTypes(t *testing.T) {
	testCases := []struct {
		name     string
		builder  func() *CSPBuilder
		expected string
	}{
		{
			name: "require-trusted-types-for outputs script keyword quoted",
			builder: func() *CSPBuilder {
				return NewCSPBuilder().RequireTrustedTypesFor()
			},
			expected: "require-trusted-types-for 'script'",
		},
		{
			name: "trusted-types with single policy name unquoted",
			builder: func() *CSPBuilder {
				return NewCSPBuilder().TrustedTypes(PolicyName("default"))
			},
			expected: "trusted-types default",
		},
		{
			name: "trusted-types with multiple policy names",
			builder: func() *CSPBuilder {
				return NewCSPBuilder().TrustedTypes(PolicyName("default"), PolicyName("dompurify"))
			},
			expected: "trusted-types default dompurify",
		},
		{
			name: "trusted-types none is quoted",
			builder: func() *CSPBuilder {
				return NewCSPBuilder().TrustedTypesNone()
			},
			expected: "trusted-types 'none'",
		},
		{
			name: "trusted-types with wildcard unquoted",
			builder: func() *CSPBuilder {
				return NewCSPBuilder().TrustedTypes(Wildcard)
			},
			expected: "trusted-types *",
		},
		{
			name: "trusted-types with allow-duplicates quoted",
			builder: func() *CSPBuilder {
				return NewCSPBuilder().TrustedTypesWithDuplicates(PolicyName("foo"), PolicyName("bar"))
			},
			expected: "trusted-types foo bar 'allow-duplicates'",
		},
		{
			name: "trusted-types wildcard with allow-duplicates",
			builder: func() *CSPBuilder {
				return NewCSPBuilder().TrustedTypesWithDuplicates(Wildcard)
			},
			expected: "trusted-types * 'allow-duplicates'",
		},
		{
			name: "both trusted types directives together",
			builder: func() *CSPBuilder {
				return NewCSPBuilder().
					DefaultSrc(Self).
					RequireTrustedTypesFor().
					TrustedTypes(PolicyName("default"))
			},
			expected: "default-src 'self'; require-trusted-types-for 'script'; trusted-types default",
		},
		{
			name: "policy name with special characters",
			builder: func() *CSPBuilder {
				return NewCSPBuilder().TrustedTypes(PolicyName("my-policy_v2"))
			},
			expected: "trusted-types my-policy_v2",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.builder().Build()
			if result != tc.expected {
				t.Errorf("Build() = %q, want %q", result, tc.expected)
			}
		})
	}
}

func TestTrustedTypesKeywords(t *testing.T) {

	testCases := []struct {
		source   Source
		expected bool
	}{
		{source: Script, expected: true},
		{source: AllowDuplicates, expected: true},
		{source: Wildcard, expected: false},
	}

	for _, tc := range testCases {
		t.Run(string(tc.source), func(t *testing.T) {
			if got := isKeyword(tc.source); got != tc.expected {
				t.Errorf("isKeyword(%q) = %v, want %v", tc.source, got, tc.expected)
			}
		})
	}
}

func TestCSPBuilder_Presets(t *testing.T) {
	t.Run("WithStrictPolicy", func(t *testing.T) {
		builder := NewCSPBuilder().WithStrictPolicy()
		result := builder.Build()

		expectations := []string{
			"default-src 'self'",
			"script-src 'strict-dynamic' {{REQUEST_TOKEN}}",
			"style-src 'self' {{REQUEST_TOKEN}}",
			"object-src 'none'",
			"base-uri 'self'",
			"frame-ancestors 'self'",
			"upgrade-insecure-requests",
		}

		for _, exp := range expectations {
			if !strings.Contains(result, exp) {
				t.Errorf("WithStrictPolicy() missing %q in output: %s", exp, result)
			}
		}

		if !builder.UsesRequestTokens() {
			t.Error("WithStrictPolicy() should use request tokens")
		}
	})

	t.Run("WithRelaxedPolicy", func(t *testing.T) {
		builder := NewCSPBuilder().WithRelaxedPolicy()
		result := builder.Build()

		expectations := []string{
			"default-src 'self'",
			"script-src 'self' 'unsafe-inline' 'unsafe-eval'",
			"style-src 'self' 'unsafe-inline'",
			"img-src 'self' data: https:",
			"font-src 'self' data:",
			"connect-src 'self'",
			"frame-ancestors 'self'",
		}

		for _, exp := range expectations {
			if !strings.Contains(result, exp) {
				t.Errorf("WithRelaxedPolicy() missing %q in output: %s", exp, result)
			}
		}

		if builder.UsesRequestTokens() {
			t.Error("WithRelaxedPolicy() should not use request tokens")
		}
	})

	t.Run("WithAPIPolicy", func(t *testing.T) {
		builder := NewCSPBuilder().WithAPIPolicy()
		result := builder.Build()

		expectations := []string{
			"default-src 'none'",
			"frame-ancestors 'none'",
			"base-uri 'none'",
			"form-action 'none'",
		}

		for _, exp := range expectations {
			if !strings.Contains(result, exp) {
				t.Errorf("WithAPIPolicy() missing %q in output: %s", exp, result)
			}
		}

		if builder.UsesRequestTokens() {
			t.Error("WithAPIPolicy() should not use request tokens")
		}
	})

	t.Run("presets can be extended", func(t *testing.T) {

		result := NewCSPBuilder().
			WithStrictPolicy().
			ConnectSrc(Self, Host("api.example.com")).
			Build()

		if !strings.Contains(result, "connect-src 'self' api.example.com") {
			t.Errorf("Preset should be extendable, got: %s", result)
		}
	})
}

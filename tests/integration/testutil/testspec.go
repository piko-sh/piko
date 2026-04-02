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

package testutil

import (
	"fmt"
	"os"

	"piko.sh/piko/wdk/json"
)

// TestSpec defines the settings for integration test cases.
// It supports several test types through optional fields.
type TestSpec struct {
	// EntryPoint is the file path where test execution begins.
	EntryPoint string `json:"entryPoint,omitempty"`

	// RequestURL is the URL to visit before running the test.
	RequestURL string `json:"requestURL,omitempty"`

	// ErrorContains is the expected text that must appear in the error message.
	ErrorContains string `json:"errorContains,omitempty"`

	// ModuleName is the Go module path being tested.
	ModuleName string `json:"moduleName,omitempty"`

	// Description is a simple explanation of what the test checks.
	Description string `json:"description"`

	// Transformer specifies which image transformer to use ("imaging" or "vips").
	// If empty, defaults to "imaging". Tests can specify "vips" to test libvips
	// backend.
	Transformer string `json:"transformer,omitempty"`

	// ErrorChecks contains the expected errors to check in the test.
	ErrorChecks []ErrorCheck `json:"errorChecks,omitempty"`

	// VariantURLChecks is a list of URL checks to run for each variant.
	VariantURLChecks []VariantURLCheck `json:"variantURLChecks,omitempty"`

	// AssertNodes contains checks to verify properties of AST nodes.
	AssertNodes []NodeAssertion `json:"assertNodes,omitempty"`

	// Stages lists the test stages to run in order.
	Stages []StageConfig `json:"stages,omitempty"`

	// Transformations lists the transformation checks to apply.
	Transformations []TransformationCheck `json:"transformations,omitempty"`

	// DeletionChecks contains checks that verify proper cleanup of deleted
	// resources.
	DeletionChecks []DeletionCheck `json:"deletionChecks,omitempty"`

	// DeduplicationChecks lists checks that find repeated documentation.
	DeduplicationChecks []DeduplicationCheck `json:"deduplicationChecks,omitempty"`

	// InvalidationChecks lists the checks that must fail for the test to pass.
	InvalidationChecks []InvalidationCheck `json:"invalidationChecks,omitempty"`

	// Assets specifies checks to run against found asset files.
	Assets []AssetCheck `json:"assets,omitempty"`

	// BatchChecks contains the batch check settings to run.
	BatchChecks []BatchCheck `json:"batchChecks,omitempty"`

	// ResponsiveChecks specifies the responsive viewport checks to perform.
	ResponsiveChecks []ResponsiveCheck `json:"responsiveChecks,omitempty"`

	// RenderChecks contains the checks to run on render output.
	RenderChecks []RenderCheck `json:"renderChecks,omitempty"`

	// CompressionChecks specifies checks for response compression.
	CompressionChecks []CompressionCheck `json:"compressionChecks,omitempty"`

	// MinificationChecks contains checks to verify minified asset output.
	MinificationChecks []MinificationCheck `json:"minificationChecks,omitempty"`

	// HTTPChecks lists HTTP checks to run as part of this test.
	HTTPChecks []HTTPCheck `json:"httpChecks,omitempty"`

	// Diagnostics contains the diagnostic checks to run for this test.
	Diagnostics []DiagnosticCheck `json:"diagnostics,omitempty"`

	// ExpectedStatus is the HTTP status code the test expects; 0 defaults to 200.
	ExpectedStatus int `json:"expectedStatus,omitempty"`

	// ExpectedDiagnostics is the number of diagnostic messages the test expects.
	ExpectedDiagnostics int `json:"expectedDiagnostics,omitempty"`

	// IsPage indicates whether this spec represents a full page.
	IsPage bool `json:"isPage,omitempty"`

	// ShouldError indicates whether the test expects an error to occur.
	ShouldError bool `json:"shouldError,omitempty"`

	// IsFragment indicates whether this test represents a partial code snippet.
	IsFragment bool `json:"isFragment,omitempty"`

	// VerifyBuild indicates whether to check the build after it finishes.
	VerifyBuild bool `json:"verifyBuild,omitempty"`

	// UpdateGoldens indicates whether to update golden files instead of
	// comparing them.
	UpdateGoldens bool `json:"updateGoldens,omitempty"`
}

// DiagnosticCheck defines an expected diagnostic message for test validation.
type DiagnosticCheck struct {
	// Severity is the expected severity level ("error" or "warning").
	Severity string `json:"severity"`

	// MessageContains is a substring that must appear in the message.
	MessageContains string `json:"messageContains"`

	// OnLine is the expected line number; nil means any line matches.
	OnLine *int `json:"onLine,omitempty"`

	// InFile is the file path where the diagnostic occurs; empty if not
	// file-specific.
	InFile string `json:"inFile,omitempty"`
}

// AssetCheck defines an asset to store and verify.
type AssetCheck struct {
	// SourcePath is the file path relative to the testdata source directory.
	SourcePath string `json:"sourcePath"`

	// ExpectedArtefactID is the artefact ID that should exist after storage.
	ExpectedArtefactID string `json:"expectedArtefactID"`
}

// RenderCheck defines a check that runs when a template is rendered.
type RenderCheck struct {
	// TemplateSrc is the value of the src attribute in the template.
	TemplateSrc string `json:"templateSrc"`

	// ExpectLookupID is the artefact ID that should be looked up.
	ExpectLookupID string `json:"expectLookupID"`
}

// NodeAssertion defines an assertion to run against an AST node.
type NodeAssertion struct {
	// Assert holds the assertion details to check for this node.
	Assert NodeAssertDetails `json:"assert"`

	// Select is a CSS selector that identifies the element to check.
	Select string `json:"select"`

	// Description is the human-readable explanation of the assertion.
	Description string `json:"description,omitempty"`
}

// NodeAssertDetails holds the expected values for a node assertion check.
type NodeAssertDetails struct {
	// NodeCount is the expected number of nodes.
	NodeCount *int `json:"nodeCount,omitempty"`

	// HasAttribute maps attribute names to their expected values.
	HasAttribute map[string]string `json:"hasAttribute,omitempty"`

	// ChildElementCount is the number of child elements in the node.
	ChildElementCount *int `json:"childElementCount,omitempty"`

	// TextContent is the expected text content of the node.
	TextContent string `json:"textContent,omitempty"`
}

// StageConfig holds settings for a single step in a test sequence.
type StageConfig struct {
	// Description is a short explanation of what this stage does.
	Description string `json:"description"`

	// ExpectedGolden is the path to the golden file that holds the expected
	// output.
	ExpectedGolden string `json:"expectedGolden"`

	// Stage is the processing stage number.
	Stage int `json:"stage"`

	// DelayBeforeRenderMs is the delay in milliseconds before rendering; 0 means
	// no delay.
	DelayBeforeRenderMs int `json:"delayBeforeRenderMs,omitempty"`

	// ExpectChange indicates whether this stage is expected to change files.
	ExpectChange bool `json:"expectChange,omitempty"`

	// ExpectCacheHit indicates whether the stage should use a cached result.
	ExpectCacheHit bool `json:"expectCacheHit,omitempty"`

	// ExpectCacheMiss indicates whether the stage should expect a cache miss.
	ExpectCacheMiss bool `json:"expectCacheMiss,omitempty"`
}

// TransformationCheck defines an image transformation to run and verify.
type TransformationCheck struct {
	// Params maps parameter names to their values for the transformation.
	Params map[string]string `json:"params"`

	// ProfileName is the name of the profile to use for this check.
	ProfileName string `json:"profileName"`

	// CapabilityName is the name of the capability being checked.
	CapabilityName string `json:"capabilityName"`

	// GoldenFile is the path to the expected output file for comparison.
	GoldenFile string `json:"goldenFile,omitempty"`

	// Description is the human-readable explanation of this check.
	Description string `json:"description,omitempty"`

	// DependsOn lists the names of checks that must pass before this check runs.
	DependsOn []string `json:"dependsOn,omitempty"`

	// Expected specifies what this transformation check should produce.
	Expected VariantExpectation `json:"expected"`

	// SourceAssetIndex is the position of the source asset in the asset list.
	SourceAssetIndex int `json:"sourceAssetIndex,omitempty"`
}

// VariantExpectation holds the expected values for a transformation output.
type VariantExpectation struct {
	// ContainsMetadata specifies metadata tags that must be present.
	ContainsMetadata map[string]string `json:"containsMetadata,omitempty"`

	// StartsWithDataURL checks that the output begins with a data URL prefix.
	StartsWithDataURL string `json:"startsWithDataURL,omitempty"`

	// MimeType is the expected MIME type of the output.
	MimeType string `json:"mimeType,omitempty"`

	// Status is the expected variant status (e.g. "READY", "STALE").
	Status string `json:"status,omitempty"`

	// ExactSizeBytes is the expected output size in bytes for fixed-size outputs.
	ExactSizeBytes int64 `json:"exactSizeBytes,omitempty"`

	// MaxSizeBytes is the largest expected output size in bytes.
	MaxSizeBytes int64 `json:"maxSizeBytes,omitempty"`

	// MinWidth is the smallest allowed image width in pixels.
	MinWidth int `json:"minWidth,omitempty"`

	// MaxWidth is the maximum expected image width in pixels.
	MaxWidth int `json:"maxWidth,omitempty"`

	// MinHeight is the smallest expected image height in pixels.
	MinHeight int `json:"minHeight,omitempty"`

	// MaxHeight is the maximum expected image height in pixels.
	MaxHeight int `json:"maxHeight,omitempty"`

	// ExactWidth specifies the exact image width in pixels.
	ExactWidth int `json:"exactWidth,omitempty"`

	// ExactHeight is the exact image height in pixels that is expected.
	ExactHeight int `json:"exactHeight,omitempty"`

	// MinSizeBytes is the smallest expected output size in bytes.
	MinSizeBytes int64 `json:"minSizeBytes,omitempty"`

	// OutputNotEmpty indicates whether the output must have content.
	OutputNotEmpty bool `json:"outputNotEmpty,omitempty"`
}

// DeletionCheck defines a deletion operation and the results it should produce.
type DeletionCheck struct {
	// ArtefactID is the unique identifier of the artefact to delete.
	ArtefactID string `json:"artefactID"`

	// ErrorContains is a substring that the error message should contain.
	ErrorContains string `json:"errorContains,omitempty"`

	// Description provides context about what this deletion check tests.
	Description string `json:"description,omitempty"`

	// ExpectGCHints is the number of GC hints expected after deletion.
	ExpectGCHints int `json:"expectGCHints,omitempty"`

	// ExpectBlobsRemaining is the number of blobs expected to remain after
	// deduplication.
	ExpectBlobsRemaining int `json:"expectBlobsRemaining,omitempty"`

	// ExpectError indicates whether the deletion should fail.
	ExpectError bool `json:"expectError,omitempty"`
}

// DeduplicationCheck defines a test case for checking that duplicate assets
// share the same blob storage.
type DeduplicationCheck struct {
	// Description provides context about what this check tests.
	Description string `json:"description,omitempty"`

	// Assets lists the asset paths that should have the same content.
	Assets []AssetCheck `json:"assets"`

	// ExpectRefCount is the expected reference count for the shared blob.
	ExpectRefCount int `json:"expectRefCount,omitempty"`

	// ExpectSingleBlob when true means all assets must share one blob.
	ExpectSingleBlob bool `json:"expectSingleBlob,omitempty"`
}

// InvalidationCheck describes a test case for checking how variants respond
// when an artefact is changed.
type InvalidationCheck struct {
	// ArtefactID is the identifier of the artefact to update.
	ArtefactID string `json:"artefactID"`

	// ModifySourcePath is the path to the new source file to upload.
	ModifySourcePath string `json:"modifySourcePath,omitempty"`

	// Description provides context for what this check tests.
	Description string `json:"description,omitempty"`

	// ExpectStaleVariants lists variant IDs that should become stale.
	ExpectStaleVariants []string `json:"expectStaleVariants,omitempty"`

	// ExpectReadyVariants lists variant IDs that should remain in the READY state.
	ExpectReadyVariants []string `json:"expectReadyVariants,omitempty"`
}

// ErrorCheck defines a test case for error handling behaviour.
type ErrorCheck struct {
	// Params holds the operation parameters; values may be invalid.
	Params map[string]string `json:"params,omitempty"`

	// Operation is the action to test: "store", "transform", "delete", or "get".
	Operation string `json:"operation"`

	// ArtefactID is the artefact identifier to use; it may be invalid.
	ArtefactID string `json:"artefactID,omitempty"`

	// SourcePath is the source file path; it may be empty or invalid.
	SourcePath string `json:"sourcePath,omitempty"`

	// ErrorContains is a substring that must appear in the error message.
	ErrorContains string `json:"errorContains,omitempty"`

	// Description gives details about what this error check tests.
	Description string `json:"description,omitempty"`

	// ExpectError indicates whether the operation is expected to fail.
	ExpectError bool `json:"expectError"`
}

// BatchCheck defines a test case for a batch operation.
type BatchCheck struct {
	// SearchTags specifies tag key-value pairs to match when searching.
	SearchTags map[string]string `json:"searchTags,omitempty"`

	// Operation is the batch action type: "getMultiple", "listAll", or "search".
	Operation string `json:"operation"`

	// Description provides context for what this batch check tests.
	Description string `json:"description,omitempty"`

	// ArtefactIDs lists the IDs to fetch when using getMultiple.
	ArtefactIDs []string `json:"artefactIDs,omitempty"`

	// ExpectIDs lists the artefact IDs that must appear in the results.
	ExpectIDs []string `json:"expectIDs,omitempty"`

	// ExpectCount is the expected number of results.
	ExpectCount int `json:"expectCount,omitempty"`
}

// ResponsiveCheck defines a test case for responsive image variant creation. It
// checks that multiple density and width variants can be made from one source.
type ResponsiveCheck struct {
	// Format specifies the output format used for all variants.
	Format string `json:"format,omitempty"`

	// GoldenPrefix is the prefix for golden file names (e.g., "hero" ->
	// "hero-x1.webp").
	GoldenPrefix string `json:"goldenPrefix,omitempty"`

	// Description provides context about what this responsive check tests.
	Description string `json:"description,omitempty"`

	// Densities lists the pixel density multipliers (e.g., "x1", "x2", "x3").
	Densities []string `json:"densities"`

	// ExpectedVariants lists the expected values for each generated variant.
	ExpectedVariants []ResponsiveVariantExpectation `json:"expectedVariants"`

	// SourceAssetIndex specifies which asset to use as the source; default is 0.
	SourceAssetIndex int `json:"sourceAssetIndex,omitempty"`

	// BaseWidth is the base display width before density multiplication.
	BaseWidth int `json:"baseWidth"`

	// Quality is the output quality for all variants; 0 means default quality.
	Quality int `json:"quality,omitempty"`
}

// ResponsiveVariantExpectation defines expectations for a single responsive
// variant.
type ResponsiveVariantExpectation struct {
	// Density is the density multiplier (e.g., "x1", "x2").
	Density string `json:"density"`

	// ExpectedWidth is the expected output width (baseWidth * density).
	ExpectedWidth int `json:"expectedWidth"`

	// ExpectedHeight is the expected output height. Optional; if not set, it is
	// calculated from the aspect ratio.
	ExpectedHeight int `json:"expectedHeight,omitempty"`

	// MinSizeBytes is the smallest expected output size in bytes.
	MinSizeBytes int64 `json:"minSizeBytes,omitempty"`

	// MaxSizeBytes is the largest expected output size in bytes.
	MaxSizeBytes int64 `json:"maxSizeBytes,omitempty"`
}

// VariantURLCheck defines a test for variant URL resolution. This verifies that
// generated variant URLs (e.g., for srcset) correctly resolve to the actual
// variant data in the registry.
type VariantURLCheck struct {
	// TransformParams maps parameter names to their values for URL changes.
	TransformParams map[string]string `json:"transformParams"`

	// ProfileName is the name of the profile to use for URL checks.
	ProfileName string `json:"profileName"`

	// ExpectedURLPattern is the regex pattern that the URL must match.
	ExpectedURLPattern string `json:"expectedURLPattern,omitempty"`

	// GoldenFile is the path to the golden file used for comparison.
	GoldenFile string `json:"goldenFile,omitempty"`

	// Description is a clear explanation of the URL check for users.
	Description string `json:"description,omitempty"`

	// Expected specifies what response this URL check should receive.
	Expected VariantExpectation `json:"expected"`

	// SourceAssetIndex is the position of the source asset in the assets list.
	SourceAssetIndex int `json:"sourceAssetIndex,omitempty"`
}

// CompressionCheck defines a test case for checking compression behaviour.
type CompressionCheck struct {
	// Params holds compression settings as key-value pairs (e.g., {"level": "9"}).
	Params map[string]string `json:"params,omitempty"`

	// CapabilityName is the compression type, either "gzip" or "brotli".
	CapabilityName string `json:"capabilityName"`

	// GoldenFile is the path for saving or comparing output, relative to golden/.
	GoldenFile string `json:"goldenFile,omitempty"`

	// Description explains what this compression check tests.
	Description string `json:"description,omitempty"`

	// Expected specifies what the compressed output should contain.
	Expected CompressionExpectation `json:"expected"`

	// SourceAssetIndex specifies which asset to compress; default is 0.
	SourceAssetIndex int `json:"sourceAssetIndex,omitempty"`
}

// CompressionExpectation holds the expected values for compressed output.
type CompressionExpectation struct {
	// MimeType is the expected output MIME type.
	MimeType string `json:"mimeType,omitempty"`

	// MaxRatio is the maximum compression ratio (output/input); 0.5 means 50%.
	MaxRatio float64 `json:"maxRatio,omitempty"`

	// MinSizeBytes is the smallest expected output size in bytes.
	MinSizeBytes int64 `json:"minSizeBytes,omitempty"`

	// MaxSizeBytes is the largest expected output size in bytes.
	MaxSizeBytes int64 `json:"maxSizeBytes,omitempty"`

	// OutputNotEmpty indicates whether the output must have content.
	OutputNotEmpty bool `json:"outputNotEmpty,omitempty"`
}

// MinificationCheck defines a minification capability test scenario.
type MinificationCheck struct {
	// Params holds key-value pairs for minification settings.
	Params map[string]string `json:"params,omitempty"`

	// CapabilityName is the name of the capability being checked.
	CapabilityName string `json:"capabilityName"`

	// GoldenFile is the path to the file that holds the expected output.
	GoldenFile string `json:"goldenFile,omitempty"`

	// Description provides a short explanation of what this check does.
	Description string `json:"description,omitempty"`

	// Expected specifies what the minification check should verify.
	Expected MinificationExpectation `json:"expected"`

	// SourceAssetIndex is the position of the source asset in the assets list.
	SourceAssetIndex int `json:"sourceAssetIndex,omitempty"`
}

// MinificationExpectation defines the conditions that minified output must
// meet.
type MinificationExpectation struct {
	// MimeType is the expected MIME type of the output.
	MimeType string `json:"mimeType,omitempty"`

	// ContainsString is a string that must appear in the output.
	ContainsString string `json:"containsString,omitempty"`

	// NotContainsString specifies a string that must not appear in the output.
	NotContainsString string `json:"notContainsString,omitempty"`

	// MaxRatio is the highest allowed ratio of output size to input size.
	MaxRatio float64 `json:"maxRatio,omitempty"`

	// MinSizeBytes is the smallest expected output size in bytes.
	MinSizeBytes int64 `json:"minSizeBytes,omitempty"`

	// MaxSizeBytes is the largest expected output size in bytes.
	MaxSizeBytes int64 `json:"maxSizeBytes,omitempty"`

	// OutputNotEmpty indicates whether to check that the output is not empty.
	OutputNotEmpty bool `json:"outputNotEmpty,omitempty"`
}

// HTTPCheck defines a test case for checking HTTP requests and responses.
// It is used to test PKC serving and asset endpoints.
type HTTPCheck struct {
	// RequestPath is the URL path to request (e.g.,
	// "/_piko/assets/testmodule/components/MyComponent").
	RequestPath string `json:"requestPath"`

	// ExpectedContentType is the expected Content-Type header; optional.
	ExpectedContentType string `json:"expectedContentType,omitempty"`

	// ExpectedBodyContains is a substring that the response body should contain.
	// This field is optional.
	ExpectedBodyContains string `json:"expectedBodyContains,omitempty"`

	// Description provides details about what this HTTP check tests.
	Description string `json:"description,omitempty"`

	// ExpectedStatus is the HTTP status code that the health check expects.
	ExpectedStatus int `json:"expectedStatus"`
}

// LoadTestSpec reads and parses a testspec.json file from the given path.
//
// Takes path (string) which specifies the file path to the testspec.json file.
//
// Returns *TestSpec which contains the parsed test specification.
// Returns error when the file cannot be read or the JSON is invalid.
func LoadTestSpec(path string) (*TestSpec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading testspec.json: %w", err)
	}

	var spec TestSpec
	if err := json.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("parsing testspec.json: %w", err)
	}

	return &spec, nil
}

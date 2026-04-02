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

package browser_provider_chromedp

import (
	"fmt"
	"path/filepath"

	"piko.sh/piko/wdk/json"
	"piko.sh/piko/wdk/safedisk"
)

// LoadTestSpecOption configures the behaviour of LoadTestSpec.
type LoadTestSpecOption func(*loadTestSpecConfig)

// loadTestSpecConfig holds optional settings for LoadTestSpec.
type loadTestSpecConfig struct {
	// sandboxFactory creates sandboxes for filesystem access. When nil,
	// safedisk.NewNoOpSandbox is used as a fallback.
	sandboxFactory safedisk.Factory
}

// WithTestSpecSandboxFactory sets a factory for creating sandboxes when
// loading a test spec file.
//
// Takes factory (safedisk.Factory) which creates sandboxes for filesystem
// access.
//
// Returns LoadTestSpecOption which configures the sandbox factory.
func WithTestSpecSandboxFactory(factory safedisk.Factory) LoadTestSpecOption {
	return func(c *loadTestSpecConfig) {
		c.sandboxFactory = factory
	}
}

// TestSpec defines the configuration for an E2E browser test case.
type TestSpec struct {
	// Partials defines a set of partial configurations with staged variants.
	Partials map[string]PartialConfig `json:"partials,omitempty"`

	// AdditionalGoldenFiles maps URL paths to golden file names for checking.
	AdditionalGoldenFiles map[string]string `json:"additionalGoldenFiles,omitempty"`

	// Description is a short text that explains what this test checks.
	Description string `json:"description"`

	// RequestURL is the starting URL path to visit (e.g., "/main").
	RequestURL string `json:"requestURL"`

	// ErrorContains is a substring expected in the error
	// message when ShouldError is true.
	ErrorContains string `json:"errorContains,omitempty"`

	// BrowserSteps is a list of browser actions to run in order.
	BrowserSteps []BrowserStep `json:"browserSteps,omitempty"`

	// PKCComponents lists the PKC components to check on the page.
	PKCComponents []PKCComponentConfig `json:"pkcComponents,omitempty"`

	// NetworkMocks defines mock network responses for specific endpoints.
	NetworkMocks []NetworkMock `json:"networkMocks,omitempty"`

	// ExpectedStatus is the expected HTTP status code for initial page load.
	// Defaults to 200.
	ExpectedStatus int `json:"expectedStatus,omitempty"`

	// ShouldError indicates whether loading the page should fail.
	ShouldError bool `json:"shouldError,omitempty"`

	// TLS indicates that the server under test uses TLS/HTTPS. When true,
	// the test harness uses an https:// base URL and launches Chrome with
	// --ignore-certificate-errors to accept self-signed certificates.
	TLS bool `json:"tls,omitempty"`

	// RequiresMarkdown indicates that the test needs the markdown collection
	// provider. When true, the harness configures piko.WithMarkdownParser
	// during both code generation and server execution.
	RequiresMarkdown bool `json:"requiresMarkdown,omitempty"`
}

// BrowserStep defines a single browser action to execute within a test.
type BrowserStep struct {
	// Expected is the expected value for check actions. It can be a string,
	// number, or object.
	Expected any `json:"expected,omitempty"`

	// EventDetail is the data sent with dispatchEvent.
	EventDetail map[string]any `json:"eventDetail,omitempty"`

	// Data holds query parameters for partial reload.
	Data map[string]any `json:"data,omitempty"`

	// Attribute is an alias for Name, used by the checkAttribute action.
	Attribute string `json:"attribute,omitempty"`

	// Level is the console log level for
	// checkConsoleMessage (e.g., "error", "warn", "log").
	Level string `json:"level,omitempty"`

	// Action specifies the type of browser action to perform.
	Action string `json:"action"`

	// Selector is the CSS selector or p-ref name for the target element.
	Selector string `json:"selector,omitempty"`

	// Value holds the input for fill, setValue, wait, and type actions.
	Value string `json:"value,omitempty"`

	// Key is the key or key combination for press action
	// (e.g., "Enter", "Shift+Enter", "Control+b").
	Key string `json:"key,omitempty"`

	// GoldenFile is the name of the golden file for the captureDOM action.
	GoldenFile string `json:"goldenFile,omitempty"`

	// EventName is the event name for dispatchEvent or triggerBusEvent actions.
	EventName string `json:"eventName,omitempty"`

	// Name is the attribute or property name for
	// checkAttribute or checkStyle actions.
	Name string `json:"name,omitempty"`

	// Message is the expected message text for
	// checkConsoleMessage (partial match).
	Message string `json:"message,omitempty"`

	// Contains specifies a substring to match in checkAttribute actions.
	Contains string `json:"contains,omitempty"`

	// PartialName is the partial name for the triggerPartialReload action.
	PartialName string `json:"partialName,omitempty"`

	// Files is a list of file paths for setFiles action, relative to the test
	// source directory.
	Files []string `json:"files,omitempty"`

	// Stage is the step number for the applyStage action.
	Stage int `json:"stage,omitempty"`

	// Timeout is the wait time in milliseconds; 0 means no timeout.
	Timeout int `json:"timeout,omitempty"`

	// RefreshLevel sets how much to reload: 0, 1, 2, or 3.
	RefreshLevel int `json:"refreshLevel,omitempty"`

	// Offset is the character position for the setCursor action.
	Offset int `json:"offset,omitempty"`

	// Start is the start offset for the setSelection action.
	Start int `json:"start,omitempty"`

	// End is the end offset for the setSelection action.
	End int `json:"end,omitempty"`

	// Width is the viewport width in pixels for the setViewport action.
	Width int `json:"width,omitempty"`

	// Height is the viewport height in pixels for the setViewport action.
	Height int `json:"height,omitempty"`

	// ToEnd indicates whether to collapse selection to the end (true) or
	// to the start (false).
	ToEnd bool `json:"toEnd,omitempty"`

	// ExcludeShadowRoots disables serialisation of shadow DOM content in
	// captureDOM actions, where shadow roots are otherwise included as
	// <template shadowrootmode="open"> elements and setting this to true
	// captures only the light DOM.
	ExcludeShadowRoots bool `json:"excludeShadowRoots,omitempty"`
}

// ExpectedString returns the expected value as a string.
//
// Returns string which contains the expected value formatted as text, or an
// empty string if no expected value is set.
func (s *BrowserStep) ExpectedString() string {
	if s.Expected == nil {
		return ""
	}
	if str, ok := s.Expected.(string); ok {
		return str
	}
	return fmt.Sprintf("%v", s.Expected)
}

// ExpectedInt returns the expected value as an integer.
//
// Returns int which is the expected value converted to an integer, or zero if
// the value is nil or cannot be converted.
func (s *BrowserStep) ExpectedInt() int {
	if s.Expected == nil {
		return 0
	}
	switch v := s.Expected.(type) {
	case int:
		return v
	case float64:
		return int(v)
	case string:
		var i int
		_, _ = fmt.Sscanf(v, "%d", &i)
		return i
	default:
		return 0
	}
}

// ExpectedMap returns the expected value as a map.
//
// Returns map[string]any which contains the expected value, or nil if
// Expected is nil or not a map type.
func (s *BrowserStep) ExpectedMap() map[string]any {
	if s.Expected == nil {
		return nil
	}
	if m, ok := s.Expected.(map[string]any); ok {
		return m
	}
	return nil
}

// AttributeName returns the attribute name, preferring Attribute over Name
// for clarity.
//
// Returns string which is the attribute identifier for this step.
func (s *BrowserStep) AttributeName() string {
	if s.Attribute != "" {
		return s.Attribute
	}
	return s.Name
}

// PartialConfig defines configuration for a partial with staged variants.
type PartialConfig struct {
	// Endpoint is a custom path for this partial; empty uses the default.
	Endpoint string `json:"endpoint,omitempty"`

	// Stages defines the file variants for each stage.
	Stages []PartialStage `json:"stages,omitempty"`
}

// PartialStage defines a single stage variant for a partial.
type PartialStage struct {
	// File is the file name for this stage (e.g. "counter.pk_1").
	File string `json:"file"`

	// Stage is the stage number (0, 1, 2, and so on).
	Stage int `json:"stage"`
}

// PKCComponentConfig defines a PKC component to verify on the page.
type PKCComponentConfig struct {
	// TagName is the custom element tag name (e.g. "pp-counter").
	TagName string `json:"tagName"`

	// Selector is the CSS selector that finds the component (e.g., "#my-counter").
	Selector string `json:"selector,omitempty"`
}

// NetworkMock defines a mock network response for testing.
type NetworkMock struct {
	// Path is the URL path to mock (e.g. "/partials/counter").
	Path string `json:"path"`

	// Body is the response body to return.
	Body string `json:"body,omitempty"`

	// Status is the HTTP status code to return.
	Status int `json:"status"`

	// Delay is the wait time in milliseconds before responding; 0 means no delay.
	Delay int `json:"delay,omitempty"`
}

// LoadTestSpec reads and parses a testspec.json file for E2E browser tests.
//
// Takes path (string) which specifies the file path to the testspec.json file.
// Takes opts (...LoadTestSpecOption) which provides optional configuration
// such as WithTestSpecSandboxFactory.
//
// Returns *TestSpec which contains the parsed test specification with defaults
// applied.
// Returns error when the file cannot be read or the JSON is invalid.
func LoadTestSpec(path string, opts ...LoadTestSpecOption) (*TestSpec, error) {
	var cfg loadTestSpecConfig
	for _, opt := range opts {
		opt(&cfg)
	}

	directory := filepath.Dir(path)
	filename := filepath.Base(path)

	var sandbox safedisk.Sandbox
	var err error
	if cfg.sandboxFactory != nil {
		sandbox, err = cfg.sandboxFactory.Create("browser-testspec-read", directory, safedisk.ModeReadOnly)
	} else {
		sandbox, err = safedisk.NewNoOpSandbox(directory, safedisk.ModeReadOnly)
	}
	if err != nil {
		return nil, fmt.Errorf("creating sandbox for testspec: %w", err)
	}
	defer func() { _ = sandbox.Close() }()

	data, err := sandbox.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("reading testspec.json: %w", err)
	}

	var spec TestSpec
	if err := json.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("parsing testspec.json: %w", err)
	}

	if spec.ExpectedStatus == 0 {
		spec.ExpectedStatus = DefaultExpectedStatus
	}

	return &spec, nil
}

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

package pml_dto

import (
	"fmt"
	"strings"
)

// ValidationLevel defines how strict the PikoML validator should be.
type ValidationLevel string

const (
	// ValidationStrict causes the build to fail on any PikoML error.
	ValidationStrict ValidationLevel = "strict"

	// ValidationSoft logs errors as warnings but allows the build to continue.
	ValidationSoft ValidationLevel = "soft"

	// ValidationSkip disables all PikoML validation checks.
	ValidationSkip ValidationLevel = "skip"
)

// UnmarshalYAML implements the yaml.Unmarshaler interface to allow
// case-insensitive parsing of the validation level from a YAML configuration
// file.
//
// Takes unmarshal (func(...)) which decodes the YAML value into a target.
//
// Returns error when the YAML value cannot be decoded or is not one of
// 'strict', 'soft', or 'skip'.
func (v *ValidationLevel) UnmarshalYAML(unmarshal func(any) error) error {
	var levelString string
	if err := unmarshal(&levelString); err != nil {
		return fmt.Errorf("unmarshalling validation level: %w", err)
	}

	switch strings.ToLower(levelString) {
	case "strict":
		*v = ValidationStrict
	case "soft":
		*v = ValidationSoft
	case "skip":
		*v = ValidationSkip
	default:
		return fmt.Errorf("invalid validation level: '%s', must be one of 'strict', 'soft', or 'skip'", levelString)
	}
	return nil
}

// Config holds the project-wide configuration for the PikoML engine.
// This struct is designed to be unmarshalled from a `pml` section in
// `piko.yaml`.
type Config struct {
	// OverrideAttributes sets custom default attributes for PikoML components.
	//
	// These override the component's built-in defaults. Explicit attributes on
	// a component tag override both built-in and config defaults.
	OverrideAttributes map[string]map[string]string `yaml:"overrideAttributes"`

	// CustomComponents maps custom tag names (e.g., "pml-custom-card")
	// to Go import paths for user-defined PikoML components. This is
	// for future extension and is not yet implemented in the core engine.
	CustomComponents map[string]string `yaml:"customComponents"`

	// ValidationLevel sets how strict the PikoML validator is.
	// Valid values are 'strict', 'soft', or 'skip'; default is 'soft'.
	ValidationLevel ValidationLevel `yaml:"validationLevel"`

	// Breakpoint defines the screen width at which responsive layouts
	// should apply, such as stacked columns. Default is "480px".
	Breakpoint string `yaml:"breakpoint"`

	// AssetServePath is the URL path prefix for serving assets in preview mode
	// (e.g. "/_piko/assets"). Only used when PreviewMode is true.
	AssetServePath string `yaml:"-"`

	// ClearDefaultAttributes disables the built-in default attributes
	// that each component defines, causing only OverrideAttributes from
	// config (along with inline attributes) to be used, giving full
	// control over defaults. Defaults to false.
	ClearDefaultAttributes bool `yaml:"clearDefaultAttributes"`

	// Beautify enables pretty-printing of HTML output; mutually
	// exclusive with Minify.
	Beautify bool `yaml:"beautify"`

	// Minify enables minification of the final HTML output.
	// Mutually exclusive with Beautify.
	Minify bool `yaml:"minify"`

	// PreviewMode indicates that the output is for browser preview, not for
	// sending as an email. When true, local image paths are resolved to served
	// asset URLs instead of CID references.
	PreviewMode bool `yaml:"-"`
}

// DefaultConfig returns a new Config with sensible default values.
//
// Returns *Config which contains default settings ready for use.
func DefaultConfig() *Config {
	return &Config{
		ValidationLevel:        ValidationSoft,
		Breakpoint:             "480px",
		ClearDefaultAttributes: false,
		OverrideAttributes:     make(map[string]map[string]string),
		CustomComponents:       make(map[string]string),
		Beautify:               false,
		Minify:                 false,
	}
}

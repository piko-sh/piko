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

// Package main compiles fuzz case components and generates application assets.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"piko.sh/piko"
	"piko.sh/piko/tests/fuzz/internal/runner/componentcheck"
	"piko.sh/piko/tests/fuzz/internal/shippedhost"
	"piko.sh/piko/wdk/markdown/markdown_provider_goldmark"
)

const (
	// flagCompileComponents verifies the case components and stops before generation.
	flagCompileComponents = "--compile-components"

	// flagManifest generates only the manifests.
	flagManifest = "--manifest"
)

var (
	// errUsage reports arguments the worker does not accept.
	errUsage = errors.New("usage: generate-worker [--compile-components | --manifest]")
)

// invocation is the work selected by the command arguments.
type invocation struct {
	// mode is the generation mode.
	mode string

	// componentsOnly stops after the component check.
	componentsOnly bool
}

// run compiles the selected components and generates application assets and manifests.
//
// Takes arguments ([]string) which are the command arguments after the program name.
//
// Returns error which reports invalid arguments, or a component compilation, host option
// or generation failure, or nil on success.
func run(ctx context.Context, arguments []string) error {
	selected, err := parseArguments(arguments)
	if err != nil {
		return err
	}
	for tag := range strings.SplitSeq(os.Getenv("PIKO_FUZZ_TAG"), ",") {
		if err := componentcheck.Verify(ctx, tag); err != nil {
			return fmt.Errorf("pkc-compile: %w", err)
		}
	}
	if selected.componentsOnly {
		return nil
	}
	options, err := hostOptions()
	if err != nil {
		return err
	}
	server := piko.New(options...)
	defer server.Close()
	return server.Generate(ctx, selected.mode)
}

// parseArguments selects the work from the command arguments.
//
// Takes arguments ([]string) which are the command arguments after the program name.
//
// Returns invocation which is the selected work.
// Returns error which wraps errUsage for an unknown or repeated argument.
func parseArguments(arguments []string) (invocation, error) {
	selected := invocation{mode: piko.GenerateModeAll, componentsOnly: false}
	switch {
	case len(arguments) == 0:
		return selected, nil
	case len(arguments) > 1:
		return selected, fmt.Errorf("%w: got %d arguments", errUsage, len(arguments))
	}
	switch arguments[0] {
	case flagCompileComponents:
		selected.componentsOnly = true
	case flagManifest:
		selected.mode = piko.GenerateModeManifest
	default:
		return selected, fmt.Errorf("%w: unknown argument %q", errUsage, arguments[0])
	}
	return selected, nil
}

// hostOptions builds the application options the case environment selects.
//
// Returns []piko.Option which configures the generating server.
// Returns error which reports a missing shipped component, or nil on success.
func hostOptions() ([]piko.Option, error) {
	options := []piko.Option{piko.WithCSSReset(piko.WithCSSResetComplete()), piko.WithHealthEnabled(false)}
	if os.Getenv("PIKO_FUZZ_SHIPPED") == "true" {
		shipped, err := shippedhost.Options()
		if err != nil {
			return nil, err
		}
		options = append(options, shipped...)
	}
	if os.Getenv("PIKO_FUZZ_PARTIAL_COMPONENTS") != "" {
		var website piko.WebsiteConfig
		website.I18n = piko.I18nConfig{DefaultLocale: "en", Locales: []string{"en", "fr"}, Strategy: "prefix_except_default"}
		options = append(options, piko.WithWebsiteConfig(website))
	}
	if os.Getenv("PIKO_FUZZ_COLLECTIONS") == "true" {
		options = append(options, piko.WithMarkdownParser(markdown_provider_goldmark.NewParser()))
	}
	return options, nil
}

// main compiles components and generates application assets and manifests.
func main() {
	if err := run(context.Background(), os.Args[1:]); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		if errors.Is(err, errUsage) {
			os.Exit(2)
		}
		os.Exit(1)
	}
}

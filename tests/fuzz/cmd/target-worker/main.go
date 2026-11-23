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

// Package main runs one isolated subsystem check against the saved choice recording.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime/debug"
	"strings"

	"piko.sh/fuzzkit/tape"
	"piko.sh/json"
	"piko.sh/piko/tests/fuzz/internal/casefile"
	"piko.sh/piko/tests/fuzz/internal/shimcontracts"
	"piko.sh/piko/tests/fuzz/internal/textlimit"
)

const (
	// maxDetailBytes bounds the failure detail reported to the campaign.
	maxDetailBytes = 8192

	// unclassifiedProperty identifies product failures that name no semantic property.
	unclassifiedProperty = "unclassified"

	// panicProperty identifies a check that panicked.
	panicProperty = "panic"
)

var (
	// errUnknownSubsystem reports a saved case naming an unregistered subsystem.
	errUnknownSubsystem = errors.New("unknown subsystem")

	// checks maps target names to isolated subsystem checks.
	checks = map[string]check{
		"captcha-adapters":     shimCheck("captcha-adapters", shimcontracts.RunCaptchaAdapters),
		"auth-middleware":      shimCheck("auth", shimcontracts.RunAuth),
		"csrf-contracts":       shimCheck("csrf", shimcontracts.RunCSRF),
		"rate-limit-contracts": shimCheck("rate-limit", shimcontracts.RunRateLimit),
		"captcha-contracts":    shimCheck("captcha", shimcontracts.RunCaptcha),
		"source-diagnostics":   checkDiagnosticPair,
		"actions":              checkActions,
		"annotator":            checkAnnotator,
		"assets":               checkAssets,
		"ast":                  checkAST,
		"binder":               checkBinder,
		"binder-safe":          checkBinder,
		"cache":                checkCache,
		"search":               checkSearch,
		"collections":          checkCollections,
		"documents":            checkDocuments,
		"formatter":            checkFormatter,
		"i18n":                 checkTranslations,
		"inspector":            checkInspector,
		"orchestrator":         checkTaskReference,
		"persistence":          checkPersistence,
		"registry":             checkRegistry,
		"resolver":             checkResolver,
		"routing":              checkRouting,
		"storage":              checkStorage,
		"syntax":               checkSyntax,
		"utilities":            checkUtilities,
	}
)

// propertyError identifies the invariant independently of source names, values and paths.
type propertyError struct {
	// Err holds the underlying failure.
	Err error

	// ID identifies the violated semantic property.
	ID string

	// Source identifies the checked source file.
	Source string
}

// infrastructureError marks a harness-side failure, such as a temporary directory that
// could not be created, which says nothing about Piko's behaviour.
type infrastructureError struct {
	// Err holds the underlying harness failure.
	Err error
}

// failureSubject is an error that names its violated property directly.
type failureSubject interface {
	error

	// FailureSubject returns the violated property's stable identity.
	//
	// Returns string which identifies the violated property.
	FailureSubject() string
}

// check runs one subsystem property against a saved case.
type check func(context.Context, *tape.Tape, *casefile.Case) error

// targetResult is the machine-readable failure line read by the campaign.
type targetResult struct {
	// Property identifies the violated semantic property, and is empty for infrastructure
	// failures.
	Property string `json:",omitempty"`

	// Source identifies the checked source file, when known.
	Source string

	// Detail holds the bounded failure message.
	Detail string

	// Infrastructure reports a harness-side failure rather than a Piko finding.
	Infrastructure bool `json:",omitempty"`
}

// Error formats the recorded failure.
//
// Returns string which is the formatted failure message.
func (e *propertyError) Error() string { return e.Err.Error() }

// Unwrap exposes the underlying failure.
//
// Returns error which is the underlying failure.
func (e *propertyError) Unwrap() error { return e.Err }

// Error formats the harness failure.
//
// Returns string which is the underlying failure message.
func (e *infrastructureError) Error() string { return e.Err.Error() }

// Unwrap exposes the underlying harness failure.
//
// Returns error which is the underlying failure.
func (e *infrastructureError) Unwrap() error { return e.Err }

// infrastructure marks a harness-side failure so the campaign does not record it as a
// Piko finding.
//
// Takes format (string) which is the failure-message format.
// Takes args (...any) which supplies the formatted failure values.
//
// Returns error which is the formatted non-nil harness failure.
func infrastructure(format string, args ...any) error {
	return &infrastructureError{Err: fmt.Errorf(format, args...)}
}

// property creates a failure with a stable property identity.
//
// Takes id (string) which identifies the violated semantic property.
// Takes format (string) which is the failure-message format.
// Takes args (...any) which supplies the formatted failure values.
//
// Returns error which is the formatted non-nil semantic property failure.
func property(id, format string, args ...any) error {
	return &propertyError{ID: id, Source: "", Err: fmt.Errorf(format, args...)}
}

// shimCheck adapts a byte-recording contract check to a subsystem target.
//
// Takes domain (string) which identifies the checked subsystem.
// Takes runContract (func(context.Context, []byte) error) which checks one subsystem
// against its recorded choice bytes.
//
// Returns check which runs the contract check against a saved case's recording.
func shimCheck(domain string, runContract func(context.Context, []byte) error) check {
	return func(ctx context.Context, _ *tape.Tape, savedCase *casefile.Case) error {
		err := runContract(ctx, savedCase.Data)
		if err == nil {
			return nil
		}
		return &propertyError{ID: contractSubject(domain, err), Source: "", Err: err}
	}
}

// contractSubject derives the property identity of a contract failure.
//
// A nonempty FailureSubject names the subject. An empty FailureSubject uses the subsystem
// name. For errors without FailureSubject, use the first colon-delimited word when
// present or the subsystem name otherwise.
//
// Takes domain (string) which is the fallback subsystem identity.
// Takes err (error) which is the non-nil contract failure.
//
// Returns string which identifies the violated property.
func contractSubject(domain string, err error) string {
	if subject, ok := errors.AsType[failureSubject](err); ok {
		if name := subject.FailureSubject(); name != "" {
			return name
		}
		return domain
	}
	subject, _, found := strings.Cut(err.Error(), ":")
	if !found || subject == "" || strings.ContainsAny(subject, " \n") {
		return domain
	}
	return subject
}

// run dispatches the saved case to its registered subsystem check.
//
// Takes savedCase (*casefile.Case) which holds the saved sources, choices and expected
// results.
//
// Returns error which reports an unknown target, invalid saved input or semantic property
// failure.
func run(ctx context.Context, savedCase *casefile.Case) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			_, _ = fmt.Fprintf(os.Stderr, "target-worker: panic in %s: %v\n%s", savedCase.Target, recovered, debug.Stack())
			err = &propertyError{ID: panicProperty, Source: "", Err: fmt.Errorf("panic: %v", recovered)}
		}
	}()
	subsystemCheck, ok := checks[savedCase.Target]
	if !ok {
		return infrastructure("%w %q", errUnknownSubsystem, savedCase.Target)
	}
	return subsystemCheck(ctx, tape.Replay(&tape.Recording{Data: savedCase.Data}), savedCase)
}

// resultOf converts a check failure into its bounded machine-readable result.
//
// Takes err (error) which is the non-nil check failure.
//
// Returns targetResult which carries the property identity, source and bounded detail, or
// the infrastructure flag for a harness-side failure.
func resultOf(err error) targetResult {
	result := targetResult{
		Property:       unclassifiedProperty,
		Source:         "",
		Detail:         textlimit.Truncate(err.Error(), maxDetailBytes),
		Infrastructure: false,
	}
	if failure, ok := errors.AsType[*propertyError](err); ok {
		result.Property = failure.ID
		result.Source = failure.Source
		return result
	}
	if _, ok := errors.AsType[*infrastructureError](err); ok {
		result.Property = ""
		result.Infrastructure = true
	}
	return result
}

// main runs one isolated subsystem check from saved case metadata.
func main() {
	if len(os.Args) != 2 {
		_, _ = fmt.Fprintln(os.Stderr, "usage: target-worker CASE_DIRECTORY")
		os.Exit(2)
	}
	savedCase, err := casefile.Read(os.Args[1])
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "loading target input:", err)
		os.Exit(2)
	}
	if err := run(context.Background(), &savedCase); err != nil {
		raw, marshalErr := json.Marshal(resultOf(err))
		if marshalErr != nil {
			_, _ = fmt.Fprintln(os.Stderr, "encoding target result:", marshalErr, "for failure:", err)
			os.Exit(2)
		}
		_, _ = fmt.Fprintf(os.Stderr, "PIKO_TARGET_RESULT %s\n", raw)
		os.Exit(1)
	}
}

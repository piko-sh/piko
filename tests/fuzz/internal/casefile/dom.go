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

package casefile

import (
	"errors"
	"fmt"
	"slices"
)

const (
	// maxDOMSteps bounds the operations in one DOM plan.
	maxDOMSteps = 24

	// maxDOMFields bounds the observed fields of one DOM step.
	maxDOMFields = 64

	// maxDOMIdentityBytes bounds a DOM step or field identity.
	maxDOMIdentityBytes = 128

	// maxDOMValueBytes bounds a DOM operation value or query argument.
	maxDOMValueBytes = 256

	// maxDOMSelectorBytes bounds a DOM target, host or selector.
	maxDOMSelectorBytes = 1024

	// narrowViewportWidth is the narrow viewport width in pixels.
	narrowViewportWidth = 420

	// wideViewportWidth is the wide viewport width in pixels.
	wideViewportWidth = 960
)

var (
	// domKeys lists the keyboard keys a DOM step may press.
	domKeys = []string{"Home", "End", "Tab", "Enter", "Escape"}

	// domOperations lists the browser operations a DOM step may perform.
	domOperations = []string{"inspect", "click", "focus", "type", "key", "viewport"}

	// domReads lists the DOM properties a query may read.
	domReads = []string{
		"text", "attr", "present", "value", "namespace", "tag", "parent", "identity", "focus", "selection", "open",
		"checked", "valid", "slot", "cell", "rows", "style", "form", "visible", "image", "pixel", "media",
	}
)

// DOMQuery selects a public DOM property in a document or shadow root.
type DOMQuery struct {
	// Host selects the shadow host whose root is queried, or the document when empty.
	Host string

	// Selector selects the queried element.
	Selector string

	// Read names the public DOM property to read.
	Read string

	// Argument qualifies the read, such as an attribute name.
	Argument string
}

// DOMField pairs a DOM query with its expected value.
type DOMField struct {
	// ID identifies the recorded item.
	ID string

	// Query selects the DOM property to read.
	Query DOMQuery

	// Want holds the independently expected value.
	Want []string `json:",omitempty"`
}

// DOMStep specifies a browser operation and its expected DOM fields.
type DOMStep struct {
	// ID identifies the browser operation.
	ID string

	// Op names the browser operation.
	Op string

	// Target selects the operation's target element.
	Target string

	// Host selects the shadow host containing the target, or the document when empty.
	Host string

	// Value holds the operation's input value.
	Value string

	// Fields holds the expected field values.
	Fields []DOMField
}

// DOMPlan specifies a sequence of browser operations and observations.
type DOMPlan struct {
	// Path is the page route opened before the journey.
	Path string

	// ReadyHost selects the shadow host awaited before the journey.
	ReadyHost string

	// ReadyElement selects the element awaited before the journey.
	ReadyElement string

	// Steps holds the ordered interaction steps.
	Steps []DOMStep
}

// DOMState records a step's URL and DOM query results.
type DOMState struct {
	// Values holds the DOM query results by field ID.
	Values map[string][]string

	// Step identifies the recorded step.
	Step string

	// Path is the step's browser URL.
	Path string
}

// DOMTrace records the browser's execution of a DOM plan.
type DOMTrace struct {
	// Engine names the browser engine that executed the plan.
	Engine string `json:",omitempty"`

	// Version identifies the browser version that executed the plan.
	Version string `json:",omitempty"`

	// DriverError records the browser driver's failure.
	DriverError string `json:",omitempty"`

	// States holds the ordered interaction snapshots.
	States []DOMState
}

// DriverPlan passes selectors and input only.
//
// Returns DOMPlan which holds validated browser instructions with expected values
// removed.
// Returns error which reports an invalid or unsupported saved DOM operation, or nil on
// success.
func (plan DOMPlan) DriverPlan() (DOMPlan, error) {
	if !LocalBrowserPath(plan.Path) || plan.ReadyElement == "" || len(plan.Steps) == 0 || len(plan.Steps) > maxDOMSteps {
		return DOMPlan{}, errors.New("invalid DOM scenario route, readiness or length")
	}
	plan.Steps = slices.Clone(plan.Steps)
	identities := map[string]bool{}
	for stepIndex := range plan.Steps {
		step := &plan.Steps[stepIndex]
		if err := validateDOMStep(step, identities); err != nil {
			return DOMPlan{}, err
		}
		identities[step.ID] = true
		if err := stripDOMFields(step); err != nil {
			return DOMPlan{}, err
		}
	}
	return plan, nil
}

// ViewportWidth returns a bounded desktop viewport for responsive component checks.
//
// Returns int which is the bounded desktop viewport width in pixels.
func (step DOMStep) ViewportWidth() int {
	if step.Value == "narrow" {
		return narrowViewportWidth
	}
	return wideViewportWidth
}

// validateDOMStep rejects a DOM step with an invalid identity, bounds or operation.
//
// Takes step (*DOMStep) which is the saved DOM step to validate.
// Takes identities (map[string]bool) which records the step identities already used.
//
// Returns error which reports the invalid step, or nil when the step is valid.
func validateDOMStep(step *DOMStep, identities map[string]bool) error {
	if step.ID == "" || identities[step.ID] || len(step.Fields) == 0 || len(step.Fields) > maxDOMFields {
		return fmt.Errorf("invalid DOM step %q", step.ID)
	}
	if len(step.Value) > maxDOMValueBytes || len(step.Target) > maxDOMSelectorBytes ||
		len(step.Host) > maxDOMSelectorBytes || len(step.ID) > maxDOMIdentityBytes {
		return fmt.Errorf("DOM input exceeds limits at %q", step.ID)
	}
	if step.Op == "key" && !slices.Contains(domKeys, step.Value) {
		return fmt.Errorf("unknown DOM key %q", step.Value)
	}
	if !validDOMOperation(step) {
		return fmt.Errorf("invalid DOM operation %q", step.Op)
	}
	return nil
}

// validDOMOperation reports whether a step's operation, target and value agree.
//
// Takes step (*DOMStep) which is the saved DOM step to check.
//
// Returns bool which reports whether the operation is known and complete.
func validDOMOperation(step *DOMStep) bool {
	if !slices.Contains(domOperations, step.Op) {
		return false
	}
	if step.Op != "inspect" && step.Op != "viewport" && step.Target == "" {
		return false
	}
	return step.Op != "viewport" || step.Value == "wide" || step.Value == "narrow"
}

// stripDOMFields validates a step's fields and removes their expected values.
//
// Takes step (*DOMStep) which is the step whose fields are cloned and stripped.
//
// Returns error which reports an invalid or oversized field, or nil on success.
func stripDOMFields(step *DOMStep) error {
	fields := map[string]bool{}
	step.Fields = slices.Clone(step.Fields)
	for fieldIndex := range step.Fields {
		field := &step.Fields[fieldIndex]
		if len(field.ID) > maxDOMIdentityBytes || len(field.Query.Selector) > maxDOMSelectorBytes ||
			len(field.Query.Host) > maxDOMSelectorBytes || len(field.Query.Argument) > maxDOMValueBytes {
			return fmt.Errorf("DOM query exceeds limits at %q", step.ID)
		}
		if field.ID == "" || fields[field.ID] || field.Query.Selector == "" || !slices.Contains(domReads, field.Query.Read) {
			return fmt.Errorf("invalid DOM field %s/%s", step.ID, field.ID)
		}
		fields[field.ID] = true
		field.Want = nil
	}
	return nil
}

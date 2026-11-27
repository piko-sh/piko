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

package generator_dto

import (
	"errors"
	"fmt"

	"piko.sh/piko/internal/annotator/annotator_dto"
)

// ClientScriptKind says what a client script belongs to. The zero value is not a valid
// kind, so a target that was never set is rejected rather than read as a page.
type ClientScriptKind uint8

const (
	// ClientScriptPage is the script of a page. It is scoped to the page root and lives for
	// one page visit.
	ClientScriptPage ClientScriptKind = iota + 1

	// ClientScriptPartial is the script of a partial. It is scoped to the root element of
	// each instance of the partial.
	ClientScriptPartial

	// ClientScriptModule is a plain module, such as the generated actions file. It is not
	// scoped to any element and gets no instance scaffolding.
	ClientScriptModule
)

var (
	// ErrInvalidClientScriptTarget is returned when a client script target has an unknown
	// kind, or a partial name that does not match its kind.
	ErrInvalidClientScriptTarget = errors.New("invalid client script target")

	// clientScriptKindNames maps each known kind to the name used in messages.
	clientScriptKindNames = map[ClientScriptKind]string{
		ClientScriptPage:    "page",
		ClientScriptPartial: "partial",
		ClientScriptModule:  "module",
	}
)

// ClientScriptTarget describes the component a client script belongs to. It decides how
// the generated JavaScript finds the element its script is scoped to.
type ClientScriptTarget struct {
	// PartialName is the partial_name attribute of the partial's root element. It is set for
	// partial scripts only.
	PartialName string

	// Kind says what the script belongs to.
	Kind ClientScriptKind
}

// String returns the name of the kind, or its number when the kind is not known.
//
// Returns string which names the kind.
func (k ClientScriptKind) String() string {
	if name, found := clientScriptKindNames[k]; found {
		return name
	}
	return fmt.Sprintf("ClientScriptKind(%d)", k)
}

// Validate checks that the target has a known kind, and that only a partial carries a
// partial name.
//
// Returns error which wraps ErrInvalidClientScriptTarget when the target is not valid.
func (t ClientScriptTarget) Validate() error {
	switch t.Kind {
	case ClientScriptPartial:
		if t.PartialName == "" {
			return fmt.Errorf("%w: partial script without a partial name", ErrInvalidClientScriptTarget)
		}
		return nil
	case ClientScriptPage, ClientScriptModule:
		if t.PartialName != "" {
			return fmt.Errorf("%w: %s script with partial name %q", ErrInvalidClientScriptTarget, t.Kind, t.PartialName)
		}
		return nil
	default:
		return fmt.Errorf("%w: unknown kind %s", ErrInvalidClientScriptTarget, t.Kind)
	}
}

// PageScriptTarget returns the target of a page's script.
//
// Returns ClientScriptTarget which targets a page.
func PageScriptTarget() ClientScriptTarget {
	return ClientScriptTarget{Kind: ClientScriptPage, PartialName: ""}
}

// PartialScriptTarget returns the target of a partial's script.
//
// Takes partialName (string) which is the partial_name attribute of the partial's root
// element.
//
// Returns ClientScriptTarget which targets the named partial.
func PartialScriptTarget(partialName string) ClientScriptTarget {
	return ClientScriptTarget{Kind: ClientScriptPartial, PartialName: partialName}
}

// ModuleScriptTarget returns the target of a plain module, such as the generated actions
// file.
//
// Returns ClientScriptTarget which targets a module.
func ModuleScriptTarget() ClientScriptTarget {
	return ClientScriptTarget{Kind: ClientScriptModule, PartialName: ""}
}

// ClientScriptTargetFor returns the target of a component's client script.
//
// A page, an error page, an email or a PDF owns its whole document, so its script is a
// page script. A component that is none of these is a partial, and the annotator gives
// every partial a partial name, wherever its source lives.
//
// Takes component (*annotator_dto.VirtualComponent) which is the component the script
// belongs to.
//
// Returns ClientScriptTarget which targets the component.
// Returns error which wraps ErrInvalidClientScriptTarget when the component is nil, has
// more than one document kind, is a document with a partial name, or is a partial without
// one.
func ClientScriptTargetFor(component *annotator_dto.VirtualComponent) (ClientScriptTarget, error) {
	if component == nil {
		return ClientScriptTarget{}, fmt.Errorf("%w: no component", ErrInvalidClientScriptTarget)
	}

	documentKinds := countDocumentKinds(component)
	if documentKinds > 1 {
		return ClientScriptTarget{}, fmt.Errorf("%w: component %q has %d document kinds",
			ErrInvalidClientScriptTarget, component.HashedName, documentKinds)
	}

	target := PartialScriptTarget(component.PartialName)
	if documentKinds == 1 {
		target = ClientScriptTarget{Kind: ClientScriptPage, PartialName: component.PartialName}
	}

	if err := target.Validate(); err != nil {
		return ClientScriptTarget{}, fmt.Errorf("component %q: %w", component.HashedName, err)
	}
	return target, nil
}

// countDocumentKinds counts the kinds of whole document a component is marked as. An
// error page counts as a page.
//
// Takes component (*annotator_dto.VirtualComponent) which is the component to check.
//
// Returns int which is the number of document kinds set.
func countDocumentKinds(component *annotator_dto.VirtualComponent) int {
	count := 0
	for _, isKind := range []bool{component.IsPage || component.IsErrorPage, component.IsEmail, component.IsPdf} {
		if isKind {
			count++
		}
	}
	return count
}

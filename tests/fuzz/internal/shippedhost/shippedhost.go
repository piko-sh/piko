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

// Package shippedhost selects the shipped Piko components that fuzz application hosts
// register when a case exercises the shipped component catalogue.
package shippedhost

import (
	"errors"
	"fmt"
	"slices"

	"piko.sh/piko"
	"piko.sh/piko/components"
)

var (
	// ErrMissingComponent reports a selected tag that the shipped catalogue does not define.
	ErrMissingComponent = errors.New("shipped component missing from catalogue")

	// tags lists the shipped components a fuzz host registers, in registration order.
	tags = []string{"piko-counter", "piko-card", "example-greeting", "m2-data-table-pagination", "m3e-switch", "piko-dev-widget"}
)

// Tags returns the shipped component tags a fuzz host registers, in registration order.
//
// Returns []string which is a copy of the selected tags.
func Tags() []string {
	return slices.Clone(tags)
}

// Options returns the host options that register the selected shipped components.
//
// Returns []piko.Option which registers each selected component once, in a fixed order.
// Returns error which wraps ErrMissingComponent when a selected tag is absent from the
// shipped and development catalogues, or nil on success.
func Options() ([]piko.Option, error) {
	return optionsFor(slices.Concat(components.All(), components.Dev()), tags)
}

// optionsFor builds registration options for the selected tags from a catalogue.
//
// Takes catalogue ([]piko.ComponentDefinition) which lists the available definitions.
// Takes selected ([]string) which lists the tags to register.
//
// Returns []piko.Option which registers each selected component in the selected order.
// Returns error which wraps ErrMissingComponent for every absent tag, or nil on success.
func optionsFor(catalogue []piko.ComponentDefinition, selected []string) ([]piko.Option, error) {
	options := make([]piko.Option, 0, len(selected))
	var missing []error
	for _, tag := range selected {
		index := slices.IndexFunc(catalogue, func(definition piko.ComponentDefinition) bool { return definition.TagName == tag })
		if index < 0 {
			missing = append(missing, fmt.Errorf("%w: %s", ErrMissingComponent, tag))
			continue
		}
		options = append(options, piko.WithComponents(catalogue[index]))
	}
	if err := errors.Join(missing...); err != nil {
		return nil, err
	}
	return options, nil
}

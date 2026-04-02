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

package bootstrap

// StructValidator defines the minimal interface that a struct validator must
// satisfy. It is implemented by the playground validator in the
// validation_provider_playground WDK module, but any implementation that
// provides a Struct(any) error method will work.
type StructValidator interface {
	// Struct validates a struct's exposed fields based on validation tags.
	//
	// Takes s (any) which is the struct to validate.
	//
	// Returns error when any field fails its validation constraint.
	Struct(s any) error
}

// WithValidator sets a custom validator that replaces the default one.
//
// Takes v (StructValidator) which is the custom validator to use.
//
// Returns Option which sets the container to use the custom validator.
func WithValidator(v StructValidator) Option {
	return func(c *Container) {
		c.validatorOverride = v
	}
}

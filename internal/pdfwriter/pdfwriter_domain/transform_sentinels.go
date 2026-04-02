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

package pdfwriter_domain

import "errors"

var (
	errTransformerNil = errors.New("transformer cannot be nil")

	errTransformerNameEmpty = errors.New("transformer name cannot be empty")

	errRegistryNil = errors.New("transformer registry cannot be nil")

	// ErrTooManyEncryptors is returned when more than one encryption
	// transformer is present in a chain.
	ErrTooManyEncryptors = errors.New("at most one encryption transformer is allowed per chain")

	// ErrTooManySigners is returned when more than one signing transformer
	// is present in a chain.
	ErrTooManySigners = errors.New("at most one signing transformer is allowed per chain")

	// ErrTooManyPdfALevels is returned when more than one PDF/A conversion
	// transformer is present in a chain.
	ErrTooManyPdfALevels = errors.New("at most one PDF/A transformer is allowed per chain")

	// ErrSigningBeforeEncryption is returned when a signing transformer has
	// a lower priority than an encryption transformer.
	ErrSigningBeforeEncryption = errors.New("signing transformer must have higher priority than encryption transformer")

	// ErrLinearisationWithPdfA1 is returned when linearisation and PDF/A-1b
	// are both present, since PDF/A-1 forbids linearisation hints.
	ErrLinearisationWithPdfA1 = errors.New("linearisation is not compatible with PDF/A-1b")

	// ErrTransformerAlreadyRegistered is returned when a transformer with
	// the same name has already been added to the registry.
	ErrTransformerAlreadyRegistered = errors.New("transformer already registered")

	// ErrTransformerNotFound is returned when no transformer with the
	// requested name exists in the registry.
	ErrTransformerNotFound = errors.New("transformer not found")

	// ErrInvalidRootBox is returned when the layout result's RootBox is
	// not the expected *LayoutBox type.
	ErrInvalidRootBox = errors.New("RootBox is not *LayoutBox")

	// ErrTemplatePath is returned when the builder's template path has
	// not been set before calling Do().
	ErrTemplatePath = errors.New("template path is required; call Template() before Do()")
)

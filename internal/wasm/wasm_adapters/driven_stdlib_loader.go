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

package wasm_adapters

import (
	"errors"
	"fmt"

	"piko.sh/piko/internal/inspector/inspector_dto"
	"piko.sh/piko/internal/wasm/wasm_domain"
)

var _ wasm_domain.StdlibLoaderPort = (*stdlibLoader)(nil)

// stdlibLoader loads standard library type data from an embedded FlatBuffer.
// It implements StdlibLoaderPort.
type stdlibLoader struct {
	// data holds the cached stdlib type information; nil until Load is called.
	data *inspector_dto.TypeData

	// loadFunc returns the raw stdlib type data; nil means preloaded data is used.
	loadFunc func() ([]byte, error)

	// decoder converts raw bytes into TypeData; nil causes Load to fail.
	decoder func([]byte) (*inspector_dto.TypeData, error)

	// packageList holds the standard library package import paths.
	packageList []string
}

// StdlibLoaderOption configures how the standard library loader behaves.
type StdlibLoaderOption func(*stdlibLoader)

// Load returns the pre-bundled stdlib TypeData.
//
// Returns *inspector_dto.TypeData which contains the standard library type
// information.
// Returns error when no load function is configured, no decoder is configured,
// or loading or decoding fails.
func (l *stdlibLoader) Load() (*inspector_dto.TypeData, error) {
	if l.data != nil {
		return l.data, nil
	}

	if l.loadFunc == nil {
		return nil, errors.New("no load function configured and no preloaded data")
	}

	rawData, err := l.loadFunc()
	if err != nil {
		return nil, fmt.Errorf("failed to load stdlib data: %w", err)
	}

	if l.decoder == nil {
		return nil, errors.New("no decoder configured")
	}

	data, err := l.decoder(rawData)
	if err != nil {
		return nil, fmt.Errorf("failed to decode stdlib data: %w", err)
	}

	l.data = data
	l.packageList = make([]string, 0, len(data.Packages))
	for packagePath := range data.Packages {
		l.packageList = append(l.packageList, packagePath)
	}

	return l.data, nil
}

// GetPackageList returns the list of standard library packages.
//
// Returns []string which contains the package import paths.
func (l *stdlibLoader) GetPackageList() []string {
	return l.packageList
}

// WithLoadFunc sets a custom function to load the raw stdlib data. Use it
// for testing or when using a different data source.
//
// Takes loadFunction (func() ([]byte, error)) which provides the raw stdlib data.
//
// Returns StdlibLoaderOption which sets up the loader to use the custom load
// function.
func WithLoadFunc(loadFunction func() ([]byte, error)) StdlibLoaderOption {
	return func(l *stdlibLoader) {
		l.loadFunc = loadFunction
	}
}

// WithDecoder sets a custom decoder for the standard library data.
//
// Takes decodeFunction (func(...)) which converts raw bytes into type data.
//
// Returns StdlibLoaderOption which configures the loader to use the custom
// decoder.
func WithDecoder(decodeFunction func([]byte) (*inspector_dto.TypeData, error)) StdlibLoaderOption {
	return func(l *stdlibLoader) {
		l.decoder = decodeFunction
	}
}

// NewStdlibLoader creates a new stdlib loader.
//
// Takes opts (...StdlibLoaderOption) which configures the loader behaviour.
//
// Returns wasm_domain.StdlibLoaderPort which is the configured loader ready for
// use.
func NewStdlibLoader(opts ...StdlibLoaderOption) wasm_domain.StdlibLoaderPort {
	l := &stdlibLoader{}

	for _, opt := range opts {
		opt(l)
	}

	return l
}

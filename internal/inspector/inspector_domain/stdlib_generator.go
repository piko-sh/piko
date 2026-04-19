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

package inspector_domain

// This file provides functionality for generating TypeData containing
// standard library and Piko framework types. This is used by:
// 1. The lite builder tests (generates stdlib at test suite startup)
// 2. The WASM stdlib FBS generator tool (generates embedded stdlib.fbs)

import (
	"context"
	"fmt"
	"go/token"
	"os"
	"time"

	"golang.org/x/tools/go/packages"
	"piko.sh/piko/internal/inspector/inspector_dto"
	"piko.sh/piko/internal/logger/logger_domain"
)

var (
	// StdlibPackages defines the full set of stdlib packages to include.
	// These cover the types most commonly used in Piko templates.
	StdlibPackages = []string{
		"time",
		"context",
		"errors",
		"fmt",
		"strings",
		"strconv",
		"bytes",
		"bufio",
		"net/http",
		"net/url",
		"io",
		"io/fs",
		"mime",
		"mime/multipart",
		"encoding/json",
		"encoding/xml",
		"encoding/base64",
		"encoding/hex",
		"sort",
		"slices",
		"maps",
		"container/list",
		"container/heap",
		"text/template",
		"html",
		"html/template",
		"regexp",
		"unicode",
		"unicode/utf8",
		"os",
		"path",
		"path/filepath",
		"math",
		"math/rand",
		"math/rand/v2",
		"crypto/rand",
	}

	// PikoPackages defines the Piko framework packages to include.
	PikoPackages = []string{
		"piko.sh/piko",
		"piko.sh/piko/wdk/maths",
	}
)

// GenerateStdlibTypeData creates TypeData containing standard library and Piko types
// using go/packages for loading and type-checking. The helper is called once at
// test suite startup for lite builder tests and is used by the WASM stdlib FBS
// generator tool.
//
// Returns *inspector_dto.TypeData which contains the type information for
// standard library and Piko packages in DTO format.
// Returns error when package loading or type-checking fails.
func GenerateStdlibTypeData(ctx context.Context) (*inspector_dto.TypeData, error) {
	return GenerateStdlibTypeDataWithPackages(ctx, StdlibPackages, PikoPackages)
}

// GenerateStdlibTypeDataWithPackages creates TypeData for the specified
// packages. Use it in tests that need a minimal stdlib subset.
//
// Takes stdlibPackages ([]string) which specifies standard
// library package patterns.
// Takes pikoPackages ([]string) which specifies piko
// package patterns to include.
//
// Returns *inspector_dto.TypeData which contains the serialised type data.
// Returns error when loading, serialising, or validating the packages fails.
func GenerateStdlibTypeDataWithPackages(ctx context.Context, stdlibPackages, pikoPackages []string) (*inspector_dto.TypeData, error) {
	ctx, span, l := log.Span(ctx, "GenerateStdlibTypeData",
		logger_domain.Int("stdlib_pkg_count", len(stdlibPackages)),
		logger_domain.Int("piko_pkg_count", len(pikoPackages)),
	)
	defer span.End()

	startTime := time.Now()
	l.Internal("Starting stdlib TypeData generation...")

	patterns := make([]string, 0, len(stdlibPackages)+len(pikoPackages))
	patterns = append(patterns, stdlibPackages...)
	patterns = append(patterns, pikoPackages...)

	loadedPackages, err := loadStdlibPackages(ctx, patterns)
	if err != nil {
		return nil, fmt.Errorf("failed to load stdlib packages: %w", err)
	}

	l.Internal("Loaded stdlib packages",
		logger_domain.Int("loaded_count", len(loadedPackages)),
		logger_domain.Int64("duration_ms", time.Since(startTime).Milliseconds()),
	)

	typeData, err := extractAndEncode(loadedPackages, "")
	if err != nil {
		return nil, fmt.Errorf("failed to encode stdlib packages: %w", err)
	}

	if err := validate(typeData); err != nil {
		return nil, fmt.Errorf("generated stdlib TypeData failed validation: %w", err)
	}

	l.Internal("Stdlib TypeData generation complete",
		logger_domain.Int("package_count", len(typeData.Packages)),
		logger_domain.Int64("total_duration_ms", time.Since(startTime).Milliseconds()),
	)

	return typeData, nil
}

// loadStdlibPackages loads the given packages from the Go standard library.
// Unlike the main builder which uses overlays, this loads packages straight
// from the Go installation and module cache.
//
// Takes patterns ([]string) which lists the package patterns to load.
//
// Returns []*packages.Package which holds the loaded package data.
// Returns error when packages.Load fails.
func loadStdlibPackages(ctx context.Context, patterns []string) ([]*packages.Package, error) {
	ctx, l := logger_domain.From(ctx, log)
	l.Internal("Loading stdlib packages...",
		logger_domain.Int("pattern_count", len(patterns)),
	)

	config := &packages.Config{
		Mode: packages.NeedName | packages.NeedTypes | packages.NeedDeps |
			packages.NeedTypesInfo | packages.NeedSyntax | packages.NeedImports,
		Context:    ctx,
		Logf:       nil,
		Dir:        "",
		Fset:       token.NewFileSet(),
		Env:        os.Environ(),
		BuildFlags: nil,
		Tests:      false,
		ParseFile:  nil,
		Overlay:    nil,
	}

	config.Env = append(config.Env, "GOTOOLCHAIN=auto")

	loadedPackages, err := packages.Load(config, patterns...)
	if err != nil {
		return nil, fmt.Errorf("packages.Load failed: %w", err)
	}

	var errorCount int
	packages.Visit(loadedPackages, nil, func(pkg *packages.Package) {
		for _, err := range pkg.Errors {
			l.Warn("Package loading warning",
				logger_domain.String("package", pkg.PkgPath),
				logger_domain.String("error", err.Error()),
			)
			errorCount++
		}
	})

	if errorCount > 0 {
		l.Warn("Some packages had errors during loading",
			logger_domain.Int("error_count", errorCount),
		)
	}

	return loadedPackages, nil
}

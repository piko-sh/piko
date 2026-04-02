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

package wasm_data

import (
	_ "embed"
	"errors"
	"fmt"
	"sync"

	"piko.sh/piko/internal/inspector/inspector_adapters"
	"piko.sh/piko/internal/inspector/inspector_dto"
)

var (
	// StdlibFBS contains pre-generated stdlib TypeData in FlatBuffers binary format.
	//
	// This file is generated at build time by the stdlib generator tool. If the
	// file does not exist, the embed will fail at compile time.
	//
	// FlatBuffers format is used for:
	//   - Zero-copy access to data without full deserialisation
	//   - Faster startup times compared to JSON parsing
	//   - Smaller binary size
	//
	// To generate:
	// go run ./cmd/stdlib-generator -output internal/wasm/wasm_data/stdlib.bin
	//go:embed stdlib.bin
	StdlibFBS []byte

	// DefaultStdlibPackages is the list of stdlib packages included in the bundle.
	DefaultStdlibPackages = []string{
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
	}

	// GetStdlibTypeData returns the pre-generated stdlib TypeData.
	// The data is cached after the first call.
	//
	// Returns *inspector_dto.TypeData which contains the standard library type
	// information.
	// Returns error when decoding the embedded stdlib data fails.
	GetStdlibTypeData = sync.OnceValues(func() (*inspector_dto.TypeData, error) {
		return decodeStdlib(StdlibFBS)
	})
)

// GetStdlibPackageList returns the list of stdlib packages in the bundle.
//
// Returns []string which contains the default standard library package names.
func GetStdlibPackageList() []string {
	return DefaultStdlibPackages
}

// decodeStdlib decodes the stdlib FlatBuffers data into TypeData.
//
// Takes data ([]byte) which contains the FlatBuffers-encoded stdlib type data.
//
// Returns *inspector_dto.TypeData which contains the decoded type information.
// Returns error when the data is empty or cannot be decoded.
func decodeStdlib(data []byte) (*inspector_dto.TypeData, error) {
	if len(data) == 0 {
		return nil, errors.New("stdlib data is empty (not generated?)")
	}

	typeData, err := inspector_adapters.DecodeTypeDataFromFBS(data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode stdlib FBS data: %w", err)
	}

	return typeData, nil
}

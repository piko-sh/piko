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

package inventory

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"slices"
	"strings"

	"piko.sh/piko/tests/fuzz/internal/publiccontracts"
)

var (
	// exampleRequirements maps authored examples to required tools and services.
	exampleRequirements = map[string]string{
		"002_contact_form":       "private Piko action host with form binding and server validation",
		"010_progress_tracker":   "private action host with an SSE stream and Chromium",
		"011_live_notifications": "private local event bus, SSE host and Chromium",
		"012_file_upload":        "private S3-compatible LocalStack service, bucket and matching local credentials",
		"016_cached_api":         "private action host with its configured response cache",
		"017_rate_limited_api":   "private action host with its configured rate limiter",
		"025_database_duckdb":    "local DuckDB engine, generated SQL accessors and a private database file",
	}

	// exampleProbes maps examples to source-backed public operations.
	exampleProbes = map[string]exampleProbe{
		"015_markdown_blog": {
			source:       "pages/index.pk",
			marker:       ">Posts</h1>",
			operation:    "read the Posts index and follow an authored article",
			request:      "/",
			prerequisite: "local Markdown collection content and parser",
		},
		"018_builtin_components": {
			source:       "pages/index.pk",
			marker:       ">Built-in components</h1>",
			operation:    "read the built-in component showcase index",
			request:      "/",
			prerequisite: "generated component assets",
		},
		"019_m3e_components": {
			source:       "pages/index.pk",
			marker:       `ClientRedirect: "/foundations/ripple"`,
			operation:    "follow the redirect to the ripple demonstration",
			request:      "/",
			prerequisite: "local M3E assets",
		},
		"020_m3e_recipe_app": {
			source:       "pages/index.pk",
			marker:       `href="/recipe/the-grate-burger"`,
			operation:    "open the featured recipe link",
			request:      "/",
			prerequisite: "local M3E assets and recipe collection",
		},
		"021_live_playground": {
			source:       "pages/index.pk",
			marker:       "<pp-playground>",
			operation:    "mount the playground and compile the initial preview",
			request:      "/",
			prerequisite: "local WASM compiler and playground module",
		},
		"022_database_sqlite": {
			source:       "pages/index.pk",
			marker:       `id="add-form"`,
			operation:    "create a task and observe the persisted row",
			request:      "/",
			prerequisite: "private SQLite database and generated accessors",
		},
		"023_database_mysql": {
			source:       "pages/index.pk",
			marker:       `id="create-form"`,
			operation:    "create a blog post and open its detail route",
			request:      "/",
			prerequisite: "private local MySQL service as declared in testspec",
		},
		"024_database_postgres": {
			source:       "pages/index.pk",
			marker:       ">Product Analytics Dashboard</h1>",
			operation:    "read the product dashboard and create an order",
			request:      "/",
			prerequisite: "private local PostgreSQL service as declared in testspec",
		},
		"026_email_contact": {
			source:       "pages/index.pk",
			marker:       `id="contact-form"`,
			operation:    "submit a contact message and inspect a captured delivery",
			request:      "/",
			prerequisite: "private local email capture sink",
		},
		"027_pdf_invoice": {
			source:       "pages/index.pk",
			marker:       "Invoice Generator",
			operation:    "generate an invoice and compare its rendered rows",
			request:      "/",
			prerequisite: "local PDF renderer and extractor",
		},
		"028_analytics_ecommerce": {
			source:       "pages/index.pk",
			marker:       "<pp-cart>",
			operation:    "purchase a cart item and observe the captured event",
			request:      "/",
			prerequisite: "private local analytics collector",
		},
		"030_captcha": {
			source:       "pages/index.pk",
			marker:       ">Captcha Examples</h1>",
			operation:    "open the HMAC challenge and submit its local verification form",
			request:      "/",
			prerequisite: "local HMAC challenge provider; third-party widgets require separate credentials",
		},
	}
)

// ExampleContract describes an actual authored spec or a source-backed representative
// operation.
type ExampleContract struct {
	// Name is the example directory name.
	Name string

	// Source is the file that defines the example's operation.
	Source string

	// SpecSource is the authored specification path, or empty without one.
	SpecSource string

	// Request is the request path the operation starts from.
	Request string

	// Operation describes the public-consumer recipe.
	Operation string

	// Prerequisite describes required tools or services.
	Prerequisite string `json:",omitempty"`

	// Steps holds the ordered interaction steps.
	Steps []json.RawMessage `json:",omitempty"`

	// Status is the HTTP response status.
	Status int

	// AuthoredSpec reports an authored browser specification.
	AuthoredSpec bool

	// ScheduledLocal reports a scheduled local recipe.
	ScheduledLocal bool
}

// exampleProbe selects a source-backed public operation.
type exampleProbe struct {
	// source is the example-relative file holding the operation marker.
	source string

	// marker is the text that proves the operation still exists.
	marker string

	// operation describes the public operation.
	operation string

	// request is the request path the operation starts from.
	request string

	// prerequisite describes required tools or services.
	prerequisite string
}

// exampleSpecification holds the fields read from an authored example specification.
type exampleSpecification struct {
	// Description describes the authored operation.
	Description string

	// RequestURL is the request path the operation starts from.
	RequestURL string

	// BrowserSteps holds the ordered interaction steps.
	BrowserSteps []json.RawMessage

	// ExpectedStatus is the expected HTTP response status.
	ExpectedStatus int

	// RequiresDocker reports a required local Docker service.
	RequiresDocker bool

	// TLS reports a self-signed HTTPS server.
	TLS bool
}

// ExampleContracts finds authored example specifications and source-backed recipes.
//
// Takes repo (fs.FS) which exposes the source tree to inventory.
//
// Returns []ExampleContract which lists authored contracts and source-backed recipes
// sorted by name.
// Returns error which reports authored-source or specification access and decoding
// failure, or nil on success.
func ExampleContracts(repo fs.FS) ([]ExampleContract, error) {
	directories, err := fs.ReadDir(repo, "examples/scenarios")
	if err != nil {
		return nil, fmt.Errorf("reading examples: %w", err)
	}
	var result []ExampleContract
	for _, directory := range directories {
		if !directory.IsDir() {
			continue
		}
		name := directory.Name()
		base := path.Join("examples/scenarios", name)
		authored, err := hasAuthoredFiles(repo, base)
		if err != nil {
			return nil, err
		}
		if !authored {
			continue
		}
		var row ExampleContract
		row.Name = name
		row.ScheduledLocal = slices.Contains(publiccontracts.LocalExamples, name)
		row.Source = path.Join(base, "testspec.json")
		if err := applySpecification(repo, &row); err != nil {
			return nil, err
		}
		if len(row.Steps) == 0 {
			if err := applyProbe(repo, &row, base); err != nil {
				return nil, err
			}
		}
		result = append(result, row)
	}
	slices.SortFunc(result, func(left, right ExampleContract) int { return strings.Compare(left.Name, right.Name) })
	return result, nil
}

// applySpecification fills a contract from the example's authored specification.
//
// Takes repo (fs.FS) which exposes the source tree to inventory.
// Takes row (*ExampleContract) which names the example and its specification path.
//
// Returns error which reports an unreadable or undecodable specification, or nil when it
// is applied or absent.
func applySpecification(repo fs.FS, row *ExampleContract) error {
	data, err := fs.ReadFile(repo, row.Source)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("reading %s: %w", row.Source, err)
	}
	var spec exampleSpecification
	if err := json.Unmarshal(data, &spec); err != nil {
		return fmt.Errorf("%s: %w", row.Source, err)
	}
	row.AuthoredSpec = true
	row.SpecSource = row.Source
	row.Request = spec.RequestURL
	row.Operation = spec.Description
	row.Status = spec.ExpectedStatus
	row.Steps = spec.BrowserSteps
	if len(row.Steps) > 0 {
		row.Prerequisite = "private Piko app and Chromium; see authored setup and fixtures"
	}
	if requirement, ok := exampleRequirements[row.Name]; ok {
		row.Prerequisite = requirement
	}
	if spec.TLS {
		row.Prerequisite = "private self-signed HTTPS server"
	}
	if spec.RequiresDocker {
		row.Prerequisite = "private local Docker service declared by the authored spec"
	}
	return nil
}

// applyProbe fills a contract from the example's source-backed representative operation.
//
// Takes repo (fs.FS) which exposes the source tree to inventory.
// Takes row (*ExampleContract) which names the example.
// Takes base (string) which is the example directory.
//
// Returns error which reports a missing probe, an unreadable source or a lost operation
// marker, or nil on success.
func applyProbe(repo fs.FS, row *ExampleContract, base string) error {
	probe, ok := exampleProbes[row.Name]
	if !ok {
		return fmt.Errorf("example %s needs a representative interaction or explicit prerequisite", row.Name)
	}
	source := path.Join(base, "src", probe.source)
	data, err := fs.ReadFile(repo, source)
	if err != nil {
		return fmt.Errorf("reading operation source %s: %w", source, err)
	}
	if !strings.Contains(string(data), probe.marker) {
		return fmt.Errorf("%s lost operation marker %q", source, probe.marker)
	}
	row.Source = source
	row.Request = probe.request
	row.Operation = probe.operation
	row.Prerequisite = probe.prerequisite
	return nil
}

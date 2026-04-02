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

package config

import (
	"fmt"
	"strings"
)

// ReportingEndpoint defines a reporting endpoint for the
// Reporting-Endpoints HTTP header, used by CSP report-to directives
// and other reporting APIs.
type ReportingEndpoint struct {
	// Name is the endpoint group name used by report-to directives.
	Name string `json:"name" yaml:"name"`

	// URL is the destination URL for violation reports.
	// Must be HTTPS for browsers to accept it.
	URL string `json:"url" yaml:"url"`
}

// ReportingConfig configures the HTTP Reporting-Endpoints header. This is
// separate from CSP as it is used by multiple web platform features: CSP
// report-to directive, Network Error Logging, Deprecation Reports, and Crash
// Reports.
type ReportingConfig struct {
	// Enabled controls whether the Reporting-Endpoints header is set.
	// Default: false.
	Enabled *bool `json:"enabled" yaml:"enabled" default:"false" env:"PIKO_REPORTING_ENABLED" flag:"reportingEnabled" usage:"Enable Reporting-Endpoints header."`

	// Endpoints lists the named reporting endpoints.
	// Each endpoint can be used by its name in CSP report-to directives.
	Endpoints []ReportingEndpoint `json:"endpoints" yaml:"endpoints"`
}

// BuildHeader generates the Reporting-Endpoints header value.
//
// Returns string which is the formatted header value, or empty if the config
// is disabled or has no endpoints.
//
// The output format follows RFC 8941 (Structured Field Values):
//
//	csp-violations="https://example.com/csp",
//	deprecations="https://example.com/deprecations"
func (c *ReportingConfig) BuildHeader() string {
	if c.Enabled == nil || !*c.Enabled || len(c.Endpoints) == 0 {
		return ""
	}
	var parts []string
	for _, ep := range c.Endpoints {
		if ep.Name != "" && ep.URL != "" {
			parts = append(parts, fmt.Sprintf("%s=%q", ep.Name, ep.URL))
		}
	}
	return strings.Join(parts, ", ")
}

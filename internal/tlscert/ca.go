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

package tlscert

import (
	"crypto/x509"
	"fmt"

	"piko.sh/piko/wdk/safedisk"
)

// LoadClientCAs reads a PEM file from the provided sandbox and creates an
// x509.CertPool for client certificate verification.
//
// Takes sandbox (safedisk.Sandbox) which provides sandboxed file access to
// the directory containing the CA file.
// Takes filename (string) which is the name of the PEM-encoded CA bundle
// within the sandbox.
//
// Returns *x509.CertPool which contains the parsed CA certificates.
// Returns error when the file cannot be read or contains no valid certificates.
func LoadClientCAs(sandbox safedisk.Sandbox, filename string) (*x509.CertPool, error) {
	pemData, err := sandbox.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("reading client CA file %s: %w", filename, err)
	}

	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(pemData) {
		return nil, fmt.Errorf("no valid certificates found in %s", filename)
	}
	return pool, nil
}

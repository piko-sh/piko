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

package cli

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"path/filepath"

	"google.golang.org/grpc/credentials"
	"piko.sh/piko/wdk/safedisk"
)

const (
	// caCertFilename is the expected CA certificate file name in the certs
	// directory. This file is required.
	caCertFilename = "ca.pem"

	// clientCertFilename is the expected client certificate file name in the
	// certs directory. This file is optional; when present alongside
	// clientKeyFilename, mTLS is enabled.
	clientCertFilename = "client.pem"

	// clientKeyFilename is the expected client private key file name in the
	// certs directory.
	clientKeyFilename = "client-key.pem"
)

// loadTLSCredentials loads TLS transport credentials from an opinionated
// certificate directory layout.
//
// Required file:
//   - ca.pem: CA certificate for server verification.
//
// Optional files (both must be present for mTLS):
//   - client.pem: Client certificate.
//   - client-key.pem: Client private key.
//
// Takes factory (safedisk.Factory) which creates sandboxes for filesystem
// access.
// Takes directory (string) which is the path to the certificate directory.
//
// Returns credentials.TransportCredentials which is the configured TLS
// credentials for gRPC.
// Returns error when the CA file cannot be loaded or client cert is invalid.
func loadTLSCredentials(factory safedisk.Factory, directory string) (credentials.TransportCredentials, error) {
	sandbox, err := factory.Create("tls-credentials", directory, safedisk.ModeReadOnly)
	if err != nil {
		return nil, fmt.Errorf("creating sandbox for certs directory %s: %w", directory, err)
	}
	defer func() { _ = sandbox.Close() }()

	caPEM, err := sandbox.ReadFile(caCertFilename)
	if err != nil {
		return nil, fmt.Errorf("reading CA certificate %s: %w", filepath.Join(directory, caCertFilename), err)
	}

	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(caPEM) {
		return nil, fmt.Errorf("no valid certificates found in %s", filepath.Join(directory, caCertFilename))
	}

	tlsConfig := &tls.Config{
		RootCAs:    certPool,
		MinVersion: tls.VersionTLS12,
	}

	clientCertPath := filepath.Join(directory, clientCertFilename)
	clientKeyPath := filepath.Join(directory, clientKeyFilename)

	if fileExists(clientCertPath) && fileExists(clientKeyPath) {
		clientCert, err := tls.LoadX509KeyPair(clientCertPath, clientKeyPath)
		if err != nil {
			return nil, fmt.Errorf("loading client certificate: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{clientCert}
	}

	return credentials.NewTLS(tlsConfig), nil
}

// fileExists reports whether the named file exists and is not a directory.
//
// Takes path (string) which is the file path to check.
//
// Returns bool which is true when the path exists and is a regular file.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

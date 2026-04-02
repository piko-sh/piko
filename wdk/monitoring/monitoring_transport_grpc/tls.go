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

package monitoring_transport_grpc

import (
	"context"
	"crypto/tls"
	"fmt"
	"path/filepath"

	"google.golang.org/grpc/credentials"
	"piko.sh/piko/internal/tlscert"
	"piko.sh/piko/wdk/safedisk"
)

// buildGRPCTLSCredentials creates gRPC transport credentials from the resolved
// TLS configuration. It loads the server certificate (with optional hot-reload)
// and optionally configures client CA verification for mTLS.
//
// Takes tlsValues (tlscert.TLSValues) which provides the resolved TLS settings.
//
// Returns credentials.TransportCredentials which is the configured TLS
// credentials for gRPC.
// Returns func() error which is a cleanup callback to release TLS resources.
// Returns error when certificate or CA loading fails.
func buildGRPCTLSCredentials(ctx context.Context, tlsValues tlscert.TLSValues, factory safedisk.Factory) (credentials.TransportCredentials, func() error, error) {
	loader, err := tlscert.NewCertificateLoader(ctx, tlsValues.CertFile, tlsValues.KeyFile, tlsValues.HotReload)
	if err != nil {
		return nil, nil, fmt.Errorf("loading TLS certificates: %w", err)
	}

	tlsConfig := &tls.Config{
		GetCertificate: loader.GetCertificate,
		ClientAuth:     tlsValues.ClientAuthType,
		MinVersion:     max(tlsValues.MinVersion, tls.VersionTLS12),
	}

	if tlsValues.ClientCAFile != "" {
		caDir := filepath.Dir(tlsValues.ClientCAFile)
		caName := filepath.Base(tlsValues.ClientCAFile)

		var caSandbox safedisk.Sandbox
		var err error
		if factory != nil {
			caSandbox, err = factory.Create("monitoring-grpc-client-ca", caDir, safedisk.ModeReadOnly)
		} else {
			caSandbox, err = safedisk.NewNoOpSandbox(caDir, safedisk.ModeReadOnly)
		}
		if err != nil {
			_ = loader.Close()
			return nil, nil, fmt.Errorf("creating sandbox for client CA directory: %w", err)
		}
		defer func() { _ = caSandbox.Close() }()

		pool, err := tlscert.LoadClientCAs(caSandbox, caName)
		if err != nil {
			_ = loader.Close()
			return nil, nil, fmt.Errorf("loading client CA: %w", err)
		}
		tlsConfig.ClientCAs = pool
	}

	return credentials.NewTLS(tlsConfig), loader.Close, nil
}

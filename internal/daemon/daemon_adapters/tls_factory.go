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

package daemon_adapters

import (
	"context"
	"fmt"
	"path/filepath"

	"piko.sh/piko/internal/daemon/daemon_domain"
	"piko.sh/piko/internal/tlscert"
	"piko.sh/piko/wdk/safedisk"
)

// SandboxFactory creates a sandbox for a given directory with the specified
// access mode. When nil is passed as a factory to NewServerAdapterFromTLSConfig,
// the default safedisk.NewNoOpSandbox is used.
type SandboxFactory func(name, baseDir string, mode safedisk.Mode) (safedisk.Sandbox, error)

// NewServerAdapterFromTLSConfig creates the appropriate ServerAdapter based
// on the resolved TLS configuration.
//
// Takes tlsValues (tlscert.TLSValues) which provides the resolved TLS
// settings.
// Takes purpose (serverPurpose) which identifies the server role for logging.
// Takes sandboxFactory (SandboxFactory) which creates sandboxes for loading
// client CA files. When nil, a default no-op sandbox is used.
//
// Returns daemon_domain.ServerAdapter which is the configured adapter.
// Returns func() error which is a cleanup callback that releases
// TLS resources when called.
// Returns error when certificate loading fails.
func NewServerAdapterFromTLSConfig(
	ctx context.Context,
	tlsValues tlscert.TLSValues,
	purpose serverPurpose,
	sandboxFactory SandboxFactory,
) (daemon_domain.ServerAdapter, func() error, error) {
	noopCleanup := func() error { return nil }

	if !tlsValues.Enabled() {
		adapter := &driverHTTPServerAdapter{purpose: purpose}
		return adapter, noopCleanup, nil
	}

	adapterConfig, cleanup, err := buildTLSAdapterConfig(ctx, tlsValues, sandboxFactory)
	if err != nil {
		return nil, nil, err
	}

	adapter := &driverHTTPServerAdapter{
		purpose:   purpose,
		tlsConfig: adapterConfig,
	}
	return adapter, cleanup, nil
}

// buildTLSAdapterConfig constructs the TLSAdapterConfig from resolved TLS
// values, handling certificate loading and client CA setup.
//
// Takes tlsValues (tlscert.TLSValues) which provides the resolved TLS
// settings.
// Takes sandboxFactory (SandboxFactory) which creates sandboxes for loading
// CA files.
//
// Returns *TLSAdapterConfig which is the constructed config.
// Returns func() error which is a cleanup callback that releases
// allocated resources when called.
// Returns error when certificate or CA loading fails.
func buildTLSAdapterConfig(ctx context.Context, tlsValues tlscert.TLSValues, sandboxFactory SandboxFactory) (*TLSAdapterConfig, func() error, error) {
	switch tlsValues.Mode {
	case tlscert.TLSModeCertFile:
		return buildCertFileTLSConfig(ctx, tlsValues, sandboxFactory)

	default:
		return nil, nil, fmt.Errorf("unsupported TLS mode: %d", tlsValues.Mode)
	}
}

// buildCertFileTLSConfig creates a TLS config from certificate files.
//
// Takes tlsValues (tlscert.TLSValues) which provides the resolved TLS
// settings.
// Takes sandboxFactory (SandboxFactory) which creates sandboxes for loading
// CA files. When nil, falls back to safedisk.NewNoOpSandbox.
//
// Returns *TLSAdapterConfig which is the constructed TLS config.
// Returns func() error which closes the certificate loader when
// called to release TLS resources.
// Returns error when certificate or CA loading fails.
func buildCertFileTLSConfig(ctx context.Context, tlsValues tlscert.TLSValues, sandboxFactory SandboxFactory) (*TLSAdapterConfig, func() error, error) {
	loader, err := newCertificateLoader(ctx, tlsValues.CertFile, tlsValues.KeyFile, tlsValues.HotReload)
	if err != nil {
		return nil, nil, fmt.Errorf("loading TLS certificates: %w", err)
	}

	config := &TLSAdapterConfig{
		GetCertificate: loader.GetCertificate,
		ClientAuth:     tlsValues.ClientAuthType,
		MinVersion:     tlsValues.MinVersion,
		NextProtos:     []string{"h2", "http/1.1"},
	}

	if tlsValues.ClientCAFile != "" {
		caDir := filepath.Dir(tlsValues.ClientCAFile)
		caName := filepath.Base(tlsValues.ClientCAFile)

		var caSandbox safedisk.Sandbox
		if sandboxFactory != nil {
			caSandbox, err = sandboxFactory("tls-client-ca", caDir, safedisk.ModeReadOnly)
		} else {
			caSandbox, err = safedisk.NewNoOpSandbox(caDir, safedisk.ModeReadOnly)
		}
		if err != nil {
			_ = loader.Close()
			return nil, nil, fmt.Errorf("creating sandbox for client CA directory: %w", err)
		}
		defer func() { _ = caSandbox.Close() }()

		clientCAs, err := tlscert.LoadClientCAs(caSandbox, caName)
		if err != nil {
			_ = loader.Close()
			return nil, nil, fmt.Errorf("loading client CA: %w", err)
		}
		config.ClientCAs = clientCAs
	}

	return config, loader.Close, nil
}

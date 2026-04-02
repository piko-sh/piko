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
	"fmt"

	"piko.sh/piko/cmd/piko/internal/tui/tui_adapters/provider_grpc"
	"piko.sh/piko/wdk/safedisk"
)

// connect establishes a gRPC connection using the global options.
//
// Takes factory (safedisk.Factory) which creates sandboxes for filesystem
// access.
// Takes opts (*GlobalOptions) which provides the endpoint and timeout.
//
// Returns *provider_grpc.Connection which holds gRPC service clients.
// Returns error when the connection cannot be established.
func connect(factory safedisk.Factory, opts *GlobalOptions) (*provider_grpc.Connection, error) {
	connOpts := []provider_grpc.Option{
		provider_grpc.WithDialTimeout(opts.Timeout),
	}

	if opts.CertsDir != "" {
		creds, err := loadTLSCredentials(factory, opts.CertsDir)
		if err != nil {
			return nil, fmt.Errorf("loading TLS credentials from %s: %w", opts.CertsDir, err)
		}
		connOpts = append(connOpts, provider_grpc.WithTransportCredentials(creds))
	}

	conn, err := provider_grpc.NewConnection(opts.Endpoint, connOpts...)
	if err != nil {
		return nil, fmt.Errorf("connecting to %s: %w; "+
			"ensure your Piko app is running with piko.WithMonitoring() enabled, "+
			"you can specify a different endpoint with --endpoint / -e", opts.Endpoint, err)
	}
	return conn, nil
}

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

	"piko.sh/piko/internal/tlscert"
)

// newCertificateLoader creates a CertificateLoader with the daemon's OTEL
// metrics wired as callbacks.
//
// Takes certFile (string) which is the path to the PEM-encoded certificate.
// Takes keyFile (string) which is the path to the PEM-encoded private key.
// Takes hotReload (bool) which enables file watching for automatic reloads.
//
// Returns *tlscert.CertificateLoader which is ready to provide certificates.
// Returns error when the initial certificate cannot be loaded.
func newCertificateLoader(ctx context.Context, certFile, keyFile string, hotReload bool) (*tlscert.CertificateLoader, error) {
	return tlscert.NewCertificateLoader(ctx, certFile, keyFile, hotReload,
		tlscert.WithOnReload(func() {
			tlsCertificateReloadCount.Add(context.Background(), 1)
		}),
		tlscert.WithOnError(func(_ error) {
			tlsCertificateReloadErrorCount.Add(context.Background(), 1)
		}),
	)
}

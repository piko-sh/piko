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
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/fsnotify/fsnotify"
	"piko.sh/piko/internal/logger/logger_domain"
)

const (
	// CertReloadDebounce is the debounce duration for certificate hot-reload.
	// This allows both cert and key files to settle after being written.
	CertReloadDebounce = 500 * time.Millisecond

	// certExpiryWarningDays is the number of days before expiry at which a
	// warning is logged.
	certExpiryWarningDays = 30
)

// log is the package-level logger for the tlscert package.
var log = logger_domain.GetLogger("piko/internal/tlscert")

// LoaderOption configures the CertificateLoader.
type LoaderOption func(*CertificateLoader)

// CertificateLoader manages loading and reloading TLS certificates from disk.
// It supports atomic hot-reload via fsnotify file watching and provides the
// GetCertificate callback for tls.Config.
type CertificateLoader struct {
	// cert holds the current certificate, swapped atomically on reload.
	cert atomic.Pointer[tls.Certificate]

	// watcher monitors certificate file directories for changes.
	watcher *fsnotify.Watcher

	// stopCh signals the watch loop to stop.
	stopCh chan struct{}

	// onReload is called after a successful certificate reload.
	onReload func()

	// onError is called after a failed certificate reload attempt.
	onError func(error)

	// certFile is the path to the PEM-encoded certificate.
	certFile string

	// keyFile is the path to the PEM-encoded private key.
	keyFile string
}

// NewCertificateLoader loads the initial certificate pair and optionally starts
// a file watcher for hot-reload.
//
// Takes ctx (context.Context) which carries logging context for trace/request
// ID propagation.
// Takes certFile (string) which is the path to the PEM-encoded certificate.
// Takes keyFile (string) which is the path to the PEM-encoded private key.
// Takes hotReload (bool) which enables watching for file changes.
// Takes opts (...LoaderOption) which provides optional callbacks.
//
// Returns *CertificateLoader which is ready to provide certificates.
// Returns error when the initial certificate cannot be loaded.
func NewCertificateLoader(ctx context.Context, certFile, keyFile string, hotReload bool, opts ...LoaderOption) (*CertificateLoader, error) {
	cl := &CertificateLoader{
		certFile: certFile,
		keyFile:  keyFile,
		stopCh:   make(chan struct{}),
	}

	for _, opt := range opts {
		opt(cl)
	}

	if err := cl.loadCertificate(); err != nil {
		return nil, fmt.Errorf("loading initial certificate: %w", err)
	}

	cl.checkCertificateExpiry(ctx)

	if hotReload {
		if err := cl.startWatcher(ctx); err != nil {
			return nil, fmt.Errorf("starting certificate watcher: %w", err)
		}
	}

	return cl, nil
}

// GetCertificate returns the currently loaded certificate. It implements the
// tls.Config.GetCertificate callback signature.
//
// Takes hello (*tls.ClientHelloInfo) which contains the TLS handshake info
// including SNI server name.
//
// Returns *tls.Certificate which is the current server certificate.
// Returns error when no certificate is loaded.
func (cl *CertificateLoader) GetCertificate(_ *tls.ClientHelloInfo) (*tls.Certificate, error) {
	cert := cl.cert.Load()
	if cert == nil {
		return nil, errors.New("no TLS certificate loaded")
	}
	return cert, nil
}

// Close stops the file watcher if it is running.
//
// Returns error when the watcher cannot be closed.
func (cl *CertificateLoader) Close() error {
	close(cl.stopCh)
	if cl.watcher != nil {
		return cl.watcher.Close()
	}
	return nil
}

// loadCertificate loads the certificate pair from disk and stores it
// atomically.
//
// Returns error when the X509 key pair cannot be loaded.
func (cl *CertificateLoader) loadCertificate() error {
	cert, err := tls.LoadX509KeyPair(cl.certFile, cl.keyFile)
	if err != nil {
		return fmt.Errorf("loading X509 key pair (cert=%s, key=%s): %w",
			cl.certFile, cl.keyFile, err)
	}
	cl.cert.Store(&cert)
	return nil
}

// checkCertificateExpiry logs a warning if the certificate expires soon.
//
// Takes ctx (context.Context) which carries logging context for trace/request
// ID propagation.
func (cl *CertificateLoader) checkCertificateExpiry(ctx context.Context) {
	_, l := logger_domain.From(ctx, log)
	cert := cl.cert.Load()
	if cert == nil || cert.Leaf == nil {
		if cert != nil && len(cert.Certificate) > 0 {
			parsed, err := x509.ParseCertificate(cert.Certificate[0])
			if err != nil {
				return
			}
			daysUntilExpiry := int(time.Until(parsed.NotAfter).Hours() / 24)
			if daysUntilExpiry < certExpiryWarningDays {
				l.Warn("TLS certificate expires soon",
					logger_domain.Int("days_until_expiry", daysUntilExpiry),
					logger_domain.String("not_after", parsed.NotAfter.Format(time.RFC3339)),
				)
			} else {
				l.Internal("TLS certificate loaded",
					logger_domain.String("not_after", parsed.NotAfter.Format(time.RFC3339)),
					logger_domain.String("subject", parsed.Subject.CommonName),
				)
			}
		}
	}
}

// startWatcher creates a file watcher on the directories containing the
// cert and key files and starts a background goroutine to handle reload
// events.
//
// Takes ctx (context.Context) which carries logging context for trace/request
// ID propagation.
//
// Returns error when the watcher cannot be created or a directory cannot
// be watched.
//
// Concurrency: Spawns a goroutine that runs the watch loop until Close
// is called.
func (cl *CertificateLoader) startWatcher(ctx context.Context) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("creating fsnotify watcher: %w", err)
	}
	cl.watcher = watcher

	certDir := filepath.Dir(cl.certFile)
	keyDir := filepath.Dir(cl.keyFile)

	if err := watcher.Add(certDir); err != nil {
		_ = watcher.Close()
		return fmt.Errorf("watching directory %s: %w", certDir, err)
	}
	if certDir != keyDir {
		if err := watcher.Add(keyDir); err != nil {
			_ = watcher.Close()
			return fmt.Errorf("watching directory %s: %w", keyDir, err)
		}
	}

	go cl.watchLoop(ctx)
	_, l := logger_domain.From(ctx, log)
	l.Internal("Certificate hot-reload watcher started",
		logger_domain.String("cert_file", cl.certFile),
		logger_domain.String("key_file", cl.keyFile),
	)

	return nil
}

// watchLoop handles file system events and reloads certificates with
// debouncing.
//
// Takes ctx (context.Context) which carries logging context for trace/request
// ID propagation.
func (cl *CertificateLoader) watchLoop(ctx context.Context) {
	var debounceTimer *time.Timer

	for {
		select {
		case <-cl.stopCh:
			if debounceTimer != nil {
				debounceTimer.Stop()
			}
			return

		case event, ok := <-cl.watcher.Events:
			if !ok {
				return
			}
			debounceTimer = cl.handleWatchEvent(ctx, event, debounceTimer)

		case err, ok := <-cl.watcher.Errors:
			if !ok {
				return
			}
			_, l := logger_domain.From(ctx, log)
			l.Warn("Certificate watcher error", logger_domain.Error(err))
		}
	}
}

// handleWatchEvent processes a single file system event. If the event is a
// write or create for a watched certificate file, it resets the debounce
// timer to trigger a reload.
//
// Takes ctx (context.Context) which carries logging context for trace/request
// ID propagation.
// Takes event (fsnotify.Event) which is the file system event to process.
// Takes debounceTimer (*time.Timer) which is the current debounce timer, or
// nil if none is active.
//
// Returns *time.Timer which is the updated debounce timer.
func (cl *CertificateLoader) handleWatchEvent(ctx context.Context, event fsnotify.Event, debounceTimer *time.Timer) *time.Timer {
	if event.Op&(fsnotify.Write|fsnotify.Create) == 0 {
		return debounceTimer
	}

	if !cl.isCertificateFile(event.Name) {
		return debounceTimer
	}

	if debounceTimer != nil {
		debounceTimer.Stop()
	}
	return time.AfterFunc(CertReloadDebounce, func() {
		cl.reloadCertificate(ctx)
	})
}

// isCertificateFile reports whether the given path matches the cert or key
// file by base name.
//
// Takes name (string) which is the file path to check.
//
// Returns bool which is true when the base name matches the cert or key file.
func (cl *CertificateLoader) isCertificateFile(name string) bool {
	basename := filepath.Base(name)
	return basename == filepath.Base(cl.certFile) || basename == filepath.Base(cl.keyFile)
}

// reloadCertificate attempts to reload the certificate from disk. On failure,
// the old certificate is kept.
//
// Takes ctx (context.Context) which carries logging context for trace/request
// ID propagation.
func (cl *CertificateLoader) reloadCertificate(ctx context.Context) {
	_, l := logger_domain.From(ctx, log)
	if err := cl.loadCertificate(); err != nil {
		l.Warn("Failed to reload TLS certificate, keeping current certificate",
			logger_domain.Error(err),
		)
		if cl.onError != nil {
			cl.onError(err)
		}
		return
	}

	cl.checkCertificateExpiry(ctx)
	l.Notice("TLS certificate reloaded successfully")
	if cl.onReload != nil {
		cl.onReload()
	}
}

// WithOnReload sets a callback invoked after a successful certificate
// reload.
//
// Takes callback (func()) which is invoked after each successful reload.
//
// Returns LoaderOption which configures the reload callback.
func WithOnReload(callback func()) LoaderOption {
	return func(cl *CertificateLoader) {
		cl.onReload = callback
	}
}

// WithOnError sets a callback invoked after a failed certificate reload
// attempt.
//
// Takes callback (func(error)) which is invoked with the reload error.
//
// Returns LoaderOption which configures the error callback.
func WithOnError(callback func(error)) LoaderOption {
	return func(cl *CertificateLoader) {
		cl.onError = callback
	}
}

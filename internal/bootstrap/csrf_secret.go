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

package bootstrap

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"sync"
	"time"

	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/safeconv"
	"piko.sh/piko/wdk/safedisk"
)

const (
	// csrfSecretFileName is the name of the file where the auto-generated CSRF
	// secret is stored.
	csrfSecretFileName = "piko_csrf_secret"

	// csrfSecretLength is the number of random bytes to generate for the CSRF
	// secret. 32 bytes = 256 bits of entropy, which is cryptographically secure.
	csrfSecretLength = 32

	// csrfSecretFilePermissions is the file permission mode for the CSRF secret
	// file. Owner read/write only (0o600).
	csrfSecretFilePermissions = 0o600

	// fallbackPIDShift is the bit shift applied to the process ID in fallback
	// secret generation.
	fallbackPIDShift = 16

	// fallbackRotateBits is the number of bits used for rotating the time value
	// in fallback secret generation.
	fallbackRotateBits = 63

	// fallbackGoldenRatio is the golden ratio constant used to mix entropy in
	// fallback generation. Derived from 2^32 / phi (the golden ratio).
	fallbackGoldenRatio = 0x9E3779B9
)

var (
	// csrfSecretOnce guards single resolution of the CSRF secret.
	csrfSecretOnce sync.Once

	// resolvedCSRFSecret holds the cached result of CSRF secret resolution.
	resolvedCSRFSecret []byte

	// testSandboxCreator is an optional test hook for sandbox creation that allows
	// tests to inject a MockSandbox for error path testing. When set, it is called
	// instead of createTempSandbox.
	//
	// WARNING: This is for testing only and should never be set in production.
	testSandboxCreator func() (safedisk.Sandbox, error)
)

// resolveCSRFSecret returns the CSRF secret to use for the application.
//
// It uses the following order of preference:
//  1. If configSecret is not empty (set via env var or config file), use it.
//  2. If a secret file exists in the temp folder, read and use it.
//  3. Create a new random secret, save it to the temp folder, and use it.
//
// The temp file method keeps the secret the same across restarts during local
// development. Containers get fresh secrets each time they start, since /tmp
// is usually cleared when a container stops.
//
// Takes configSecret (string) which is the value from settings, may be empty.
//
// Returns []byte which is the CSRF secret, always non-empty.
func resolveCSRFSecret(configSecret string) []byte {
	csrfSecretOnce.Do(func() {
		resolvedCSRFSecret = doResolveCSRFSecret(configSecret)
	})

	return resolvedCSRFSecret
}

// doResolveCSRFSecret performs the actual secret resolution logic.
// This is separate from resolveCSRFSecret to allow testing without sync.Once.
//
// When configSecret is not empty, it is used directly. Otherwise, the function
// tries to read a saved secret from a temporary file. If that fails, it creates
// a new secret and tries to save it for future use.
//
// Takes configSecret (string) which specifies a secret to use directly.
//
// Returns []byte which is the resolved or newly created CSRF secret.
func doResolveCSRFSecret(configSecret string) []byte {
	_, l := logger_domain.From(context.Background(), log)

	if configSecret != "" {
		l.Notice("Using CSRF secret from configuration")

		return []byte(configSecret)
	}

	var sandbox safedisk.Sandbox
	var err error
	if testSandboxCreator != nil {
		sandbox, err = testSandboxCreator()
	} else {
		sandbox, err = createTempSandbox()
	}
	if err != nil {
		l.Warn("Failed to create temp sandbox, generating ephemeral CSRF secret",
			logger_domain.Error(err),
		)

		return generateCSRFSecret()
	}
	defer func() { _ = sandbox.Close() }()

	secret, err := readCSRFSecretFromSandbox(sandbox)
	if err == nil {
		l.Notice("Using persisted CSRF secret from temp file",
			logger_domain.String("path", sandbox.Root()),
		)

		return secret
	}

	secret = generateCSRFSecret()

	err = writeCSRFSecretToSandbox(sandbox, secret)
	if err != nil {
		l.Warn("Generated ephemeral CSRF secret (failed to persist to temp file)",
			logger_domain.Error(err),
		)
	} else {
		l.Notice("Generated new CSRF secret and persisted to temp file",
			logger_domain.String("path", sandbox.Root()),
		)
	}

	return secret
}

// createTempSandbox creates a sandboxed file system for the system temp folder.
// This is used for reading and writing the CSRF secret file safely.
//
// Returns safedisk.Sandbox which provides safe access to the temp folder.
// Returns error when the sandbox factory cannot be created or started.
func createTempSandbox() (safedisk.Sandbox, error) {
	return createSystemTempSandbox("csrf-secret")
}

// getCSRFSecretPath returns the path to the CSRF secret file.
//
// Returns string which is the file path where the CSRF secret is stored.
func getCSRFSecretPath() string {
	return csrfSecretFileName
}

// readCSRFSecretFromSandbox reads and decodes a hex-encoded CSRF secret from
// the sandbox.
//
// Takes sandbox (safedisk.Sandbox) which provides sandboxed file access.
//
// Returns []byte which is the decoded secret.
// Returns error when the file cannot be read, is empty, or contains invalid
// hex data.
func readCSRFSecretFromSandbox(sandbox safedisk.Sandbox) ([]byte, error) {
	data, err := sandbox.ReadFile(csrfSecretFileName)
	if err != nil {
		return nil, fmt.Errorf("reading CSRF secret file: %w", err)
	}

	if len(data) == 0 {
		return nil, os.ErrNotExist
	}

	secret := make([]byte, len(data)/2)

	_, err = hex.Decode(secret, data)
	if err != nil {
		return nil, fmt.Errorf("decoding CSRF secret hex data: %w", err)
	}

	return secret, nil
}

// writeCSRFSecretToSandbox writes a CSRF secret to the sandbox as a hex-encoded
// file.
//
// Takes sandbox (safedisk.Sandbox) which provides sandboxed file access.
// Takes secret ([]byte) which contains the raw secret bytes to write.
//
// Returns error when the file cannot be written.
func writeCSRFSecretToSandbox(sandbox safedisk.Sandbox, secret []byte) error {
	encoded := hex.AppendEncode(nil, secret)

	return sandbox.WriteFile(csrfSecretFileName, encoded, csrfSecretFilePermissions)
}

// generateCSRFSecret creates a new random secret using secure random bytes.
//
// When the secure random source fails, falls back to a less secure method.
//
// Returns []byte which is the generated secret.
func generateCSRFSecret() []byte {
	secret := make([]byte, csrfSecretLength)

	_, err := rand.Read(secret)
	if err != nil {
		_, l := logger_domain.From(context.Background(), log)
		l.Error("crypto/rand failed, using fallback", logger_domain.Error(err))

		return generateFallbackSecret()
	}

	return secret
}

// generateFallbackSecret creates a secret using a less secure method when
// crypto/rand fails. This should never happen on modern systems.
//
// Uses time-based entropy mixed with process ID. While not safe for
// cryptographic use, it provides enough randomness across different runs
// and processes.
//
// Returns []byte which contains the generated fallback secret.
func generateFallbackSecret() []byte {
	now := safeconv.Int64ToUint64(time.Now().UnixNano())
	pid := safeconv.IntToUint64(os.Getpid())

	fallback := make([]byte, csrfSecretLength)
	for i := range fallback {
		mixed := now ^ (pid << fallbackPIDShift) ^ safeconv.IntToUint64(i*fallbackGoldenRatio)
		fallback[i] = byte(mixed >> ((i % 8) * 8)) //nolint:gosec // intentional byte extraction
		now = (now >> 1) | (now << fallbackRotateBits)
	}

	return fallback
}

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

	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/safedisk"
)

const (
	// captchaSecretFileName is the name of the file where the auto-generated
	// captcha HMAC secret is stored.
	captchaSecretFileName = "piko_captcha_secret"

	// captchaSecretLength is the number of random bytes for the HMAC secret.
	captchaSecretLength = 32

	// captchaSecretFilePermissions is the file permission mode for the captcha
	// secret file. Owner read/write only (0o600).
	captchaSecretFilePermissions = 0o600
)

var (
	// captchaSecretOnce guards single resolution of the captcha secret.
	captchaSecretOnce sync.Once

	// resolvedCaptchaSecret holds the cached result of captcha secret
	// resolution.
	resolvedCaptchaSecret []byte

	// testCaptchaSandboxCreator is an optional test hook for sandbox creation
	// that allows tests to inject a MockSandbox for error path testing. When
	// set, it is called instead of createCaptchaTempSandbox.
	//
	// WARNING: This is for testing only and should never be set in production.
	testCaptchaSandboxCreator func() (safedisk.Sandbox, error)
)

// resolveCaptchaSecret returns the captcha HMAC secret to use.
//
// It uses the following order of preference:
//  1. If configSecret is not empty (set via env var or config file), use it.
//  2. If a secret file exists in the temp folder, read and use it.
//  3. Create a new random secret, save it to the temp folder, and use it.
//
// The temp file method keeps the secret the same across restarts during local
// development, so challenge tokens remain valid. Containers get fresh secrets
// each time they start, since /tmp is usually cleared when a container stops.
//
// Takes configSecret (string) which is the secret from configuration, or empty
// to use auto-resolution.
//
// Returns []byte which is the resolved HMAC secret.
func resolveCaptchaSecret(configSecret string) []byte {
	captchaSecretOnce.Do(func() {
		resolvedCaptchaSecret = doResolveCaptchaSecret(configSecret)
	})

	return resolvedCaptchaSecret
}

// doResolveCaptchaSecret performs the actual secret resolution logic.
//
// Takes configSecret (string) which is the secret from configuration, or empty
// to use auto-resolution.
//
// Returns []byte which is the resolved secret.
func doResolveCaptchaSecret(configSecret string) []byte {
	_, l := logger_domain.From(context.Background(), log)

	if configSecret != "" {
		l.Notice("Using captcha secret from configuration")
		return []byte(configSecret)
	}

	var sandbox safedisk.Sandbox
	var err error
	if testCaptchaSandboxCreator != nil {
		sandbox, err = testCaptchaSandboxCreator()
	} else {
		sandbox, err = createCaptchaTempSandbox()
	}
	if err != nil {
		l.Warn("Failed to create temp sandbox for captcha secret; generating ephemeral secret. "+
			"Tokens will be invalidated on restart. Set security.captchaSecretKey for persistence.",
			logger_domain.Error(err))
		return generateCaptchaSecret()
	}
	defer func() { _ = sandbox.Close() }()

	secret, err := readCaptchaSecretFromSandbox(sandbox)
	if err == nil {
		l.Notice("Using persisted captcha secret from temp file",
			logger_domain.String("path", sandbox.Root()))
		return secret
	}

	secret = generateCaptchaSecret()

	err = writeCaptchaSecretToSandbox(sandbox, secret)
	if err != nil {
		l.Warn("Generated ephemeral captcha secret (failed to persist to temp file). "+
			"Tokens will be invalidated on restart.",
			logger_domain.Error(err))
	} else {
		l.Notice("Generated new captcha secret and persisted to temp file",
			logger_domain.String("path", sandbox.Root()))
	}

	return secret
}

// createCaptchaTempSandbox creates a sandboxed file system for the system temp
// folder.
//
// Returns safedisk.Sandbox which provides sandboxed access to the temp folder.
// Returns error when the sandbox cannot be created.
func createCaptchaTempSandbox() (safedisk.Sandbox, error) {
	return createSystemTempSandbox("captcha-secret")
}

// readCaptchaSecretFromSandbox reads and decodes a hex-encoded captcha secret
// from the sandbox.
//
// Takes sandbox (safedisk.Sandbox) which provides access to the secret file.
//
// Returns []byte which is the decoded secret.
// Returns error when the file cannot be read or decoded.
func readCaptchaSecretFromSandbox(sandbox safedisk.Sandbox) ([]byte, error) {
	data, err := sandbox.ReadFile(captchaSecretFileName)
	if err != nil {
		return nil, fmt.Errorf("reading captcha secret file: %w", err)
	}

	if len(data) == 0 {
		return nil, os.ErrNotExist
	}

	secret := make([]byte, len(data)/2)

	_, err = hex.Decode(secret, data)
	if err != nil {
		return nil, fmt.Errorf("decoding captcha secret hex data: %w", err)
	}

	if len(secret) != captchaSecretLength {
		return nil, fmt.Errorf("captcha secret has invalid length %d, expected %d", len(secret), captchaSecretLength)
	}

	return secret, nil
}

// writeCaptchaSecretToSandbox writes a captcha secret to the sandbox as a
// hex-encoded file.
//
// Takes sandbox (safedisk.Sandbox) which provides access to the temp folder.
// Takes secret ([]byte) which is the secret to persist.
//
// Returns error when the file cannot be written.
func writeCaptchaSecretToSandbox(sandbox safedisk.Sandbox, secret []byte) error {
	encoded := hex.AppendEncode(nil, secret)

	return sandbox.WriteFile(captchaSecretFileName, encoded, captchaSecretFilePermissions)
}

// generateCaptchaSecret creates a new random secret using secure random bytes.
//
// Returns []byte which is the generated secret.
func generateCaptchaSecret() []byte {
	secret := make([]byte, captchaSecretLength)
	if _, err := rand.Read(secret); err != nil {
		_, l := logger_domain.From(context.Background(), log)
		l.Error("crypto/rand failed for captcha secret", logger_domain.Error(err))
		return generateFallbackSecret()
	}

	return secret
}

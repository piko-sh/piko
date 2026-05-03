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

package storage_domain

import (
	"encoding/base64"
	"errors"
	"strings"
	"testing"
	"time"
)

func testSecret() []byte {
	return []byte("test-secret-key-32-bytes-long!!!")
}

func shortSecret() []byte {
	return []byte("short")
}

func validTokenData() PresignTokenData {
	return PresignTokenData{
		TempKey:     "tmp/test-file.png",
		Repository:  "media",
		ContentType: "image/png",
		MaxSize:     1024 * 1024,
		ExpiresAt:   time.Now().Add(15 * time.Minute).Unix(),
		RID:         "test-rid-123456789",
	}
}

func expiredTokenData() PresignTokenData {
	data := validTokenData()
	data.ExpiresAt = time.Now().Add(-1 * time.Hour).Unix()
	return data
}

func TestGeneratePresignToken(t *testing.T) {
	tests := []struct {
		wantErr   error
		checkFunc func(t *testing.T, tok string)
		name      string
		secret    []byte
		data      PresignTokenData
	}{
		{
			name:   "valid token generation",
			secret: testSecret(),
			data:   validTokenData(),
			checkFunc: func(t *testing.T, tok string) {

				if !strings.Contains(tok, presignTokenDelimiter) {
					t.Error("token should contain delimiter")
				}

				parts := strings.SplitN(tok, presignTokenDelimiter, 2)
				if len(parts) != 2 {
					t.Errorf("expected 2 parts, got %d", len(parts))
				}

				_, err := base64.RawURLEncoding.DecodeString(parts[0])
				if err != nil {
					t.Errorf("payload should be valid base64url: %v", err)
				}

				_, err = base64.RawURLEncoding.DecodeString(parts[1])
				if err != nil {
					t.Errorf("signature should be valid base64url: %v", err)
				}
			},
		},
		{
			name:    "short secret returns error",
			secret:  shortSecret(),
			data:    validTokenData(),
			wantErr: ErrPresignSecretTooShort,
		},
		{
			name:    "empty secret returns error",
			secret:  []byte{},
			data:    validTokenData(),
			wantErr: ErrPresignSecretTooShort,
		},
		{
			name:    "nil secret returns error",
			secret:  nil,
			data:    validTokenData(),
			wantErr: ErrPresignSecretTooShort,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tok, err := GeneratePresignToken(tc.secret, tc.data)

			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Errorf("expected error %v, got %v", tc.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if tc.checkFunc != nil {
				tc.checkFunc(t, tok)
			}
		})
	}
}

func TestParseAndVerifyPresignToken(t *testing.T) {
	secret := testSecret()

	tests := []struct {
		wantErr   error
		setupFunc func() string
		checkFunc func(t *testing.T, data *PresignTokenData)
		name      string
		secret    []byte
	}{
		{
			name:   "valid token verification",
			secret: secret,
			setupFunc: func() string {
				tok, _ := GeneratePresignToken(secret, validTokenData())
				return tok
			},
			checkFunc: func(t *testing.T, data *PresignTokenData) {
				if data.TempKey != "tmp/test-file.png" {
					t.Errorf("expected TempKey 'tmp/test-file.png', got %q", data.TempKey)
				}
				if data.Repository != "media" {
					t.Errorf("expected Repository 'media', got %q", data.Repository)
				}
				if data.ContentType != "image/png" {
					t.Errorf("expected ContentType 'image/png', got %q", data.ContentType)
				}
				if data.MaxSize != 1024*1024 {
					t.Errorf("expected MaxSize 1048576, got %d", data.MaxSize)
				}
			},
		},
		{
			name:   "expired token returns error",
			secret: secret,
			setupFunc: func() string {
				tok, _ := GeneratePresignToken(secret, expiredTokenData())
				return tok
			},
			wantErr: ErrPresignTokenExpired,
		},
		{
			name:   "invalid token format returns error",
			secret: secret,
			setupFunc: func() string {
				return "invalid-token-without-delimiter"
			},
			wantErr: ErrPresignTokenInvalid,
		},
		{
			name:   "tampered signature returns error",
			secret: secret,
			setupFunc: func() string {
				tok, _ := GeneratePresignToken(secret, validTokenData())
				parts := strings.SplitN(tok, presignTokenDelimiter, 2)

				return parts[0] + presignTokenDelimiter + "tampered-signature"
			},
			wantErr: ErrPresignTokenSignature,
		},
		{
			name:   "tampered payload returns error",
			secret: secret,
			setupFunc: func() string {
				tok, _ := GeneratePresignToken(secret, validTokenData())
				parts := strings.SplitN(tok, presignTokenDelimiter, 2)

				return "dGFtcGVyZWQ" + presignTokenDelimiter + parts[1]
			},
			wantErr: ErrPresignTokenSignature,
		},
		{
			name:   "wrong secret returns error",
			secret: []byte("different-secret-key-32-bytes!!!"),
			setupFunc: func() string {
				tok, _ := GeneratePresignToken(secret, validTokenData())
				return tok
			},
			wantErr: ErrPresignTokenSignature,
		},
		{
			name:   "short secret returns error",
			secret: shortSecret(),
			setupFunc: func() string {
				tok, _ := GeneratePresignToken(secret, validTokenData())
				return tok
			},
			wantErr: ErrPresignSecretTooShort,
		},
		{
			name:   "invalid base64 payload returns error",
			secret: secret,
			setupFunc: func() string {
				tok, _ := GeneratePresignToken(secret, validTokenData())
				parts := strings.SplitN(tok, presignTokenDelimiter, 2)

				return "!!!invalid-base64!!!" + presignTokenDelimiter + parts[1]
			},
			wantErr: ErrPresignTokenSignature,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tok := tc.setupFunc()
			data, err := ParseAndVerifyPresignToken(tc.secret, tok)

			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Errorf("expected error %v, got %v", tc.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if tc.checkFunc != nil {
				tc.checkFunc(t, data)
			}
		})
	}
}

func TestPresignTokenData_IsExpired(t *testing.T) {
	tests := []struct {
		name      string
		expiresAt int64
		want      bool
	}{
		{
			name:      "future expiry is not expired",
			expiresAt: time.Now().Add(1 * time.Hour).Unix(),
			want:      false,
		},
		{
			name:      "past expiry is expired",
			expiresAt: time.Now().Add(-1 * time.Hour).Unix(),
			want:      true,
		},
		{
			name:      "just expired is expired",
			expiresAt: time.Now().Add(-1 * time.Second).Unix(),
			want:      true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			data := PresignTokenData{
				ExpiresAt: tc.expiresAt,
			}

			if got := data.IsExpired(); got != tc.want {
				t.Errorf("IsExpired() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestGeneratePresignRID(t *testing.T) {
	t.Run("generates unique random identifiers", func(t *testing.T) {
		seen := make(map[string]bool)
		iterations := 100

		for range iterations {
			rid, err := GeneratePresignRID()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if seen[rid] {
				t.Errorf("duplicate random identifier generated: %s", rid)
			}
			seen[rid] = true
		}
	})

	t.Run("generates valid base64url", func(t *testing.T) {
		rid, err := GeneratePresignRID()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		decoded, err := base64.RawURLEncoding.DecodeString(rid)
		if err != nil {
			t.Errorf("random identifier should be valid base64url: %v", err)
		}

		if len(decoded) != presignRIDLengthBytes {
			t.Errorf("expected %d bytes, got %d", presignRIDLengthBytes, len(decoded))
		}
	})
}

func TestParseAndVerifyPresignTokenRejectsOversizeInput(t *testing.T) {
	secret := testSecret()

	huge := strings.Repeat("A", presignMaxTokenLength+1)

	_, err := ParseAndVerifyPresignToken(secret, huge)
	if err == nil {
		t.Fatalf("expected error for oversize token, got nil")
	}
	if !errors.Is(err, ErrPresignTokenTooLarge) {
		t.Errorf("expected ErrPresignTokenTooLarge, got %v", err)
	}

	_, err = ParseAndVerifyPresignDownloadToken(secret, huge)
	if err == nil {
		t.Fatalf("expected error for oversize download token, got nil")
	}
	if !errors.Is(err, ErrPresignTokenTooLarge) {
		t.Errorf("expected ErrPresignTokenTooLarge for download, got %v", err)
	}

	tok, err := GeneratePresignToken(secret, validTokenData())
	if err != nil {
		t.Fatalf("setup: failed to generate valid token: %v", err)
	}
	if len(tok) > presignMaxTokenLength {
		t.Fatalf("legitimate token exceeded cap: %d > %d", len(tok), presignMaxTokenLength)
	}
	if _, err := ParseAndVerifyPresignToken(secret, tok); err != nil {
		t.Errorf("legitimate token rejected: %v", err)
	}
}

func TestTokenRoundTrip(t *testing.T) {
	secret := testSecret()
	originalData := validTokenData()

	tok, err := GeneratePresignToken(secret, originalData)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	parsedData, err := ParseAndVerifyPresignToken(secret, tok)
	if err != nil {
		t.Fatalf("failed to parse token: %v", err)
	}

	if parsedData.TempKey != originalData.TempKey {
		t.Errorf("TempKey mismatch: got %q, want %q", parsedData.TempKey, originalData.TempKey)
	}
	if parsedData.Repository != originalData.Repository {
		t.Errorf("Repository mismatch: got %q, want %q", parsedData.Repository, originalData.Repository)
	}
	if parsedData.ContentType != originalData.ContentType {
		t.Errorf("ContentType mismatch: got %q, want %q", parsedData.ContentType, originalData.ContentType)
	}
	if parsedData.MaxSize != originalData.MaxSize {
		t.Errorf("MaxSize mismatch: got %d, want %d", parsedData.MaxSize, originalData.MaxSize)
	}
	if parsedData.ExpiresAt != originalData.ExpiresAt {
		t.Errorf("ExpiresAt mismatch: got %d, want %d", parsedData.ExpiresAt, originalData.ExpiresAt)
	}
	if parsedData.RID != originalData.RID {
		t.Errorf("RID mismatch: got %q, want %q", parsedData.RID, originalData.RID)
	}
}

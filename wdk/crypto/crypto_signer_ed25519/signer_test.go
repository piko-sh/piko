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

package crypto_signer_ed25519

import (
	"errors"
	"testing"
)

func TestGenerateSignVerifyRoundTrip(t *testing.T) {
	t.Parallel()
	signer, verifier, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair error: %v", err)
	}
	message := []byte("hello world")
	signature := signer.Sign(message)
	if err := verifier.Verify(message, signature); err != nil {
		t.Fatalf("Verify after Sign should succeed, got %v", err)
	}
}

func TestVerifyRejectsTamperedMessage(t *testing.T) {
	t.Parallel()
	signer, verifier, _ := GenerateKeyPair()
	signature := signer.Sign([]byte("payload"))
	err := verifier.Verify([]byte("payload-tampered"), signature)
	if !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("Verify with tampered message returned %v, want ErrInvalidSignature", err)
	}
}

func TestVerifyRejectsTamperedSignature(t *testing.T) {
	t.Parallel()
	signer, verifier, _ := GenerateKeyPair()
	signature := signer.Sign([]byte("payload"))
	signature[0] ^= 0x01
	err := verifier.Verify([]byte("payload"), signature)
	if !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("Verify with tampered signature returned %v, want ErrInvalidSignature", err)
	}
}

func TestVerifyRejectsWrongKey(t *testing.T) {
	t.Parallel()
	signerA, _, _ := GenerateKeyPair()
	_, verifierB, _ := GenerateKeyPair()
	signature := signerA.Sign([]byte("payload"))
	err := verifierB.Verify([]byte("payload"), signature)
	if !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("Verify with wrong key returned %v, want ErrInvalidSignature", err)
	}
}

func TestPEMRoundTrip(t *testing.T) {
	t.Parallel()
	original, _, _ := GenerateKeyPair()

	privatePEM, err := original.MarshalPrivateKeyPEM()
	if err != nil {
		t.Fatalf("MarshalPrivateKeyPEM error: %v", err)
	}
	reloaded, err := LoadSignerFromPEM(privatePEM)
	if err != nil {
		t.Fatalf("LoadSignerFromPEM error: %v", err)
	}

	message := []byte("after round trip")
	signatureOriginal := original.Sign(message)
	signatureReloaded := reloaded.Sign(message)

	if string(signatureOriginal) != string(signatureReloaded) {
		t.Fatalf("reloaded signer produces different signature for same input")
	}
}

func TestVerifierPEMRoundTrip(t *testing.T) {
	t.Parallel()
	signer, verifier, _ := GenerateKeyPair()
	publicPEM, err := verifier.MarshalPublicKeyPEM()
	if err != nil {
		t.Fatalf("MarshalPublicKeyPEM error: %v", err)
	}
	reloadedVerifier, err := LoadVerifierFromPEM(publicPEM)
	if err != nil {
		t.Fatalf("LoadVerifierFromPEM error: %v", err)
	}
	signature := signer.Sign([]byte("after public-pem round trip"))
	if err := reloadedVerifier.Verify([]byte("after public-pem round trip"), signature); err != nil {
		t.Fatalf("reloaded verifier failed to verify signature: %v", err)
	}
}

func TestLoadSignerFromPEMRejectsMalformed(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		input   []byte
		errKind error
	}{
		{"empty", []byte(""), ErrInvalidPEMBlock},
		{"not pem", []byte("hello"), ErrInvalidPEMBlock},
		{"wrong block type", []byte("-----BEGIN RSA PRIVATE KEY-----\nMIIB\n-----END RSA PRIVATE KEY-----\n"), ErrInvalidPEMBlock},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := LoadSignerFromPEM(tt.input)
			if !errors.Is(err, tt.errKind) {
				t.Fatalf("LoadSignerFromPEM(%q) error %v, want %v", tt.name, err, tt.errKind)
			}
		})
	}
}

func TestSignerVerifierAccessor(t *testing.T) {
	t.Parallel()
	signer, expected, _ := GenerateKeyPair()
	derived := signer.Verifier()
	if derived.Fingerprint() != expected.Fingerprint() {
		t.Fatalf("Signer.Verifier() fingerprint mismatch")
	}
}

func TestFingerprintIsStable(t *testing.T) {
	t.Parallel()
	_, verifierA, _ := GenerateKeyPair()
	verifierB, err := NewVerifier(verifierA.PublicKey())
	if err != nil {
		t.Fatalf("NewVerifier error: %v", err)
	}
	if verifierA.Fingerprint() != verifierB.Fingerprint() {
		t.Fatalf("identical public keys produced different fingerprints")
	}
}

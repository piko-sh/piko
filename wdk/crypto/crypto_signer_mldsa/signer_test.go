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

package crypto_signer_mldsa_test

import (
	"bytes"
	"errors"
	"testing"

	"piko.sh/piko/wdk/crypto/crypto_signer_mldsa"
)

func TestSignVerifyRoundTrip(t *testing.T) {
	t.Parallel()

	signer, verifier, err := crypto_signer_mldsa.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair: %v", err)
	}

	message := []byte("ratify the audit segment")
	signature, err := signer.Sign(message)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	if err := verifier.Verify(message, signature); err != nil {
		t.Errorf("Verify of a genuine signature should succeed, got %v", err)
	}
}

func TestVerifyRejectsTamperedSignature(t *testing.T) {
	t.Parallel()

	signer, verifier, err := crypto_signer_mldsa.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair: %v", err)
	}
	message := []byte("authentic message")
	signature, err := signer.Sign(message)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}

	signature[len(signature)/2] ^= 0xFF
	if err := verifier.Verify(message, signature); !errors.Is(err, crypto_signer_mldsa.ErrInvalidSignature) {
		t.Errorf("a tampered signature must fail with ErrInvalidSignature, got %v", err)
	}
}

func TestVerifyRejectsForeignKey(t *testing.T) {
	t.Parallel()

	signer, _, err := crypto_signer_mldsa.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair: %v", err)
	}
	_, foreignVerifier, err := crypto_signer_mldsa.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair (foreign): %v", err)
	}

	message := []byte("signed by the first key")
	signature, err := signer.Sign(message)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	if err := foreignVerifier.Verify(message, signature); !errors.Is(err, crypto_signer_mldsa.ErrInvalidSignature) {
		t.Errorf("a foreign key must not verify the signature, got %v", err)
	}
}

func TestMarshalRoundTrip(t *testing.T) {
	t.Parallel()

	signer, verifier, err := crypto_signer_mldsa.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair: %v", err)
	}

	privateBytes, err := signer.MarshalBinary()
	if err != nil {
		t.Fatalf("Signer.MarshalBinary: %v", err)
	}
	publicBytes, err := verifier.MarshalBinary()
	if err != nil {
		t.Fatalf("Verifier.MarshalBinary: %v", err)
	}

	restoredSigner, err := crypto_signer_mldsa.NewSignerFromBytes(privateBytes)
	if err != nil {
		t.Fatalf("NewSignerFromBytes: %v", err)
	}
	restoredVerifier, err := crypto_signer_mldsa.NewVerifierFromBytes(publicBytes)
	if err != nil {
		t.Fatalf("NewVerifierFromBytes: %v", err)
	}

	message := []byte("survives a reload")
	signature, err := restoredSigner.Sign(message)
	if err != nil {
		t.Fatalf("restored Sign: %v", err)
	}
	if err := restoredVerifier.Verify(message, signature); err != nil {
		t.Errorf("restored keypair should verify its own signature, got %v", err)
	}

	derivedVerifier, err := signer.Verifier()
	if err != nil {
		t.Fatalf("Signer.Verifier: %v", err)
	}
	derivedBytes, err := derivedVerifier.MarshalBinary()
	if err != nil {
		t.Fatalf("derived MarshalBinary: %v", err)
	}
	if !bytes.Equal(derivedBytes, publicBytes) {
		t.Error("Signer.Verifier should yield the same public key as the generated verifier")
	}
}

func TestNewFromBytesRejectsGarbage(t *testing.T) {
	t.Parallel()

	if _, err := crypto_signer_mldsa.NewSignerFromBytes([]byte("too short")); !errors.Is(err, crypto_signer_mldsa.ErrInvalidKey) {
		t.Errorf("garbage private key bytes must fail with ErrInvalidKey, got %v", err)
	}
	if _, err := crypto_signer_mldsa.NewVerifierFromBytes([]byte("too short")); !errors.Is(err, crypto_signer_mldsa.ErrInvalidKey) {
		t.Errorf("garbage public key bytes must fail with ErrInvalidKey, got %v", err)
	}
}

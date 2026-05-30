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
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
)

const (
	// pemBlockPrivateKey is the PEM block type tag used by Signer.MarshalPrivateKeyPEM for
	// Ed25519 private keys; operators can identify piko keys by this tag when inspecting
	// on-disk material.
	pemBlockPrivateKey = "ED25519 PRIVATE KEY"

	// pemBlockPublicKey is the PEM block type tag used by Verifier.MarshalPublicKeyPEM for
	// Ed25519 public keys; operators can identify piko verifier material by this tag.
	pemBlockPublicKey = "ED25519 PUBLIC KEY"
)

var (
	// ErrInvalidSignature is returned when a signature did not verify against the supplied
	// public key and message. Distinguished from PEM/parsing errors so hosts can surface
	// "tampering" distinct from "malformed input.".
	ErrInvalidSignature = errors.New("crypto_signer_ed25519: signature did not verify")

	// ErrInvalidPEMBlock indicates the supplied PEM data is not a well-formed Ed25519 key
	// block.
	ErrInvalidPEMBlock = errors.New("crypto_signer_ed25519: invalid PEM block")

	// ErrUnsupportedKeyType indicates a PEM block decoded successfully but did not contain
	// an Ed25519 key.
	ErrUnsupportedKeyType = errors.New("crypto_signer_ed25519: not an Ed25519 key")
)

// Signer holds an Ed25519 private key and produces detached signatures. Construct via
// NewSigner, LoadSignerFromPEM, or LoadSignerFromFile.
//
// A Signer is safe for concurrent use.
type Signer struct {
	// privateKey is the 64-byte Ed25519 private key used to produce detached signatures.
	privateKey ed25519.PrivateKey
}

// Verifier holds an Ed25519 public key and checks signatures. Construct via NewVerifier,
// LoadVerifierFromPEM, or LoadVerifierFromFile.
//
// A Verifier is safe for concurrent use.
type Verifier struct {
	// publicKey is the 32-byte Ed25519 public key used to validate detached signatures.
	publicKey ed25519.PublicKey
}

// GenerateKeyPair generates a fresh Ed25519 keypair using crypto/rand. Used by
// operator-side CLI commands to mint signing identities.
//
// Returns the signer (holding the private key), its verifier (holding the matching public
// key), and any rand-source error.
func GenerateKeyPair() (*Signer, *Verifier, error) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("crypto_signer_ed25519: generating keypair: %w", err)
	}
	return &Signer{privateKey: privateKey}, &Verifier{publicKey: publicKey}, nil
}

// NewSigner wraps a raw Ed25519 private key in a Signer.
//
// Takes privateKey (ed25519.PrivateKey) which must be 64 bytes (Ed25519's full private
// key size including the public key half).
//
// Returns a *Signer or an error when the key length is wrong.
func NewSigner(privateKey ed25519.PrivateKey) (*Signer, error) {
	if len(privateKey) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("crypto_signer_ed25519: private key length %d, want %d", len(privateKey), ed25519.PrivateKeySize)
	}
	return &Signer{privateKey: privateKey}, nil
}

// NewVerifier wraps a raw Ed25519 public key in a Verifier.
//
// Takes publicKey (ed25519.PublicKey) which must be 32 bytes.
//
// Returns a *Verifier or an error when the key length is wrong.
func NewVerifier(publicKey ed25519.PublicKey) (*Verifier, error) {
	if len(publicKey) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("crypto_signer_ed25519: public key length %d, want %d", len(publicKey), ed25519.PublicKeySize)
	}
	return &Verifier{publicKey: publicKey}, nil
}

// Sign returns a detached signature over the supplied message bytes. The returned slice
// is 64 bytes; hosts choose their own encoding for transport.
//
// Takes message ([]byte) which is the data to sign.
//
// Returns the signature bytes. Always succeeds for a well-formed Signer.
func (s *Signer) Sign(message []byte) []byte {
	return ed25519.Sign(s.privateKey, message)
}

// Verifier returns the matching public-key Verifier for this Signer. Used to
// round-trip-check that a signer was constructed correctly without exposing the private
// key.
//
// Returns a freshly-allocated Verifier wrapping the matching public key.
func (s *Signer) Verifier() *Verifier {
	publicKey, ok := s.privateKey.Public().(ed25519.PublicKey)
	if !ok {
		return nil
	}
	return &Verifier{publicKey: publicKey}
}

// MarshalPrivateKeyPEM serialises the signer's private key to PEM format. Use by
// operator-side CLI commands when persisting a freshly-generated keypair.
//
// Returns the PEM-encoded bytes or a marshalling error.
func (s *Signer) MarshalPrivateKeyPEM() ([]byte, error) {
	pkcs8, err := x509.MarshalPKCS8PrivateKey(s.privateKey)
	if err != nil {
		return nil, fmt.Errorf("crypto_signer_ed25519: marshalling private key: %w", err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: pemBlockPrivateKey, Bytes: pkcs8}), nil
}

// Verify checks that signature was produced by the matching private key over message.
// Returns nil on success, ErrInvalidSignature on a verifiable mismatch, or other errors
// for malformed input.
//
// Takes message ([]byte) which is the original signed data.
// Takes signature ([]byte) which is the detached signature to check.
//
// Returns nil when the signature is valid for (publicKey, message), ErrInvalidSignature
// otherwise.
func (v *Verifier) Verify(message, signature []byte) error {
	if !ed25519.Verify(v.publicKey, message, signature) {
		return ErrInvalidSignature
	}
	return nil
}

// PublicKey returns a copy of the wrapped public key bytes. Useful for constructing
// fingerprints or for direct stdlib interop.
//
// Returns a 32-byte slice that is safe for the caller to retain.
func (v *Verifier) PublicKey() ed25519.PublicKey {
	out := make(ed25519.PublicKey, ed25519.PublicKeySize)
	copy(out, v.publicKey)
	return out
}

// Fingerprint returns the SHA-256 of the public key, formatted as "sha256:<64-hex>". Used
// by hosts to refer to a signing identity without including the full key bytes (e.g., the
// signing_key_ref field in pinkas.toml).
//
// Returns the fingerprint string.
func (v *Verifier) Fingerprint() string {
	sum := sha256.Sum256(v.publicKey)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// MarshalPublicKeyPEM serialises the verifier's public key to PEM format. Used by
// operator-side CLI commands to share verifier identities with peers / endpoints.
//
// Returns the PEM-encoded bytes or a marshalling error.
func (v *Verifier) MarshalPublicKeyPEM() ([]byte, error) {
	pkix, err := x509.MarshalPKIXPublicKey(v.publicKey)
	if err != nil {
		return nil, fmt.Errorf("crypto_signer_ed25519: marshalling public key: %w", err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: pemBlockPublicKey, Bytes: pkix}), nil
}

// LoadSignerFromPEM parses a PEM block produced by Signer.MarshalPrivateKeyPEM.
//
// Takes data ([]byte) which is the PEM-encoded private key.
//
// Returns the Signer or a parse / type error.
//
//nolint:dupl // mirrors LoadVerifierFromPEM; identical PEM/X.509 paths kept in lockstep
func LoadSignerFromPEM(data []byte) (*Signer, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("%w: no PEM block found", ErrInvalidPEMBlock)
	}
	if block.Type != pemBlockPrivateKey {
		return nil, fmt.Errorf("%w: expected %q, got %q", ErrInvalidPEMBlock, pemBlockPrivateKey, block.Type)
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("crypto_signer_ed25519: parsing PKCS8 private key: %w", err)
	}
	privateKey, ok := parsed.(ed25519.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("%w: parsed key type %T is not Ed25519", ErrUnsupportedKeyType, parsed)
	}
	return NewSigner(privateKey)
}

// LoadVerifierFromPEM parses a PEM block produced by Verifier.MarshalPublicKeyPEM.
//
// Takes data ([]byte) which is the PEM-encoded public key.
//
// Returns the Verifier or a parse / type error.
//
//nolint:dupl // mirrors LoadSignerFromPEM; see its rationale
func LoadVerifierFromPEM(data []byte) (*Verifier, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("%w: no PEM block found", ErrInvalidPEMBlock)
	}
	if block.Type != pemBlockPublicKey {
		return nil, fmt.Errorf("%w: expected %q, got %q", ErrInvalidPEMBlock, pemBlockPublicKey, block.Type)
	}
	parsed, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("crypto_signer_ed25519: parsing PKIX public key: %w", err)
	}
	publicKey, ok := parsed.(ed25519.PublicKey)
	if !ok {
		return nil, fmt.Errorf("%w: parsed key type %T is not Ed25519", ErrUnsupportedKeyType, parsed)
	}
	return NewVerifier(publicKey)
}

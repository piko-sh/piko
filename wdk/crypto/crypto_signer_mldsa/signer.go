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

package crypto_signer_mldsa

import (
	"errors"
	"fmt"

	"github.com/cloudflare/circl/sign"
	"github.com/cloudflare/circl/sign/mldsa/mldsa65"
)

var (
	// scheme is ML-DSA-65 (FIPS 204, NIST security category 3), the balanced parameter set
	// and a sensible pairing for a classical Ed25519 signer in a hybrid scheme. Signing uses
	// no context string (the empty context), matching the plain detached-signature contract
	// of the classical signer.
	scheme = mldsa65.Scheme()
)

var (
	// ErrInvalidSignature is returned when a signature did not verify against the supplied
	// public key and message. Distinguished from key-parsing errors so hosts can surface
	// "tampering" distinct from "malformed input".
	ErrInvalidSignature = errors.New("crypto_signer_mldsa: signature did not verify")

	// ErrInvalidKey indicates the supplied bytes are not a well-formed ML-DSA-65 key for
	// this scheme.
	ErrInvalidKey = errors.New("crypto_signer_mldsa: invalid key bytes")
)

// Signer holds an ML-DSA-65 private key and produces detached signatures.
//
// Construct via GenerateKeyPair or NewSignerFromBytes. Safe for concurrent use.
type Signer struct {
	// privateKey is the ML-DSA-65 private key used to produce signatures.
	privateKey sign.PrivateKey
}

// Verifier holds an ML-DSA-65 public key and checks signatures.
//
// Construct via GenerateKeyPair, Signer.Verifier, or NewVerifierFromBytes. Safe for
// concurrent use.
type Verifier struct {
	// publicKey is the ML-DSA-65 public key used to validate signatures.
	publicKey sign.PublicKey
}

// GenerateKeyPair generates a fresh ML-DSA-65 keypair.
//
// Returns the signer (holding the private key), its verifier (holding the matching public
// key), and any key-generation error.
func GenerateKeyPair() (*Signer, *Verifier, error) {
	publicKey, privateKey, err := scheme.GenerateKey()
	if err != nil {
		return nil, nil, fmt.Errorf("crypto_signer_mldsa: generating keypair: %w", err)
	}
	return &Signer{privateKey: privateKey}, &Verifier{publicKey: publicKey}, nil
}

// NewSignerFromBytes reconstructs a Signer from the bytes produced by
// Signer.MarshalBinary.
//
// Takes data ([]byte) which is the private-key encoding from Signer.MarshalBinary.
//
// Returns a *Signer, or an error wrapping ErrInvalidKey when the bytes are not a
// well-formed ML-DSA-65 private key.
func NewSignerFromBytes(data []byte) (*Signer, error) {
	privateKey, err := scheme.UnmarshalBinaryPrivateKey(data)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidKey, err)
	}
	return &Signer{privateKey: privateKey}, nil
}

// NewVerifierFromBytes reconstructs a Verifier from the bytes produced by
// Verifier.MarshalBinary.
//
// Takes data ([]byte) which is the public-key encoding from Verifier.MarshalBinary.
//
// Returns a *Verifier, or an error wrapping ErrInvalidKey when the bytes are not a
// well-formed ML-DSA-65 public key.
func NewVerifierFromBytes(data []byte) (*Verifier, error) {
	publicKey, err := scheme.UnmarshalBinaryPublicKey(data)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidKey, err)
	}
	return &Verifier{publicKey: publicKey}, nil
}

// Sign returns a detached ML-DSA-65 signature over the supplied message bytes.
//
// The error return mirrors the broader Signer contract so a host can treat the classical
// and post-quantum signers uniformly; ML-DSA signing does not surface an error for a
// well-formed Signer.
//
// Takes message ([]byte) which is the data to sign.
//
// Returns the detached signature bytes and a nil error for a well-formed Signer.
func (s *Signer) Sign(message []byte) ([]byte, error) {
	return scheme.Sign(s.privateKey, message, nil), nil
}

// Verifier returns the matching public-key Verifier for this Signer, without exposing the
// private key.
//
// Returns the *Verifier, or an error wrapping ErrInvalidKey when the private key has no
// ML-DSA public half.
func (s *Signer) Verifier() (*Verifier, error) {
	publicKey, ok := s.privateKey.Public().(sign.PublicKey)
	if !ok {
		return nil, fmt.Errorf("%w: private key has no ML-DSA public half", ErrInvalidKey)
	}
	return &Verifier{publicKey: publicKey}, nil
}

// MarshalBinary serialises the signer's private key to raw bytes for persistence.
//
// The byte layout is the scheme's canonical encoding.
//
// Returns the canonical private-key encoding and any error from the scheme encoder.
func (s *Signer) MarshalBinary() ([]byte, error) {
	return s.privateKey.MarshalBinary()
}

// Verify checks that signature was produced by the matching private key over message.
//
// Takes message ([]byte) which is the original signed data.
// Takes signature ([]byte) which is the detached signature to check.
//
// Returns nil when the signature is valid, or ErrInvalidSignature on a mismatch or a
// malformed signature.
func (v *Verifier) Verify(message, signature []byte) error {
	if !scheme.Verify(v.publicKey, message, signature, nil) {
		return ErrInvalidSignature
	}
	return nil
}

// MarshalBinary serialises the verifier's public key to raw bytes.
//
// Use it to share a verifier identity with peers.
//
// Returns the canonical public-key encoding and any error from the scheme encoder.
func (v *Verifier) MarshalBinary() ([]byte, error) {
	return v.publicKey.MarshalBinary()
}

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

package security_domain

import (
	"crypto/hmac"
	"crypto/sha256"
	"hash"
	"sync"
)

// csrfSigner computes CSRF signatures with one service's secret key. Each signer owns its
// own pool of HMAC instances, so services configured with different keys never share
// keyed state.
type csrfSigner struct {
	// pool reuses HMAC-SHA256 instances keyed with secretKey. Reset clears the running state
	// but keeps the key, so reuse is safe.
	pool sync.Pool

	// secretKey is a private copy of the signing key.
	secretKey []byte
}

// sum computes the truncated HMAC-SHA256 signature of payload.
//
// Takes payload ([]byte) which is the data to sign.
//
// Returns []byte which is the truncated signature.
func (s *csrfSigner) sum(payload []byte) []byte {
	mac, ok := s.pool.Get().(hash.Hash)
	if !ok {
		mac = hmac.New(sha256.New, s.secretKey)
	}
	_, _ = mac.Write(payload)
	fullSignature := mac.Sum(nil)
	mac.Reset()
	s.pool.Put(mac)
	return fullSignature[:signatureLengthBytes]
}

// newCSRFSigner creates a signer for secretKey.
//
// Takes secretKey ([]byte) which is the HMAC key. The signer keeps its own copy.
//
// Returns *csrfSigner which signs with the key.
// Returns error when the key is empty.
func newCSRFSigner(secretKey []byte) (*csrfSigner, error) {
	if len(secretKey) == 0 {
		return nil, errMissingSecret
	}
	signer := &csrfSigner{pool: sync.Pool{}, secretKey: append([]byte(nil), secretKey...)}
	signer.pool.New = func() any {
		return hmac.New(sha256.New, signer.secretKey)
	}
	return signer, nil
}

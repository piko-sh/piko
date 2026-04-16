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

package driven_transform_pades

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/asn1"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	// tsaTimeout is the HTTP timeout for TSA requests.
	tsaTimeout = 30 * time.Second

	// tsaContentType is the MIME type for RFC 3161 timestamp requests.
	tsaContentType = "application/timestamp-query"

	// tsaAcceptType is the MIME type for RFC 3161 timestamp responses.
	tsaAcceptType = "application/timestamp-reply"

	// tsaMaxResponseSize is the maximum TSA response size (1 MB).
	tsaMaxResponseSize = 1 << 20
)

var (
	// ErrTSAHTTPFailure indicates the TSA server returned a non-200
	// HTTP status code.
	ErrTSAHTTPFailure = errors.New("TSA returned non-200 HTTP status")

	// ErrTSAErrorStatus indicates the TSA response contained an error
	// status.
	ErrTSAErrorStatus = errors.New("TSA returned error status in response")

	// ErrTSAEmptyToken indicates the TSA response did not contain a
	// timestamp token.
	ErrTSAEmptyToken = errors.New("TSA response contains no timestamp token")
)

// RFC 3161 ASN.1 structures.

// timeStampReq is an RFC 3161 TimeStampReq (section 2.4.1).
//
//nolint:govet // ASN.1 field order
type timeStampReq struct {
	// Version holds the TSP protocol version number.
	Version int

	// MessageImprint holds the hash algorithm and hashed message.
	MessageImprint messageImprint

	// CertReq specifies whether the TSA should include its certificate
	// in the response.
	CertReq bool `asn1:"optional"`
}

// messageImprint is the hash algorithm and hash value of the datum to
// be timestamped (RFC 3161 section 2.4.1).
type messageImprint struct {
	// HashAlgorithm holds the algorithm used to hash the message.
	HashAlgorithm algorithmIdentifier

	// HashedMessage holds the hash value of the datum to timestamp.
	HashedMessage []byte
}

// algorithmIdentifier is an ASN.1 AlgorithmIdentifier without parameters.
type algorithmIdentifier struct {
	// Algorithm holds the algorithm OID.
	Algorithm asn1.ObjectIdentifier
}

// timeStampResp is an RFC 3161 TimeStampResp (section 2.4.2).
//
//nolint:govet // ASN.1 field order
type timeStampResp struct {
	// Status holds the PKI status information from the TSA.
	Status pkiStatusInfo

	// TimeStampToken holds the optional DER-encoded timestamp token.
	TimeStampToken asn1.RawValue `asn1:"optional"`
}

// pkiStatusInfo is the status field of a TimeStampResp.
type pkiStatusInfo struct {
	// Status holds the integer status code from the TSA response.
	Status int
}

// requestTimestamp sends an RFC 3161 timestamp request to the TSA and
// returns the DER-encoded TimeStampToken (a CMS ContentInfo). The
// token proves that the given signature existed at the time of
// timestamping.
//
// Takes ctx (context.Context) which carries cancellation and timeout.
// Takes tsaURL (string) which is the TSA endpoint URL.
// Takes signature ([]byte) which is the signature value to timestamp.
//
// Returns []byte which is the DER-encoded TimeStampToken.
// Returns error when the request fails or the TSA returns an error.
func requestTimestamp(ctx context.Context, tsaURL string, signature []byte) ([]byte, error) {
	digest := sha256.Sum256(signature)

	req := timeStampReq{
		Version: 1,
		MessageImprint: messageImprint{
			HashAlgorithm: algorithmIdentifier{Algorithm: oidSHA256},
			HashedMessage: digest[:],
		},
		CertReq: true,
	}

	reqDER, err := asn1.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("encoding timestamp request: %w", err)
	}

	ctx, cancel := context.WithTimeoutCause(ctx, tsaTimeout, errors.New("TSA request timed out"))
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, tsaURL, bytes.NewReader(reqDER))
	if err != nil {
		return nil, fmt.Errorf("creating HTTP request: %w", err)
	}
	httpReq.Header.Set("Content-Type", tsaContentType)
	httpReq.Header.Set("Accept", tsaAcceptType)

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("sending timestamp request to %s: %w", tsaURL, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("TSA returned HTTP %d: %w", resp.StatusCode, ErrTSAHTTPFailure)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, tsaMaxResponseSize))
	if err != nil {
		return nil, fmt.Errorf("reading TSA response: %w", err)
	}

	var tsResp timeStampResp
	rest, err := asn1.Unmarshal(body, &tsResp)
	if err != nil {
		return nil, fmt.Errorf("decoding timestamp response: %w", err)
	}
	if len(rest) > 0 {
		return nil, errors.New("trailing bytes in timestamp response")
	}

	if tsResp.Status.Status > 1 {
		return nil, fmt.Errorf("TSA returned error status %d: %w", tsResp.Status.Status, ErrTSAErrorStatus)
	}

	if len(tsResp.TimeStampToken.FullBytes) == 0 {
		return nil, fmt.Errorf("TSA response contains no timestamp token: %w", ErrTSAEmptyToken)
	}

	return tsResp.TimeStampToken.FullBytes, nil
}

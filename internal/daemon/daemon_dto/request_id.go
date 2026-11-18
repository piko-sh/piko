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

package daemon_dto

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"piko.sh/safeconv"
	"piko.sh/sharded"
)

const (
	// requestIDPrefixLength is the number of random characters in the request ID prefix.
	// This gives enough uniqueness when the process restarts.
	requestIDPrefixLength = 10

	// requestIDCounterPadding is the number of digits to pad the request counter to. With 6
	// digits, the counter can reach 999,999 before it goes past the padding width.
	requestIDCounterPadding = 6

	// firstRequestIDCounter is the first counter value handed out. Zero is kept free so that
	// PikoRequestCtx can use it to mean "no generated ID".
	firstRequestIDCounter = 1

	// baseDecimal is the base value for formatting numbers as decimal strings.
	baseDecimal = 10

	// maxForwardedRequestIDLength bounds an id accepted from a proxy.
	maxForwardedRequestIDLength = 128

	// forwardedRequestIDPrefix marks an id that came from a client rather than from this
	// process, so a reader tracing an incident can tell which ids the server issued.
	forwardedRequestIDPrefix = "fwd/"
)

var (
	// requestIDPrefix is a unique prefix for this process instance, formatted as
	// "hostname/random10chars".
	requestIDPrefix string

	// requestIDSequence hands out the unique counter values for request IDs. It is nil only
	// when requestIDSequenceErr is set.
	requestIDSequence *sharded.Sequence

	// requestIDSequenceErr holds the error from creating requestIDSequence, if any.
	requestIDSequenceErr error

	// errNilRequestIDSequence is returned when there is no sequence to take a counter from.
	errNilRequestIDSequence = errors.New("request id sequence is nil")
)

// NextRequestIDCounter returns a process-unique counter when available.
//
// Values are unique for the life of the process and never zero, but they are not in
// arrival order and may have gaps, because each processor hands out values from its own
// block.
//
// Callers should use FallbackRequestIDCounter if no counter is available.
//
// Returns uint64 which is the next unique counter value, zero when error is set.
// Returns error which wraps sharded.ErrSequenceExhausted when no value is left, or the
// creation error when the sequence could not be made.
func NextRequestIDCounter() (uint64, error) {
	return nextRequestIDCounterFrom(requestIDSequence, requestIDSequenceErr)
}

// FallbackRequestIDCounter returns a random, non-zero counter value for use when
// NextRequestIDCounter fails. The value is not guaranteed to be unique, but a collision
// is very unlikely.
//
// Returns uint64 which is never zero.
func FallbackRequestIDCounter() uint64 {
	var buffer [8]byte
	if _, err := rand.Read(buffer[:]); err != nil {
		return safeconv.Int64ToUint64(time.Now().UnixNano()) | firstRequestIDCounter
	}
	return binary.LittleEndian.Uint64(buffer[:]) | firstRequestIDCounter
}

// FormatRequestID formats a counter value into the full request ID string
// "hostname/random10chars-NNNNNN". This is called lazily by PikoRequestCtx.RequestID when
// the request ID is actually read.
//
// Takes counter (uint64) which is the raw counter value to format.
//
// Returns string which is the formatted request ID.
func FormatRequestID(counter uint64) string {
	var counterBuf [20]byte
	counterString := strconv.AppendUint(counterBuf[:0], counter, baseDecimal)

	padLen := max(requestIDCounterPadding-len(counterString), 0)

	totalLen := len(requestIDPrefix) + 1 + padLen + len(counterString)
	buffer := make([]byte, totalLen)

	n := copy(buffer, requestIDPrefix)
	buffer[n] = '-'
	n++
	for i := range padLen {
		buffer[n+i] = '0'
	}
	n += padLen
	copy(buffer[n:], counterString)

	return string(buffer)
}

func init() {
	hostname, err := os.Hostname()
	if hostname == "" || err != nil {
		hostname = "localhost"
	}

	var buffer [12]byte
	_, _ = rand.Read(buffer[:])
	b64 := base64.StdEncoding.EncodeToString(buffer[:])
	b64 = strings.NewReplacer("+", "", "/", "").Replace(b64)

	requestIDPrefix = hostname + "/" + b64[:requestIDPrefixLength]

	requestIDSequence, requestIDSequenceErr = sharded.NewSequence(sharded.WithStart(firstRequestIDCounter))
}

// AcceptForwardedRequestID reports whether a request id received from a proxy may be used
// as-is, and returns it unchanged when it may.
//
// Takes forwarded (string) which is the raw header value.
//
// Returns string which is the accepted id, empty when the value was rejected.
// Returns bool which reports whether the value was accepted.
func AcceptForwardedRequestID(forwarded string) (string, bool) {
	if forwarded == "" || len(forwarded) > maxForwardedRequestIDLength {
		return "", false
	}

	for index := range len(forwarded) {
		if !isRequestIDByte(forwarded[index]) {
			return "", false
		}
	}

	return forwardedRequestIDPrefix + forwarded, true
}

// IsForwardedRequestID reports whether a request id came from a client rather than being
// issued by this process.
//
// Takes requestID (string) which is the id to classify.
//
// Returns bool which is true when the id was forwarded by a trusted proxy.
func IsForwardedRequestID(requestID string) bool {
	return strings.HasPrefix(requestID, forwardedRequestIDPrefix)
}

// isRequestIDByte reports whether a byte may appear in an accepted request id.
//
// Takes candidate (byte) which is the byte to classify.
//
// Returns bool which is true for the accepted character set.
func isRequestIDByte(candidate byte) bool {
	switch {
	case candidate >= 'a' && candidate <= 'z',
		candidate >= 'A' && candidate <= 'Z',
		candidate >= '0' && candidate <= '9':
		return true
	default:

		return candidate == '-' || candidate == '_' || candidate == '.'
	}
}

// nextRequestIDCounterFrom takes the next counter value from sequence.
//
// Takes sequence (*sharded.Sequence) which hands out the values.
// Takes creationErr (error) which is the error from creating sequence, if any.
//
// Returns uint64 which is the next unique counter value, zero when error is set.
// Returns error which explains why no value could be taken.
func nextRequestIDCounterFrom(sequence *sharded.Sequence, creationErr error) (uint64, error) {
	if creationErr != nil {
		return 0, fmt.Errorf("creating request id sequence: %w", creationErr)
	}
	if sequence == nil {
		return 0, errNilRequestIDSequence
	}
	counter, err := sequence.Next()
	if err != nil {
		return 0, fmt.Errorf("taking next request id counter: %w", err)
	}
	return counter, nil
}

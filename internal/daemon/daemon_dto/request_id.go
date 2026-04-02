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
	"os"
	"strconv"
	"strings"
	"sync/atomic"
)

const (
	// requestIDPrefixLength is the number of random characters in the request ID
	// prefix. This gives enough uniqueness when the process restarts.
	requestIDPrefixLength = 10

	// requestIDCounterPadding is the number of digits to pad the request counter
	// to. With 6 digits, the counter can reach 999,999 before it goes past the
	// padding width.
	requestIDCounterPadding = 6

	// baseDecimal is the base value for formatting numbers as decimal strings.
	baseDecimal = 10
)

var (
	// requestIDPrefix is a unique prefix for this process instance, formatted as
	// "hostname/random10chars".
	requestIDPrefix string

	// requestIDCounter is an atomic counter for generating unique request IDs.
	requestIDCounter atomic.Uint64
)

// NextRequestIDCounter atomically increments and returns the next
// request ID counter value. The middleware stores this lightweight
// uint64 in the context instead of a pre-formatted string; the
// string is produced lazily by FormatRequestID only when actually
// read.
//
// Returns uint64 which is the next unique counter value.
func NextRequestIDCounter() uint64 {
	return requestIDCounter.Add(1)
}

// FormatRequestID formats a counter value into the full request ID
// string "hostname/random10chars-NNNNNN". This is called lazily by
// PikoRequestCtx.RequestID when the request ID is actually read.
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
}

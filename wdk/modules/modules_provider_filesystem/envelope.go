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

package modules_provider_filesystem

import (
	"encoding/binary"
	"errors"
	"fmt"

	"piko.sh/piko/wdk/modules"
	"piko.sh/piko/wdk/safeconv"
)

const (
	// envelopeMagicLen is the byte length of envelopeMagic.
	envelopeMagicLen = 6

	// envelopeLengthPrefixSize is the byte length of the big-endian uint32 length prefix
	// that follows the magic.
	envelopeLengthPrefixSize = 4

	// envelopeMinDescriptorBytes is the minimum descriptor byte count a well-formed envelope
	// can carry.
	envelopeMinDescriptorBytes = 1

	// envelopeMinBytecodeBytes is the minimum bytecode byte count a well-formed envelope can
	// carry.
	envelopeMinBytecodeBytes = 1

	// envelopeMinLen is the minimum number of bytes a well-formed envelope can occupy: magic
	// + length prefix + at least 1 byte of descriptor + at least 1 byte of bytecode.
	envelopeMinLen = envelopeMagicLen +
		envelopeLengthPrefixSize +
		envelopeMinDescriptorBytes +
		envelopeMinBytecodeBytes
)

var (
	// envelopeMagic is the magic prefix for the .pkbundle file format: 6 bytes literal
	// "PKBND\x01". The trailing 0x01 is the envelope version; bump it for any incompatible
	// wire-format change.
	envelopeMagic = []byte{'P', 'K', 'B', 'N', 'D', 0x01}

	// errInvalidEnvelope is the sentinel returned for any malformed envelope shape: too
	// short, bad magic, length prefix that overflows the file. Wrapped with
	// modules.ErrModuleNotFound by the caller when appropriate.
	errInvalidEnvelope = errors.New("modules_provider_filesystem: invalid bundle envelope")
)

// MarshalEnvelope serialises a bundle to the .pkbundle wire format described in the
// package doc.
//
// Takes bundle (*modules.ModuleBundle) which must be Validate()-clean.
//
// Returns the envelope bytes ready for atomic write, or any marshalling error from the
// descriptor.
func MarshalEnvelope(bundle *modules.ModuleBundle) ([]byte, error) {
	if err := bundle.Validate(); err != nil {
		return nil, err
	}
	descriptorBytes, err := bundle.Descriptor.MarshalCanonicalJSON()
	if err != nil {
		return nil, fmt.Errorf("modules_provider_filesystem: marshalling descriptor: %w", err)
	}
	out := make([]byte, 0, envelopeMagicLen+envelopeLengthPrefixSize+len(descriptorBytes)+len(bundle.Bytecode))
	out = append(out, envelopeMagic...)
	descriptorLenPrefix := make([]byte, envelopeLengthPrefixSize)
	binary.BigEndian.PutUint32(descriptorLenPrefix, safeconv.IntToUint32(len(descriptorBytes)))
	out = append(out, descriptorLenPrefix...)
	out = append(out, descriptorBytes...)
	out = append(out, bundle.Bytecode...)
	return out, nil
}

// UnmarshalEnvelope parses .pkbundle bytes into a modules.ModuleBundle. The descriptor is
// decoded and validated; the bytecode is returned as a fresh slice (not shared with the
// input buffer).
//
// Takes data ([]byte) which is the raw envelope content.
//
// Returns the bundle or a parse / validation error.
func UnmarshalEnvelope(data []byte) (*modules.ModuleBundle, error) {
	if len(data) < envelopeMinLen {
		return nil, fmt.Errorf("%w: %d bytes, need at least %d", errInvalidEnvelope, len(data), envelopeMinLen)
	}
	if !envelopeMagicMatches(data) {
		return nil, fmt.Errorf("%w: bad magic prefix", errInvalidEnvelope)
	}
	descriptorLen := binary.BigEndian.Uint32(data[envelopeMagicLen : envelopeMagicLen+envelopeLengthPrefixSize])
	headerLen := envelopeMagicLen + envelopeLengthPrefixSize
	if uint64(headerLen)+uint64(descriptorLen) > uint64(len(data)) {
		return nil, fmt.Errorf("%w: descriptor length %d overflows envelope (%d bytes total)", errInvalidEnvelope, descriptorLen, len(data))
	}
	descriptorEnd := headerLen + int(descriptorLen)
	descriptor, err := modules.UnmarshalDescriptor(data[headerLen:descriptorEnd])
	if err != nil {
		return nil, err
	}
	if descriptorEnd >= len(data) {
		return nil, fmt.Errorf("%w: bytecode section empty", errInvalidEnvelope)
	}
	bytecode := make([]byte, len(data)-descriptorEnd)
	copy(bytecode, data[descriptorEnd:])
	return &modules.ModuleBundle{
		Descriptor: descriptor,
		Bytecode:   bytecode,
	}, nil
}

// envelopeMagicMatches reports whether the first envelopeMagicLen bytes of data match
// envelopeMagic byte-for-byte.
//
// Takes data ([]byte) which is the envelope candidate.
//
// Returns bool which is true when the prefix matches.
func envelopeMagicMatches(data []byte) bool {
	if len(data) < envelopeMagicLen {
		return false
	}
	for i := range envelopeMagicLen {
		if data[i] != envelopeMagic[i] {
			return false
		}
	}
	return true
}

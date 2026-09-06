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
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math"

	"piko.sh/piko/wdk/modules"
	"piko.sh/piko/wdk/safeconv"
)

const (
	// envelopeMagicLength counts the five-byte envelopeMagicPrefix and the following version
	// byte.
	envelopeMagicLength = 6

	// envelopeVersionIndex is the offset of the version byte within the magic.
	envelopeVersionIndex = 5

	// envelopeLengthPrefixSize is the byte length of each big-endian uint32 section length
	// prefix.
	envelopeLengthPrefixSize = 4

	// envelopeVersionOne identifies the original layout containing the descriptor length,
	// descriptor, and bytecode running to the end of the file, in that order. It carries no
	// types export.
	envelopeVersionOne byte = 0x01

	// envelopeVersionTwo identifies the current layout with three length-prefixed sections
	// holding the descriptor, the bytecode and the types export, in that order.
	envelopeVersionTwo byte = 0x02

	// envelopeMinLength counts the magic, one length prefix, and at least one byte each for
	// the descriptor and bytecode that a well-formed envelope requires.
	envelopeMinLength = envelopeMagicLength + envelopeLengthPrefixSize + 1 + 1
)

var (
	// envelopeMagicPrefix is the version-independent start of every .pkbundle file. The
	// version byte that follows it selects the section layout.
	envelopeMagicPrefix = []byte{'P', 'K', 'B', 'N', 'D'}

	// errInvalidEnvelope is the sentinel returned for any malformed envelope shape: too
	// short, bad magic, unknown version, or a section length that overflows the file.
	errInvalidEnvelope = errors.New("modules_provider_filesystem: invalid bundle envelope")
)

// envelopeReader walks the length-prefixed sections of a version two envelope, bounding
// every declared length against the bytes that remain.
type envelopeReader struct {
	// data is the complete envelope.
	data []byte

	// offset is the index of the next unread byte in data.
	offset int
}

// readSection reads one length-prefixed section.
//
// Takes name (string) which labels the section in error messages.
//
// Returns []byte which is a view of the section within the envelope.
// Returns error when the length prefix is truncated or the declared length exceeds the
// remaining input.
func (r *envelopeReader) readSection(name string) ([]byte, error) {
	remaining := len(r.data) - r.offset
	if remaining < envelopeLengthPrefixSize {
		return nil, fmt.Errorf("%w: %s length prefix truncated (%d bytes remain)", errInvalidEnvelope, name, remaining)
	}
	declared := binary.BigEndian.Uint32(r.data[r.offset : r.offset+envelopeLengthPrefixSize])
	r.offset += envelopeLengthPrefixSize
	remaining -= envelopeLengthPrefixSize
	if uint64(declared) > uint64(remaining) {
		return nil, fmt.Errorf("%w: %s length %d overflows the %d remaining bytes", errInvalidEnvelope, name, declared, remaining)
	}
	length := safeconv.Uint64ToInt(uint64(declared))
	section := r.data[r.offset : r.offset+length]
	r.offset += length
	return section, nil
}

// MarshalEnvelope serialises a bundle to the current .pkbundle wire format described in
// the package doc.
//
// Takes bundle (*modules.ModuleBundle) which must be Validate()-clean.
//
// Returns []byte which holds the envelope ready for atomic write.
// Returns error when the bundle is invalid, the descriptor cannot be marshalled, or a
// section is too large for its length prefix.
func MarshalEnvelope(bundle *modules.ModuleBundle) ([]byte, error) {
	if err := bundle.Validate(); err != nil {
		return nil, err
	}
	descriptorBytes, err := bundle.Descriptor.MarshalCanonicalJSON()
	if err != nil {
		return nil, fmt.Errorf("modules_provider_filesystem: marshalling descriptor: %w", err)
	}
	sections := [][]byte{descriptorBytes, bundle.Bytecode, bundle.TypesExport}
	total := envelopeMagicLength
	for _, section := range sections {
		if uint64(len(section)) > math.MaxUint32 {
			return nil, fmt.Errorf("modules_provider_filesystem: envelope section of %d bytes exceeds the format limit", len(section))
		}
		total += envelopeLengthPrefixSize + len(section)
	}
	out := make([]byte, 0, total)
	out = append(out, envelopeMagicPrefix...)
	out = append(out, envelopeVersionTwo)
	for _, section := range sections {
		out = binary.BigEndian.AppendUint32(out, safeconv.IntToUint32(len(section)))
		out = append(out, section...)
	}
	return out, nil
}

// UnmarshalEnvelope parses .pkbundle bytes into a modules.ModuleBundle.
//
// Both envelope versions are accepted; a version one envelope yields an empty
// TypesExport. The descriptor is decoded and validated, and the bytecode and types export
// are returned as fresh slices that do not share the input buffer.
//
// Takes data ([]byte) which is the raw envelope content.
//
// Returns *modules.ModuleBundle which is the parsed bundle.
// Returns error when the envelope is malformed or the descriptor is invalid.
func UnmarshalEnvelope(data []byte) (*modules.ModuleBundle, error) {
	if len(data) < envelopeMinLength {
		return nil, fmt.Errorf("%w: %d bytes, need at least %d", errInvalidEnvelope, len(data), envelopeMinLength)
	}
	if !bytes.HasPrefix(data, envelopeMagicPrefix) {
		return nil, fmt.Errorf("%w: bad magic prefix", errInvalidEnvelope)
	}
	switch version := data[envelopeVersionIndex]; version {
	case envelopeVersionOne:
		return unmarshalEnvelopeVersionOne(data)
	case envelopeVersionTwo:
		return unmarshalEnvelopeVersionTwo(data)
	default:
		return nil, fmt.Errorf("%w: unsupported envelope version %d", errInvalidEnvelope, version)
	}
}

// unmarshalEnvelopeVersionOne parses the original layout, in which the bytecode runs to
// the end of the file and no types export is stored.
//
// Takes data ([]byte) which is the envelope, already checked for length and magic.
//
// Returns *modules.ModuleBundle which carries an empty TypesExport.
// Returns error when the descriptor section overflows the file, the descriptor is
// invalid, or the bytecode section is empty.
func unmarshalEnvelopeVersionOne(data []byte) (*modules.ModuleBundle, error) {
	reader := &envelopeReader{data: data, offset: envelopeMagicLength}
	descriptorBytes, err := reader.readSection("descriptor")
	if err != nil {
		return nil, err
	}
	descriptor, err := modules.UnmarshalDescriptor(descriptorBytes)
	if err != nil {
		return nil, err
	}
	if reader.offset >= len(data) {
		return nil, fmt.Errorf("%w: bytecode section empty", errInvalidEnvelope)
	}
	return &modules.ModuleBundle{
		Descriptor:  descriptor,
		Bytecode:    bytes.Clone(data[reader.offset:]),
		TypesExport: nil,
	}, nil
}

// unmarshalEnvelopeVersionTwo parses the current layout of three length-prefixed
// sections.
//
// Takes data ([]byte) which is the envelope, already checked for length and magic.
//
// Returns *modules.ModuleBundle which is the parsed bundle.
// Returns error when a section overflows the file, trailing bytes follow the last
// section, the descriptor is invalid, or the bytecode section is empty.
func unmarshalEnvelopeVersionTwo(data []byte) (*modules.ModuleBundle, error) {
	reader := &envelopeReader{data: data, offset: envelopeMagicLength}
	descriptorBytes, err := reader.readSection("descriptor")
	if err != nil {
		return nil, err
	}
	bytecode, err := reader.readSection("bytecode")
	if err != nil {
		return nil, err
	}
	typesExport, err := reader.readSection("types export")
	if err != nil {
		return nil, err
	}
	if trailing := len(data) - reader.offset; trailing != 0 {
		return nil, fmt.Errorf("%w: %d unexpected trailing bytes", errInvalidEnvelope, trailing)
	}
	if len(bytecode) == 0 {
		return nil, fmt.Errorf("%w: bytecode section empty", errInvalidEnvelope)
	}
	descriptor, err := modules.UnmarshalDescriptor(descriptorBytes)
	if err != nil {
		return nil, err
	}
	var typesExportCopy []byte
	if len(typesExport) > 0 {
		typesExportCopy = bytes.Clone(typesExport)
	}
	return &modules.ModuleBundle{
		Descriptor:  descriptor,
		Bytecode:    bytes.Clone(bytecode),
		TypesExport: typesExportCopy,
	}, nil
}

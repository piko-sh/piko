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

package modules_domain

import (
	"encoding/binary"
	"errors"
	"fmt"
	"slices"
	"strings"

	"piko.sh/piko/wdk/safeconv"
)

const (
	// uint16Size is the byte length of a big-endian uint16 prefix in the wire format.
	uint16Size = 2

	// uint32Size is the byte length of a big-endian uint32 prefix in the wire format.
	uint32Size = 4

	// uint16MaxLen is the largest value encodable in a uint16 length prefix.
	uint16MaxLen = 0xFFFF
)

// TypesExportEntry carries per-package metadata for a single non-main package that the
// bundle makes importable.
//
// One entry per such package; multi-package modules (e.g. testify with
// assert/require/mock) produce multiple entries.
type TypesExportEntry struct {
	// FuncTable maps exported function names to their indices.
	//
	// The bytecode wire format only preserves a single entrypoints map (the last compiled
	// package's), so per-package indices ship alongside the types data - without them, a
	// multi-package module can register only the last sub-package's functions correctly.
	FuncTable map[string]uint16

	// ImportPath is the full import path under which the package is registered (e.g.
	// "github.com/stretchr/testify/assert").
	ImportPath string

	// Data is the gcexportdata.Write payload for the entry's go/types.Package, used to
	// rebuild type metadata when the bundle is loaded.
	Data []byte
}

// EncodeTypesExport serialises entries to TypesExport TLV bytes.
//
// Wire format:
//
//	count(uint32-BE)
//	  ( pathLen(uint16-BE) path
//	    dataLen(uint32-BE) data
//	    funcCount(uint32-BE)
//	      ( nameLen(uint16-BE) name funcIndex(uint16-BE) )*
//	  )*
//
// Entries are sorted by ImportPath, and each entry's funcTable names are sorted, so the
// encoded byte sequence is deterministic: bundle fingerprints stay stable across re-pack
// runs that walk maps in a different iteration order.
//
// Takes entries ([]TypesExportEntry) which are the per-package metadata records to
// encode.
//
// Returns []byte which is nil (not an empty slice) when entries is empty so
// ModuleBundle.TypesExport reads as "absent" downstream.
// Returns error when an ImportPath or function name exceeds the length limits encodable
// in the wire format.
func EncodeTypesExport(entries []TypesExportEntry) ([]byte, error) {
	if len(entries) == 0 {
		return nil, nil
	}
	sorted := make([]TypesExportEntry, len(entries))
	copy(sorted, entries)
	slices.SortFunc(sorted, func(a, b TypesExportEntry) int {
		return strings.Compare(a.ImportPath, b.ImportPath)
	})

	size := uint32Size
	for _, entry := range sorted {
		if len(entry.ImportPath) > uint16MaxLen {
			return nil, fmt.Errorf("modules_domain: TypesExport path too long: %d bytes", len(entry.ImportPath))
		}
		size += uint16Size + len(entry.ImportPath) + uint32Size + len(entry.Data) + uint32Size
		for name := range entry.FuncTable {
			if len(name) > uint16MaxLen {
				return nil, fmt.Errorf("modules_domain: TypesExport function name too long: %d bytes", len(name))
			}
			size += uint16Size + len(name) + uint16Size
		}
	}
	out := make([]byte, 0, size)
	out = binary.BigEndian.AppendUint32(out, safeconv.IntToUint32(len(sorted)))
	for _, entry := range sorted {
		out = binary.BigEndian.AppendUint16(out, safeconv.IntToUint16(len(entry.ImportPath)))
		out = append(out, entry.ImportPath...)
		out = binary.BigEndian.AppendUint32(out, safeconv.IntToUint32(len(entry.Data)))
		out = append(out, entry.Data...)
		names := make([]string, 0, len(entry.FuncTable))
		for name := range entry.FuncTable {
			names = append(names, name)
		}
		slices.Sort(names)
		out = binary.BigEndian.AppendUint32(out, safeconv.IntToUint32(len(names)))
		for _, name := range names {
			out = binary.BigEndian.AppendUint16(out, safeconv.IntToUint16(len(name)))
			out = append(out, name...)
			out = binary.BigEndian.AppendUint16(out, entry.FuncTable[name])
		}
	}
	return out, nil
}

// DecodeTypesExport reverses EncodeTypesExport.
//
// Empty data returns a nil slice and a nil error: empty TypesExport means "no per-package
// type metadata was packaged", which is a valid backward-compat state, not an error.
//
// Takes data ([]byte) which is the TLV-encoded payload.
//
// Returns []TypesExportEntry which is the decoded set.
// Returns error when data is truncated or declares fields extending past the end of the
// buffer.
func DecodeTypesExport(data []byte) ([]TypesExportEntry, error) {
	if len(data) == 0 {
		return nil, nil
	}
	if len(data) < uint32Size {
		return nil, errors.New("modules_domain: TypesExport truncated header")
	}
	count := binary.BigEndian.Uint32(data[:uint32Size])
	offset := uint32Size
	out := make([]TypesExportEntry, 0, count)
	for range count {
		entry, nextOffset, err := decodeTypesExportEntry(data, offset)
		if err != nil {
			return nil, err
		}
		out = append(out, entry)
		offset = nextOffset
	}
	if offset != len(data) {
		return nil, fmt.Errorf("modules_domain: TypesExport trailing %d bytes after %d entries", len(data)-offset, count)
	}
	return out, nil
}

// decodeTypesExportEntry decodes a single TLV entry beginning at offset.
//
// Takes data ([]byte) which is the full TLV payload.
// Takes offset (int) which is the start position of the entry.
//
// Returns TypesExportEntry which is the decoded entry.
// Returns int which is the offset immediately after the decoded entry.
// Returns error when any sub-field is truncated.
func decodeTypesExportEntry(data []byte, offset int) (TypesExportEntry, int, error) {
	path, offset, err := readLengthPrefixedString(data, offset, uint16Size, "path")
	if err != nil {
		return TypesExportEntry{}, 0, err
	}
	payload, offset, err := readLengthPrefixedBytes(data, offset, uint32Size, "data")
	if err != nil {
		return TypesExportEntry{}, 0, err
	}
	funcTable, offset, err := decodeFuncTable(data, offset)
	if err != nil {
		return TypesExportEntry{}, 0, err
	}
	return TypesExportEntry{
		ImportPath: path,
		Data:       payload,
		FuncTable:  funcTable,
	}, offset, nil
}

// decodeFuncTable decodes the (uint32 count, [name, uint16 index]*) tail section of a
// TypesExport entry.
//
// Takes data ([]byte) which is the full TLV payload.
// Takes offset (int) which marks the start of the funcTable count prefix.
//
// Returns map[string]uint16 which is the populated function table.
// Returns int which is the offset after the last function-name/index pair.
// Returns error when any sub-field is truncated.
func decodeFuncTable(data []byte, offset int) (map[string]uint16, int, error) {
	if offset+uint32Size > len(data) {
		return nil, 0, fmt.Errorf("modules_domain: TypesExport truncated funcTable count at offset %d", offset)
	}
	funcCount := binary.BigEndian.Uint32(data[offset : offset+uint32Size])
	offset += uint32Size
	funcTable := make(map[string]uint16, funcCount)
	for range funcCount {
		funcName, nextOffset, err := readLengthPrefixedString(data, offset, uint16Size, "funcName")
		if err != nil {
			return nil, 0, err
		}
		offset = nextOffset
		if offset+uint16Size > len(data) {
			return nil, 0, fmt.Errorf("modules_domain: TypesExport truncated funcIndex at offset %d", offset)
		}
		funcTable[funcName] = binary.BigEndian.Uint16(data[offset : offset+uint16Size])
		offset += uint16Size
	}
	return funcTable, offset, nil
}

// readLengthPrefixedBytes reads a length-prefixed byte slice from data.
//
// Takes data ([]byte) which holds the encoded payload.
// Takes offset (int) which marks the start of the length prefix.
// Takes prefixSize (int) which is uint16Size or uint32Size.
// Takes fieldName (string) which is used to build truncation error messages.
//
// Returns []byte which is a fresh copy of the payload bytes.
// Returns int which is the offset after the payload.
// Returns error when the prefix or payload is truncated.
func readLengthPrefixedBytes(data []byte, offset, prefixSize int, fieldName string) ([]byte, int, error) {
	if offset+prefixSize > len(data) {
		return nil, 0, fmt.Errorf("modules_domain: TypesExport truncated %s length at offset %d", fieldName, offset)
	}
	var payloadLen int
	switch prefixSize {
	case uint16Size:
		payloadLen = int(binary.BigEndian.Uint16(data[offset : offset+prefixSize]))
	case uint32Size:
		payloadLen = int(binary.BigEndian.Uint32(data[offset : offset+prefixSize]))
	default:
		return nil, 0, fmt.Errorf("modules_domain: TypesExport unsupported prefix size %d", prefixSize)
	}
	offset += prefixSize
	if offset+payloadLen > len(data) {
		return nil, 0, fmt.Errorf("modules_domain: TypesExport truncated %s at offset %d (need %d bytes)", fieldName, offset, payloadLen)
	}
	payload := make([]byte, payloadLen)
	copy(payload, data[offset:offset+payloadLen])
	return payload, offset + payloadLen, nil
}

// readLengthPrefixedString reads a length-prefixed UTF-8 string from data.
//
// Takes data ([]byte) which holds the encoded payload.
// Takes offset (int) which marks the start of the length prefix.
// Takes prefixSize (int) which is uint16Size or uint32Size.
// Takes fieldName (string) which is used to build truncation error messages.
//
// Returns string which is the decoded string.
// Returns int which is the offset after the string body.
// Returns error when the prefix or body is truncated.
func readLengthPrefixedString(data []byte, offset, prefixSize int, fieldName string) (string, int, error) {
	if offset+prefixSize > len(data) {
		return "", 0, fmt.Errorf("modules_domain: TypesExport truncated %s length at offset %d", fieldName, offset)
	}
	var bodyLen int
	switch prefixSize {
	case uint16Size:
		bodyLen = int(binary.BigEndian.Uint16(data[offset : offset+prefixSize]))
	case uint32Size:
		bodyLen = int(binary.BigEndian.Uint32(data[offset : offset+prefixSize]))
	default:
		return "", 0, fmt.Errorf("modules_domain: TypesExport unsupported prefix size %d", prefixSize)
	}
	offset += prefixSize
	if offset+bodyLen > len(data) {
		return "", 0, fmt.Errorf("modules_domain: TypesExport truncated %s at offset %d (need %d bytes)", fieldName, offset, bodyLen)
	}
	return string(data[offset : offset+bodyLen]), offset + bodyLen, nil
}

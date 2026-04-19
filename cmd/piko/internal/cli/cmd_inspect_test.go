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

package cli

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunInspect_MissingArgs(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		arguments []string
	}{
		{name: "no arguments", arguments: nil},
		{name: "type only", arguments: []string{"manifest"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var stdout, stderr bytes.Buffer
			code := RunInspectWithIO(tc.arguments, &stdout, &stderr)
			assert.Equal(t, 1, code, "exit code = %d, want 1", code)
			assert.Contains(t, stderr.String(), "piko inspect", "stderr should contain usage text, got: %s", stderr.String())
		})
	}
}

func TestRunInspect_UnknownType(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	code := RunInspectWithIO([]string{"nonexistent", "file.bin"}, &stdout, &stderr)
	assert.Equal(t, 1, code, "exit code = %d, want 1", code)
	assert.Contains(t, stderr.String(), "Unknown type", "stderr should mention unknown type, got: %s", stderr.String())
	assert.Contains(t, stderr.String(), "nonexistent", "stderr should include the type name, got: %s", stderr.String())
}

func TestRunInspect_UnknownFlag(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	code := RunInspectWithIO([]string{"wal", "file.wal", "--bogus"}, &stdout, &stderr)
	assert.Equal(t, 1, code, "exit code = %d, want 1", code)
	assert.Contains(t, stderr.String(), "--bogus", "stderr should mention the unknown flag, got: %s", stderr.String())
}

func TestRunInspect_EffectiveOnNonWAL(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	code := RunInspectWithIO([]string{"manifest", "file.bin", "--effective"}, &stdout, &stderr)
	assert.Equal(t, 1, code, "exit code = %d, want 1", code)
	assert.Contains(t, stderr.String(), "--effective", "stderr should mention --effective, got: %s", stderr.String())
}

func TestRunInspect_FileNotFound(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	code := RunInspectWithIO([]string{"manifest", "/nonexistent/path/file.bin"}, &stdout, &stderr)
	assert.Equal(t, 1, code, "exit code = %d, want 1", code)
	assert.Contains(t, stderr.String(), "Error", "stderr should contain error, got: %s", stderr.String())
}

func TestRunInspect_Usage(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer
	inspectUsage(&buffer)
	output := buffer.String()

	types := []string{"manifest", "i18n", "collection", "search", "wal"}
	for _, typ := range types {
		assert.Contains(t, output, typ, "usage text should contain type %q", typ)
	}

	assert.Contains(t, output, "--compact", "usage text should mention --compact flag")
	assert.Contains(t, output, "--effective", "usage text should mention --effective flag")
	assert.Contains(t, output, "--parse-values", "usage text should mention --parse-values flag")
}

func TestInspectHandlers_AllRegistered(t *testing.T) {
	t.Parallel()

	expectedTypes := []string{"manifest", "i18n", "collection", "search", "wal"}
	for _, typ := range expectedTypes {
		_, ok := inspectHandlers[typ]
		assert.True(t, ok, "inspectHandlers missing type %q", typ)
	}
}

func TestRunInspect_WAL(t *testing.T) {
	t.Parallel()

	data := buildTestWALEntry("my-key", "my-value", 1, 1700000000000000000, 0, nil)

	directory := t.TempDir()
	err := os.WriteFile(filepath.Join(directory, "test.wal"), data, 0o600)
	require.NoError(t, err)

	var stdout, stderr bytes.Buffer
	code := RunInspectWithIO([]string{"wal", filepath.Join(directory, "test.wal")}, &stdout, &stderr)
	require.Equal(t, 0, code, "exit code = %d, stderr: %s", code, stderr.String())

	output := stdout.String()
	assert.Contains(t, output, "my-key", "output should contain key, got: %s", output)
	assert.Contains(t, output, "my-value", "output should contain value, got: %s", output)
	assert.Contains(t, output, "SET", "output should contain operation, got: %s", output)
	assert.Contains(t, output, `"entryCount": 1`, "output should contain entryCount, got: %s", output)
}

func TestRunInspect_WAL_Empty(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	err := os.WriteFile(filepath.Join(directory, "empty.wal"), nil, 0o600)
	require.NoError(t, err)

	var stdout, stderr bytes.Buffer
	code := RunInspectWithIO([]string{"wal", filepath.Join(directory, "empty.wal")}, &stdout, &stderr)
	require.Equal(t, 0, code, "exit code = %d, stderr: %s", code, stderr.String())

	assert.Contains(t, stdout.String(), `"entryCount": 0`, "output should contain entryCount 0, got: %s", stdout.String())
}

func TestRunInspect_WAL_Effective(t *testing.T) {
	t.Parallel()

	data := slices.Concat(
		buildTestWALEntry("key1", "v1", 1, 1000000000000000000, 0, nil),
		buildTestWALEntry("key2", "only", 1, 1000000000000000001, 0, nil),
		buildTestWALEntry("key1", "v2", 1, 1000000000000000002, 0, nil),
	)

	directory := t.TempDir()
	err := os.WriteFile(filepath.Join(directory, "test.wal"), data, 0o600)
	require.NoError(t, err)

	walFile := filepath.Join(directory, "test.wal")

	var stdout, stderr bytes.Buffer
	code := RunInspectWithIO([]string{"wal", walFile, "--effective"}, &stdout, &stderr)
	require.Equal(t, 0, code, "exit code = %d, stderr: %s", code, stderr.String())

	output := stdout.String()
	assert.Contains(t, output, `"entryCount": 2`, "effective should have 2 entries, got: %s", output)
	assert.Contains(t, output, `"originalEntryCount": 3`, "should report original count of 3, got: %s", output)

	assert.NotContains(t, output, `"v1"`, "effective should not contain superseded value v1, got: %s", output)
	assert.Contains(t, output, `"v2"`, "effective should contain latest value v2, got: %s", output)

	stdout.Reset()
	stderr.Reset()
	code = RunInspectWithIO([]string{"--effective", "wal", walFile}, &stdout, &stderr)
	require.Equal(t, 0, code, "flag-first: exit code = %d, stderr: %s", code, stderr.String())
	assert.Contains(t, stdout.String(), `"entryCount": 2`, "flag-first: effective should have 2 entries, got: %s", stdout.String())
}

func TestRunInspect_WAL_ParseValues(t *testing.T) {
	t.Parallel()

	jsonValue := `{"name":"widget","count":3}`
	data := buildTestWALEntry("my-key", jsonValue, 1, 1700000000000000000, 0, nil)

	directory := t.TempDir()
	walFile := filepath.Join(directory, "test.wal")
	err := os.WriteFile(walFile, data, 0o600)
	require.NoError(t, err)

	var stdout, stderr bytes.Buffer
	code := RunInspectWithIO([]string{"wal", walFile}, &stdout, &stderr)
	require.Equal(t, 0, code, "exit code = %d, stderr: %s", code, stderr.String())

	assert.Contains(t, stdout.String(), `"value": "{\"name\":\"widget\"`, "without --parse-values, value should be a string, got: %s", stdout.String())

	stdout.Reset()
	stderr.Reset()
	code = RunInspectWithIO([]string{"wal", walFile, "--parse-values"}, &stdout, &stderr)
	require.Equal(t, 0, code, "exit code = %d, stderr: %s", code, stderr.String())
	output := stdout.String()

	assert.Contains(t, output, `"name": "widget"`, "with --parse-values, value should be parsed JSON, got: %s", output)
	assert.Contains(t, output, `"count": 3`, "with --parse-values, parsed value should have count, got: %s", output)
}

func TestRunInspect_WAL_ParseValues_NonJSON(t *testing.T) {
	t.Parallel()

	data := buildTestWALEntry("key", "just-a-string", 1, 1700000000000000000, 0, nil)

	directory := t.TempDir()
	walFile := filepath.Join(directory, "test.wal")
	err := os.WriteFile(walFile, data, 0o600)
	require.NoError(t, err)

	var stdout, stderr bytes.Buffer
	code := RunInspectWithIO([]string{"wal", walFile, "--parse-values"}, &stdout, &stderr)
	require.Equal(t, 0, code, "exit code = %d, stderr: %s", code, stderr.String())
	assert.Contains(t, stdout.String(), `"just-a-string"`, "non-JSON value should remain a string, got: %s", stdout.String())
}

func TestParseWALValues(t *testing.T) {
	t.Parallel()

	result := parseWALValues(walInspectResult{
		EntryCount: 3,
		FileSize:   100,
		Entries: []walInspectEntry{
			{Index: 0, Key: "json-obj", Value: `{"a":1}`},
			{Index: 1, Key: "json-arr", Value: `[1,2,3]`},
			{Index: 2, Key: "plain", Value: "hello"},
		},
	})

	_, ok := result.Entries[0].Value.(map[string]any)
	assert.True(t, ok, "entry 0 value should be map, got %T", result.Entries[0].Value)

	_, ok = result.Entries[1].Value.([]any)
	assert.True(t, ok, "entry 1 value should be slice, got %T", result.Entries[1].Value)

	v, ok := result.Entries[2].Value.(string)
	assert.False(t, !ok || v != "hello", "entry 2 value should be string 'hello', got %v (%T)", result.Entries[2].Value, result.Entries[2].Value)
}

func TestParseWALValues_ScalarsUnchanged(t *testing.T) {
	t.Parallel()

	result := parseWALValues(walInspectResult{
		EntryCount: 2,
		FileSize:   50,
		Entries: []walInspectEntry{
			{Index: 0, Key: "num", Value: "42"},
			{Index: 1, Key: "bool", Value: "true"},
		},
	})

	v, ok := result.Entries[0].Value.(string)
	assert.False(t, !ok || v != "42", "numeric string should stay as string, got %v (%T)", result.Entries[0].Value, result.Entries[0].Value)
	v, ok = result.Entries[1].Value.(string)
	assert.False(t, !ok || v != "true", "boolean string should stay as string, got %v (%T)", result.Entries[1].Value, result.Entries[1].Value)
}

func TestEffectiveWALResult_DeduplicatesKeys(t *testing.T) {
	t.Parallel()

	full := walInspectResult{
		EntryCount: 4,
		FileSize:   1000,
		Entries: []walInspectEntry{
			{Index: 0, Operation: "SET", Key: "a", Value: "a1"},
			{Index: 1, Operation: "SET", Key: "b", Value: "b1"},
			{Index: 2, Operation: "SET", Key: "a", Value: "a2"},
			{Index: 3, Operation: "SET", Key: "c", Value: "c1"},
		},
	}

	result := effectiveWALResult(full)

	assert.Equal(t, 3, result.EntryCount, "entryCount = %d, want 3", result.EntryCount)
	assert.Equal(t, 4, result.OriginalEntryCount, "originalEntryCount = %d, want 4", result.OriginalEntryCount)
	assert.Equal(t, 1000, result.FileSize, "fileSize = %d, want 1000", result.FileSize)

	keys := make([]string, len(result.Entries))
	for i, e := range result.Entries {
		keys[i] = e.Key + "=" + e.Value.(string)
	}
	want := "b=b1,a=a2,c=c1"
	got := strings.Join(keys, ",")
	assert.Equal(t, want, got, "effective entries = %s, want %s", got, want)
}

func TestEffectiveWALResult_RespectsDelete(t *testing.T) {
	t.Parallel()

	full := walInspectResult{
		EntryCount: 3,
		FileSize:   500,
		Entries: []walInspectEntry{
			{Index: 0, Operation: "SET", Key: "a", Value: "val"},
			{Index: 1, Operation: "SET", Key: "b", Value: "val"},
			{Index: 2, Operation: "DELETE", Key: "a"},
		},
	}

	result := effectiveWALResult(full)

	assert.Equal(t, 2, result.EntryCount, "entryCount = %d, want 2", result.EntryCount)

	assert.Equal(t, "DELETE", result.Entries[1].Operation, "entry for 'a' op = %s, want DELETE", result.Entries[1].Operation)
}

func TestEffectiveWALResult_DiscardBeforeClear(t *testing.T) {
	t.Parallel()

	full := walInspectResult{
		EntryCount: 4,
		FileSize:   800,
		Entries: []walInspectEntry{
			{Index: 0, Operation: "SET", Key: "old", Value: "stale"},
			{Index: 1, Operation: "CLEAR", Key: ""},
			{Index: 2, Operation: "SET", Key: "new1", Value: "fresh1"},
			{Index: 3, Operation: "SET", Key: "new2", Value: "fresh2"},
		},
	}

	result := effectiveWALResult(full)

	assert.Equal(t, 2, result.EntryCount, "entryCount = %d, want 2", result.EntryCount)
	for _, e := range result.Entries {
		assert.NotEqual(t, "old", e.Key, "entry before CLEAR should be discarded")
	}
}

func TestEffectiveWALResult_Empty(t *testing.T) {
	t.Parallel()

	result := effectiveWALResult(walInspectResult{FileSize: 0})
	assert.Equal(t, 0, result.EntryCount, "entryCount = %d, want 0", result.EntryCount)
}

func TestDisplayBytes(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		want string
		in   []byte
	}{
		{name: "empty", in: nil, want: ""},
		{name: "valid utf8", in: []byte("hello"), want: "hello"},
		{name: "invalid utf8", in: []byte{0xff, 0xfe}, want: "0xfffe"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := displayBytes(tc.in)
			assert.Equal(t, tc.want, got, "displayBytes(%v) = %q, want %q", tc.in, got, tc.want)
		})
	}
}

const (
	walFormatVersionForTest uint8 = 1
)

func buildTestWALEntry(key, value string, op uint8, timestamp, expiresAt int64, tags []string) []byte {
	keyBytes := []byte(key)
	valueBytes := []byte(value)

	tagsSize := 0
	for _, tag := range tags {
		tagsSize += 2 + len(tag)
	}

	payloadSize := 1 + 1 + 8 + 8 + 4 + len(keyBytes) + 4 + len(valueBytes) + 2 + tagsSize
	payload := make([]byte, payloadSize)

	offset := 0
	payload[offset] = walFormatVersionForTest
	offset++
	payload[offset] = op
	offset++
	binary.BigEndian.PutUint64(payload[offset:], uint64(timestamp))
	offset += 8
	binary.BigEndian.PutUint64(payload[offset:], uint64(expiresAt))
	offset += 8
	binary.BigEndian.PutUint32(payload[offset:], uint32(len(keyBytes)))
	offset += 4
	copy(payload[offset:], keyBytes)
	offset += len(keyBytes)
	binary.BigEndian.PutUint32(payload[offset:], uint32(len(valueBytes)))
	offset += 4
	copy(payload[offset:], valueBytes)
	offset += len(valueBytes)
	binary.BigEndian.PutUint16(payload[offset:], uint16(len(tags)))
	offset += 2
	for _, tag := range tags {
		binary.BigEndian.PutUint16(payload[offset:], uint16(len(tag)))
		offset += 2
		copy(payload[offset:], tag)
		offset += len(tag)
	}

	checksum := crc32.Checksum(payload, crc32.MakeTable(crc32.IEEE))

	crcAndPayload := make([]byte, 4+len(payload))
	binary.BigEndian.PutUint32(crcAndPayload[:4], checksum)
	copy(crcAndPayload[4:], payload)

	result := make([]byte, 4+len(crcAndPayload))
	binary.BigEndian.PutUint32(result[:4], uint32(len(crcAndPayload)))
	copy(result[4:], crcAndPayload)

	return result
}

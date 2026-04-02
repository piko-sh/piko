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
			if code != 1 {
				t.Errorf("exit code = %d, want 1", code)
			}
			if !strings.Contains(stderr.String(), "piko inspect") {
				t.Errorf("stderr should contain usage text, got: %s", stderr.String())
			}
		})
	}
}

func TestRunInspect_UnknownType(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	code := RunInspectWithIO([]string{"nonexistent", "file.bin"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "Unknown type") {
		t.Errorf("stderr should mention unknown type, got: %s", stderr.String())
	}
	if !strings.Contains(stderr.String(), "nonexistent") {
		t.Errorf("stderr should include the type name, got: %s", stderr.String())
	}
}

func TestRunInspect_UnknownFlag(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	code := RunInspectWithIO([]string{"wal", "file.wal", "--bogus"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "--bogus") {
		t.Errorf("stderr should mention the unknown flag, got: %s", stderr.String())
	}
}

func TestRunInspect_EffectiveOnNonWAL(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	code := RunInspectWithIO([]string{"manifest", "file.bin", "--effective"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "--effective") {
		t.Errorf("stderr should mention --effective, got: %s", stderr.String())
	}
}

func TestRunInspect_FileNotFound(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	code := RunInspectWithIO([]string{"manifest", "/nonexistent/path/file.bin"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "Error") {
		t.Errorf("stderr should contain error, got: %s", stderr.String())
	}
}

func TestRunInspect_Usage(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer
	inspectUsage(&buffer)
	output := buffer.String()

	types := []string{"manifest", "i18n", "collection", "search", "wal"}
	for _, typ := range types {
		if !strings.Contains(output, typ) {
			t.Errorf("usage text should contain type %q", typ)
		}
	}

	if !strings.Contains(output, "--compact") {
		t.Error("usage text should mention --compact flag")
	}
	if !strings.Contains(output, "--effective") {
		t.Error("usage text should mention --effective flag")
	}
	if !strings.Contains(output, "--parse-values") {
		t.Error("usage text should mention --parse-values flag")
	}
}

func TestInspectHandlers_AllRegistered(t *testing.T) {
	t.Parallel()

	expectedTypes := []string{"manifest", "i18n", "collection", "search", "wal"}
	for _, typ := range expectedTypes {
		if _, ok := inspectHandlers[typ]; !ok {
			t.Errorf("inspectHandlers missing type %q", typ)
		}
	}
}

func TestRunInspect_WAL(t *testing.T) {
	t.Parallel()

	data := buildTestWALEntry("my-key", "my-value", 1, 1700000000000000000, 0, nil)

	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "test.wal"), data, 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := RunInspectWithIO([]string{"wal", filepath.Join(directory, "test.wal")}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr: %s", code, stderr.String())
	}

	output := stdout.String()
	if !strings.Contains(output, "my-key") {
		t.Errorf("output should contain key, got: %s", output)
	}
	if !strings.Contains(output, "my-value") {
		t.Errorf("output should contain value, got: %s", output)
	}
	if !strings.Contains(output, "SET") {
		t.Errorf("output should contain operation, got: %s", output)
	}
	if !strings.Contains(output, `"entryCount": 1`) {
		t.Errorf("output should contain entryCount, got: %s", output)
	}
}

func TestRunInspect_WAL_Empty(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "empty.wal"), nil, 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := RunInspectWithIO([]string{"wal", filepath.Join(directory, "empty.wal")}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr: %s", code, stderr.String())
	}

	if !strings.Contains(stdout.String(), `"entryCount": 0`) {
		t.Errorf("output should contain entryCount 0, got: %s", stdout.String())
	}
}

func TestRunInspect_WAL_Effective(t *testing.T) {
	t.Parallel()

	data := slices.Concat(
		buildTestWALEntry("key1", "v1", 1, 1000000000000000000, 0, nil),
		buildTestWALEntry("key2", "only", 1, 1000000000000000001, 0, nil),
		buildTestWALEntry("key1", "v2", 1, 1000000000000000002, 0, nil),
	)

	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "test.wal"), data, 0o600); err != nil {
		t.Fatal(err)
	}

	walFile := filepath.Join(directory, "test.wal")

	var stdout, stderr bytes.Buffer
	code := RunInspectWithIO([]string{"wal", walFile, "--effective"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr: %s", code, stderr.String())
	}

	output := stdout.String()
	if !strings.Contains(output, `"entryCount": 2`) {
		t.Errorf("effective should have 2 entries, got: %s", output)
	}
	if !strings.Contains(output, `"originalEntryCount": 3`) {
		t.Errorf("should report original count of 3, got: %s", output)
	}

	if strings.Contains(output, `"v1"`) {
		t.Errorf("effective should not contain superseded value v1, got: %s", output)
	}
	if !strings.Contains(output, `"v2"`) {
		t.Errorf("effective should contain latest value v2, got: %s", output)
	}

	stdout.Reset()
	stderr.Reset()
	code = RunInspectWithIO([]string{"--effective", "wal", walFile}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("flag-first: exit code = %d, stderr: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"entryCount": 2`) {
		t.Errorf("flag-first: effective should have 2 entries, got: %s", stdout.String())
	}
}

func TestRunInspect_WAL_ParseValues(t *testing.T) {
	t.Parallel()

	jsonValue := `{"name":"widget","count":3}`
	data := buildTestWALEntry("my-key", jsonValue, 1, 1700000000000000000, 0, nil)

	directory := t.TempDir()
	walFile := filepath.Join(directory, "test.wal")
	if err := os.WriteFile(walFile, data, 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := RunInspectWithIO([]string{"wal", walFile}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr: %s", code, stderr.String())
	}

	if !strings.Contains(stdout.String(), `"value": "{\"name\":\"widget\"`) {
		t.Errorf("without --parse-values, value should be a string, got: %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = RunInspectWithIO([]string{"wal", walFile, "--parse-values"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr: %s", code, stderr.String())
	}
	output := stdout.String()

	if !strings.Contains(output, `"name": "widget"`) {
		t.Errorf("with --parse-values, value should be parsed JSON, got: %s", output)
	}
	if !strings.Contains(output, `"count": 3`) {
		t.Errorf("with --parse-values, parsed value should have count, got: %s", output)
	}
}

func TestRunInspect_WAL_ParseValues_NonJSON(t *testing.T) {
	t.Parallel()

	data := buildTestWALEntry("key", "just-a-string", 1, 1700000000000000000, 0, nil)

	directory := t.TempDir()
	walFile := filepath.Join(directory, "test.wal")
	if err := os.WriteFile(walFile, data, 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := RunInspectWithIO([]string{"wal", walFile, "--parse-values"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"just-a-string"`) {
		t.Errorf("non-JSON value should remain a string, got: %s", stdout.String())
	}
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

	if _, ok := result.Entries[0].Value.(map[string]any); !ok {
		t.Errorf("entry 0 value should be map, got %T", result.Entries[0].Value)
	}

	if _, ok := result.Entries[1].Value.([]any); !ok {
		t.Errorf("entry 1 value should be slice, got %T", result.Entries[1].Value)
	}

	if v, ok := result.Entries[2].Value.(string); !ok || v != "hello" {
		t.Errorf("entry 2 value should be string 'hello', got %v (%T)", result.Entries[2].Value, result.Entries[2].Value)
	}
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

	if v, ok := result.Entries[0].Value.(string); !ok || v != "42" {
		t.Errorf("numeric string should stay as string, got %v (%T)", result.Entries[0].Value, result.Entries[0].Value)
	}
	if v, ok := result.Entries[1].Value.(string); !ok || v != "true" {
		t.Errorf("boolean string should stay as string, got %v (%T)", result.Entries[1].Value, result.Entries[1].Value)
	}
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

	if result.EntryCount != 3 {
		t.Errorf("entryCount = %d, want 3", result.EntryCount)
	}
	if result.OriginalEntryCount != 4 {
		t.Errorf("originalEntryCount = %d, want 4", result.OriginalEntryCount)
	}
	if result.FileSize != 1000 {
		t.Errorf("fileSize = %d, want 1000", result.FileSize)
	}

	keys := make([]string, len(result.Entries))
	for i, e := range result.Entries {
		keys[i] = e.Key + "=" + e.Value.(string)
	}
	want := "b=b1,a=a2,c=c1"
	got := strings.Join(keys, ",")
	if got != want {
		t.Errorf("effective entries = %s, want %s", got, want)
	}
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

	if result.EntryCount != 2 {
		t.Errorf("entryCount = %d, want 2", result.EntryCount)
	}

	if result.Entries[1].Operation != "DELETE" {
		t.Errorf("entry for 'a' op = %s, want DELETE", result.Entries[1].Operation)
	}
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

	if result.EntryCount != 2 {
		t.Errorf("entryCount = %d, want 2", result.EntryCount)
	}
	for _, e := range result.Entries {
		if e.Key == "old" {
			t.Error("entry before CLEAR should be discarded")
		}
	}
}

func TestEffectiveWALResult_Empty(t *testing.T) {
	t.Parallel()

	result := effectiveWALResult(walInspectResult{FileSize: 0})
	if result.EntryCount != 0 {
		t.Errorf("entryCount = %d, want 0", result.EntryCount)
	}
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
			if got != tc.want {
				t.Errorf("displayBytes(%v) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

const walFormatVersionForTest uint8 = 1

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

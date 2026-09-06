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
	"context"
	"encoding/binary"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/wdk/modules"
	"piko.sh/piko/wdk/safedisk"
)

func helperBundle(path, version string) *modules.ModuleBundle {
	return &modules.ModuleBundle{
		Descriptor: &modules.ModuleDescriptor{
			SchemaVersion: modules.DescriptorVersion,
			Ref:           modules.ModuleRef{Path: path, Version: version},
			Capabilities:  modules.CapabilitySet{{Axis: "network"}},
		},
		Bytecode: []byte("piko-bytecode-" + path + "-" + version),
	}
}

func versionOneEnvelope(t *testing.T, bundle *modules.ModuleBundle) []byte {
	t.Helper()
	descriptorBytes, err := bundle.Descriptor.MarshalCanonicalJSON()
	require.NoError(t, err)
	out := append([]byte{}, envelopeMagicPrefix...)
	out = append(out, envelopeVersionOne)
	out = binary.BigEndian.AppendUint32(out, uint32(len(descriptorBytes)))
	out = append(out, descriptorBytes...)
	return append(out, bundle.Bytecode...)
}

func newTestProvider(t *testing.T, options ...Option) *Provider {
	t.Helper()
	provider, err := New(t.TempDir(), options...)
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, provider.Close()) })
	return provider
}

func TestEnvelopeRoundTripPreservesBundle(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name        string
		typesExport []byte
	}{
		{name: "with types export", typesExport: []byte("types-export-payload")},
		{name: "without types export", typesExport: nil},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			original := helperBundle("example.com/mod", "v1.2.3")
			original.TypesExport = testCase.typesExport
			wantFingerprint, err := original.Fingerprint()
			require.NoError(t, err)

			envelope, err := MarshalEnvelope(original)
			require.NoError(t, err)
			assert.Equal(t, envelopeVersionTwo, envelope[envelopeVersionIndex])

			roundTrip, err := UnmarshalEnvelope(envelope)
			require.NoError(t, err)
			assert.Equal(t, original.Bytecode, roundTrip.Bytecode)
			assert.Equal(t, original.TypesExport, roundTrip.TypesExport)
			assert.Equal(t, original.Descriptor.Ref, roundTrip.Descriptor.Ref)

			gotFingerprint, err := roundTrip.Fingerprint()
			require.NoError(t, err)
			assert.Equal(t, wantFingerprint, gotFingerprint)
		})
	}
}

func TestUnmarshalEnvelopeReadsVersionOne(t *testing.T) {
	t.Parallel()
	original := helperBundle("example.com/legacy", "v0.1.0")
	roundTrip, err := UnmarshalEnvelope(versionOneEnvelope(t, original))
	require.NoError(t, err)
	assert.Equal(t, original.Bytecode, roundTrip.Bytecode)
	assert.Empty(t, roundTrip.TypesExport)
	assert.Equal(t, original.Descriptor.Ref, roundTrip.Descriptor.Ref)
}

func TestUnmarshalEnvelopeRejectsMalformed(t *testing.T) {
	t.Parallel()
	valid, err := MarshalEnvelope(helperBundle("example.com/mod", "v1"))
	require.NoError(t, err)
	descriptorLength := int(binary.BigEndian.Uint32(valid[envelopeMagicLength:]))
	bytecodePrefixOffset := envelopeMagicLength + envelopeLengthPrefixSize + descriptorLength

	withBytecodeLength := func(length uint32) []byte {
		mutated := append([]byte{}, valid...)
		binary.BigEndian.PutUint32(mutated[bytecodePrefixOffset:], length)
		return mutated
	}
	withTypesLength := func(length uint32) []byte {
		mutated := append([]byte{}, valid...)
		binary.BigEndian.PutUint32(mutated[len(mutated)-envelopeLengthPrefixSize:], length)
		return mutated
	}
	withVersion := func(version byte) []byte {
		mutated := append([]byte{}, valid...)
		mutated[envelopeVersionIndex] = version
		return mutated
	}
	emptyBytecode := append([]byte{}, valid[:bytecodePrefixOffset]...)
	emptyBytecode = binary.BigEndian.AppendUint32(emptyBytecode, 0)
	emptyBytecode = binary.BigEndian.AppendUint32(emptyBytecode, 0)
	versionOneWithoutBytecode := versionOneEnvelope(t, helperBundle("example.com/mod", "v1"))
	versionOneWithoutBytecode = versionOneWithoutBytecode[:len(versionOneWithoutBytecode)-len(helperBundle("example.com/mod", "v1").Bytecode)]

	testCases := []struct {
		name string
		data []byte
	}{
		{name: "empty", data: []byte("")},
		{name: "short", data: []byte("PKBND")},
		{name: "bad magic", data: []byte("XXXXXX\x00\x00\x00\x05hello-world")},
		{name: "unknown version", data: withVersion(0x09)},
		{name: "truncated descriptor length", data: []byte("PKBND\x02\x00\x00\x00\x00\x00")},
		{name: "descriptor length overflows file", data: []byte("PKBND\x01\x00\x00\xff\xffabc")},
		{name: "bytecode length overflows file", data: withBytecodeLength(1 << 30)},
		{name: "types export length overflows file", data: withTypesLength(1 << 30)},
		{name: "trailing bytes", data: append(append([]byte{}, valid...), 0x00)},
		{name: "missing types export section", data: valid[:len(valid)-envelopeLengthPrefixSize]},
		{name: "empty bytecode section", data: emptyBytecode},
		{name: "version one without bytecode", data: versionOneWithoutBytecode},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			_, err := UnmarshalEnvelope(testCase.data)
			require.Error(t, err)
		})
	}
}

func TestMarshalEnvelopeRejectsInvalidBundle(t *testing.T) {
	t.Parallel()
	bundle := helperBundle("example.com/mod", "v1")
	bundle.Bytecode = nil
	_, err := MarshalEnvelope(bundle)
	require.Error(t, err)
}

func TestWriteThenResolveRoundTrip(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name    string
		version string
	}{
		{name: "explicit version", version: "v1.2.3"},
		{name: "empty version stored as latest", version: ""},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			provider := newTestProvider(t)
			bundle := helperBundle("example.com/mod", testCase.version)
			bundle.TypesExport = []byte("types")
			require.NoError(t, provider.Write(bundle))

			got, err := provider.Resolve(context.Background(), bundle.Descriptor.Ref)
			require.NoError(t, err)
			assert.Equal(t, bundle.Bytecode, got.Bytecode)
			assert.Equal(t, bundle.TypesExport, got.TypesExport)
		})
	}
}

func TestResolveErrors(t *testing.T) {
	t.Parallel()
	cancelled, cancel := context.WithCancelCause(context.Background())
	cancel(errors.New("test cancelled the resolve"))

	testCases := []struct {
		ctx     context.Context
		wantErr error
		name    string
		ref     modules.ModuleRef
	}{
		{name: "missing module", ctx: context.Background(), ref: modules.ModuleRef{Path: "missing", Version: "v1"}, wantErr: modules.ErrModuleNotFound},
		{name: "empty path", ctx: context.Background(), ref: modules.ModuleRef{Version: "v1"}, wantErr: modules.ErrModuleNotFound},
		{name: "cancelled context", ctx: cancelled, ref: modules.ModuleRef{Path: "example.com/mod"}, wantErr: context.Canceled},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			provider := newTestProvider(t)
			_, err := provider.Resolve(testCase.ctx, testCase.ref)
			require.ErrorIs(t, err, testCase.wantErr)
		})
	}
}

func TestResolveRejectsTraversalPath(t *testing.T) {
	t.Parallel()
	provider := newTestProvider(t)
	_, err := provider.Resolve(context.Background(), modules.ModuleRef{Path: "../escape", Version: "v1"})
	require.Error(t, err)
	assert.NotErrorIs(t, err, modules.ErrModuleNotFound)
}

func TestResolveEnforcesBundleSizeLimit(t *testing.T) {
	t.Parallel()
	bundle := helperBundle("example.com/mod", "v1")
	envelope, err := MarshalEnvelope(bundle)
	require.NoError(t, err)
	size := int64(len(envelope))

	testCases := []struct {
		name    string
		limit   int64
		wantErr bool
	}{
		{name: "limit equals file size", limit: size, wantErr: false},
		{name: "limit one byte short", limit: size - 1, wantErr: true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			provider := newTestProvider(t, WithMaxBundleBytes(testCase.limit))
			require.NoError(t, provider.Write(bundle))
			_, err := provider.Resolve(context.Background(), bundle.Descriptor.Ref)
			if testCase.wantErr {
				require.ErrorIs(t, err, errBundleTooLarge)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestWithMaxBundleBytesIgnoresNonPositive(t *testing.T) {
	t.Parallel()
	provider := NewWithSandbox(safedisk.NewMockSandbox("/bundles", safedisk.ModeReadWrite), WithMaxBundleBytes(0))
	assert.Equal(t, defaultMaxBundleBytes, provider.maxBundleBytes)
}

func TestResolveReportsOpenFailure(t *testing.T) {
	t.Parallel()
	openFailure := errors.New("open failed")
	sandbox := safedisk.NewMockSandbox("/bundles", safedisk.ModeReadWrite)
	sandbox.OpenErr = openFailure
	_, err := NewWithSandbox(sandbox).Resolve(context.Background(), modules.ModuleRef{Path: "example.com/mod", Version: "v1"})
	require.ErrorIs(t, err, openFailure)
}

func TestWriteReportsSandboxFailures(t *testing.T) {
	t.Parallel()
	mkdirFailure := errors.New("mkdir failed")
	writeFailure := errors.New("write failed")

	testCases := []struct {
		configure func(sandbox *safedisk.MockSandbox)
		bundle    *modules.ModuleBundle
		wantErr   error
		name      string
	}{
		{
			name:      "mkdir failure",
			configure: func(sandbox *safedisk.MockSandbox) { sandbox.MkdirAllErr = mkdirFailure },
			bundle:    helperBundle("example.com/mod", "v1"),
			wantErr:   mkdirFailure,
		},
		{
			name:      "write failure",
			configure: func(sandbox *safedisk.MockSandbox) { sandbox.WriteFileAtomicErr = writeFailure },
			bundle:    helperBundle("example.com/mod", "v1"),
			wantErr:   writeFailure,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			sandbox := safedisk.NewMockSandbox("/bundles", safedisk.ModeReadWrite)
			testCase.configure(sandbox)
			require.ErrorIs(t, NewWithSandbox(sandbox).Write(testCase.bundle), testCase.wantErr)
		})
	}
}

func TestWriteRejectsInvalidBundles(t *testing.T) {
	t.Parallel()
	traversal := helperBundle("../escape", "v1")
	missingBytecode := helperBundle("example.com/mod", "v1")
	missingBytecode.Bytecode = nil

	testCases := []struct {
		bundle *modules.ModuleBundle
		name   string
	}{
		{name: "traversal path", bundle: traversal},
		{name: "missing bytecode", bundle: missingBytecode},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			provider := NewWithSandbox(safedisk.NewMockSandbox("/bundles", safedisk.ModeReadWrite))
			require.Error(t, provider.Write(testCase.bundle))
		})
	}
}

func TestCloseOnlyReleasesOwnedSandbox(t *testing.T) {
	t.Parallel()
	sandbox := safedisk.NewMockSandbox("/bundles", safedisk.ModeReadWrite)
	require.NoError(t, NewWithSandbox(sandbox).Close())
	assert.Zero(t, sandbox.CallCounts["Close"])

	closeFailure := errors.New("close failed")
	owned := NewWithSandbox(sandbox)
	owned.ownsSandbox = true
	sandbox.CloseErr = closeFailure
	require.ErrorIs(t, owned.Close(), closeFailure)
	assert.Equal(t, 1, sandbox.CallCounts["Close"])
}

func TestNewRejectsEmptyRoot(t *testing.T) {
	t.Parallel()
	_, err := New("")
	require.Error(t, err)
}

func TestEncodeModuleFolder(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name    string
		path    string
		want    string
		wantErr bool
	}{
		{name: "nested path", path: "github.com/foo/bar", want: "github.com__foo__bar"},
		{name: "traversal", path: "../escape", wantErr: true},
		{name: "empty", path: "", wantErr: true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			got, err := encodeModuleFolder(testCase.path)
			if testCase.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, testCase.want, got)
		})
	}
}

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

package daemon_frontend

import (
	"io/fs"
	"slices"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRuntimeVersion_IsAValidVersionOfTheEmbeddedBundles(t *testing.T) {
	t.Parallel()

	version := RuntimeVersion()

	require.NotEmpty(t, version, "the embedded bundles must always yield a runtime version")
	assert.True(t, IsRuntimeVersion(version))
	assert.Equal(t, version, RuntimeVersion(), "the version must be stable across calls")
}

func TestRuntimeVersion_FingerprintsEveryEmbeddedSidecar(t *testing.T) {
	t.Parallel()

	sidecars, err := fingerprintedSidecars(runtimeSidecars)
	require.NoError(t, err)
	require.NotEmpty(t, sidecars)

	for _, name := range []string{
		"built/ppframework.core.es.js.map.etag",
		"built/ppframework.core.es.js.sri",
		"built/ppframework.core.min.es.js.sri",
		"built/ppframework.components.es.js.sri",
		"built/ppframework.animation.es.js.sri",
	} {
		assert.Contains(t, sidecars, name, "the runtime version must cover %s", name)
	}
}

func TestComputeRuntimeVersion(t *testing.T) {
	t.Parallel()

	base := fstest.MapFS{
		"built/a.js.sri":      {Data: []byte("sha384-aaa\n")},
		"built/b.js.sri":      {Data: []byte("sha384-bbb")},
		"built/a.js.map.etag": {Data: []byte(`"map-1"`)},
	}

	baseVersion, err := computeRuntimeVersion(base)
	require.NoError(t, err)
	require.True(t, IsRuntimeVersion(baseVersion))

	testCases := []struct {
		wantErr     error
		sidecars    fstest.MapFS
		name        string
		wantSame    bool
		wantAnError bool
	}{
		{
			name: "surrounding whitespace in a sidecar does not change the version",
			sidecars: fstest.MapFS{
				"built/a.js.sri":      {Data: []byte("  sha384-aaa  ")},
				"built/b.js.sri":      {Data: []byte("sha384-bbb\n")},
				"built/a.js.map.etag": {Data: []byte(`"map-1"` + "\n")},
			},
			wantSame: true,
		},
		{
			name: "a changed source map changes the version",
			sidecars: fstest.MapFS{
				"built/a.js.sri":      {Data: []byte("sha384-aaa")},
				"built/b.js.sri":      {Data: []byte("sha384-bbb")},
				"built/a.js.map.etag": {Data: []byte(`"map-2"`)},
			},
			wantSame: false,
		},
		{
			name: "a changed bundle hash changes the version",
			sidecars: fstest.MapFS{
				"built/a.js.sri": {Data: []byte("sha384-aaa")},
				"built/b.js.sri": {Data: []byte("sha384-ccc")},
			},
			wantSame: false,
		},
		{
			name: "a renamed bundle changes the version",
			sidecars: fstest.MapFS{
				"built/a.js.sri": {Data: []byte("sha384-aaa")},
				"built/c.js.sri": {Data: []byte("sha384-bbb")},
			},
			wantSame: false,
		},
		{
			name: "an added bundle changes the version",
			sidecars: fstest.MapFS{
				"built/a.js.sri": {Data: []byte("sha384-aaa")},
				"built/b.js.sri": {Data: []byte("sha384-bbb")},
				"built/c.js.sri": {Data: []byte("sha384-ccc")},
			},
			wantSame: false,
		},
		{
			name: "files that are not sidecars are ignored",
			sidecars: fstest.MapFS{
				"built/a.js.sri":      {Data: []byte("sha384-aaa")},
				"built/b.js.sri":      {Data: []byte("sha384-bbb")},
				"built/a.js.map.etag": {Data: []byte(`"map-1"`)},
				"built/b.js.etag":     {Data: []byte(`"changed"`)},
				"built/b.js":          {Data: []byte("console.log(1)")},
			},
			wantSame: true,
		},
		{
			name:        "no sidecars is an error",
			sidecars:    fstest.MapFS{"built/a.js": {Data: []byte("x")}},
			wantErr:     errNoRuntimeSidecars,
			wantAnError: true,
		},
		{
			name:        "source maps without bundle sidecars is an error",
			sidecars:    fstest.MapFS{"built/a.js.map.etag": {Data: []byte(`"map-1"`)}},
			wantErr:     errNoRuntimeSidecars,
			wantAnError: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			version, versionErr := computeRuntimeVersion(testCase.sidecars)
			if testCase.wantAnError {
				require.ErrorIs(t, versionErr, testCase.wantErr)
				assert.Empty(t, version)
				return
			}
			require.NoError(t, versionErr)
			assert.True(t, IsRuntimeVersion(version))
			if testCase.wantSame {
				assert.Equal(t, baseVersion, version)
			} else {
				assert.NotEqual(t, baseVersion, version)
			}
		})
	}
}

func TestIsRuntimeVersion(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		segment string
		want    bool
	}{
		{name: "sixteen lowercase hex characters", segment: "0123456789abcdef", want: true},
		{name: "too short", segment: "0123456789abcde", want: false},
		{name: "too long", segment: "0123456789abcdef0", want: false},
		{name: "uppercase hex", segment: "0123456789ABCDEF", want: false},
		{name: "not hex", segment: "0123456789abcdeg", want: false},
		{name: "a file name", segment: "ppframework.core", want: false},
		{name: "empty", segment: "", want: false},
		{name: "the typegen directory", segment: "ts", want: false},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, testCase.want, IsRuntimeVersion(testCase.segment))
		})
	}
}

func TestSplitRuntimeVersion(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		assetPath   string
		wantVersion string
		wantFile    string
		wantOK      bool
	}{
		{
			name:        "versioned bundle",
			assetPath:   "0123456789abcdef/ppframework.core.es.js",
			wantVersion: "0123456789abcdef",
			wantFile:    "ppframework.core.es.js",
			wantOK:      true,
		},
		{
			name:        "versioned source map",
			assetPath:   "0123456789abcdef/ppframework.core.es.js.map",
			wantVersion: "0123456789abcdef",
			wantFile:    "ppframework.core.es.js.map",
			wantOK:      true,
		},
		{name: "unversioned bundle", assetPath: "ppframework.core.es.js", wantOK: false},
		{name: "directory that is not a version", assetPath: "ts/actions.d.ts", wantOK: false},
		{name: "empty", assetPath: "", wantOK: false},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			version, fileName, ok := SplitRuntimeVersion(testCase.assetPath)
			assert.Equal(t, testCase.wantOK, ok)
			assert.Equal(t, testCase.wantVersion, version)
			assert.Equal(t, testCase.wantFile, fileName)
		})
	}
}

func TestRuntimeAssetURLFor(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "/_piko/dist/0123456789abcdef/ppframework.core.es.js",
		runtimeAssetURLFor("0123456789abcdef", "ppframework.core.es.js"))
	assert.Equal(t, "/_piko/dist/ppframework.core.es.js",
		runtimeAssetURLFor("", "ppframework.core.es.js"),
		"without a version the unversioned URL is the safe fallback")
}

func TestModuleURLs_CarryTheirVersions(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "/_piko/dist/ppframework.components.es.js", ComponentsModuleImportURL(),
		"compiled components import the components runtime without a version, since they do not know the build")
	assert.Equal(t, "/_piko/dist/ppframework.animation.es.js", AnimationModuleImportURL())
	assert.Equal(t, "/_piko/dist/"+CoreRuntimeBuildProduction.RuntimeVersion()+"/ppframework.components.es.js", CoreRuntimeBuildProduction.ComponentsModuleURL())
	assert.Equal(t, "/_piko/dist/ppframework.core.es.js", CoreModuleImportURL(),
		"generated modules import the core without a version, since they do not know the build")
	assert.Equal(t, "/_piko/dist/"+CoreRuntimeBuildProduction.RuntimeVersion()+"/ppframework.core.es.js", CoreRuntimeBuildProduction.CoreModuleURL())
}

func TestStripRuntimeVersion(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		text string
		want string
	}{
		{
			name: "versioned import",
			text: `import {piko} from "/_piko/dist/0123456789abcdef/ppframework.core.es.js";`,
			want: `import {piko} from "/_piko/dist/ppframework.core.es.js";`,
		},
		{
			name: "several URLs",
			text: `<link href="/_piko/dist/0123456789abcdef/a.js"><script src="/_piko/dist/fedcba9876543210/b.js">`,
			want: `<link href="/_piko/dist/a.js"><script src="/_piko/dist/b.js">`,
		},
		{
			name: "unversioned URLs are left alone",
			text: `<script src="/_piko/dist/ppframework.core.es.js"></script>`,
			want: `<script src="/_piko/dist/ppframework.core.es.js"></script>`,
		},
		{
			name: "a segment that only looks like part of a version is left alone",
			text: `/_piko/dist/0123456789abcde/x.js and /_piko/dist/`,
			want: `/_piko/dist/0123456789abcde/x.js and /_piko/dist/`,
		},
		{
			name: "text without bundle URLs",
			text: "plain text",
			want: "plain text",
		},
		{
			name: "empty",
			text: "",
			want: "",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, testCase.want, StripRuntimeVersion(testCase.text))
		})
	}
}

func TestRuntimeVersionError_IsNilForTheEmbeddedBundles(t *testing.T) {
	t.Parallel()

	assert.NoError(t, runtimeVersionError())
}

func TestComputeRuntimeVersion_DoesNotDependOnListingOrder(t *testing.T) {
	t.Parallel()

	sidecars := fstest.MapFS{
		"built/a.js.sri": {Data: []byte("sha384-aaa")},
		"built/b.js.sri": {Data: []byte("sha384-bbb")},
		"built/c.js.sri": {Data: []byte("sha384-ccc")},
	}

	sorted, err := computeRuntimeVersion(sidecars)
	require.NoError(t, err)

	reversed, err := computeRuntimeVersion(reversedListingFS{MapFS: sidecars})
	require.NoError(t, err)

	assert.Equal(t, sorted, reversed)
}

type reversedListingFS struct {
	fstest.MapFS
}

func (reversed reversedListingFS) ReadDir(name string) ([]fs.DirEntry, error) {
	entries, err := reversed.MapFS.ReadDir(name)
	if err != nil {
		return nil, err
	}
	slices.Reverse(entries)
	return entries, nil
}

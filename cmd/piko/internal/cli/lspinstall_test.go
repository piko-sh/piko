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
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/json"
)

func TestAssetName(t *testing.T) {
	t.Parallel()

	plat := platform{os: "linux", arch: "amd64", archiveExt: "tar.gz", binaryFile: "pikopls"}
	got := assetName("0.0.0-alpha.26", plat)
	want := "pikopls-0.0.0-alpha.26-linux-amd64.tar.gz"
	assert.Equal(t, want, got, "assetName() = %q, want %q", got, want)
}

func TestCurrentPlatform(t *testing.T) {
	t.Parallel()

	plat, err := currentPlatform()
	if err != nil {
		t.Skipf("host platform unsupported: %v", err)
	}
	assert.False(t, plat.os == "" || plat.arch == "" || plat.binaryFile == "" || plat.archiveExt == "", "incomplete platform: %+v", plat)
}

func TestExtractFromTarGz(t *testing.T) {
	t.Parallel()

	content := []byte("#!/bin/sh\necho pikopls\n")
	archive := makeTarGz(t, "pikopls", content)

	got, err := extractFromTarGz(archive, "pikopls", maxExtractedBinaryBytes)
	require.NoError(t, err, "extractFromTarGz() = %v", err)
	assert.True(t, bytes.Equal(got, content), "extracted content does not match")
}

func TestExtractFromTarGz_MissingBinary(t *testing.T) {
	t.Parallel()

	archive := makeTarGz(t, "something-else", []byte("x"))
	_, err := extractFromTarGz(archive, "pikopls", maxExtractedBinaryBytes)
	require.ErrorIs(t, err, errBinaryNotInArchive)
}

func TestEnsureInstalled_AlreadyOnPath(t *testing.T) {
	original := lookPath
	t.Cleanup(func() { lookPath = original })
	lookPath = func(string) (string, error) { return "/usr/local/bin/pikopls", nil }

	status, binaryPath := ensureInstalled(t.Context(), "1.2.3")
	assert.Contains(t, status, "already on PATH", "status = %q, want it to report the binary is already on PATH", status)
	assert.Equal(t, "/usr/local/bin/pikopls", binaryPath)
}

func TestEnsureInstalled_DownloadsAndInstalls(t *testing.T) {
	plat, err := currentPlatform()
	if err != nil {
		t.Skipf("host platform unsupported: %v", err)
	}

	const version = "9.9.9"
	content := []byte("fake pikopls executable")

	var archive []byte
	if plat.archiveExt == "zip" {
		archive = makeZip(t, plat.binaryFile, content)
	} else {
		archive = makeTarGz(t, plat.binaryFile, content)
	}

	asset := assetName(version, plat)
	serveRelease(t, version, map[string][]byte{
		asset:              archive,
		checksumsAssetName: checksumListing(asset, archive),
	})

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PATH", filepath.Join(home, ".local", "bin"))

	status, binaryPath := ensureInstalled(t.Context(), version)

	installed := filepath.Join(home, ".local", "bin", plat.binaryFile)
	got, err := os.ReadFile(installed)
	require.NoError(t, err, "binary not installed (status %q): %v", status, err)
	assert.True(t, bytes.Equal(got, content), "installed binary content does not match the archived binary")
	assert.Contains(t, status, "installed pikopls "+version, "status = %q, want it to confirm install of %s", status, version)
	assert.Equal(t, installed, binaryPath)
}

func TestEnsureInstalled_FailsClosedOnChecksumProblems(t *testing.T) {
	plat, err := currentPlatform()
	if err != nil {
		t.Skipf("host platform unsupported: %v", err)
	}

	const version = "9.9.9"
	archive := makeTarGz(t, plat.binaryFile, []byte("fake pikopls executable"))
	if plat.archiveExt == "zip" {
		archive = makeZip(t, plat.binaryFile, []byte("fake pikopls executable"))
	}
	asset := assetName(version, plat)

	testCases := []struct {
		files map[string][]byte
		name  string
	}{
		{
			name: "mismatched checksum",
			files: map[string][]byte{
				asset:              archive,
				checksumsAssetName: checksumListing(asset, []byte("different bytes")),
			},
		},
		{
			name: "checksum file missing",
			files: map[string][]byte{
				asset: archive,
			},
		},
		{
			name: "asset not listed",
			files: map[string][]byte{
				asset:              archive,
				checksumsAssetName: checksumListing("some-other-asset.tar.gz", archive),
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			serveRelease(t, version, testCase.files)

			home := t.TempDir()
			t.Setenv("HOME", home)
			t.Setenv("PATH", filepath.Join(home, ".local", "bin"))

			status, binaryPath := ensureInstalled(t.Context(), version)

			assert.Empty(t, binaryPath)
			assert.Contains(t, status, "could not download pikopls")
			assert.Contains(t, status, "checksum")
			_, statErr := os.Stat(filepath.Join(home, ".local", "bin", plat.binaryFile))
			assert.ErrorIs(t, statErr, fs.ErrNotExist, "an unverified binary must not be installed")
		})
	}
}

func TestDownloadAndExtract_ChecksumSentinels(t *testing.T) {
	plat := platform{os: "linux", arch: "amd64", archiveExt: "tar.gz", binaryFile: "pikopls"}
	const version = "1.0.0"
	asset := assetName(version, plat)
	archive := makeTarGz(t, "pikopls", []byte("binary"))

	serveRelease(t, version, map[string][]byte{
		asset:              archive,
		checksumsAssetName: checksumListing(asset, []byte("tampered")),
	})
	_, err := downloadAndExtract(t.Context(), version, plat)
	require.ErrorIs(t, err, errChecksumMismatch)

	serveRelease(t, version, map[string][]byte{asset: archive})
	_, err = downloadAndExtract(t.Context(), version, plat)
	require.ErrorIs(t, err, errChecksumUnavailable)

	serveRelease(t, version, map[string][]byte{
		asset:              archive,
		checksumsAssetName: checksumListing(asset, archive),
	})
	binary, err := downloadAndExtract(t.Context(), version, plat)
	require.NoError(t, err)
	assert.Equal(t, []byte("binary"), binary)
}

func TestHTTPGetBounded_RejectsOversizedBody(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(bytes.Repeat([]byte("x"), 4096))
	}))
	t.Cleanup(server.Close)

	_, err := httpGetBounded(t.Context(), testGet(server.URL, 1024))
	require.ErrorIs(t, err, errDownloadTooLarge)

	body, err := httpGetBounded(t.Context(), testGet(server.URL, 4096))
	require.NoError(t, err)
	assert.Len(t, body, 4096)
}

func TestExtractFromZip_RejectsZipBomb(t *testing.T) {
	t.Parallel()

	const limit = 64 * 1024
	bomb := makeZip(t, "pikopls.exe", make([]byte, 4*limit))
	assert.Less(t, len(bomb), limit, "the archive compresses far below the limit")

	_, err := extractFromZip(bomb, "pikopls.exe", limit)
	require.ErrorIs(t, err, errBinaryTooLarge)

	small, err := extractFromZip(makeZip(t, "pikopls.exe", []byte("ok")), "pikopls.exe", limit)
	require.NoError(t, err)
	assert.Equal(t, []byte("ok"), small)
}

func TestExtractFromTarGz_RejectsOversizedBinary(t *testing.T) {
	t.Parallel()

	const limit = 64 * 1024
	bomb := makeTarGz(t, "pikopls", make([]byte, 4*limit))
	assert.Less(t, len(bomb), limit, "the archive compresses far below the limit")

	_, err := extractFromTarGz(bomb, "pikopls", limit)
	require.ErrorIs(t, err, errBinaryTooLarge)
}

func TestLookupChecksum(t *testing.T) {
	t.Parallel()

	digest := strings.Repeat("ab", 32)
	listing := []byte("\n" + digest + "  pikopls-1.0.0-linux-amd64.tar.gz\n" +
		strings.Repeat("cd", 32) + " *pikopls-1.0.0-windows-amd64.zip\n" +
		"not-hex  broken.tar.gz\n")

	found, ok := lookupChecksum(listing, "pikopls-1.0.0-linux-amd64.tar.gz")
	assert.True(t, ok)
	assert.Equal(t, digest, found)

	found, ok = lookupChecksum(listing, "pikopls-1.0.0-windows-amd64.zip")
	assert.True(t, ok, "binary-mode entries are recognised")
	assert.Equal(t, strings.Repeat("cd", 32), found)

	_, ok = lookupChecksum(listing, "broken.tar.gz")
	assert.False(t, ok, "malformed digests are ignored")

	_, ok = lookupChecksum(listing, "absent.tar.gz")
	assert.False(t, ok)
}

func TestPinLSPCommand(t *testing.T) {
	t.Parallel()

	pluginDir := t.TempDir()
	original := `{
  "piko": {
    "command": "pikopls",
    "extensionToLanguage": {".pk": "piko", ".pkc": "piko-component"},
    "initializationOptions": {"goBridge": true}
  }
}`
	require.NoError(t, os.WriteFile(filepath.Join(pluginDir, lspConfigFileName), []byte(original), 0o644))

	binaryPath := filepath.Join(t.TempDir(), "bin", "pikopls")
	require.NoError(t, pinLSPCommand(pluginDir, binaryPath))

	rewritten, err := os.ReadFile(filepath.Join(pluginDir, lspConfigFileName))
	require.NoError(t, err)

	var document map[string]map[string]any
	require.NoError(t, json.Unmarshal(rewritten, &document))
	assert.Equal(t, binaryPath, document["piko"]["command"])
	assert.Equal(t, map[string]any{".pk": "piko", ".pkc": "piko-component"}, document["piko"]["extensionToLanguage"])
	assert.Equal(t, map[string]any{"goBridge": true}, document["piko"]["initializationOptions"])
}

func TestPinCommandInLSPConfig_RejectsMissingServer(t *testing.T) {
	t.Parallel()

	_, err := pinCommandInLSPConfig([]byte(`{"other": {"command": "x"}}`), "/abs/pikopls")
	require.ErrorIs(t, err, errLSPConfigMissingServer)

	_, err = pinCommandInLSPConfig([]byte(`not json`), "/abs/pikopls")
	require.Error(t, err)
}

func serveRelease(t *testing.T, version string, files map[string][]byte) {
	t.Helper()

	prefix := "/v" + version + "/"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		content, ok := files[strings.TrimPrefix(r.URL.Path, prefix)]
		if !ok || !strings.HasPrefix(r.URL.Path, prefix) {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write(content)
	}))
	t.Cleanup(server.Close)

	originalURL, originalLook, originalAPI := releaseBaseURL, lookPath, releasesAPIURL
	t.Cleanup(func() { releaseBaseURL, lookPath, releasesAPIURL = originalURL, originalLook, originalAPI })
	releaseBaseURL = server.URL
	releasesAPIURL = server.URL + "/releases"
	lookPath = func(string) (string, error) { return "", exec.ErrNotFound }
}

func checksumListing(asset string, content []byte) []byte {
	digest := sha256.Sum256(content)
	return []byte(hex.EncodeToString(digest[:]) + "  " + asset + "\n")
}

func makeTarGz(t *testing.T, name string, content []byte) []byte {
	t.Helper()

	var buffer bytes.Buffer
	gzw := gzip.NewWriter(&buffer)
	tw := tar.NewWriter(gzw)

	header := &tar.Header{Name: name, Mode: 0o755, Size: int64(len(content)), Typeflag: tar.TypeReg}
	err := tw.WriteHeader(header)
	require.NoError(t, err)
	_, err = tw.Write(content)
	require.NoError(t, err)
	err = tw.Close()
	require.NoError(t, err)
	err = gzw.Close()
	require.NoError(t, err)
	return buffer.Bytes()
}

func makeZip(t *testing.T, name string, content []byte) []byte {
	t.Helper()

	var buffer bytes.Buffer
	zw := zip.NewWriter(&buffer)
	writer, err := zw.Create(name)
	require.NoError(t, err)
	_, err = writer.Write(content)
	require.NoError(t, err)
	err = zw.Close()
	require.NoError(t, err)
	return buffer.Bytes()
}

func TestPlatformFor(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		goos     string
		goarch   string
		expected platform
		wantErr  bool
	}{
		{name: "linux amd64", goos: "linux", goarch: "amd64", expected: platform{os: "linux", arch: "amd64", archiveExt: "tar.gz", binaryFile: "pikopls"}},
		{name: "darwin arm64", goos: "darwin", goarch: "arm64", expected: platform{os: "darwin", arch: "arm64", archiveExt: "tar.gz", binaryFile: "pikopls"}},
		{name: "windows amd64", goos: "windows", goarch: "amd64", expected: platform{os: "windows", arch: "amd64", archiveExt: "zip", binaryFile: "pikopls.exe"}},
		{name: "unsupported operating system", goos: "plan9", goarch: "amd64", wantErr: true},
		{name: "unsupported architecture", goos: "linux", goarch: "riscv64", wantErr: true},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got, err := platformFor(testCase.goos, testCase.goarch)
			if testCase.wantErr {
				require.ErrorIs(t, err, errUnsupportedPlatform)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, testCase.expected, got)
		})
	}
}

func TestValidateVersion(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		version string
		wantErr bool
	}{
		{name: "release", version: "1.2.3"},
		{name: "prerelease with build metadata", version: "0.0.0-alpha.26+build.7"},
		{name: "empty", version: "", wantErr: true},
		{name: "path traversal", version: "../../evil", wantErr: true},
		{name: "query string", version: "1.0.0?x=y", wantErr: true},
		{name: "development build marker", version: "(devel)", wantErr: true},
		{name: "too long", version: strings.Repeat("1", maxVersionLength+1), wantErr: true},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			err := validateVersion(testCase.version)
			if testCase.wantErr {
				require.ErrorIs(t, err, errInvalidVersion)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestPickLatestVersion(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		expected string
		releases []githubRelease
		wantErr  bool
	}{
		{
			name: "first stable release wins over a newer prerelease",
			releases: []githubRelease{
				{TagName: "v2.0.0-rc.1", Prerelease: true},
				{TagName: "v1.9.0"},
			},
			expected: "1.9.0",
		},
		{
			name: "newest prerelease is the fallback",
			releases: []githubRelease{
				{TagName: "v2.0.0-rc.2", Prerelease: true},
				{TagName: "v2.0.0-rc.1", Prerelease: true},
			},
			expected: "2.0.0-rc.2",
		},
		{
			name: "drafts and unusable tags are skipped",
			releases: []githubRelease{
				{TagName: "v3.0.0", Draft: true},
				{TagName: ""},
				{TagName: "v../evil"},
				{TagName: "v1.0.0"},
			},
			expected: "1.0.0",
		},
		{name: "no releases", releases: nil, wantErr: true},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got, err := pickLatestVersion(testCase.releases)
			if testCase.wantErr {
				require.ErrorIs(t, err, errNoPublishedRelease)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, testCase.expected, got)
		})
	}
}

func TestFetchBinary_FallsBackToLatestRelease(t *testing.T) {
	plat := platform{os: "linux", arch: "amd64", archiveExt: "tar.gz", binaryFile: "pikopls"}
	const latest = "2.0.0"
	asset := assetName(latest, plat)
	archive := makeTarGz(t, "pikopls", []byte("latest binary"))
	listing, err := json.Marshal([]githubRelease{{TagName: "v" + latest}})
	require.NoError(t, err)

	serveRelease(t, latest, map[string][]byte{
		asset:              archive,
		checksumsAssetName: checksumListing(asset, archive),
	})
	serveReleasesAPI(t, listing)

	binary, resolved, err := fetchBinary(t.Context(), "(devel)", plat)
	require.NoError(t, err)
	assert.Equal(t, latest, resolved)
	assert.Equal(t, []byte("latest binary"), binary)

	_, _, err = fetchBinary(t.Context(), "v"+latest+"-missing", plat)
	require.NoError(t, err, "an unreleased exact version falls back to the latest release")
}

func TestFetchBinary_LatestReleaseLacksPlatform(t *testing.T) {
	plat := platform{os: "linux", arch: "amd64", archiveExt: "tar.gz", binaryFile: "pikopls"}
	listing, err := json.Marshal([]githubRelease{{TagName: "v1.0.0"}})
	require.NoError(t, err)

	serveRelease(t, "1.0.0", map[string][]byte{})
	serveReleasesAPI(t, listing)

	_, _, err = fetchBinary(t.Context(), "1.0.0", plat)
	require.ErrorIs(t, err, errReleaseMissingPlatform)
}

func TestFetchBinary_LatestLookupFails(t *testing.T) {
	plat := platform{os: "linux", arch: "amd64", archiveExt: "tar.gz", binaryFile: "pikopls"}
	serveRelease(t, "1.0.0", map[string][]byte{})

	_, _, err := fetchBinary(t.Context(), "1.0.0", plat)
	require.ErrorIs(t, err, errUnexpectedStatus)
}

func TestLatestVersion_RejectsMalformedListing(t *testing.T) {
	serveRelease(t, "1.0.0", map[string][]byte{})
	serveReleasesAPI(t, []byte("not json"))

	_, err := latestVersion(t.Context())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decoding releases")
}

func TestHTTPGetBounded(t *testing.T) {
	t.Parallel()

	t.Run("reports a non-200 status as a sentinel error", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte("rate limited"))
		}))
		t.Cleanup(server.Close)

		_, err := httpGetBounded(t.Context(), testGet(server.URL, 1024))
		require.ErrorIs(t, err, errUnexpectedStatus)
		assert.Contains(t, err.Error(), "403")
	})

	t.Run("sends the accept header when one is set", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(r.Header.Get("Accept") + "|" + r.Header.Get("User-Agent")))
		}))
		t.Cleanup(server.Close)

		request := testGet(server.URL, 1024)
		request.accept = githubAPIAccept
		body, err := httpGetBounded(t.Context(), request)
		require.NoError(t, err)
		assert.Equal(t, githubAPIAccept+"|"+userAgent, string(body))
	})

	t.Run("reports the timeout cause", func(t *testing.T) {
		t.Parallel()

		release := make(chan struct{})
		server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			select {
			case <-release:
			case <-r.Context().Done():
			}
		}))
		t.Cleanup(func() {
			close(release)
			server.Close()
		})

		request := testGet(server.URL, 1024)
		request.timeout = 50 * time.Millisecond
		_, err := httpGetBounded(t.Context(), request)
		require.ErrorIs(t, err, errDownloadTimeout)
	})

	t.Run("rejects a malformed URL", func(t *testing.T) {
		t.Parallel()

		_, err := httpGetBounded(t.Context(), testGet("http://bad host/", 1024))
		require.Error(t, err)
	})
}

func TestWithContextCause(t *testing.T) {
	t.Parallel()

	baseErr := errors.New("transport failed")

	live := t.Context()
	assert.Same(t, baseErr, withContextCause(live, baseErr), "a live context adds nothing")

	cause := errors.New("took too long")
	ended, cancel := context.WithCancelCause(t.Context())
	cancel(cause)
	joined := withContextCause(ended, baseErr)
	require.ErrorIs(t, joined, baseErr)
	require.ErrorIs(t, joined, cause)

	alreadyWrapped := fmt.Errorf("wrapped: %w", cause)
	assert.Same(t, alreadyWrapped, withContextCause(ended, alreadyWrapped), "the cause is not added twice")
}

func TestOverflowErrorReader(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("overflow")

	exact := &overflowErrorReader{reader: strings.NewReader("abcd"), overflowErr: sentinel, remaining: 4}
	data, err := io.ReadAll(exact)
	require.NoError(t, err, "a stream exactly at the limit ends cleanly")
	assert.Equal(t, "abcd", string(data))

	over := &overflowErrorReader{reader: strings.NewReader("abcdef"), overflowErr: sentinel, remaining: 4}
	_, err = io.ReadAll(over)
	require.ErrorIs(t, err, sentinel, "a stream past the limit fails rather than looking shorter")
}

func TestExtractFromTarGz_StreamOverflow(t *testing.T) {
	t.Parallel()

	const limit = 1024
	var buffer bytes.Buffer
	gzipWriter := gzip.NewWriter(&buffer)
	tarWriter := tar.NewWriter(gzipWriter)
	padding := make([]byte, 2*tarFramingHeadroom)
	require.NoError(t, tarWriter.WriteHeader(&tar.Header{Name: "padding", Mode: 0o644, Size: int64(len(padding)), Typeflag: tar.TypeReg}))
	_, err := tarWriter.Write(padding)
	require.NoError(t, err)
	require.NoError(t, tarWriter.Close())
	require.NoError(t, gzipWriter.Close())

	_, err = extractFromTarGz(buffer.Bytes(), "pikopls", limit)
	require.ErrorIs(t, err, errBinaryTooLarge, "a bomb in an earlier entry is cut off with a clear error")
}

func TestExtractFromTarGz_TooManyEntries(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer
	gzipWriter := gzip.NewWriter(&buffer)
	tarWriter := tar.NewWriter(gzipWriter)
	for index := range maxArchiveEntries + 1 {
		require.NoError(t, tarWriter.WriteHeader(&tar.Header{Name: fmt.Sprintf("entry-%d", index), Mode: 0o644, Typeflag: tar.TypeReg}))
	}
	require.NoError(t, tarWriter.Close())
	require.NoError(t, gzipWriter.Close())

	_, err := extractFromTarGz(buffer.Bytes(), "pikopls", maxExtractedBinaryBytes)
	require.ErrorIs(t, err, errTooManyArchiveEntries)
}

func TestExtractFromTarGz_RejectsNonGzip(t *testing.T) {
	t.Parallel()

	_, err := extractFromTarGz([]byte("not gzip"), "pikopls", maxExtractedBinaryBytes)
	require.Error(t, err)
}

func TestExtractFromZip_MissingBinaryAndCorruptArchive(t *testing.T) {
	t.Parallel()

	_, err := extractFromZip(makeZip(t, "other.exe", []byte("x")), "pikopls.exe", maxExtractedBinaryBytes)
	require.ErrorIs(t, err, errBinaryNotInArchive)

	_, err = extractFromZip([]byte("not a zip"), "pikopls.exe", maxExtractedBinaryBytes)
	require.Error(t, err)
}

func TestExtractBinary_DispatchesOnArchiveFormat(t *testing.T) {
	t.Parallel()

	zipPlatform := platform{os: "windows", arch: "amd64", archiveExt: zipArchiveExtension, binaryFile: "pikopls.exe"}
	binary, err := extractBinary(makeZip(t, "pikopls.exe", []byte("zip")), zipPlatform)
	require.NoError(t, err)
	assert.Equal(t, []byte("zip"), binary)

	tarPlatform := platform{os: "linux", arch: "amd64", archiveExt: tarGzArchiveExtension, binaryFile: "pikopls"}
	binary, err = extractBinary(makeTarGz(t, "pikopls", []byte("tar")), tarPlatform)
	require.NoError(t, err)
	assert.Equal(t, []byte("tar"), binary)
}

func TestPinCommandInLSPConfig_PreservesOtherFields(t *testing.T) {
	t.Parallel()

	original := []byte(`{"piko": {"command": "pikopls", "startupTimeout": 12345678901234567}, "other": [1, 2]}`)

	rewritten, err := pinCommandInLSPConfig(original, "/abs/pikopls")
	require.NoError(t, err)
	assert.Contains(t, string(rewritten), "12345678901234567", "large integers survive byte for byte")
	assert.Contains(t, string(rewritten), `"/abs/pikopls"`)
	assert.True(t, bytes.HasSuffix(rewritten, []byte("\n")))

	var document map[string]any
	require.NoError(t, json.Unmarshal(rewritten, &document))
	assert.Equal(t, []any{float64(1), float64(2)}, document["other"])
}

func TestPinCommandInLSPConfig_RejectsMalformedServerEntry(t *testing.T) {
	t.Parallel()

	_, err := pinCommandInLSPConfig([]byte(`{"piko": "pikopls"}`), "/abs/pikopls")
	require.ErrorIs(t, err, errLSPConfigMissingServer)

	_, err = pinCommandInLSPConfig([]byte(`{"piko": null}`), "/abs/pikopls")
	require.ErrorIs(t, err, errLSPConfigMissingServer)
}

func TestPinLSPCommand_Errors(t *testing.T) {
	t.Parallel()

	require.ErrorIs(t, pinLSPCommand(t.TempDir(), " "), errEmptyBinaryPath)

	err := pinLSPCommand(t.TempDir(), "/abs/pikopls")
	require.Error(t, err, "a plugin directory without .lsp.json cannot be pinned")

	pluginDirectory := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(pluginDirectory, lspConfigFileName), []byte(`{"other": {}}`), 0o644))
	require.ErrorIs(t, pinLSPCommand(pluginDirectory, "/abs/pikopls"), errLSPConfigMissingServer)
}

func TestChooseInstallDirectory(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	localBin := filepath.Join(home, ".local", "bin")

	t.Setenv("PATH", localBin)
	directory, onPath, err := chooseInstallDirectory(t.Context())
	require.NoError(t, err)
	assert.Equal(t, localBin, directory)
	assert.True(t, onPath)
	_, statErr := os.Stat(filepath.Join(localBin, writeProbeFileName))
	assert.ErrorIs(t, statErr, fs.ErrNotExist, "the write probe is removed")

	t.Setenv("PATH", t.TempDir())
	directory, onPath, err = chooseInstallDirectory(t.Context())
	require.NoError(t, err)
	assert.Equal(t, localBin, directory)
	assert.False(t, onPath)
}

func TestIsWritable_RejectsUncreatableDirectory(t *testing.T) {
	t.Parallel()

	blocker := filepath.Join(t.TempDir(), "file")
	require.NoError(t, os.WriteFile(blocker, []byte("x"), 0o600))

	assert.False(t, isWritable(t.Context(), filepath.Join(blocker, "child")))
}

func TestEnsureInstalled_ReportsDownloadFailure(t *testing.T) {
	if _, err := currentPlatform(); err != nil {
		t.Skipf("host platform unsupported: %v", err)
	}
	serveRelease(t, "9.9.9", map[string][]byte{})

	status, binaryPath := ensureInstalled(t.Context(), "9.9.9")
	assert.Empty(t, binaryPath)
	assert.Contains(t, status, "could not download pikopls")
}

func serveReleasesAPI(t *testing.T, listing []byte) {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(listing)
	}))
	t.Cleanup(server.Close)

	originalAPI := releasesAPIURL
	t.Cleanup(func() { releasesAPIURL = originalAPI })
	releasesAPIURL = server.URL
}

func testGet(url string, maxBytes int64) boundedGet {
	return boundedGet{
		url:          url,
		accept:       "",
		maxBytes:     maxBytes,
		tooLargeErr:  errDownloadTooLarge,
		timeout:      downloadTimeout,
		timeoutCause: errDownloadTimeout,
	}
}

type failingCloser struct{}

func (failingCloser) Close() error {
	return errors.New("close failed")
}

func TestCloseInto_JoinsCloseErrors(t *testing.T) {
	t.Parallel()

	original := errors.New("original")
	err := original
	closeInto(failingCloser{}, "resource", &err)
	require.ErrorIs(t, err, original)
	assert.Contains(t, err.Error(), "closing resource: close failed")

	var clean error
	closeInto(io.NopCloser(nil), "resource", &clean)
	assert.NoError(t, clean)
}

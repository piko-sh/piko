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
	stdjson "encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"time"

	"piko.sh/piko/internal/json"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/safeconv"
	"piko.sh/safedisk"
)

const (
	// binaryName is the language-server executable, both on PATH and inside release
	// archives. Go names the binary after the cmd/pikopls package's last path segment, so
	// `go install piko.sh/piko/cmd/pikopls@latest` produces a `pikopls` binary directly.
	binaryName = "pikopls"

	// downloadTimeout bounds each release download. Release binaries are tens of
	// megabytes, so this is generous compared with the API lookup.
	downloadTimeout = 90 * time.Second

	// apiTimeout bounds the GitHub API lookup used for the latest-release fallback.
	apiTimeout = 10 * time.Second

	// installBinaryPerm makes the installed language-server binary executable by its owner.
	installBinaryPerm = 0o755

	// writeProbePerm is the permission for the transient directory writability probe file.
	writeProbePerm = 0o600

	// writeProbeFileName is the transient file created to check that a directory is
	// writable.
	writeProbeFileName = ".piko-write-test"

	// lspConfigPerm is the permission for the rewritten .lsp.json plugin config.
	lspConfigPerm = 0o644

	// checksumsAssetName is the release asset listing the SHA-256 of every other asset, in
	// sha256sum format.
	checksumsAssetName = "checksums-sha256.txt"

	// lspConfigFileName is the Claude Code plugin's language-server config file.
	lspConfigFileName = ".lsp.json"

	// lspConfigServerKey is the server entry in .lsp.json whose command is pinned.
	lspConfigServerKey = "piko"

	// lspConfigCommandKey is the field holding the language-server executable.
	lspConfigCommandKey = "command"

	// installSandboxPurpose names the sandbox used for install and config writes.
	installSandboxPurpose = "lsp-install"

	// userAgent identifies the CLI to the release host.
	userAgent = "piko-cli"

	// githubAPIAccept is the media type GitHub recommends for REST API requests.
	githubAPIAccept = "application/vnd.github+json"

	// zipArchiveExtension is the release archive extension used on Windows.
	zipArchiveExtension = "zip"

	// tarGzArchiveExtension is the release archive extension used on Unix-like systems.
	tarGzArchiveExtension = "tar.gz"

	// windowsGOOS is the runtime.GOOS value for Windows.
	windowsGOOS = "windows"

	// systemBinDirectory is the shared install location tried after ~/.local/bin.
	systemBinDirectory = "/usr/local/bin"

	// maxArchiveBytes caps a downloaded release archive. Release archives are tens of
	// megabytes, so this leaves generous headroom while bounding a hostile server.
	maxArchiveBytes = 256 * 1024 * 1024

	// maxExtractedBinaryBytes caps the decompressed language-server executable, guarding
	// against decompression bombs.
	maxExtractedBinaryBytes = 512 * 1024 * 1024

	// maxReleasesAPIBytes caps the GitHub releases listing response.
	maxReleasesAPIBytes = 32 * 1024 * 1024

	// maxChecksumsBytes caps the checksums file, which lists a few dozen assets.
	maxChecksumsBytes = 1024 * 1024

	// maxLSPConfigBytes caps the .lsp.json config read back for rewriting.
	maxLSPConfigBytes = 1024 * 1024

	// maxArchiveEntries caps how many entries an archive may hold before extraction gives
	// up. Release archives hold a handful of files.
	maxArchiveEntries = 4096

	// maxVersionLength caps a release version taken from the CLI or the releases API
	// before it is placed in a download URL.
	maxVersionLength = 128

	// tarFramingHeadroom is the allowance for tar headers and padding on top of the
	// extracted binary when capping the whole decompressed tar stream.
	tarFramingHeadroom = 1024 * 1024
)

var (
	// releaseBaseURL is the GitHub release-download root. It is a var so tests can point it
	// at a local server.
	releaseBaseURL = "https://github.com/piko-sh/piko/releases/download"

	// releasesAPIURL lists releases newest-first for the latest-release fallback. It is a
	// var so tests can point it at a local server.
	releasesAPIURL = "https://api.github.com/repos/piko-sh/piko/releases?per_page=100"

	// lookPath is indirected so tests can simulate an empty PATH.
	lookPath = exec.LookPath

	// releaseOSTokens maps runtime.GOOS onto the operating-system token in asset names.
	releaseOSTokens = map[string]string{"darwin": "darwin", "linux": "linux", windowsGOOS: windowsGOOS}

	// releaseArchTokens maps runtime.GOARCH onto the architecture token in asset names.
	releaseArchTokens = map[string]string{"amd64": "amd64", "arm64": "arm64"}

	// errDownloadTooLarge is returned when a release download exceeds its byte budget.
	errDownloadTooLarge = errors.New("download exceeded byte budget")

	// errDownloadTimeout is the cause attached when a release download runs past
	// downloadTimeout.
	errDownloadTimeout = errors.New("release download timed out")

	// errReleaseLookupTimeout is the cause attached when the releases API lookup runs past
	// apiTimeout.
	errReleaseLookupTimeout = errors.New("release lookup timed out")

	// errUnexpectedStatus is returned when the release host answers with a status other
	// than 200 OK.
	errUnexpectedStatus = errors.New("unexpected HTTP status")

	// errBinaryTooLarge is returned when the executable inside a release archive
	// decompresses beyond maxExtractedBinaryBytes.
	errBinaryTooLarge = errors.New("extracted binary exceeded byte budget")

	// errBinaryNotInArchive is returned when a release archive does not hold the
	// language-server executable.
	errBinaryNotInArchive = errors.New("language server binary not found in archive")

	// errTooManyArchiveEntries is returned when an archive holds more than
	// maxArchiveEntries entries.
	errTooManyArchiveEntries = errors.New("archive holds too many entries")

	// errChecksumUnavailable is returned when the release checksums file cannot be fetched
	// or does not list the downloaded asset. Installation fails closed rather than running
	// an unverified binary.
	errChecksumUnavailable = errors.New("release checksum unavailable")

	// errChecksumMismatch is returned when a downloaded archive's SHA-256 does not match
	// the published checksum.
	errChecksumMismatch = errors.New("release checksum mismatch")

	// errUnsupportedPlatform is returned when no release binary is published for the host
	// operating system or architecture.
	errUnsupportedPlatform = errors.New("unsupported platform")

	// errNoPublishedRelease is returned when the releases API lists no usable release.
	errNoPublishedRelease = errors.New("no published releases found")

	// errReleaseMissingPlatform is returned when the latest release has no binary for the
	// host platform.
	errReleaseMissingPlatform = errors.New("release has no binary for this platform")

	// errInvalidVersion is returned when a release version is empty, too long, or holds
	// characters that are not allowed in a release tag.
	errInvalidVersion = errors.New("invalid release version")

	// errEmptyBinaryPath is returned when the language-server path to pin is empty.
	errEmptyBinaryPath = errors.New("language server path is empty")

	// errLSPConfigMissingServer is returned when .lsp.json has no piko server entry to pin.
	errLSPConfigMissingServer = errors.New("language-server config has no piko entry")
)

// githubRelease is the subset of a GitHub release listing entry used to pick the latest
// published version.
type githubRelease struct {
	// TagName is the release tag, such as v1.2.3.
	TagName string `json:"tag_name"`

	// Draft is true for unpublished draft releases.
	Draft bool `json:"draft"`

	// Prerelease is true for alpha, beta, and release-candidate releases.
	Prerelease bool `json:"prerelease"`
}

// platform holds the release-asset tokens for the host operating system and architecture.
type platform struct {
	// os is the GOOS-derived token used in asset names (darwin, linux, windows).
	os string

	// arch is the GOARCH-derived token used in asset names (amd64, arm64).
	arch string

	// archiveExt is the archive extension for the platform (tar.gz or zip).
	archiveExt string

	// binaryFile is the executable name inside the archive (pikopls or pikopls.exe).
	binaryFile string
}

// boundedGet describes one size- and time-bounded HTTP GET.
type boundedGet struct {
	// tooLargeErr is the sentinel wrapped when the body exceeds maxBytes.
	tooLargeErr error

	// timeoutCause is the cause attached to the request context when timeout elapses.
	timeoutCause error

	// url is the address to fetch.
	url string

	// accept is the Accept header value, or empty to send none.
	accept string

	// maxBytes is the largest accepted response body.
	maxBytes int64

	// timeout bounds the whole request, including reading the body.
	timeout time.Duration
}

// overflowErrorReader passes reads through until limit bytes have been read and then
// returns overflowErr if more data follows, so a capped stream fails loudly instead of
// looking like a clean, shorter stream.
type overflowErrorReader struct {
	// reader is the stream being capped.
	reader io.Reader

	// overflowErr is returned once the stream holds more than the limit.
	overflowErr error

	// remaining is how many bytes may still be read before the limit is reached.
	remaining int64
}

// Read reads up to the remaining budget, returning overflowErr when the stream goes on
// past the limit.
//
// Takes buffer ([]byte) which receives the read bytes.
//
// Returns int which is the number of bytes read.
// Returns error which is overflowErr past the limit, or the inner read error.
func (r *overflowErrorReader) Read(buffer []byte) (int, error) {
	if r.remaining <= 0 {
		var probe [1]byte
		count, err := r.reader.Read(probe[:])
		if count > 0 {
			return 0, r.overflowErr
		}
		return 0, err
	}
	if int64(len(buffer)) > r.remaining {
		buffer = buffer[:r.remaining]
	}
	count, err := r.reader.Read(buffer)
	r.remaining -= int64(count)
	return count, err
}

// ensureInstalled guarantees pikopls is available for editors and agents to launch.
//
// Takes version (string) which is the running CLI version used to pick the release asset.
//
// Returns status (string) which is a single-line status describing what happened. It
// never reports a hard failure: the plugin still works once the user installs the binary
// by hand, so a failed download is shown as guidance rather than an error.
// Returns binaryPath (string) which is the absolute path of the usable pikopls
// executable, or empty when none is available.
func ensureInstalled(ctx context.Context, version string) (status string, binaryPath string) {
	if path, err := lookPath(binaryName); err == nil {
		return fmt.Sprintf("pikopls already on PATH (%s)", path), absolutePath(path)
	}

	plat, err := currentPlatform()
	if err != nil {
		return fmt.Sprintf("pikopls not installed: %v - build it with 'make build-lsp' and copy it onto PATH", err), ""
	}

	binary, resolvedVersion, err := fetchBinary(ctx, version, plat)
	if err != nil {
		return fmt.Sprintf("could not download pikopls (%v) - install it manually from https://github.com/piko-sh/piko/releases", err), ""
	}

	installed, onPath, err := install(ctx, binary, plat)
	if err != nil {
		return fmt.Sprintf("downloaded pikopls %s but could not install it: %v", resolvedVersion, err), ""
	}

	if !onPath {
		return fmt.Sprintf("installed pikopls %s to %s - add that directory to your PATH", resolvedVersion, installed), absolutePath(installed)
	}
	return fmt.Sprintf("installed pikopls %s to %s", resolvedVersion, installed), absolutePath(installed)
}

// pinLSPCommand rewrites the plugin's .lsp.json so its piko server launches binaryPath
// by absolute path. This keeps the language server working when its directory is not on
// the PATH that the editor or agent inherits.
//
// Takes pluginDirectory (string) which is the installed plugin directory holding
// .lsp.json.
// Takes binaryPath (string) which is the pikopls executable to launch.
//
// Returns error when binaryPath is empty, or the config cannot be read, parsed, or
// written, or has no piko entry.
func pinLSPCommand(pluginDirectory, binaryPath string) (err error) {
	if strings.TrimSpace(binaryPath) == "" {
		return errEmptyBinaryPath
	}

	sandbox, err := openInstallSandbox(pluginDirectory)
	if err != nil {
		return err
	}
	defer closeSandbox(sandbox, &err)

	data, _, err := sandbox.ReadFileLimit(lspConfigFileName, maxLSPConfigBytes)
	if err != nil {
		return fmt.Errorf("reading %s: %w", lspConfigFileName, err)
	}

	rewritten, err := pinCommandInLSPConfig(data, absolutePath(binaryPath))
	if err != nil {
		return err
	}

	if writeErr := sandbox.WriteFileAtomic(lspConfigFileName, rewritten, lspConfigPerm); writeErr != nil {
		return fmt.Errorf("writing %s: %w", lspConfigFileName, writeErr)
	}
	return nil
}

// currentPlatform maps the host GOOS/GOARCH onto release-asset tokens.
//
// Returns platform which holds the asset tokens for the host.
// Returns error which wraps errUnsupportedPlatform when the host has no published binary.
func currentPlatform() (platform, error) {
	return platformFor(runtime.GOOS, runtime.GOARCH)
}

// platformFor maps an operating system and architecture onto release-asset tokens.
//
// Takes goos (string) which is the operating system, as runtime.GOOS reports it.
// Takes goarch (string) which is the architecture, as runtime.GOARCH reports it.
//
// Returns platform which holds the asset tokens.
// Returns error which wraps errUnsupportedPlatform when no binary is published for the
// pair.
func platformFor(goos, goarch string) (platform, error) {
	osToken, ok := releaseOSTokens[goos]
	if !ok {
		return platform{}, fmt.Errorf("%w: operating system %q", errUnsupportedPlatform, goos)
	}

	archToken, ok := releaseArchTokens[goarch]
	if !ok {
		return platform{}, fmt.Errorf("%w: architecture %q", errUnsupportedPlatform, goarch)
	}

	if goos == windowsGOOS {
		return platform{os: osToken, arch: archToken, archiveExt: zipArchiveExtension, binaryFile: binaryName + ".exe"}, nil
	}
	return platform{os: osToken, arch: archToken, archiveExt: tarGzArchiveExtension, binaryFile: binaryName}, nil
}

// assetName builds the release-asset filename for a version on a platform.
//
// The version is embedded without a leading "v" (e.g.
// pikopls-0.0.0-alpha.26-linux-amd64.tar.gz).
//
// Takes version (string) which is the release version without a leading "v".
// Takes plat (platform) which holds the host asset tokens.
//
// Returns string which is the asset filename.
func assetName(version string, plat platform) string {
	return fmt.Sprintf("%s-%s-%s-%s.%s", binaryName, version, plat.os, plat.arch, plat.archiveExt)
}

// fetchBinary downloads and extracts the pikopls executable. It tries the exact CLI
// version first and falls back to the latest published release, which covers
// development builds whose version was never released.
//
// Takes version (string) which is the running CLI version (with or without "v").
// Takes plat (platform) which holds the host asset tokens.
//
// Returns []byte which is the extracted executable.
// Returns string which is the version actually downloaded.
// Returns error when neither the exact nor the latest release could be fetched, or when
// a checksum check fails.
func fetchBinary(ctx context.Context, version string, plat platform) ([]byte, string, error) {
	wanted := strings.TrimPrefix(strings.TrimSpace(version), "v")

	if validateVersion(wanted) == nil {
		binary, err := downloadAndExtract(ctx, wanted, plat)
		if err == nil {
			return binary, wanted, nil
		}
		if isChecksumFailure(err) {
			return nil, "", err
		}
	}

	latest, err := latestVersion(ctx)
	if err != nil {
		return nil, "", fmt.Errorf("no release for version %q and latest lookup failed: %w", wanted, err)
	}
	if latest == wanted {
		return nil, "", fmt.Errorf("%w: release %q, %s/%s", errReleaseMissingPlatform, latest, plat.os, plat.arch)
	}

	binary, err := downloadAndExtract(ctx, latest, plat)
	if err != nil {
		return nil, "", err
	}
	return binary, latest, nil
}

// downloadAndExtract fetches one release archive, verifies it against the release's
// published SHA-256 checksums, and returns the executable inside it.
//
// Takes version (string) which is the release version without a leading "v".
// Takes plat (platform) which holds the host asset tokens.
//
// Returns []byte which is the extracted executable.
// Returns error when the download fails, the checksum cannot be verified, or the archive
// lacks the binary.
func downloadAndExtract(ctx context.Context, version string, plat platform) ([]byte, error) {
	asset := assetName(version, plat)

	archive, err := httpGetBounded(ctx, boundedGet{
		url:          releaseAssetURL(version, asset),
		maxBytes:     maxArchiveBytes,
		tooLargeErr:  errDownloadTooLarge,
		timeout:      downloadTimeout,
		timeoutCause: errDownloadTimeout,
		accept:       "",
	})
	if err != nil {
		return nil, err
	}

	if err := verifyReleaseChecksum(ctx, version, asset, archive); err != nil {
		return nil, err
	}

	binary, err := extractBinary(archive, plat)
	if err != nil {
		return nil, fmt.Errorf("extracting %s: %w", asset, err)
	}
	return binary, nil
}

// verifyReleaseChecksum checks archive against the SHA-256 published for asset in the
// release's checksums file. It fails closed when the file or its entry is missing.
//
// Takes version (string) which is the release version without a leading "v".
// Takes asset (string) which is the archive's release-asset filename.
// Takes archive ([]byte) which is the downloaded archive.
//
// Returns error wrapping errChecksumUnavailable when the checksum cannot be obtained, or
// errChecksumMismatch when it differs.
func verifyReleaseChecksum(ctx context.Context, version, asset string, archive []byte) error {
	listing, err := httpGetBounded(ctx, boundedGet{
		url:          releaseAssetURL(version, checksumsAssetName),
		maxBytes:     maxChecksumsBytes,
		tooLargeErr:  errDownloadTooLarge,
		timeout:      downloadTimeout,
		timeoutCause: errDownloadTimeout,
		accept:       "",
	})
	if err != nil {
		return fmt.Errorf("%w: fetching %s: %w", errChecksumUnavailable, checksumsAssetName, err)
	}

	expected, found := lookupChecksum(listing, asset)
	if !found {
		return fmt.Errorf("%w: %s does not list %s", errChecksumUnavailable, checksumsAssetName, asset)
	}

	digest := sha256.Sum256(archive)
	actual := hex.EncodeToString(digest[:])
	if !strings.EqualFold(expected, actual) {
		return fmt.Errorf("%w: %s has SHA-256 %s, release lists %s", errChecksumMismatch, asset, actual, expected)
	}
	return nil
}

// releaseAssetURL builds the download URL of one asset in a release.
//
// Takes version (string) which is the release version without a leading "v".
// Takes asset (string) which is the asset filename.
//
// Returns string which is the asset URL.
func releaseAssetURL(version, asset string) string {
	return fmt.Sprintf("%s/v%s/%s", releaseBaseURL, version, asset)
}

// httpGetBounded performs a single GET bounded in time and size, and returns the body.
//
// Takes request (boundedGet) which describes the URL, limits, and timeout.
//
// Returns []byte which is the response body.
// Returns error which wraps errUnexpectedStatus for a non-200 status, request.tooLargeErr
// when the body is too large, or the transport error.
func httpGetBounded(ctx context.Context, request boundedGet) ([]byte, error) {
	ctx, cancel := context.WithTimeoutCause(ctx, request.timeout, request.timeoutCause)
	defer cancel()

	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodGet, request.url, nil)
	if err != nil {
		return nil, fmt.Errorf("building request for %s: %w", request.url, err)
	}
	httpRequest.Header.Set("User-Agent", userAgent)
	if request.accept != "" {
		httpRequest.Header.Set("Accept", request.accept)
	}

	response, err := http.DefaultClient.Do(httpRequest)
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", request.url, withContextCause(ctx, err))
	}
	defer drainAndClose(response.Body)

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: GET %s: %s", errUnexpectedStatus, request.url, response.Status)
	}

	body, err := readBoundedBody(response.Body, request.maxBytes, request.tooLargeErr)
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", request.url, withContextCause(ctx, err))
	}
	return body, nil
}

// withContextCause adds the context's cancellation cause to err when the context has
// ended and err does not already carry that cause.
//
// Takes err (error) which is the error raised while the context was live.
//
// Returns error which is err, joined with the cause when one applies.
func withContextCause(ctx context.Context, err error) error {
	cause := context.Cause(ctx)
	if cause == nil || errors.Is(err, cause) {
		return err
	}
	return fmt.Errorf("%w: %w", err, cause)
}

// latestVersion resolves the newest published release, preferring a stable release over a
// prerelease, with the version returned without a leading "v".
//
// Returns string which is the latest release version without "v".
// Returns error when the lookup fails or no usable published release exists.
func latestVersion(ctx context.Context) (string, error) {
	body, err := httpGetBounded(ctx, boundedGet{
		url:          releasesAPIURL,
		accept:       githubAPIAccept,
		maxBytes:     maxReleasesAPIBytes,
		tooLargeErr:  errDownloadTooLarge,
		timeout:      apiTimeout,
		timeoutCause: errReleaseLookupTimeout,
	})
	if err != nil {
		return "", fmt.Errorf("listing releases: %w", err)
	}

	var releases []githubRelease
	if err := json.Unmarshal(body, &releases); err != nil {
		return "", fmt.Errorf("decoding releases: %w", err)
	}
	return pickLatestVersion(releases)
}

// pickLatestVersion chooses the first stable release in a newest-first listing, falling
// back to the newest prerelease. Drafts and releases with unusable tags are skipped.
//
// Takes releases ([]githubRelease) which is the newest-first release listing.
//
// Returns string which is the chosen version without a leading "v".
// Returns error which wraps errNoPublishedRelease when no release qualifies.
func pickLatestVersion(releases []githubRelease) (string, error) {
	fallback := ""
	for _, release := range releases {
		if release.Draft {
			continue
		}
		version := strings.TrimPrefix(release.TagName, "v")
		if validateVersion(version) != nil {
			continue
		}
		if !release.Prerelease {
			return version, nil
		}
		if fallback == "" {
			fallback = version
		}
	}
	if fallback == "" {
		return "", errNoPublishedRelease
	}
	return fallback, nil
}

// validateVersion checks that a release version is safe to place in a download URL:
// non-empty, bounded in length, and made only of letters, digits, '.', '-' and '+'.
//
// Takes version (string) which is the version without a leading "v".
//
// Returns error wrapping errInvalidVersion when the version is unusable.
func validateVersion(version string) error {
	if version == "" || len(version) > maxVersionLength {
		return fmt.Errorf("%w: %q", errInvalidVersion, version)
	}
	for _, character := range version {
		if !isVersionCharacter(character) {
			return fmt.Errorf("%w: %q", errInvalidVersion, version)
		}
	}
	return nil
}

// isVersionCharacter reports whether character may appear in a release version.
//
// Takes character (rune) which is the character to check.
//
// Returns bool which is true for ASCII letters, digits, '.', '-' and '+'.
func isVersionCharacter(character rune) bool {
	switch {
	case character >= 'a' && character <= 'z',
		character >= 'A' && character <= 'Z',
		character >= '0' && character <= '9':
		return true
	default:
		return character == '.' || character == '-' || character == '+'
	}
}

// extractBinary pulls the language-server executable out of a release archive.
//
// Takes archive ([]byte) which is the downloaded tar.gz or zip.
// Takes plat (platform) which names the archive format and the executable inside it.
//
// Returns []byte which is the executable.
// Returns error when the archive cannot be read or does not contain the binary.
func extractBinary(archive []byte, plat platform) ([]byte, error) {
	if plat.archiveExt == zipArchiveExtension {
		return extractFromZip(archive, plat.binaryFile, maxExtractedBinaryBytes)
	}
	return extractFromTarGz(archive, plat.binaryFile, maxExtractedBinaryBytes)
}

// extractFromTarGz finds binaryFile inside a gzip-compressed tar archive.
//
// Takes archive ([]byte) which is the tar.gz data.
// Takes binaryFile (string) which is the executable name to extract.
// Takes maxBytes (int64) which caps the decompressed executable size.
//
// Returns []byte which is the executable.
// Returns error when the archive cannot be read, lacks the binary (wrapping
// errBinaryNotInArchive), holds too many entries, or the binary exceeds maxBytes
// (wrapping errBinaryTooLarge).
func extractFromTarGz(archive []byte, binaryFile string, maxBytes int64) (binary []byte, err error) {
	gzipReader, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return nil, fmt.Errorf("opening gzip stream: %w", err)
	}
	defer closeInto(gzipReader, "gzip stream", &err)

	reader := tar.NewReader(&overflowErrorReader{
		reader:      gzipReader,
		overflowErr: errBinaryTooLarge,
		remaining:   maxDecompressedArchiveBytes(maxBytes),
	})
	for range maxArchiveEntries {
		header, nextErr := reader.Next()
		if errors.Is(nextErr, io.EOF) {
			return nil, fmt.Errorf("%w: %s", errBinaryNotInArchive, binaryFile)
		}
		if nextErr != nil {
			return nil, fmt.Errorf("reading tar entry: %w", nextErr)
		}
		if header.Typeflag != tar.TypeReg || filepath.Base(header.Name) != binaryFile {
			continue
		}
		if header.Size > maxBytes {
			return nil, fmt.Errorf("%w: %s declares %d bytes", errBinaryTooLarge, binaryFile, header.Size)
		}
		return readBoundedBody(reader, maxBytes, errBinaryTooLarge)
	}
	return nil, fmt.Errorf("%w: more than %d", errTooManyArchiveEntries, maxArchiveEntries)
}

// extractFromZip finds binaryFile inside a zip archive.
//
// Takes archive ([]byte) which is the zip data.
// Takes binaryFile (string) which is the executable name to extract.
// Takes maxBytes (int64) which caps the decompressed executable size.
//
// Returns []byte which is the executable.
// Returns error when the archive cannot be read, lacks the binary (wrapping
// errBinaryNotInArchive), holds too many entries, or the binary exceeds maxBytes
// (wrapping errBinaryTooLarge).
func extractFromZip(archive []byte, binaryFile string, maxBytes int64) ([]byte, error) {
	reader, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		return nil, fmt.Errorf("opening zip archive: %w", err)
	}
	if len(reader.File) > maxArchiveEntries {
		return nil, fmt.Errorf("%w: %d", errTooManyArchiveEntries, len(reader.File))
	}

	index := slices.IndexFunc(reader.File, func(file *zip.File) bool {
		return filepath.Base(file.Name) == binaryFile
	})
	if index < 0 {
		return nil, fmt.Errorf("%w: %s", errBinaryNotInArchive, binaryFile)
	}
	return readZipEntry(reader.File[index], binaryFile, maxBytes)
}

// readZipEntry reads one zip entry, refusing entries that declare or decompress to more
// than maxBytes.
//
// Takes file (*zip.File) which is the entry to read.
// Takes binaryFile (string) which is the entry name used in error messages.
// Takes maxBytes (int64) which caps the decompressed size.
//
// Returns data ([]byte) which is the entry content.
// Returns err (error) which wraps errBinaryTooLarge when the entry is too large, or the
// open, read or close error.
func readZipEntry(file *zip.File, binaryFile string, maxBytes int64) (data []byte, err error) {
	if file.UncompressedSize64 > safeconv.Int64ToUint64(maxBytes) {
		return nil, fmt.Errorf("%w: %s declares %d bytes", errBinaryTooLarge, binaryFile, file.UncompressedSize64)
	}
	entry, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("opening %s in zip: %w", binaryFile, err)
	}
	defer closeInto(entry, binaryFile, &err)

	return readBoundedBody(entry, maxBytes, errBinaryTooLarge)
}

// install writes the executable into a directory on PATH (preferring ~/.local/bin) and
// marks it executable. The write is atomic, so a running pikopls is replaced cleanly and
// a failed write never leaves a half-written executable behind.
//
// Takes binary ([]byte) which is the executable contents.
// Takes plat (platform) which names the on-disk executable file.
//
// Returns string which is the full path the binary was written to.
// Returns bool which is true when the install directory is already on PATH.
// Returns error when the file cannot be written.
func install(ctx context.Context, binary []byte, plat platform) (path string, onPath bool, err error) {
	directory, onPath, err := chooseInstallDirectory(ctx)
	if err != nil {
		return "", false, err
	}

	sandbox, err := openInstallSandbox(directory)
	if err != nil {
		return "", false, err
	}
	defer closeSandbox(sandbox, &err)

	if writeErr := sandbox.WriteFileAtomic(plat.binaryFile, binary, installBinaryPerm); writeErr != nil {
		return "", false, fmt.Errorf("writing language server to %q: %w", directory, writeErr)
	}
	return filepath.Join(directory, plat.binaryFile), onPath, nil
}

// openInstallSandbox opens a read-write sandbox rooted at the install directory, creating
// the directory when it does not yet exist.
//
// Takes directory (string) which is the install directory to root the sandbox at.
//
// Returns safedisk.Sandbox which scopes all file operations to directory.
// Returns error when the directory cannot be created or opened.
func openInstallSandbox(directory string) (safedisk.Sandbox, error) {
	factory, err := safedisk.NewCLIFactory(directory)
	if err != nil {
		return nil, fmt.Errorf("creating sandbox factory: %w", err)
	}
	sandbox, err := factory.Create(installSandboxPurpose, directory, safedisk.ModeReadWrite)
	if err != nil {
		return nil, fmt.Errorf("opening directory %q: %w", directory, err)
	}
	return sandbox, nil
}

// chooseInstallDirectory selects where to write the binary, preferring a candidate that
// is already on PATH so the install takes effect without further configuration.
//
// Returns string which is the chosen directory.
// Returns bool which is true when the chosen directory is on PATH.
// Returns error when the user's home directory cannot be determined.
func chooseInstallDirectory(ctx context.Context) (string, bool, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", false, fmt.Errorf("finding home directory: %w", err)
	}
	localBin := filepath.Join(home, ".local", "bin")

	pathDirectories := filepath.SplitList(os.Getenv("PATH"))

	for _, candidate := range []string{localBin, systemBinDirectory} {
		if slices.Contains(pathDirectories, candidate) && isWritable(ctx, candidate) {
			return candidate, true, nil
		}
	}
	return localBin, slices.Contains(pathDirectories, localBin), nil
}

// isWritable reports whether files can be created in directory, creating the directory
// first when it does not yet exist.
//
// Takes directory (string) which is the directory to probe.
//
// Returns bool which is true when a file can be created in directory.
func isWritable(ctx context.Context, directory string) bool {
	_, l := logger_domain.From(ctx, log)

	sandbox, err := openInstallSandbox(directory)
	if err != nil {
		return false
	}
	defer func() {
		if closeErr := sandbox.Close(); closeErr != nil {
			l.Warn("Closing install directory sandbox failed",
				logger_domain.String("directory", directory),
				logger_domain.Error(closeErr))
		}
	}()

	if err := sandbox.WriteFile(writeProbeFileName, nil, writeProbePerm); err != nil {
		return false
	}
	if err := sandbox.Remove(writeProbeFileName); err != nil {
		l.Warn("Removing install directory write probe failed",
			logger_domain.String("directory", directory),
			logger_domain.Error(err))
	}
	return true
}

// closeSandbox closes a sandbox and joins any close error into the caller's error.
//
// Takes sandbox (safedisk.Sandbox) which is the sandbox to close.
// Takes err (*error) which is the caller's named error result.
func closeSandbox(sandbox safedisk.Sandbox, err *error) {
	closeInto(sandbox, "sandbox", err)
}

// closeInto closes closer and joins any close error into the caller's error, so a
// failed close is reported instead of dropped.
//
// Takes closer (io.Closer) which is the resource to close.
// Takes what (string) which names the resource in the error message.
// Takes err (*error) which is the caller's named error result.
func closeInto(closer io.Closer, what string, err *error) {
	if closeErr := closer.Close(); closeErr != nil {
		*err = errors.Join(*err, fmt.Errorf("closing %s: %w", what, closeErr))
	}
}

// maxDecompressedArchiveBytes bounds the whole decompressed tar stream. The tar framing
// adds a small header per entry, so the cap leaves headroom above the binary limit.
//
// Takes maxBinaryBytes (int64) which is the cap on the extracted executable.
//
// Returns int64 which is the cap on the decompressed archive stream.
func maxDecompressedArchiveBytes(maxBinaryBytes int64) int64 {
	return maxBinaryBytes + tarFramingHeadroom
}

// lookupChecksum finds the SHA-256 recorded for asset in a sha256sum-format listing.
//
// Takes listing ([]byte) which is the checksums file content.
// Takes asset (string) which is the asset filename to look up.
//
// Returns string which is the hex digest recorded for asset.
// Returns bool which is true when the asset is listed with a well-formed digest.
func lookupChecksum(listing []byte, asset string) (string, bool) {
	for line := range strings.Lines(string(listing)) {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}
		digest, name := fields[0], strings.TrimPrefix(fields[1], "*")
		if name != asset || len(digest) != sha256.Size*2 {
			continue
		}
		if _, err := hex.DecodeString(digest); err != nil {
			continue
		}
		return digest, true
	}
	return "", false
}

// isChecksumFailure reports whether err came from checksum verification, which must not
// be retried against another release.
//
// Takes err (error) which is the download error to classify.
//
// Returns bool which is true for a missing or mismatched checksum.
func isChecksumFailure(err error) bool {
	return errors.Is(err, errChecksumMismatch) || errors.Is(err, errChecksumUnavailable)
}

// pinCommandInLSPConfig sets the piko server's command in a .lsp.json document. Every
// other field is kept byte for byte, so numbers and unknown settings survive the rewrite.
//
// Takes data ([]byte) which is the current .lsp.json content.
// Takes command (string) which is the executable path to launch.
//
// Returns []byte which is the rewritten, indented document.
// Returns error when the document cannot be parsed or has no piko entry.
func pinCommandInLSPConfig(data []byte, command string) ([]byte, error) {
	var document map[string]stdjson.RawMessage
	if err := json.Unmarshal(data, &document); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", lspConfigFileName, err)
	}

	rawServer, ok := document[lspConfigServerKey]
	if !ok {
		return nil, fmt.Errorf("%w: %s", errLSPConfigMissingServer, lspConfigFileName)
	}
	var server map[string]stdjson.RawMessage
	if err := json.Unmarshal(rawServer, &server); err != nil {
		return nil, fmt.Errorf("%w: %s entry %q is not an object: %w", errLSPConfigMissingServer, lspConfigFileName, lspConfigServerKey, err)
	}
	if server == nil {
		return nil, fmt.Errorf("%w: %s entry %q is null", errLSPConfigMissingServer, lspConfigFileName, lspConfigServerKey)
	}

	encodedCommand, err := json.Marshal(command)
	if err != nil {
		return nil, fmt.Errorf("encoding command: %w", err)
	}
	server[lspConfigCommandKey] = encodedCommand

	encodedServer, err := json.Marshal(server)
	if err != nil {
		return nil, fmt.Errorf("encoding %s entry: %w", lspConfigServerKey, err)
	}
	document[lspConfigServerKey] = encodedServer

	rewritten, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encoding %s: %w", lspConfigFileName, err)
	}
	return append(rewritten, '\n'), nil
}

// absolutePath resolves path to an absolute path, falling back to path unchanged when
// the working directory cannot be determined.
//
// Takes path (string) which is the path to resolve.
//
// Returns string which is the absolute path, or path itself on failure.
func absolutePath(path string) string {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return absolute
}

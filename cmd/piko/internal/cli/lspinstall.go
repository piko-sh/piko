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
	"encoding/json"
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

	"piko.sh/piko/wdk/safedisk"
)

const (
	// binaryName is the language-server executable, both on PATH and inside release
	// archives. Go names the binary after the cmd/pikopls package's last path segment, so
	// `go install piko.sh/piko/cmd/pikopls@latest` produces a `pikopls` binary directly.
	binaryName = "pikopls"

	// downloadTimeout bounds the archive download. Release binaries are tens of megabytes,
	// so this is generous compared with the API lookup.
	downloadTimeout = 90 * time.Second

	// apiTimeout bounds the GitHub API lookup used for the latest-release fallback.
	apiTimeout = 10 * time.Second

	// installBinaryPerm makes the installed language-server binary executable by its owner.
	installBinaryPerm = 0o755

	// writeProbePerm is the permission for the transient directory writability probe file.
	writeProbePerm = 0o600
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
)

// EnsureInstalled guarantees pikopls is available for editors and agents to launch.
//
// Takes version (string) which is the running CLI version used to pick the release asset.
//
// Returns string which is a human-readable, single-line status describing what happened.
// It never reports a hard failure: the plugin still works once the user installs the
// binary by hand, so a failed download is surfaced as guidance rather than an error.
func EnsureInstalled(ctx context.Context, version string) string {
	if path, err := lookPath(binaryName); err == nil {
		return fmt.Sprintf("pikopls already on PATH (%s)", path)
	}

	plat, err := currentPlatform()
	if err != nil {
		return fmt.Sprintf("pikopls not installed: %v - build it with 'make build-lsp' and copy it onto PATH", err)
	}

	binary, resolvedVersion, err := fetchBinary(ctx, version, plat)
	if err != nil {
		return fmt.Sprintf("could not download pikopls (%v) - install it manually from https://github.com/piko-sh/piko/releases", err)
	}

	installed, onPath, err := install(binary, plat)
	if err != nil {
		return fmt.Sprintf("downloaded pikopls %s but could not install it: %v", resolvedVersion, err)
	}

	if !onPath {
		return fmt.Sprintf("installed pikopls %s to %s - add that directory to your PATH", resolvedVersion, installed)
	}
	return fmt.Sprintf("installed pikopls %s to %s", resolvedVersion, installed)
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

// currentPlatform maps the host GOOS/GOARCH onto release-asset tokens.
//
// Returns platform which holds the asset tokens for the host.
// Returns error when the host operating system or architecture has no published binary.
func currentPlatform() (platform, error) {
	osToken := map[string]string{"darwin": "darwin", "linux": "linux", "windows": "windows"}[runtime.GOOS]
	if osToken == "" {
		return platform{}, fmt.Errorf("unsupported OS %q", runtime.GOOS)
	}

	archToken := map[string]string{"amd64": "amd64", "arm64": "arm64"}[runtime.GOARCH]
	if archToken == "" {
		return platform{}, fmt.Errorf("unsupported architecture %q", runtime.GOARCH)
	}

	ext := "tar.gz"
	binFile := binaryName
	if runtime.GOOS == "windows" {
		ext = "zip"
		binFile = binaryName + ".exe"
	}

	return platform{os: osToken, arch: archToken, archiveExt: ext, binaryFile: binFile}, nil
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

// fetchBinary downloads and extracts the pikopls executable, trying the exact CLI version
// first and falling back to the latest published release (covering development builds
// whose version was never released).
//
// Takes ctx (context.Context) which bounds the network operations.
// Takes version (string) which is the running CLI version (with or without "v").
// Takes plat (platform) which holds the host asset tokens.
//
// Returns []byte which is the extracted executable.
// Returns string which is the version actually downloaded.
// Returns error when neither the exact nor the latest release could be fetched.
func fetchBinary(ctx context.Context, version string, plat platform) ([]byte, string, error) {
	wanted := strings.TrimPrefix(strings.TrimSpace(version), "v")

	if wanted != "" {
		if binary, err := downloadAndExtract(ctx, wanted, plat); err == nil {
			return binary, wanted, nil
		}
	}

	latest, err := latestVersion(ctx)
	if err != nil {
		return nil, "", fmt.Errorf("no release for version %q and latest lookup failed: %w", wanted, err)
	}
	if latest == wanted {
		return nil, "", fmt.Errorf("release %q has no %s/%s binary", latest, plat.os, plat.arch)
	}

	binary, err := downloadAndExtract(ctx, latest, plat)
	if err != nil {
		return nil, "", err
	}
	return binary, latest, nil
}

// downloadAndExtract fetches one release archive and returns the executable inside it.
//
// Takes ctx (context.Context) which bounds the download.
// Takes version (string) which is the release version without a leading "v".
// Takes plat (platform) which holds the host asset tokens.
//
// Returns []byte which is the extracted executable.
// Returns error when the download fails or the archive lacks the binary.
func downloadAndExtract(ctx context.Context, version string, plat platform) ([]byte, error) {
	asset := assetName(version, plat)
	url := fmt.Sprintf("%s/v%s/%s", releaseBaseURL, version, asset)

	archive, err := download(ctx, url)
	if err != nil {
		return nil, err
	}

	binary, err := extractBinary(archive, plat)
	if err != nil {
		return nil, fmt.Errorf("extract %s: %w", asset, err)
	}
	return binary, nil
}

// download performs a single GET and returns the body, bounded by downloadTimeout.
//
// Takes ctx (context.Context) which bounds the request.
// Takes url (string) which is the asset URL.
//
// Returns []byte which is the response body.
// Returns error when the request fails or the server returns a non-200 status.
func download(ctx context.Context, url string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, downloadTimeout)
	defer cancel()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("User-Agent", "piko-cli")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: %s", url, response.Status)
	}

	return io.ReadAll(response.Body)
}

// latestVersion resolves the newest published release, preferring a stable release over a
// prerelease, with the version returned without a leading "v".
//
// Takes ctx (context.Context) which bounds the request.
//
// Returns string which is the latest release version without "v".
// Returns error when the lookup fails or no published release exists.
func latestVersion(ctx context.Context) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, apiTimeout)
	defer cancel()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, releasesAPIURL, nil)
	if err != nil {
		return "", err
	}
	request.Header.Set("User-Agent", "piko-cli")
	request.Header.Set("Accept", "application/vnd.github+json")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return "", err
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GET releases: %s", response.Status)
	}

	var releases []struct {
		TagName    string `json:"tag_name"`
		Draft      bool   `json:"draft"`
		Prerelease bool   `json:"prerelease"`
	}
	if err := json.NewDecoder(response.Body).Decode(&releases); err != nil {
		return "", err
	}

	fallback := ""
	for _, release := range releases {
		if release.Draft || release.TagName == "" {
			continue
		}
		version := strings.TrimPrefix(release.TagName, "v")
		if !release.Prerelease {
			return version, nil
		}
		if fallback == "" {
			fallback = version
		}
	}
	if fallback == "" {
		return "", errors.New("no published releases found")
	}
	return fallback, nil
}

// extractBinary pulls the language-server executable out of a release archive.
//
// Takes archive ([]byte) which is the downloaded tar.gz or zip.
// Takes plat (platform) which names the archive format and the executable inside it.
//
// Returns []byte which is the executable.
// Returns error when the archive cannot be read or does not contain the binary.
func extractBinary(archive []byte, plat platform) ([]byte, error) {
	if plat.archiveExt == "zip" {
		return extractFromZip(archive, plat.binaryFile)
	}
	return extractFromTarGz(archive, plat.binaryFile)
}

// extractFromTarGz finds binaryFile inside a gzip-compressed tar archive.
//
// Takes archive ([]byte) which is the tar.gz data.
// Takes binaryFile (string) which is the executable name to extract.
//
// Returns []byte which is the executable.
// Returns error when the archive cannot be read or lacks the binary.
func extractFromTarGz(archive []byte, binaryFile string) ([]byte, error) {
	gzr, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return nil, err
	}
	defer func() { _ = gzr.Close() }()

	reader := tar.NewReader(gzr)
	for {
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		if header.Typeflag == tar.TypeReg && filepath.Base(header.Name) == binaryFile {
			return io.ReadAll(reader)
		}
	}
	return nil, fmt.Errorf("%s not found in archive", binaryFile)
}

// extractFromZip finds binaryFile inside a zip archive.
//
// Takes archive ([]byte) which is the zip data.
// Takes binaryFile (string) which is the executable name to extract.
//
// Returns []byte which is the executable.
// Returns error when the archive cannot be read or lacks the binary.
func extractFromZip(archive []byte, binaryFile string) ([]byte, error) {
	reader, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		return nil, err
	}
	for _, file := range reader.File {
		if filepath.Base(file.Name) != binaryFile {
			continue
		}
		rc, err := file.Open()
		if err != nil {
			return nil, err
		}
		data, err := io.ReadAll(rc)
		_ = rc.Close()
		if err != nil {
			return nil, err
		}
		return data, nil
	}
	return nil, fmt.Errorf("%s not found in archive", binaryFile)
}

// install writes the executable into a directory on PATH (preferring ~/.local/bin) and
// marks it executable.
//
// Takes binary ([]byte) which is the executable contents.
// Takes plat (platform) which names the on-disk executable file.
//
// Returns string which is the full path the binary was written to.
// Returns bool which is true when the install directory is already on PATH.
// Returns error when the file cannot be written.
func install(binary []byte, plat platform) (string, bool, error) {
	dir, onPath, err := chooseInstallDir()
	if err != nil {
		return "", false, err
	}

	sandbox, err := openInstallSandbox(dir)
	if err != nil {
		return "", false, err
	}
	defer func() { _ = sandbox.Close() }()

	if err := sandbox.WriteFile(plat.binaryFile, binary, installBinaryPerm); err != nil {
		return "", false, fmt.Errorf("writing language server to %q: %w", dir, err)
	}
	return filepath.Join(dir, plat.binaryFile), onPath, nil
}

// openInstallSandbox opens a read-write sandbox rooted at the install directory, creating
// the directory when it does not yet exist.
//
// Takes dir (string) which is the install directory to root the sandbox at.
//
// Returns safedisk.Sandbox which scopes all file operations to dir.
// Returns error when the directory cannot be created or opened.
func openInstallSandbox(dir string) (safedisk.Sandbox, error) {
	factory, err := safedisk.NewCLIFactory(dir)
	if err != nil {
		return nil, fmt.Errorf("creating sandbox factory: %w", err)
	}
	sandbox, err := factory.Create("lsp-install", dir, safedisk.ModeReadWrite)
	if err != nil {
		return nil, fmt.Errorf("opening install directory %q: %w", dir, err)
	}
	return sandbox, nil
}

// chooseInstallDir selects where to write the binary, preferring a candidate that is
// already on PATH so the install takes effect without further configuration.
//
// Returns string which is the chosen directory.
// Returns bool which is true when the chosen directory is on PATH.
// Returns error when the user's home directory cannot be determined.
func chooseInstallDir() (string, bool, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", false, err
	}
	localBin := filepath.Join(home, ".local", "bin")

	pathDirs := filepath.SplitList(os.Getenv("PATH"))

	for _, candidate := range []string{localBin, "/usr/local/bin"} {
		if slices.Contains(pathDirs, candidate) && isWritable(candidate) {
			return candidate, true, nil
		}
	}
	return localBin, slices.Contains(pathDirs, localBin), nil
}

// isWritable reports whether files can be created in dir, creating the directory first
// when it does not yet exist.
//
// Takes dir (string) which is the directory to probe.
//
// Returns bool which is true when a file can be created in dir.
func isWritable(dir string) bool {
	sandbox, err := openInstallSandbox(dir)
	if err != nil {
		return false
	}
	defer func() { _ = sandbox.Close() }()

	probeName := ".piko-write-test"
	if err := sandbox.WriteFile(probeName, nil, writeProbePerm); err != nil {
		return false
	}
	_ = sandbox.Remove(probeName)
	return true
}

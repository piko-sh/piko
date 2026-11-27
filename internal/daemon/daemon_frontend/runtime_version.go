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
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"slices"
	"strings"
	"sync"
)

const (
	// RuntimeVersionHeader is the response header that names the runtime version a server
	// hands out. The browser runtime compares it with its own version and loads the full
	// page when they differ, instead of mixing code from two builds in one tab.
	RuntimeVersionHeader = "X-PP-Runtime"

	// RuntimeRecoveryURL is the fixed URL of the small classic script that reloads a page
	// once when a framework bundle or generated module fails to load. It has no version in
	// its path, so pages from every build can load it.
	RuntimeRecoveryURL = "/_piko/runtime-recovery.js"

	// RuntimeRecoveryAssetPath is the asset store path of the runtime recovery script.
	RuntimeRecoveryAssetPath = "built/runtime-recovery.js"

	// runtimeVersionBytes is the number of leading SHA-256 bytes kept in a runtime version.
	// Sixty-four bits is far more than enough to tell a handful of live builds apart.
	runtimeVersionBytes = 8

	// runtimeVersionLength is the number of lowercase hex characters in a runtime version.
	runtimeVersionLength = runtimeVersionBytes * 2

	// runtimeDistPath is the URL prefix that generated code and pages use for the framework
	// bundles.
	runtimeDistPath = "/_piko/dist/"

	// sriSidecarGlob matches the SRI sidecar of every embedded framework bundle.
	sriSidecarGlob = "built/*.sri"

	// sourceMapETagGlob matches the ETag sidecar of every embedded source map. Source maps
	// are served for good under the versioned URL too, so their bytes must be covered by the
	// version.
	sourceMapETagGlob = "built/*.map.etag"

	// coreModuleFile is the file name of the core runtime ES module.
	coreModuleFile = "ppframework.core.es.js"

	// componentsModuleFile is the file name of the PKC components ES module.
	componentsModuleFile = "ppframework.components.es.js"

	// animationModuleFile is the file name of the animation extension ES module.
	animationModuleFile = "ppframework.animation.es.js"
)

var (
	// runtimeSidecars holds the SRI sidecars of the embedded framework bundles and the ETag
	// sidecars of their source maps. They are a few bytes each, so every build flavour
	// embeds them, including WebAssembly, and the runtime version is the same wherever it is
	// computed.
	//
	//go:embed built/*.sri built/*.map.etag
	runtimeSidecars embed.FS

	// runtimeVersion computes the runtime version once, on first use.
	//
	// The bundles are embedded in the binary, so they cannot change while the process runs.
	// A rebuilt bundle always comes with a new process, which computes the version again.
	runtimeVersion = sync.OnceValues(func() (string, error) {
		return computeRuntimeVersion(runtimeSidecars)
	})

	// errNoRuntimeSidecars is returned when no SRI sidecars are found to fingerprint.
	errNoRuntimeSidecars = errors.New("no framework bundle SRI sidecars found")
)

// RuntimeVersion returns the version of the embedded framework bundles.
//
// The version is a fingerprint of every bundle's SRI hash and every source map's ETag, so
// it changes whenever any bundle's or source map's bytes change, and stays the same
// across app releases that keep the same Piko build. It names the URL path segment of the
// versioned bundles whose bytes are the same in every runtime build. The modules with
// build variants and RuntimeVersionHeader use CoreRuntimeBuild.RuntimeVersion instead.
//
// Returns string which is the version, or empty when it cannot be computed. Callers then
// fall back to the unversioned URLs, which still work but must revalidate on every use.
func RuntimeVersion() string {
	version, err := runtimeVersion()
	if err != nil {
		return ""
	}
	return version
}

// IsRuntimeVersion reports whether segment has the exact shape of a runtime version.
//
// Takes segment (string) which is a URL path segment or header value.
//
// Returns bool which is true for exactly sixteen lowercase hex characters.
func IsRuntimeVersion(segment string) bool {
	if len(segment) != runtimeVersionLength {
		return false
	}
	for index := range len(segment) {
		if !isLowerHex(segment[index]) {
			return false
		}
	}
	return true
}

// CoreModuleImportURL returns the unversioned URL of the core runtime ES module, which
// generated modules import.
//
// Generated modules are written before it is known which core runtime build the server
// will send, so they cannot name the build's versioned URL. The server answers this URL
// with a module that re-exports its own versioned core, so the page and every generated
// module share one runtime instance.
//
// Returns string which is the unversioned URL of ppframework.core.es.js.
func CoreModuleImportURL() string {
	return runtimeDistPath + coreModuleFile
}

// ComponentsModuleImportURL returns the unversioned URL of the PKC components ES module,
// which compiled components import.
//
// Like the core, its bytes depend on the build the server sends, which is not known when
// components are compiled. The server answers this URL with a module that re-exports its
// own versioned components module, so every component shares one runtime instance.
//
// Returns string which is the unversioned URL of ppframework.components.es.js.
func ComponentsModuleImportURL() string {
	return runtimeDistPath + componentsModuleFile
}

// AnimationModuleImportURL returns the unversioned URL of the animation extension ES
// module, which compiled components import.
//
// The server answers it with a module that re-exports its own versioned animation module.
//
// Returns string which is the unversioned URL of ppframework.animation.es.js.
func AnimationModuleImportURL() string {
	return runtimeDistPath + animationModuleFile
}

// SplitRuntimeVersion splits a path below the framework bundle prefix into its runtime
// version and file name.
//
// Takes assetPath (string) which is the request path after the prefix, for example
// "0123456789abcdef/ppframework.core.es.js".
//
// Returns version (string) which is the leading runtime version segment.
// Returns fileName (string) which is the rest of the path.
// Returns ok (bool) which is false when the path does not start with a runtime version
// segment, meaning it is an unversioned request.
func SplitRuntimeVersion(assetPath string) (version, fileName string, ok bool) {
	version, fileName, found := strings.Cut(assetPath, "/")
	if !found || !IsRuntimeVersion(version) {
		return "", "", false
	}
	return version, fileName, true
}

// StripRuntimeVersion removes the runtime version segment from every framework bundle URL
// in text.
//
// Output that embeds bundle URLs changes whenever the bundles are rebuilt. Comparing it
// after this call checks everything except the version itself.
//
// Takes text (string) which may contain versioned bundle URLs.
//
// Returns string which is text with each "/_piko/dist/<version>/" shortened to
// "/_piko/dist/".
func StripRuntimeVersion(text string) string {
	if !strings.Contains(text, runtimeDistPath) {
		return text
	}
	var builder strings.Builder
	builder.Grow(len(text))
	remaining := text
	for {
		before, after, found := strings.Cut(remaining, runtimeDistPath)
		if !found {
			builder.WriteString(remaining)
			return builder.String()
		}
		builder.WriteString(before)
		builder.WriteString(runtimeDistPath)
		if _, fileName, versioned := SplitRuntimeVersion(after); versioned {
			after = fileName
		}
		remaining = after
	}
}

// runtimeVersionError returns the reason the runtime version cannot be computed.
//
// Returns error which is nil when the version is available.
func runtimeVersionError() error {
	_, err := runtimeVersion()
	return err
}

// runtimeAssetURLFor builds the URL of a bundle file for a given runtime version.
//
// Takes version (string) which is the runtime version, or empty for the unversioned URL.
// Takes fileName (string) which is the bundle file name.
//
// Returns string which is the bundle URL.
func runtimeAssetURLFor(version, fileName string) string {
	if version == "" {
		return runtimeDistPath + fileName
	}
	return runtimeDistPath + version + "/" + fileName
}

// computeRuntimeVersion fingerprints the sidecars found in sidecars.
//
// Each sidecar's file name and trimmed content feed one SHA-256 hash in file name order,
// so the result does not depend on the order in which the file system lists them.
//
// Takes sidecars (fs.FS) which holds the "built/*.sri" and "built/*.map.etag" files.
//
// Returns string which is the runtime version.
// Returns error when no SRI sidecars match or a sidecar cannot be read.
func computeRuntimeVersion(sidecars fs.FS) (string, error) {
	names, err := fingerprintedSidecars(sidecars)
	if err != nil {
		return "", err
	}

	hasher := sha256.New()
	for _, name := range names {
		content, readErr := fs.ReadFile(sidecars, name)
		if readErr != nil {
			return "", fmt.Errorf("reading framework bundle sidecar %q: %w", name, readErr)
		}
		_, _ = hasher.Write([]byte(path.Base(name)))
		_, _ = hasher.Write([]byte{0})
		_, _ = hasher.Write([]byte(strings.TrimSpace(string(content))))
		_, _ = hasher.Write([]byte{'\n'})
	}
	return hex.EncodeToString(hasher.Sum(nil)[:runtimeVersionBytes]), nil
}

// fingerprintedSidecars lists the sidecars the runtime version covers, in file name
// order.
//
// Takes sidecars (fs.FS) which holds the embedded sidecar files.
//
// Returns []string which holds the sorted sidecar paths.
// Returns error when no SRI sidecars match or a listing fails.
func fingerprintedSidecars(sidecars fs.FS) ([]string, error) {
	sriNames, err := fs.Glob(sidecars, sriSidecarGlob)
	if err != nil {
		return nil, fmt.Errorf("listing framework bundle SRI sidecars: %w", err)
	}
	if len(sriNames) == 0 {
		return nil, errNoRuntimeSidecars
	}
	sourceMapNames, err := fs.Glob(sidecars, sourceMapETagGlob)
	if err != nil {
		return nil, fmt.Errorf("listing framework source map ETag sidecars: %w", err)
	}
	names := slices.Concat(sriNames, sourceMapNames)
	slices.Sort(names)
	return names, nil
}

// isLowerHex reports whether character is a lowercase hexadecimal digit.
//
// Takes character (byte) which is the character to check.
//
// Returns bool which is true for 0-9 and a-f.
func isLowerHex(character byte) bool {
	return ('0' <= character && character <= '9') || ('a' <= character && character <= 'f')
}

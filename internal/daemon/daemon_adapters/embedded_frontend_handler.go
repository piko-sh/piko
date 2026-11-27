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

package daemon_adapters

import (
	"bytes"
	"net/http"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"piko.sh/piko/internal/daemon/daemon_frontend"
	"piko.sh/piko/internal/logger/logger_domain"
)

const (
	// esModuleSuffix ends the file name of every framework bundle built as an ES module.
	esModuleSuffix = ".es.js"

	// reexportETagPrefix starts the entity tag of the module that re-exports a versioned
	// bundle from its unversioned URL. The runtime version follows it, so the tag changes
	// with every runtime build.
	reexportETagPrefix = `"reexport-`
)

// embeddedFrontendHandler serves the framework bundles embedded in the binary.
//
// Each bundle has two URLs. The versioned URL "<dist>/<runtime version>/<file>" names the
// bytes of one build, so browsers and CDNs may keep it for good, and a request for a
// version this binary does not hold is a 404 that no cache may keep. The unversioned URL
// "<dist>/<file>" stays for older pages, generated code and tools. It revalidates on
// every use, and for ES module bundles it answers with a small module that re-exports the
// versioned URL, so a page that reaches the runtime through both URLs still runs one copy
// of it.
//
// The core module, and every built-in extension module with build variants, answers with
// the bytes of the runtime build the server runs, which is the minified file in
// production and the unminified one in the dev modes. Their versioned URLs use the
// build's own runtime version, so two builds never share a cached module URL. Each source
// map resolves from the URL named in those bytes.
type embeddedFrontendHandler struct {
	// runtimeVersion is the version of the bundles embedded in this binary, or empty when it
	// is not known. Without a version only the unversioned URLs are served, as before.
	runtimeVersion string

	// coreRuntimeVersion is the runtime version of the build this handler sends. Only it
	// names the modules with build variants, but it serves the other bundles too, so
	// relative source map URLs resolve.
	coreRuntimeVersion string

	// versionedCacheControl is the Cache-Control value for versioned bundle responses.
	versionedCacheControl []string

	// coreRuntimeBuild is the build of the runtime sent at the URLs of the modules with
	// build variants.
	coreRuntimeBuild daemon_frontend.CoreRuntimeBuild
}

// ServeHTTP serves a versioned or unversioned framework bundle request.
//
// Takes w (http.ResponseWriter) which receives the bundle.
// Takes r (*http.Request) which names the bundle in its wildcard path parameter.
func (handler *embeddedFrontendHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	assetPath := chi.URLParam(r, "*")
	if version, fileName, versioned := daemon_frontend.SplitRuntimeVersion(assetPath); versioned {
		handler.serveVersioned(w, r, version, fileName)
		return
	}
	handler.serveUnversioned(w, r, assetPath)
}

// serveVersioned serves a bundle under its versioned URL.
//
// Only the bundles built into the binary are served here, because only their bytes are
// covered by the runtime version. A custom module registered at startup changes with each
// app release, so caching it for good under this URL would be wrong.
//
// Takes w (http.ResponseWriter) which receives the bundle.
// Takes r (*http.Request) which is the bundle request.
// Takes version (string) which is the requested runtime version.
// Takes fileName (string) which is the requested bundle file name.
func (handler *embeddedFrontendHandler) serveVersioned(w http.ResponseWriter, r *http.Request, version, fileName string) {
	if !handler.holdsVersion(version, fileName) {
		handler.rejectUnknownVersion(w, r, version)
		return
	}
	if !isPlainFileName(fileName) {
		http.NotFound(w, r)
		return
	}

	asset, found := findEmbeddedAsset(r, path.Join("built", handler.coreRuntimeBuild.ServedFileName(fileName)))
	if !found || !asset.Embedded {
		http.NotFound(w, r)
		return
	}
	writeEmbeddedAsset(w, r, asset, fileName, handler.versionedCacheControl)
}

// serveUnversioned serves a bundle under its unversioned URL.
//
// The bytes behind these URLs change with each release, so every response tells the
// browser to revalidate before reuse, in every mode. The strong ETag from each file's
// .etag sidecar keeps that check to a small 304 Not Modified until a deploy changes the
// bytes.
//
// Takes w (http.ResponseWriter) which receives the bundle.
// Takes r (*http.Request) which is the bundle request.
// Takes assetPath (string) which is the requested path below the bundle prefix.
func (handler *embeddedFrontendHandler) serveUnversioned(w http.ResponseWriter, r *http.Request, assetPath string) {
	basePath := path.Join("built", assetPath)
	if handler.serveReexportModule(w, r, basePath, assetPath) {
		return
	}

	asset, found := findEmbeddedAsset(r, path.Join("built", handler.coreRuntimeBuild.ServedFileName(assetPath)))
	if !found {
		http.NotFound(w, r)
		return
	}
	writeEmbeddedAsset(w, r, asset, basePath, headerValCacheNoCache)
}

// serveReexportModule answers an unversioned request for an embedded ES module bundle
// with a module that re-exports the versioned URL.
//
// ES modules are identified by URL, so serving the bundle bytes here would give a page
// that also imports the versioned URL a second, separate copy of the runtime. The
// re-export resolves both URLs to one module instance. It is relative, so it resolves
// below whatever path the unversioned request came from.
//
// Takes w (http.ResponseWriter) which receives the module.
// Takes r (*http.Request) which is the bundle request.
// Takes basePath (string) which is the asset store path of the bundle.
// Takes fileName (string) which is the bundle file name.
//
// Returns bool which is true when the re-export module was written.
func (handler *embeddedFrontendHandler) serveReexportModule(w http.ResponseWriter, r *http.Request, basePath, fileName string) bool {
	if handler.runtimeVersion == "" || !strings.HasSuffix(fileName, esModuleSuffix) || !isPlainFileName(fileName) {
		return false
	}
	asset, found := daemon_frontend.GetAsset(r.Context(), basePath)
	if !found || !asset.Embedded {
		return false
	}

	version := handler.versionFor(fileName)
	module := "export * from \"./" + version + "/" + fileName + "\";\n"
	h := w.Header()
	h[headerETag] = []string{reexportETagPrefix + version + `"`}
	h[headerCacheControl] = headerValCacheNoCache
	h[headerContentType] = headerValContentTypeJavaScript
	http.ServeContent(w, r, fileName, time.Time{}, strings.NewReader(module))
	return true
}

// holdsVersion reports whether this handler serves a file under a version segment.
//
// A module with build variants is served only under the build's version, since its bytes
// depend on the build. Every other bundle is the same in every build, so it is served
// under the bundle version and the build's version.
//
// Takes version (string) which is the requested version segment.
// Takes fileName (string) which is the requested bundle file name.
//
// Returns bool which is true when the version names bytes this handler sends.
func (handler *embeddedFrontendHandler) holdsVersion(version, fileName string) bool {
	if version == handler.coreRuntimeVersion {
		return true
	}
	return version == handler.runtimeVersion && !daemon_frontend.HasBuildVariants(fileName)
}

// versionFor returns the version segment of a bundle's versioned URL.
//
// Takes fileName (string) which is the bundle file name.
//
// Returns string which is the build's version for a module with build variants and the
// bundle version for every other bundle.
func (handler *embeddedFrontendHandler) versionFor(fileName string) string {
	if daemon_frontend.HasBuildVariants(fileName) {
		return handler.coreRuntimeVersion
	}
	return handler.runtimeVersion
}

// rejectUnknownVersion answers a request for a runtime version this binary does not hold.
//
// The response is a 404 that no cache may keep, so a retry that reaches an instance
// holding the version can still succeed. A redirect to this binary's version would not
// help, because the page pins the requested bytes with SRI, and a second module URL would
// load a second copy of the runtime.
//
// Takes w (http.ResponseWriter) which receives the 404.
// Takes r (*http.Request) which is the bundle request.
// Takes version (string) which is the requested runtime version.
func (handler *embeddedFrontendHandler) rejectUnknownVersion(w http.ResponseWriter, r *http.Request, version string) {
	ctx, l := logger_domain.From(r.Context(), log)
	unknownRuntimeVersionCount.Add(ctx, 1)
	l.Debug("Framework bundle requested for a runtime version this binary does not hold",
		logger_domain.String("requested_runtime_version", version),
		logger_domain.String("runtime_version", handler.runtimeVersion),
	)

	w.Header()[headerCacheControl] = headerValCacheNoStore
	http.NotFound(w, r)
}

// newEmbeddedFrontendHandler creates a handler for the embedded framework bundles.
//
// Takes runtimeVersion (string) which is the version of the embedded bundles, or empty
// when it is not known.
// Takes versionedCacheControl (string) which is the Cache-Control value for versioned
// bundle responses.
// Takes coreRuntimeBuild (daemon_frontend.CoreRuntimeBuild) which is the build of the
// core runtime sent at the core module URL.
//
// Returns *embeddedFrontendHandler which serves the bundles of runtimeVersion.
func newEmbeddedFrontendHandler(
	runtimeVersion, versionedCacheControl string,
	coreRuntimeBuild daemon_frontend.CoreRuntimeBuild,
) *embeddedFrontendHandler {
	return &embeddedFrontendHandler{
		runtimeVersion:        runtimeVersion,
		coreRuntimeVersion:    coreRuntimeBuild.VersionOf(runtimeVersion),
		versionedCacheControl: []string{versionedCacheControl},
		coreRuntimeBuild:      coreRuntimeBuild,
	}
}

// serveRuntimeRecovery serves the runtime recovery script at its fixed URL.
//
// Pages from every build load this one URL, so it always revalidates. Its bytes rarely
// change, so the check is nearly always a small 304 Not Modified.
//
// Takes w (http.ResponseWriter) which receives the script.
// Takes r (*http.Request) which is the script request.
func serveRuntimeRecovery(w http.ResponseWriter, r *http.Request) {
	asset, found := findEmbeddedAsset(r, daemon_frontend.RuntimeRecoveryAssetPath)
	if !found {
		http.NotFound(w, r)
		return
	}
	writeEmbeddedAsset(w, r, asset, daemon_frontend.RuntimeRecoveryAssetPath, headerValCacheNoCache)
}

// runtimeVersionMiddleware names this binary's runtime version on every dynamic response.
//
// A tab keeps running the runtime it first loaded, while pages, partials and actions
// fetched later come from whichever build the server now runs. The browser runtime
// compares this header with its own version and loads the full page when they differ,
// instead of mixing modules from two builds that may not link.
//
// Takes version (string) which is the runtime version, or empty to add no header.
//
// Returns func(http.Handler) http.Handler which sets the header before delegating.
func runtimeVersionMiddleware(version string) func(http.Handler) http.Handler {
	if version == "" {
		return func(next http.Handler) http.Handler { return next }
	}
	headerValue := []string{version}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header()[headerXPPRuntime] = headerValue
			next.ServeHTTP(w, r)
		})
	}
}

// isPlainFileName reports whether name is a single file name with no directory part.
//
// Takes name (string) which is the requested file name.
//
// Returns bool which is false for an empty name, a dot segment, or a name with a slash or
// backslash.
func isPlainFileName(name string) bool {
	return name != "" && name != "." && name != ".." && !strings.ContainsAny(name, `/\`)
}

// findEmbeddedAsset finds the best encoded variant of an embedded asset for a request.
//
// Takes r (*http.Request) which supplies the Accept-Encoding header.
// Takes basePath (string) which is the asset store path of the uncompressed asset.
//
// Returns *daemon_frontend.EmbeddedAsset which is the asset to serve.
// Returns bool which is false when neither the encoded nor the plain asset exists.
func findEmbeddedAsset(r *http.Request, basePath string) (*daemon_frontend.EmbeddedAsset, bool) {
	finalPath := daemon_frontend.DetermineBestAssetPath(r.Context(), basePath, r.Header.Get(headerAcceptEncoding))
	if asset, found := daemon_frontend.GetAsset(r.Context(), finalPath); found {
		return asset, true
	}
	return daemon_frontend.GetAsset(r.Context(), basePath)
}

// writeEmbeddedAsset writes an embedded asset with its cache headers.
//
// Conditional and range requests are answered by http.ServeContent, which sends 304 Not
// Modified when the client's If-None-Match names the asset's ETag.
//
// Takes w (http.ResponseWriter) which receives the asset.
// Takes r (*http.Request) which is the asset request.
// Takes asset (*daemon_frontend.EmbeddedAsset) which is the asset to write.
// Takes name (string) which names the asset for http.ServeContent.
// Takes cacheControl ([]string) which is the Cache-Control header value.
func writeEmbeddedAsset(w http.ResponseWriter, r *http.Request, asset *daemon_frontend.EmbeddedAsset, name string, cacheControl []string) {
	h := w.Header()
	h[headerETag] = []string{asset.ETag}
	h[headerCacheControl] = cacheControl
	h.Add(headerVary, headerAcceptEncoding)
	h[headerContentType] = []string{asset.MimeType}
	if asset.Encoding != "" {
		h[headerContentEncoding] = []string{asset.Encoding}
	}

	http.ServeContent(w, r, filepath.Base(name), time.Time{}, bytes.NewReader(asset.Content))
}

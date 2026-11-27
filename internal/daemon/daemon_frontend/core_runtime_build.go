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
	"encoding/hex"
)

const (
	// CoreRuntimeBuildDevelopment is the unminified framework runtime, built with its
	// development checks and warnings. Dev and dev-i modes send it.
	CoreRuntimeBuildDevelopment CoreRuntimeBuild = iota

	// CoreRuntimeBuildProduction is the minified framework runtime, built without the
	// development checks and warnings. Production mode sends it.
	CoreRuntimeBuildProduction
)

const (
	// coreMinifiedModuleFile is the file name of the minified core runtime ES module.
	coreMinifiedModuleFile = "ppframework.core.min.es.js"

	// componentsMinifiedModuleFile is the file name of the minified PKC components ES
	// module.
	componentsMinifiedModuleFile = "ppframework.components.min.es.js"

	// animationMinifiedModuleFile is the file name of the minified animation extension ES
	// module.
	animationMinifiedModuleFile = "ppframework.animation.min.es.js"

	// analyticsModuleFile is the file name of the analytics extension ES module.
	analyticsModuleFile = "ppframework.analytics.es.js"

	// analyticsMinifiedModuleFile is the file name of the minified analytics extension ES
	// module.
	analyticsMinifiedModuleFile = "ppframework.analytics.min.es.js"

	// modalsModuleFile is the file name of the modals extension ES module.
	modalsModuleFile = "ppframework.modals.es.js"

	// modalsMinifiedModuleFile is the file name of the minified modals extension ES module.
	modalsMinifiedModuleFile = "ppframework.modals.min.es.js"

	// toastsModuleFile is the file name of the toasts extension ES module.
	toastsModuleFile = "ppframework.toasts.es.js"

	// toastsMinifiedModuleFile is the file name of the minified toasts extension ES module.
	toastsMinifiedModuleFile = "ppframework.toasts.min.es.js"

	// devModuleFile is the file name of the dev tools extension ES module.
	devModuleFile = "ppframework.dev.es.js"

	// devMinifiedModuleFile is the file name of the minified dev tools extension ES module.
	devMinifiedModuleFile = "ppframework.dev.min.es.js"

	// builtAssetPrefix starts the asset store path of every embedded framework bundle.
	builtAssetPrefix = "built/"
)

// CoreRuntimeBuild names the build of the framework runtime the server sends. The build
// covers the core runtime and every built-in extension module that has an unminified and
// a minified file.
//
// Each of those modules has one URL, named after its unminified file, so the browser only
// ever loads one copy of it. The build decides which bytes the URL answers with, and so
// which SRI hash pages must carry.
type CoreRuntimeBuild uint8

var (
	// minifiedModuleFiles maps each module that has build variants to the minified file the
	// production build sends at its URL.
	minifiedModuleFiles = map[string]string{
		coreModuleFile:       coreMinifiedModuleFile,
		componentsModuleFile: componentsMinifiedModuleFile,
		animationModuleFile:  animationMinifiedModuleFile,
		analyticsModuleFile:  analyticsMinifiedModuleFile,
		modalsModuleFile:     modalsMinifiedModuleFile,
		toastsModuleFile:     toastsMinifiedModuleFile,
		devModuleFile:        devMinifiedModuleFile,
	}
)

// ServedFileName returns the embedded bundle whose bytes answer a request for a bundle
// file.
//
// Production swaps each module that has build variants for its minified file. Every other
// file, source maps included, is served as named, and a value that names no build sends
// every file as named.
//
// Takes requestedFile (string) which is the bundle file name in the request.
//
// Returns string which is the file name of the bytes to send.
func (build CoreRuntimeBuild) ServedFileName(requestedFile string) string {
	if build != CoreRuntimeBuildProduction {
		return requestedFile
	}
	if minifiedFile, found := minifiedModuleFiles[requestedFile]; found {
		return minifiedFile
	}
	return requestedFile
}

// CoreFileName returns the embedded file this build sends at the core module URL.
//
// Returns string which is the bundle file name.
func (build CoreRuntimeBuild) CoreFileName() string {
	return build.ServedFileName(coreModuleFile)
}

// CoreModuleSRIHash returns the SRI hash of the core runtime bytes this build sends.
//
// Returns string which is the integrity hash, or empty when SRI is disabled.
func (build CoreRuntimeBuild) CoreModuleSRIHash() string {
	return build.ModuleSRIHash(coreModuleFile)
}

// ModuleSRIHash returns the SRI hash of the bytes this build sends at a bundle URL.
//
// Takes fileName (string) which is the bundle file name in the URL.
//
// Returns string which is the integrity hash, or empty when SRI is disabled.
func (build CoreRuntimeBuild) ModuleSRIHash(fileName string) string {
	return GetSRIHash(builtAssetPrefix + build.ServedFileName(fileName))
}

// RuntimeVersion returns the runtime version a server sending this build names.
//
// It is the version in the URL of every module that has build variants, and in
// RuntimeVersionHeader. Each build has its own, so a production server and a dev server
// never share a versioned module URL, and a browser never reuses one build's cached bytes
// for the other.
//
// Returns string which is the version, or empty when the bundle version is unknown.
func (build CoreRuntimeBuild) RuntimeVersion() string {
	return build.VersionOf(RuntimeVersion())
}

// VersionOf derives this build's runtime version from a bundle version.
//
// Takes bundleVersion (string) which is the fingerprint of the embedded bundles.
//
// Returns string which is sixteen lowercase hex characters, or empty when bundleVersion
// is empty.
func (build CoreRuntimeBuild) VersionOf(bundleVersion string) string {
	if bundleVersion == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(bundleVersion + "\x00" + build.CoreFileName()))
	return hex.EncodeToString(sum[:runtimeVersionBytes])
}

// CoreModuleURL returns the versioned URL pages load this build of the core runtime from.
//
// Returns string which is the URL of ppframework.core.es.js under this build's version.
func (build CoreRuntimeBuild) CoreModuleURL() string {
	return build.ModuleURL(coreModuleFile)
}

// ComponentsModuleURL returns the versioned URL pages load this build of the PKC
// components runtime from.
//
// Returns string which is the URL of ppframework.components.es.js under this build's
// version.
func (build CoreRuntimeBuild) ComponentsModuleURL() string {
	return build.ModuleURL(componentsModuleFile)
}

// ModuleURL returns the URL of a bundle file under this build's version.
//
// Takes fileName (string) which is the bundle file name.
//
// Returns string which is the versioned URL, or the unversioned URL when the bundle
// version is unknown.
func (build CoreRuntimeBuild) ModuleURL(fileName string) string {
	return runtimeAssetURLFor(build.RuntimeVersion(), fileName)
}

// HasBuildVariants reports whether a bundle file name is a module whose bytes depend on
// the build.
//
// Takes fileName (string) which is the bundle file name.
//
// Returns bool which is true for the core runtime and the built-in extension modules that
// have an unminified and a minified file.
func HasBuildVariants(fileName string) bool {
	_, found := minifiedModuleFiles[fileName]
	return found
}

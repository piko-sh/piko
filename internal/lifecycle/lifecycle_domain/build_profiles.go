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

package lifecycle_domain

import (
	"path/filepath"
	"slices"
	"strings"

	"piko.sh/piko/internal/capabilities/capabilities_dto"
	"piko.sh/piko/internal/registry/registry_dto"
)

const (
	// profileMinified is the profile name for minified output variants.
	profileMinified = "minified"

	// profileGzip is the profile name for gzip compression.
	profileGzip = "gzip"

	// profileBrotli is the profile name for Brotli compression.
	profileBrotli = "br"

	// profileSource is the base profile name for unprocessed source files.
	profileSource = "source"

	// profileCompiledJS is the profile name for compiled JavaScript components.
	profileCompiledJS = "compiled_js"

	// profileTranspiledJS is the profile name for TypeScript-to-JavaScript
	// transpilation output.
	profileTranspiledJS = "transpiled_js"

	// blobStoreID is the storage backend identifier for cached assets.
	blobStoreID = "local_disk_cache"

	// mimeTypeJS is the MIME type for JavaScript files.
	mimeTypeJS = "application/javascript"

	// mimeTypeCSS is the MIME type for CSS stylesheets.
	mimeTypeCSS = "text/css"

	// mimeTypeSVG is the MIME type for SVG images.
	mimeTypeSVG = "image/svg+xml"

	// mimeTypeIcon is the MIME type for icon files.
	mimeTypeIcon = "image/x-icon"

	// mimeTypePNG is the MIME type for PNG image files.
	mimeTypePNG = "image/png"

	// mimeTypeWebmanifst is the MIME type for web app manifest files.
	mimeTypeWebmanifst = "application/manifest+json"

	// extensionMinJS is the file extension for pre-minified JavaScript files.
	extensionMinJS = ".min.js"

	// minifiedTypeJS is the asset type tag for minified JavaScript output.
	minifiedTypeJS = "minified-js"

	// profileSliceCapacity is the default pre-allocation capacity for
	// profile slices during build profile construction.
	profileSliceCapacity = 4
)

// profileContext holds the data needed when building file profiles.
type profileContext struct {
	// artefactID is the path or name that identifies the artefact being processed.
	artefactID string

	// ext is the file extension, including the leading dot.
	ext string
}

// profileBuilder is a function type that builds profiles from a given context.
type profileBuilder func(ctx profileContext) []registry_dto.NamedProfile

// extensionProfiles maps file extensions to their profile builders.
var extensionProfiles = map[string]profileBuilder{
	".pkc":         buildComponentProfiles,
	".ico":         buildStaticAssetProfiles,
	".png":         buildStaticAssetProfiles,
	".webmanifest": buildStaticAssetProfiles,
	".svg":         buildSVGProfiles,
	".css":         buildCSSProfiles,
	".js":          buildJSProfiles,
	".ts":          buildTypeScriptProfiles,
}

// GetProfilesForFile returns the processing profiles for a file based on its
// extension.
//
// Takes artefactID (string) which is the path or name of the file.
// Takes ignoreExt ([]string) which lists extensions to skip.
//
// Returns []registry_dto.NamedProfile which contains the output profiles for
// the file, or nil if the extension is in the ignore list or not recognised.
func GetProfilesForFile(artefactID string, ignoreExt []string) []registry_dto.NamedProfile {
	ext := strings.ToLower(filepath.Ext(artefactID))

	if slices.Contains(ignoreExt, ext) {
		return nil
	}

	builder, ok := extensionProfiles[ext]
	if !ok {
		return nil
	}

	return builder(profileContext{artefactID: artefactID, ext: ext})
}

// makeProfile creates a NamedProfile with the given parameters.
//
// Takes name (string) which specifies the unique identifier for the profile.
// Takes priority (registry_dto.ProfilePriority) which sets the execution order.
// Takes capability (string) which identifies the capability this profile uses.
// Takes dependsOn (string) which specifies a dependency profile name, or empty
// for no dependency.
// Takes tags (map[string]string) which provides key-value pairs for the
// resulting tags.
// Takes params (map[string]string) which provides key-value configuration
// parameters.
//
// Returns registry_dto.NamedProfile which is the constructed profile ready for
// use in component and compression chain building.
func makeProfile(name string, priority registry_dto.ProfilePriority, capability string, dependsOn string, tags map[string]string, params map[string]string) registry_dto.NamedProfile {
	var resultingTags registry_dto.Tags
	for k, v := range tags {
		resultingTags.SetByName(k, v)
	}

	var deps registry_dto.Dependencies
	if dependsOn != "" {
		deps.Add(dependsOn)
	}

	var profileParams registry_dto.ProfileParams
	for k, v := range params {
		profileParams.SetByName(k, v)
	}

	return registry_dto.NamedProfile{
		Name: name,
		Profile: registry_dto.DesiredProfile{
			Priority:       priority,
			CapabilityName: capability,
			Params:         profileParams,
			DependsOn:      deps,
			ResultingTags:  resultingTags,
		},
	}
}

// buildComponentProfiles creates profiles for .pkc component files.
//
// Takes ctx (profileContext) which provides the artefact ID and file extension.
//
// Returns []registry_dto.NamedProfile which contains a profile for the compiled
// JavaScript and the minify/compress chain profiles.
func buildComponentProfiles(ctx profileContext) []registry_dto.NamedProfile {
	componentName := strings.TrimSuffix(filepath.Base(ctx.artefactID), ctx.ext)

	profiles := make([]registry_dto.NamedProfile, 0, profileSliceCapacity)
	profiles = append(profiles, makeProfile(profileCompiledJS, registry_dto.PriorityNeed, capabilities_dto.CapabilityCompileComponent.String(), profileSource,
		map[string]string{
			"type":             "component-js",
			"role":             "entrypoint",
			"storageBackendId": blobStoreID,
			"mimeType":         mimeTypeJS,
			"tagName":          componentName,
			"fileExtension":    ".js",
		},
		map[string]string{"tagName": componentName, "sourcePath": ctx.artefactID},
	))

	return append(profiles, buildMinifyCompressChain(
		capabilities_dto.CapabilityMinifyJS.String(),
		profileCompiledJS,
		minifiedTypeJS,
		mimeTypeJS,
		extensionMinJS,
	)...)
}

// buildStaticAssetProfiles creates profiles for static assets that only need
// compression.
//
// Takes ctx (profileContext) which provides the file extension and other
// context for building the profile.
//
// Returns []registry_dto.NamedProfile which contains the compression profiles
// for the static asset.
func buildStaticAssetProfiles(ctx profileContext) []registry_dto.NamedProfile {
	mimeType := getMimeTypeForExtension(ctx.ext)
	return buildCompressChain(profileSource, "asset", mimeType, ctx.ext)
}

// buildSVGProfiles creates profiles for SVG files.
//
// Returns []registry_dto.NamedProfile which holds the minify and compress
// chain settings for SVG files.
func buildSVGProfiles(_ profileContext) []registry_dto.NamedProfile {
	return buildMinifyCompressChain(
		capabilities_dto.CapabilityMinifySVG.String(),
		profileSource,
		"minified-svg",
		mimeTypeSVG,
		".min.svg",
	)
}

// buildCSSProfiles creates profiles for CSS files.
//
// Returns []registry_dto.NamedProfile which contains the minify and compress
// chain for CSS processing.
func buildCSSProfiles(_ profileContext) []registry_dto.NamedProfile {
	return buildMinifyCompressChain(
		capabilities_dto.CapabilityMinifyCSS.String(),
		profileSource,
		"minified-css",
		mimeTypeCSS,
		".min.css",
	)
}

// isPreMinified reports whether a file name indicates the asset has already
// been minified (e.g. "foo.min.js", "bar.min.es.js"). Double-minifying
// breaks variable scoping in the output, so these files must skip the
// minification step.
//
// Takes name (string) which is the artefact ID or file path to check.
//
// Returns bool which is true if the file name contains a ".min" segment
// before the final ".js" extension.
func isPreMinified(name string) bool {
	base := filepath.Base(name)
	withoutJS := strings.TrimSuffix(base, ".js")
	if withoutJS == base {
		return false
	}
	return strings.HasSuffix(withoutJS, ".min") ||
		strings.Contains(withoutJS, ".min.")
}

// buildJSProfiles creates processing profiles for JavaScript files.
//
// Takes ctx (profileContext) which provides the artefact context for profile
// creation.
//
// Returns []registry_dto.NamedProfile which contains the minify and compress
// chain profiles for JavaScript processing.
func buildJSProfiles(ctx profileContext) []registry_dto.NamedProfile {
	if isPreMinified(ctx.artefactID) {
		return buildCompressChain(profileSource, minifiedTypeJS, mimeTypeJS, extensionMinJS)
	}

	if strings.HasPrefix(ctx.artefactID, "pk-js/") {
		profiles := make([]registry_dto.NamedProfile, 0, profileSliceCapacity)
		profiles = append(profiles, makeProfile("readable", registry_dto.PriorityWant,
			capabilities_dto.CapabilityCopyJS.String(), profileSource,
			map[string]string{
				"type":             "readable-pk-js",
				"storageBackendId": blobStoreID,
				"mimeType":         mimeTypeJS,
				"fileExtension":    ".js",
			},
			nil,
		))

		return append(profiles, buildMinifyCompressChainWithPriority(
			capabilities_dto.CapabilityMinifyJS.String(),
			profileSource,
			"minified-pk-js",
			mimeTypeJS,
			extensionMinJS,
			registry_dto.PriorityNeed,
			"compressed-pk-js",
		)...)
	}

	return buildMinifyCompressChain(
		capabilities_dto.CapabilityMinifyJS.String(),
		profileSource,
		minifiedTypeJS,
		mimeTypeJS,
		extensionMinJS,
	)
}

// getMimeTypeForExtension returns the MIME type for a given file extension.
//
// Takes ext (string) which is the file extension including the leading dot.
//
// Returns string which is the MIME type for the extension.
func getMimeTypeForExtension(ext string) string {
	switch ext {
	case ".png":
		return mimeTypePNG
	case ".webmanifest":
		return mimeTypeWebmanifst
	default:
		return mimeTypeIcon
	}
}

// buildTypeScriptProfiles creates profiles for TypeScript files that need
// transpilation to JavaScript, followed by minification and compression.
//
// Takes ctx (profileContext) which provides the artefact ID and file extension.
//
// Returns []registry_dto.NamedProfile which contains the transpilation profile
// followed by the minify/compress chain.
func buildTypeScriptProfiles(ctx profileContext) []registry_dto.NamedProfile {
	profiles := make([]registry_dto.NamedProfile, 0, profileSliceCapacity)
	profiles = append(profiles, makeProfile(profileTranspiledJS, registry_dto.PriorityNeed,
		capabilities_dto.CapabilityTranspileTypeScript.String(), profileSource,
		map[string]string{
			"type":             "transpiled-js",
			"storageBackendId": blobStoreID,
			"mimeType":         mimeTypeJS,
			"fileExtension":    ".js",
		},
		map[string]string{"sourcePath": ctx.artefactID},
	))

	return append(profiles, buildMinifyCompressChain(
		capabilities_dto.CapabilityMinifyJS.String(),
		profileTranspiledJS,
		minifiedTypeJS,
		mimeTypeJS,
		extensionMinJS,
	)...)
}

// NormaliseAssetArtefactID replaces transpilable source extensions in an
// artefact ID with the output extension. TypeScript ".ts" becomes ".js" so
// that the registry entry matches the path browsers will request.
//
// Takes artefactID (string) which is the computed artefact identifier.
//
// Returns string which is the normalised artefact ID.
func NormaliseAssetArtefactID(artefactID string) string {
	if base, ok := strings.CutSuffix(artefactID, ".ts"); ok {
		return base + ".js"
	}
	return artefactID
}

// buildMinifyCompressChain creates a chain of profiles for minifying and then
// compressing an asset with gzip and brotli.
//
// Takes minifyCapability (string) which specifies the capability needed to
// minify the asset.
// Takes dependsOn (string) which specifies the input profile this chain
// depends on.
// Takes minifiedType (string) which specifies the asset type after minifying.
// Takes mimeType (string) which specifies the MIME type for the asset.
// Takes minExt (string) which specifies the file extension for minified files.
//
// Returns []registry_dto.NamedProfile which contains the minify and compress
// profiles for the chain.
func buildMinifyCompressChain(
	minifyCapability, dependsOn, minifiedType, mimeType, minExt string,
) []registry_dto.NamedProfile {
	compressedType := "compressed-" + strings.TrimPrefix(minifiedType, "minified-")
	return buildMinifyCompressChainWithPriority(
		minifyCapability, dependsOn, minifiedType, mimeType, minExt,
		registry_dto.PriorityWant, compressedType,
	)
}

// buildMinifyCompressChainWithPriority creates a minify -> compress chain with
// custom priority.
//
// Takes minifyCapability (string) which specifies the capability for minifying.
// Takes dependsOn (string) which specifies the profile this chain depends on.
// Takes minifiedType (string) which specifies the type for minified output.
// Takes mimeType (string) which specifies the MIME type for the output.
// Takes minExt (string) which specifies the file extension for minified files.
// Takes minifyPriority (registry_dto.ProfilePriority) which sets the priority
// for the minify profile.
// Takes compressedType (string) which specifies the type for compressed output.
//
// Returns []registry_dto.NamedProfile which contains the minify profile
// followed by gzip and brotli compression profiles.
func buildMinifyCompressChainWithPriority(
	minifyCapability, dependsOn, minifiedType, mimeType, minExt string,
	minifyPriority registry_dto.ProfilePriority,
	compressedType string,
) []registry_dto.NamedProfile {
	return []registry_dto.NamedProfile{
		makeProfile(profileMinified, minifyPriority, minifyCapability, dependsOn,
			map[string]string{
				"type":             minifiedType,
				"storageBackendId": blobStoreID,
				"mimeType":         mimeType,
				"fileExtension":    minExt,
			},
			nil,
		),
		makeProfile(profileGzip, registry_dto.PriorityWant, capabilities_dto.CapabilityCompressGzip.String(), profileMinified,
			map[string]string{
				"type":             compressedType,
				"contentEncoding":  "gzip",
				"storageBackendId": blobStoreID,
				"mimeType":         mimeType,
				"fileExtension":    minExt + ".gz",
			},
			nil,
		),
		makeProfile(profileBrotli, registry_dto.PriorityWant, capabilities_dto.CapabilityCompressBrotli.String(), profileMinified,
			map[string]string{
				"type":             compressedType,
				"contentEncoding":  "br",
				"storageBackendId": blobStoreID,
				"mimeType":         mimeType,
				"fileExtension":    minExt + ".br",
			},
			nil,
		),
	}
}

// buildCompressChain creates a gzip and brotli profile chain from a source
// profile.
//
// Takes dependsOn (string) which specifies the source profile this chain
// depends on.
// Takes assetType (string) which identifies the type of asset being compressed.
// Takes mimeType (string) which specifies the content type of the asset.
// Takes ext (string) which provides the base file extension for compressed
// outputs.
//
// Returns []registry_dto.NamedProfile which contains the gzip and brotli
// compression profiles.
func buildCompressChain(dependsOn, assetType, mimeType, ext string) []registry_dto.NamedProfile {
	return []registry_dto.NamedProfile{
		makeProfile(profileGzip, registry_dto.PriorityWant, capabilities_dto.CapabilityCompressGzip.String(), dependsOn,
			map[string]string{
				"type":             assetType,
				"contentEncoding":  "gzip",
				"storageBackendId": blobStoreID,
				"mimeType":         mimeType,
				"fileExtension":    ext + ".gz",
			},
			nil,
		),
		makeProfile(profileBrotli, registry_dto.PriorityWant, capabilities_dto.CapabilityCompressBrotli.String(), dependsOn,
			map[string]string{
				"type":             assetType,
				"contentEncoding":  "br",
				"storageBackendId": blobStoreID,
				"mimeType":         mimeType,
				"fileExtension":    ext + ".br",
			},
			nil,
		),
	}
}

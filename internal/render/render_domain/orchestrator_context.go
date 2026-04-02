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

package render_domain

import (
	"net/http"
	"sync"

	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/daemon/daemon_dto"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/mem"
	"piko.sh/piko/internal/render/render_dto"
)

// populateLocaleFromRequest extracts the locale from the request context.
//
// Takes request (*http.Request) which provides the request context containing the
// locale value. If request is nil, the method returns without changing state.
func (rctx *renderContext) populateLocaleFromRequest(request *http.Request) {
	if request == nil {
		return
	}
	if pctx := daemon_dto.PikoRequestCtxFromContext(request.Context()); pctx != nil {
		rctx.currentLocale = pctx.Locale
	} else {
		rctx.currentLocale = ""
	}
}

// populateI18nFromConfig sets the i18n strategy and default locale from the
// site settings.
//
// Takes siteConfig (*config.WebsiteConfig) which provides the i18n settings to
// use. When nil, the strategy and locale are cleared.
func (rctx *renderContext) populateI18nFromConfig(siteConfig *config.WebsiteConfig) {
	if siteConfig != nil {
		rctx.i18nStrategy = siteConfig.I18n.Strategy
		rctx.defaultLocale = siteConfig.I18n.DefaultLocale
	} else {
		rctx.i18nStrategy = ""
		rctx.defaultLocale = ""
	}
}

// clearCaches resets all cached data so the context can be reused.
func (rctx *renderContext) clearCaches() {
	for k := range rctx.collectedCustomComponents {
		delete(rctx.collectedCustomComponents, k)
	}
	for k := range rctx.linkHeaderSet {
		delete(rctx.linkHeaderSet, k)
	}
	rctx.collectedLinkHeaders = rctx.collectedLinkHeaders[:0]
	rctx.requiredSvgSymbols = rctx.requiredSvgSymbols[:0]
	for k := range rctx.customTags {
		delete(rctx.customTags, k)
	}
	for k := range rctx.mergedAttrsCache {
		delete(rctx.mergedAttrsCache, k)
	}
	for k := range rctx.registeredDynamicAssets {
		delete(rctx.registeredDynamicAssets, k)
	}
	for k := range rctx.srcsetCache {
		delete(rctx.srcsetCache, k)
	}
}

// resetDiagnosticsAndCSRF clears all warnings, errors, and CSRF state to
// prepare for a new request.
func (rctx *renderContext) resetDiagnosticsAndCSRF() {
	rctx.diagnostics.Warnings = rctx.diagnostics.Warnings[:0]
	rctx.diagnostics.Errors = rctx.diagnostics.Errors[:0]
	rctx.csrfPair = nil
	rctx.csrfOnce = sync.Once{}
	rctx.csrfError = nil
}

// putRenderContext returns a render context to the pool after clearing its
// fields.
//
// Takes rctx (*renderContext) which is the context to clear and return.
func (*RenderOrchestrator) putRenderContext(rctx *renderContext) {
	rctx.originalCtx = nil
	rctx.httpRequest = nil
	rctx.httpResponse = nil
	rctx.skipPrerenderedHTML = false

	rctx.currentLocale = ""
	rctx.i18nStrategy = ""
	rctx.defaultLocale = ""
	rctx.isEmailMode = false

	if rctx.probeData != nil {
		render_dto.ReleaseProbeData(rctx.probeData)
		rctx.probeData = nil
	}
	rctx.componentMetadata = nil
	rctx.csrfPair = nil
	rctx.csrfError = nil

	for k := range rctx.registeredDynamicAssets {
		delete(rctx.registeredDynamicAssets, k)
	}

	for k := range rctx.srcsetCache {
		delete(rctx.srcsetCache, k)
	}

	for _, buffer := range rctx.frozenBuffers {
		*buffer = (*buffer)[:0]
		byteBufferPool.Put(buffer)
	}
	rctx.frozenBuffers = rctx.frozenBuffers[:0]

	if rctx.csrfBuf != nil {
		rctx.csrfBuf.Reset()
		csrfBufPool.Put(rctx.csrfBuf)
		rctx.csrfBuf = nil
	}

	renderContextPool.Put(rctx)
}

// getBuffer retrieves a reusable byte buffer from the pool.
// The buffer should be returned via freezeToString for safe zero-copy
// conversion.
//
// Returns *[]byte which is a buffer from the pool ready for use.
func (rctx *renderContext) getBuffer() *[]byte {
	if buffer, ok := byteBufferPool.Get().(*[]byte); ok {
		return buffer
	}
	_, l := logger_domain.From(rctx.originalCtx, log)
	l.Error("byteBufferPool returned unexpected type, allocating new instance")
	return new(make([]byte, 0, byteBufferInitialCapacity))
}

// freezeToString converts a buffer to a string without copying using
// mem.String.
//
// The buffer is not returned to the pool until the request ends via
// putRenderContext. This makes the zero-copy conversion safe because the
// buffer lifetime exceeds the string lifetime.
//
// Takes buffer (*[]byte) which is the buffer to freeze.
//
// Returns string which is the zero-copy string view of the buffer contents.
//
// SAFETY: The buffer must be obtained from getBuffer and must not be modified
// after this call.
func (rctx *renderContext) freezeToString(buffer *[]byte) string {
	rctx.frozenBuffers = append(rctx.frozenBuffers, buffer)
	return mem.String(*buffer)
}

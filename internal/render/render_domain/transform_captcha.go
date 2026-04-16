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
	"fmt"
	"slices"
	"strings"
	"sync/atomic"

	qt "github.com/valyala/quicktemplate"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/captcha/captcha_domain"
	"piko.sh/piko/internal/captcha/captcha_dto"
	"piko.sh/piko/internal/logger/logger_domain"
)

const (
	// tagPikoCaptcha is the tag name for the piko:captcha server-side element.
	tagPikoCaptcha = "piko:captcha"

	// defaultCaptchaFieldName is the default hidden input name for the captcha
	// token when no name attribute is specified.
	defaultCaptchaFieldName = "_captcha_token"

	// defaultCaptchaTheme is the default widget theme.
	defaultCaptchaTheme = "light"

	// defaultCaptchaSize is the default widget size.
	defaultCaptchaSize = "normal"
)

// captchaRenderLog is the logger for captcha rendering operations.
var captchaRenderLog = logger_domain.GetLogger("piko/internal/render/render_domain/captcha")

// captchaIDCounter generates unique IDs for captcha widget instances.
var captchaIDCounter atomic.Uint64

// captchaScriptInfo holds the script requirements collected during rendering
// for a single captcha provider, used by the probe phase to add scripts to
// the page head.
type captchaScriptInfo struct {
	// InitScriptArtefactID is the registry artefact ID for the init script.
	InitScriptArtefactID string

	// SDKScriptURLs lists external provider SDK script URLs.
	SDKScriptURLs []string
}

// WithCaptchaService returns a RenderOrchestratorOption that sets the captcha
// service used to render piko:captcha elements.
//
// Takes service (captcha_domain.CaptchaServicePort) which is the captcha
// service to configure on the orchestrator.
//
// Returns RenderOrchestratorOption which applies the service to the
// orchestrator.
func WithCaptchaService(service captcha_domain.CaptchaServicePort) RenderOrchestratorOption {
	return func(ro *RenderOrchestrator) {
		ro.captchaService = service
	}
}

// renderPikoCaptcha replaces a <piko:captcha> element with provider-specific
// HTML, emitting a pre-populated hidden input for server-side providers or a
// data-attribute container div with collected scripts for cloud providers.
//
// Takes ro (*RenderOrchestrator) which is the orchestrator holding the captcha
// service.
//
// Takes node (*ast_domain.TemplateNode) which is the piko:captcha AST element.
//
// Takes qw (*qt.Writer) which is the output writer for the rendered HTML.
//
// Takes rctx (*renderContext) which is the per-request render context for
// collecting script metadata.
//
// Returns error which is non-nil if token generation fails.
func renderPikoCaptcha(
	ro *RenderOrchestrator,
	node *ast_domain.TemplateNode,
	qw *qt.Writer,
	rctx *renderContext,
) error {
	if ro.captchaService == nil || !ro.captchaService.IsEnabled() {
		captchaRenderLog.Warn("piko:captcha element rendered but captcha service not configured")
		qw.N().S("<!-- piko:captcha: captcha service not configured -->")
		return nil
	}

	attrs := readCaptchaAttributes(node)

	var (
		provider captcha_domain.CaptchaProvider
		err      error
	)
	if attrs.providerName != "" {
		provider, err = ro.captchaService.GetProviderByName(rctx.originalCtx, attrs.providerName)
	} else {
		provider, err = ro.captchaService.GetDefaultProvider(rctx.originalCtx)
	}

	if err != nil {
		captchaRenderLog.Warn("piko:captcha provider lookup failed", logger_domain.Error(err))
		qw.N().S("<!-- piko:captcha: provider unavailable -->")
		return nil
	}

	requirements := provider.RenderRequirements()
	if requirements == nil {
		captchaRenderLog.Warn("piko:captcha provider returned nil RenderRequirements",
			logger_domain.String("provider", attrs.providerName))
		qw.N().S("<!-- piko:captcha: provider returned nil render requirements -->")
		return nil
	}

	if requirements.ServerSideToken {
		return renderServerSideCaptcha(provider, qw, attrs.fieldName, attrs.action)
	}

	attrs.elementID = generateCaptchaElementID()
	attrs.siteKey = provider.SiteKey()
	return renderClientSideCaptcha(requirements, qw, rctx, attrs)
}

// renderServerSideCaptcha outputs a pre-populated hidden input for providers
// that generate tokens at render time (e.g. HMAC challenge).
//
// Takes provider (captcha_domain.CaptchaProvider) which is the server-side
// captcha provider.
//
// Takes qw (*qt.Writer) which is the output writer for the rendered HTML.
//
// Takes fieldName (string) which is the hidden input name for the token.
//
// Takes action (string) which is the captcha action identifier.
//
// Returns error which is non-nil if challenge generation fails.
func renderServerSideCaptcha(
	provider captcha_domain.CaptchaProvider,
	qw *qt.Writer,
	fieldName, action string,
) error {
	type challengeGenerator interface {
		GenerateChallenge(action string) (string, error)
	}

	generator, ok := provider.(challengeGenerator)
	if !ok {
		captchaRenderLog.Error("piko:captcha provider claims ServerSideToken but does not implement GenerateChallenge")
		qw.N().S("<!-- piko:captcha: provider does not support server-side token generation -->")
		return nil
	}

	token, err := generator.GenerateChallenge(action)
	if err != nil {
		return fmt.Errorf("generating captcha challenge: %w", err)
	}

	qw.N().S(`<input type="hidden" name="`)
	qw.E().S(fieldName)
	qw.N().S(`" value="`)
	qw.E().S(token)
	qw.N().S(`" />`)

	return nil
}

// captchaWidgetParams holds the resolved attribute values for a single
// piko:captcha element.
type captchaWidgetParams struct {
	// providerName is the configured captcha provider identifier.
	providerName string

	// elementID is the unique DOM ID for the captcha widget container.
	elementID string

	// fieldName is the hidden input name that carries the captcha token.
	fieldName string

	// theme is the visual theme for the captcha widget (e.g. "light", "dark").
	theme string

	// size is the display size for the captcha widget (e.g. "normal", "compact").
	size string

	// action is the captcha action identifier passed to the provider.
	action string

	// siteKey is the provider-issued site key for the captcha widget.
	siteKey string
}

// renderClientSideCaptcha outputs a data-attribute container div (or hidden
// input for invisible providers) and collects the required scripts for the
// probe phase, emitting no inline scripts.
//
// Takes requirements (*captcha_dto.RenderRequirements) which describes the
// provider's rendering needs.
//
// Takes qw (*qt.Writer) which is the output writer for the rendered HTML.
//
// Takes rctx (*renderContext) which is the per-request render context for
// collecting script metadata.
//
// Takes params (captchaWidgetParams) which holds the resolved element
// attributes.
//
// Returns error which is always nil (reserved for future use).
func renderClientSideCaptcha(
	requirements *captcha_dto.RenderRequirements,
	qw *qt.Writer,
	rctx *renderContext,
	params captchaWidgetParams,
) error {
	providerName := params.providerName
	elementID := params.elementID
	fieldName := params.fieldName
	theme := params.theme
	size := params.size
	siteKey := params.siteKey
	providerType := requirements.ProviderType

	action := params.action

	if requirements.Invisible {
		qw.N().S(`<input type="hidden" name="`)
		qw.E().S(fieldName)
		qw.N().S(`" value="" pk-no-track data-captcha-provider="`)
		qw.E().S(providerType)
		qw.N().S(`" data-captcha-sitekey="`)
		qw.E().S(siteKey)
		qw.N().S(`" data-captcha-field="`)
		qw.E().S(fieldName)
		qw.N().S(`"`)
		if action != "" {
			qw.N().S(` data-captcha-action="`)
			qw.E().S(action)
			qw.N().S(`"`)
		}
		qw.N().S(` />`)
	} else {
		qw.N().S(`<div id="`)
		qw.E().S(elementID)
		qw.N().S(`" pk-no-track data-captcha-provider="`)
		qw.E().S(providerType)
		qw.N().S(`" data-captcha-sitekey="`)
		qw.E().S(siteKey)
		qw.N().S(`" data-captcha-theme="`)
		qw.E().S(theme)
		qw.N().S(`" data-captcha-size="`)
		qw.E().S(size)
		qw.N().S(`" data-captcha-field="`)
		qw.E().S(fieldName)
		qw.N().S(`"></div>`)

		qw.N().S(`<input type="hidden" name="`)
		qw.E().S(fieldName)
		qw.N().S(`" value="" pk-no-track />`)
	}

	if rctx.collectedCaptchaScripts == nil {
		rctx.collectedCaptchaScripts = make(map[string]*captchaScriptInfo)
	}

	if _, exists := rctx.collectedCaptchaScripts[providerName]; !exists {
		rctx.collectedCaptchaScripts[providerName] = &captchaScriptInfo{
			InitScriptArtefactID: fmt.Sprintf("captcha/init-%s.js", providerName),
			SDKScriptURLs:        requirements.ScriptURLs,
		}
	}

	return nil
}

// writeCaptchaScriptTags outputs the external SDK and init script tags for all
// captcha providers collected during rendering, using pre-resolved probe paths
// when available or falling back to direct path construction for headless
// renders.
//
// Takes qw (*qt.Writer) which is the output writer for the script elements.
//
// Takes rctx (*renderContext) which holds the collected captcha script metadata.
func writeCaptchaScriptTags(qw *qt.Writer, rctx *renderContext) {
	if len(rctx.collectedCaptchaScripts) == 0 {
		return
	}

	for providerName, info := range rctx.collectedCaptchaScripts {
		for _, sdkURL := range info.SDKScriptURLs {
			qw.N().S("\n")
			qw.N().S(`<script src="`)
			qw.E().S(sdkURL)
			qw.N().S(`" async defer></script>`)
		}

		servePath := resolveCaptchaInitScriptPath(rctx, providerName, info)
		qw.N().S("\n")
		qw.N().S(`<script src="`)
		qw.E().S(servePath)
		qw.N().S(`" async defer></script>`)
	}
}

// collectCaptchaWidgetScriptURLs returns all SDK and init script URLs from the
// collected captcha scripts. These are emitted as meta[name="pk-widget-script"]
// tags in the page footer so the framework can discover and load them during
// soft navigation.
//
// Takes rctx (*renderContext) which holds the collected captcha script metadata.
//
// Returns []string which contains the SDK and init script URLs, or nil if no
// captcha scripts were collected.
func collectCaptchaWidgetScriptURLs(rctx *renderContext) []string {
	if len(rctx.collectedCaptchaScripts) == 0 {
		return nil
	}

	var urls []string
	for providerName, info := range rctx.collectedCaptchaScripts {
		urls = append(urls, info.SDKScriptURLs...)
		urls = append(urls, resolveCaptchaInitScriptPath(rctx, providerName, info))
	}
	slices.Sort(urls)
	return urls
}

// resolveCaptchaInitScriptPath returns the pre-resolved serve path for a
// captcha init script from probe data, falling back to a direct path
// construction when probe data is unavailable (e.g. headless rendering).
//
// Takes rctx (*renderContext) which holds the probe data with resolved paths.
//
// Takes providerName (string) which identifies the captcha provider.
//
// Takes info (*captchaScriptInfo) which holds the artefact ID for fallback
// path construction.
//
// Returns string which is the serve path for the init script.
func resolveCaptchaInitScriptPath(rctx *renderContext, providerName string, info *captchaScriptInfo) string {
	if rctx.probeData != nil && rctx.probeData.CaptchaScripts != nil {
		if probeInfo, ok := rctx.probeData.CaptchaScripts[providerName]; ok && probeInfo.InitScriptServePath != "" {
			return probeInfo.InitScriptServePath
		}
	}
	captchaRenderLog.Warn("Captcha init script serve path not resolved by probe, using fallback",
		logger_domain.String("provider", providerName),
		logger_domain.String("artefactID", info.InitScriptArtefactID))
	return "/_piko/" + info.InitScriptArtefactID
}

// readCaptchaAttributes reads and defaults the piko:captcha element attributes.
//
// Takes node (*ast_domain.TemplateNode) which is the piko:captcha AST element
// to read attributes from.
//
// Returns captchaWidgetParams which holds the resolved attribute values with
// defaults applied for missing fields.
func readCaptchaAttributes(node *ast_domain.TemplateNode) captchaWidgetParams {
	p := captchaWidgetParams{
		providerName: getStaticAttribute(node, "provider"),
		fieldName:    getStaticAttribute(node, "name"),
		theme:        getStaticAttribute(node, "theme"),
		size:         getStaticAttribute(node, "size"),
		action:       getStaticAttribute(node, "action"),
	}
	if p.fieldName == "" {
		p.fieldName = defaultCaptchaFieldName
	}
	if p.theme == "" {
		p.theme = defaultCaptchaTheme
	}
	if p.size == "" {
		p.size = defaultCaptchaSize
	}
	return p
}

// sanitiseHTMLComment strips sequences that are forbidden or dangerous inside
// HTML comments, replacing all "--" runs and removing "<" / ">" to prevent
// spec violations and nested tag injection.
//
// Takes message (string) which is the raw text to sanitise.
//
// Returns string which is the sanitised text safe for embedding in an HTML
// comment.
func sanitiseHTMLComment(message string) string {
	message = strings.ReplaceAll(message, "--", "")
	message = strings.ReplaceAll(message, "<", "")
	message = strings.ReplaceAll(message, ">", "")
	return message
}

// getStaticAttribute returns the value of a static attribute on the node, or
// empty string if not found.
//
// Takes node (*ast_domain.TemplateNode) which is the AST node to inspect.
//
// Takes name (string) which is the attribute name to look up.
//
// Returns string which is the attribute value, or empty if not present.
func getStaticAttribute(node *ast_domain.TemplateNode, name string) string {
	for i := range node.Attributes {
		if node.Attributes[i].Name == name {
			return node.Attributes[i].Value
		}
	}
	return ""
}

// generateCaptchaElementID creates a unique ID for a captcha widget instance.
//
// Returns string which is a unique element ID of the form "piko-captcha-N".
func generateCaptchaElementID() string {
	counter := captchaIDCounter.Add(1)
	return fmt.Sprintf("piko-captcha-%d", counter)
}

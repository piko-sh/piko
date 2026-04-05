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
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"

	"github.com/go-chi/chi/v5"
	"piko.sh/piko/internal/daemon/daemon_frontend"
	"piko.sh/piko/internal/layouter/layouter_dto"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_domain"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_dto"
	"piko.sh/piko/internal/render/render_domain"
	"piko.sh/piko/internal/templater/templater_domain"
	"piko.sh/piko/internal/templater/templater_dto"
)

// DevPreviewHandler serves dev-mode preview endpoints for listing and
// rendering previewable templates. It is only mounted when the dev widget
// is enabled and is not accessible in production.
type DevPreviewHandler struct {
	// store provides access to manifest entries and their preview metadata.
	store templater_domain.ManifestStoreView

	// emailService renders email templates with CSS inlining.
	emailService templater_domain.EmailTemplateService

	// pdfService renders PDF templates to bytes.
	pdfService pdfwriter_domain.PdfWriterService

	// runner executes templates with arbitrary props.
	runner templater_domain.ManifestRunnerPort

	// renderer converts template ASTs to HTML output.
	renderer templater_domain.RendererPort

	// registry provides component metadata lookups for injecting the dev
	// widget's compiled JS on preview pages.
	registry render_domain.RegistryPort
}

// previewListResponse is the JSON response for the preview listing endpoint.
type previewListResponse struct {
	Groups []previewGroup `json:"groups"`
}

type previewGroup struct {
	Type string `json:"type"`

	Templates []previewTemplate `json:"templates"`
}

type previewTemplate struct {
	SourcePath string `json:"sourcePath"`

	Scenarios []previewScenario `json:"scenarios"`
}

type previewScenario struct {
	Name string `json:"name"`

	Description string `json:"description,omitempty"`
}

const (
	// componentTypePdf is the component type identifier for PDF templates.
	componentTypePdf = "pdf"

	// componentTypePartial is the component type identifier for partial
	// templates.
	componentTypePartial = "partial"

	// sourceExtension is the file extension for Piko source files.
	sourceExtension = ".pk"

	// previewActionButtonStyle is the inline CSS for the floating preview
	// copy button.
	previewActionButtonStyle = `position:fixed;top:16px;right:16px;z-index:99999;` +
		`width:36px;height:36px;border-radius:8px;` +
		`border:1px solid rgba(255,255,255,0.1);` +
		`background:rgba(24,24,27,0.85);` +
		`backdrop-filter:blur(8px);-webkit-backdrop-filter:blur(8px);` +
		`cursor:pointer;display:flex;align-items:center;justify-content:center;` +
		`color:#a78bfa;transition:all 0.15s;padding:0`

	// svgCopyIcon is a clipboard icon. Attributes use single quotes so the
	// SVG can be safely embedded inside double-quoted JavaScript strings.
	svgCopyIcon = `<svg width='18' height='18' viewBox='0 0 24 24' fill='none' ` +
		`stroke='currentColor' stroke-width='2' stroke-linecap='round' stroke-linejoin='round'>` +
		`<rect x='9' y='9' width='13' height='13' rx='2' ry='2'/>` +
		`<path d='M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1'/></svg>`

	// svgCheckIcon is a checkmark icon shown briefly after a successful copy.
	svgCheckIcon = `<svg width='18' height='18' viewBox='0 0 24 24' fill='none' ` +
		`stroke='currentColor' stroke-width='2' stroke-linecap='round' stroke-linejoin='round'>` +
		`<polyline points='20 6 9 17 4 12'/></svg>`
)

// copyButtonScript is the JavaScript that powers the copy-to-clipboard
// button in email previews. It is defined at package level so its CSP
// hash can be pre-computed once at init time.
var copyButtonScript = `(function(){` +
	`var b=document.getElementById("piko-copy-email");` +
	`b.addEventListener("click",function(){` +
	`var s=document.getElementById("piko-email-source").textContent;` +
	`var html=new TextDecoder().decode(Uint8Array.from(atob(s),function(c){return c.charCodeAt(0)}));` +
	`navigator.clipboard.writeText(html).then(function(){` +
	`var o=b.innerHTML;b.innerHTML="` + svgCheckIcon + `";b.style.color="#4ade80";` +
	`setTimeout(function(){b.innerHTML=o;b.style.color="#a78bfa"},1500)` +
	`})})` +
	`}())`

// copyButtonScriptCSPHash is the SHA-256 hash of copyButtonScript,
// formatted for inclusion in a Content-Security-Policy header.
var copyButtonScriptCSPHash string

func init() {
	hash := sha256.Sum256([]byte(copyButtonScript))
	copyButtonScriptCSPHash = "'sha256-" + base64.StdEncoding.EncodeToString(hash[:]) + "'"
}

// NewDevPreviewHandler creates a handler for dev preview endpoints.
//
// Takes store (templater_domain.ManifestStoreView) which provides manifest
// access.
// Takes runner (templater_domain.ManifestRunnerPort) which executes templates.
// Takes emailService (templater_domain.EmailTemplateService) which renders
// emails.
// Takes pdfService (pdfwriter_domain.PdfWriterService) which renders PDFs.
//
// Returns *DevPreviewHandler which is the initialised handler.
func NewDevPreviewHandler(
	store templater_domain.ManifestStoreView,
	runner templater_domain.ManifestRunnerPort,
	renderer templater_domain.RendererPort,
	emailService templater_domain.EmailTemplateService,
	pdfService pdfwriter_domain.PdfWriterService,
	registry render_domain.RegistryPort,
) *DevPreviewHandler {
	return &DevPreviewHandler{
		store:        store,
		runner:       runner,
		renderer:     renderer,
		emailService: emailService,
		pdfService:   pdfService,
		registry:     registry,
	}
}

// Mount registers the preview API and render routes on the router.
func (h *DevPreviewHandler) Mount(r chi.Router) {
	r.Get("/_piko/dev/api/previews", h.handleListPreviews)
	r.Get("/_piko/dev/preview/{type}/*", h.handleRenderPreview)
}

// handleListPreviews returns a JSON listing of all templates that have
// Preview functions, grouped by component type.
func (h *DevPreviewHandler) handleListPreviews(w http.ResponseWriter, _ *http.Request) {
	entries := h.store.ListPreviewEntries()

	grouped := make(map[string][]previewTemplate)
	for _, entry := range entries {
		scenarios := make([]previewScenario, 0, len(entry.Scenarios))
		for _, s := range entry.Scenarios {
			scenarios = append(scenarios, previewScenario{
				Name:        s.Name,
				Description: s.Description,
			})
		}
		grouped[entry.ComponentType] = append(grouped[entry.ComponentType], previewTemplate{
			SourcePath: entry.OriginalSourcePath,
			Scenarios:  scenarios,
		})
	}

	typeOrder := []string{"page", componentTypePartial, "email", componentTypePdf}
	groups := make([]previewGroup, 0, len(typeOrder))
	for _, t := range typeOrder {
		if templates, ok := grouped[t]; ok {
			groups = append(groups, previewGroup{
				Type:      t,
				Templates: templates,
			})
		}
	}

	writeJSON(w, http.StatusOK, previewListResponse{Groups: groups})
}

// handleRenderPreview renders a specific preview scenario for a template.
// URL format: /_piko/dev/preview/{type}/{path...}?scenario={name}
// For PDFs, /_piko/dev/preview/pdf/{path...}/render returns raw PDF bytes
// with X-Frame-Options relaxed to SAMEORIGIN so the wrapper page can embed it.
func (h *DevPreviewHandler) handleRenderPreview(w http.ResponseWriter, r *http.Request) {
	componentType := chi.URLParam(r, "type")
	rawPath := chi.URLParam(r, "*")
	scenarioName := r.URL.Query().Get("scenario")

	isPdfRender := componentType == componentTypePdf && strings.HasSuffix(rawPath, "/render")
	if isPdfRender {
		rawPath = strings.TrimSuffix(rawPath, "/render")
	}

	sourcePath := buildSourcePath(componentType, rawPath)

	scenario, err := h.findScenario(sourcePath, scenarioName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	switch componentType {
	case "email":
		h.renderEmailPreview(w, r, sourcePath, scenario)
	case componentTypePdf:
		if isPdfRender {
			w.Header().Set("X-Frame-Options", "SAMEORIGIN")
			h.renderPdfRaw(w, r, sourcePath, scenario)
		} else {
			h.renderPdfPreview(w, r, sourcePath, scenario, scenarioName)
		}
	case "page", componentTypePartial:
		h.renderComponentPreview(w, r, sourcePath, scenario, componentType)
	default:
		http.Error(w, "unknown component type", http.StatusBadRequest)
	}
}

// findScenario looks up a preview scenario by source path and scenario name.
func (h *DevPreviewHandler) findScenario(sourcePath, scenarioName string) (*templater_dto.PreviewScenario, error) {
	entry, ok := h.store.GetPageEntry(sourcePath)
	if !ok {
		return nil, fmt.Errorf("template not found: %s", sourcePath)
	}

	scenarios := entry.GetPreviewScenarios()
	if len(scenarios) == 0 {
		return nil, fmt.Errorf("template has no preview scenarios: %s", sourcePath)
	}

	if scenarioName == "" {
		return &scenarios[0], nil
	}

	for i := range scenarios {
		if scenarios[i].Name == scenarioName {
			return &scenarios[i], nil
		}
	}

	return nil, fmt.Errorf("scenario %q not found for template %s", scenarioName, sourcePath)
}

// buildSourcePath converts a component type and URL path back to a source path.
// For example, ("email", "welcome") becomes "emails/welcome.pk".
func buildSourcePath(componentType, path string) string {
	path = strings.TrimPrefix(path, "/")

	switch componentType {
	case "email":
		return "emails/" + path + sourceExtension
	case componentTypePdf:
		return "pdfs/" + path + sourceExtension
	case "page":
		return "pages/" + path + sourceExtension
	case componentTypePartial:
		return "partials/" + path + sourceExtension
	default:
		return path
	}
}

// renderEmailPreview renders an email template with the scenario's props and
// returns the CSS-inlined HTML with the dev widget injected.
func (h *DevPreviewHandler) renderEmailPreview(w http.ResponseWriter, r *http.Request, sourcePath string, scenario *templater_dto.PreviewScenario) {
	if h.emailService == nil {
		http.Error(w, "email service not available", http.StatusServiceUnavailable)
		return
	}

	result, err := h.emailService.Render(r.Context(), r, sourcePath, scenario.Props, nil, true)
	if err != nil {
		http.Error(w, fmt.Sprintf("email render failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set(headerContentType, contentTypeHTML)
	addCopyScriptHashToCSP(w)

	html := result.HTML
	injection := emailCopyButtonSnippet(result.HTML) + h.devWidgetSnippet(r.Context())
	if idx := strings.LastIndex(html, "</body>"); idx != -1 {
		html = html[:idx] + injection + html[idx:]
	} else {
		html += injection
	}

	_, _ = w.Write([]byte(html))
}

// renderPdfPreview serves an HTML wrapper page with an embedded PDF viewer
// and the dev widget.
func (h *DevPreviewHandler) renderPdfPreview(w http.ResponseWriter, r *http.Request, sourcePath string, _ *templater_dto.PreviewScenario, scenarioName string) {
	rawPath := strings.TrimPrefix(sourcePath, "pdfs/")
	rawPath = strings.TrimSuffix(rawPath, sourceExtension)

	renderURL := fmt.Sprintf("/_piko/dev/preview/pdf/%s/render", rawPath)
	if scenarioName != "" {
		renderURL += "?scenario=" + url.QueryEscape(scenarioName)
	}

	devSnippet := h.devWidgetSnippet(r.Context())

	w.Header().Set(headerContentType, contentTypeHTML)
	fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head><title>PDF Preview: %s</title></head>
<body style="margin:0; height:100vh">
  <embed src="%s" type="application/pdf" width="100%%" height="100%%">
  %s
</body>
</html>`, sourcePath, renderURL, devSnippet)
}

// renderPdfRaw renders a PDF template and returns the raw PDF bytes.
func (h *DevPreviewHandler) renderPdfRaw(w http.ResponseWriter, r *http.Request, sourcePath string, scenario *templater_dto.PreviewScenario) {
	if h.pdfService == nil {
		http.Error(w, "PDF service not available", http.StatusServiceUnavailable)
		return
	}

	result, err := h.pdfService.Render(r.Context(), r, sourcePath, scenario.Props, pdfwriter_dto.PdfConfig{
		Page: layouter_dto.PageA4,
	})
	if err != nil {
		http.Error(w, fmt.Sprintf("PDF render failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set(headerContentType, "application/pdf")
	_, _ = w.Write(result.Content)
}

// renderComponentPreview renders a page or partial with the scenario's props
// and returns the HTML.
func (h *DevPreviewHandler) renderComponentPreview(w http.ResponseWriter, r *http.Request, sourcePath string, scenario *templater_dto.PreviewScenario, componentType string) {
	if h.runner == nil || h.renderer == nil {
		http.Error(w, "template runner or renderer not available", http.StatusServiceUnavailable)
		return
	}

	syntheticRequest := httptest.NewRequest(http.MethodGet, "/", nil)
	syntheticRequest = syntheticRequest.WithContext(r.Context())

	pageDef := templater_dto.PageDefinition{
		OriginalPath:   sourcePath,
		NormalisedPath: "",
		TemplateHTML:   "",
	}

	templateAST, metadata, styling, err := h.runner.RunPartialWithProps(r.Context(), pageDef, syntheticRequest, scenario.Props)
	if err != nil {
		http.Error(w, fmt.Sprintf("render failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set(headerContentType, contentTypeHTML)

	isFragment := componentType == componentTypePartial

	renderErr := h.renderer.RenderPage(r.Context(), templater_domain.RenderPageParams{
		Writer:         w,
		ResponseWriter: w,
		Request:        syntheticRequest,
		PageDefinition: pageDef,
		TemplateAST:    templateAST,
		Metadata:       &metadata,
		IsFragment:     isFragment,
		Styling:        styling,
	})
	if renderErr != nil {
		http.Error(w, fmt.Sprintf("render failed: %v", renderErr), http.StatusInternalServerError)
		return
	}
}

// devWidgetSnippet returns an HTML snippet that injects the full dev widget
// (including its PKC component definition) and hot-reload scripts into a
// preview page.
func (h *DevPreviewHandler) devWidgetSnippet(ctx context.Context) string {
	widgetHTML := daemon_frontend.GetDevWidgetHTML()
	if widgetHTML == "" {
		return ""
	}

	coreURL := "/_piko/dist/ppframework.core.es.js"
	componentsURL := daemon_frontend.ModuleComponents.ServeURL()
	devURL := daemon_frontend.ModuleDev.ServeURL()

	var componentScript string
	if h.registry != nil {
		meta, err := h.registry.GetComponentMetadata(ctx, "piko-dev-widget")
		if err == nil && meta != nil && meta.BaseJSPath != "" {
			componentScript = `<script type="module" src="` + meta.BaseJSPath + `"></script>`
		}
	}

	return fmt.Sprintf(
		`%s`+
			`<script type="module" src="%s"></script>`+
			`<script type="module" src="%s"></script>`+
			`%s`+
			`<script type="module" src="%s"></script>`,
		widgetHTML, coreURL, componentsURL, componentScript, devURL,
	)
}

// emailCopyButtonSnippet returns an HTML snippet containing a floating button
// that copies the raw email HTML to the clipboard when clicked.
//
// The raw HTML is base64-encoded and stored in a hidden script element so
// that it can be decoded and copied without the dev widget markup. After a
// successful copy the button icon briefly changes to a checkmark.
//
// Takes rawHTML (string) which is the original email HTML before dev widget
// injection.
//
// Returns string which is the HTML snippet to inject into the preview page.
func emailCopyButtonSnippet(rawHTML string) string {
	encoded := base64.StdEncoding.EncodeToString([]byte(rawHTML))
	return `<script type="text/plain" id="piko-email-source">` + encoded + `</script>` +
		`<style>#piko-copy-email:hover{background:rgba(24,24,27,0.95)!important;transform:scale(1.05)}</style>` +
		`<button id="piko-copy-email" title="Copy email HTML" style="` + previewActionButtonStyle + `">` + svgCopyIcon + `</button>` +
		`<script>` + copyButtonScript + `</script>`
}

// addCopyScriptHashToCSP modifies the Content-Security-Policy header to
// allow the inline copy button script by adding its pre-computed hash.
// If the CSP already contains a script-src-elem directive the hash is
// appended to it; otherwise a new directive is added.
//
// Takes w (http.ResponseWriter) which provides the response headers to
// modify.
func addCopyScriptHashToCSP(w http.ResponseWriter) {
	csp := w.Header().Get("Content-Security-Policy")
	if csp == "" {
		return
	}

	directives := strings.Split(csp, ";")
	found := false
	for i, d := range directives {
		trimmed := strings.TrimSpace(d)
		if strings.HasPrefix(trimmed, "script-src-elem") {
			directives[i] = " " + trimmed + " " + copyButtonScriptCSPHash
			found = true
			break
		}
	}
	if !found {
		directives = append(directives, " script-src-elem 'self' "+copyButtonScriptCSPHash)
	}

	w.Header().Set("Content-Security-Policy", strings.Join(directives, ";"))
}

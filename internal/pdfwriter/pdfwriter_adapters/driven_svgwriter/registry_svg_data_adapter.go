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

package driven_svgwriter

import (
	"context"
	"html"
	"strings"

	"piko.sh/piko/internal/pdfwriter/pdfwriter_domain"
	"piko.sh/piko/internal/render/render_domain"
)

// SVGAssetResolver is the slice of the render registry needed to resolve an SVG asset
// reference to its parsed content.
//
// A reference is a module-relative "@/lib/icons/foo.svg" path or any other registry asset
// id. render_domain.RegistryPort satisfies it. Kept narrow so this adapter depends only
// on the one method it uses.
type SVGAssetResolver interface {
	// GetAssetRawSVG resolves an asset id to its parsed SVG data.
	//
	// Takes assetID (string) which is the registry asset id (such as a "@/..." path).
	//
	// Returns *render_domain.ParsedSvgData which holds the parsed SVG content.
	// Returns error when the asset cannot be resolved.
	GetAssetRawSVG(ctx context.Context, assetID string) (*render_domain.ParsedSvgData, error)
}

// RegistrySVGDataAdapter implements SVGDataPort by resolving asset references through the
// render registry.
//
// It resolves references the same way the web renderer's <piko:svg> does, via
// GetAssetRawSVG, then reconstructs the original <svg> markup from the parsed data so the
// PDF SVG writer can paint it as native vectors. Sources the registry does not own
// (notably data: URIs) are delegated to a fallback adapter.
//
// This is what lets <piko:svg src="@/lib/icons/mail.svg"> actually render in PDFs: the
// asset pipeline keys SVGs by their source string (including the @/ module alias), so the
// raw src is the asset id GetAssetRawSVG expects.
type RegistrySVGDataAdapter struct {
	// registry resolves an asset id to parsed SVG data. May be nil (fallback-only).
	registry SVGAssetResolver

	// fallback handles sources the registry does not, such as data: URIs; may be nil.
	fallback pdfwriter_domain.SVGDataPort
}

var (
	_ pdfwriter_domain.SVGDataPort = (*RegistrySVGDataAdapter)(nil)
)

// NewRegistrySVGDataAdapter creates an SVGDataPort backed by the render registry.
//
// Takes registry (SVGAssetResolver) which resolves asset references to parsed SVG data;
// may be nil, in which case only the fallback is consulted.
// Takes fallback (pdfwriter_domain.SVGDataPort) which handles sources the registry does
// not resolve (e.g. data: URIs); may be nil.
//
// Returns *RegistrySVGDataAdapter which implements SVGDataPort.
func NewRegistrySVGDataAdapter(registry SVGAssetResolver, fallback pdfwriter_domain.SVGDataPort) *RegistrySVGDataAdapter {
	return &RegistrySVGDataAdapter{registry: registry, fallback: fallback}
}

// GetSVGData resolves source to raw SVG XML markup.
//
// Takes ctx (context.Context) which carries cancellation and tracing.
// Takes source (string) which is the SVG source: a data: URI, a module-relative "@/..."
// asset path, or another registry asset id.
//
// Returns string which is the SVG markup.
// Returns bool which is true when source was resolved.
func (a *RegistrySVGDataAdapter) GetSVGData(ctx context.Context, source string) (string, bool) {
	if source == "" {
		return "", false
	}

	if a.fallback != nil {
		if markup, ok := a.fallback.GetSVGData(ctx, source); ok {
			return markup, true
		}
	}
	if a.registry == nil || strings.HasPrefix(source, "data:") {
		return "", false
	}
	parsed, err := a.registry.GetAssetRawSVG(ctx, source)
	if err != nil || parsed == nil {
		return "", false
	}
	return reconstructSVGMarkup(parsed), true
}

// reconstructSVGMarkup rebuilds an <svg> element string from parsed SVG data so it can be
// re-parsed by the PDF SVG renderer. The parsed form stores the element attributes (such
// as viewBox and fill) and the inner content separately; this reassembles them, ensuring
// an xmlns is present.
//
// Takes parsed (*render_domain.ParsedSvgData) which holds the SVG attributes and inner
// markup.
//
// Returns string which is the reconstructed <svg>...</svg> markup.
func reconstructSVGMarkup(parsed *render_domain.ParsedSvgData) string {
	var b strings.Builder
	b.WriteString("<svg")
	hasXMLNS := false
	for index := range parsed.Attributes {
		attr := &parsed.Attributes[index]

		if !isValidXMLAttributeName(attr.Name) {
			continue
		}
		if attr.Name == "xmlns" {
			hasXMLNS = true
		}
		b.WriteString(" ")
		b.WriteString(attr.Name)
		b.WriteString(`="`)
		b.WriteString(html.EscapeString(attr.Value))
		b.WriteString(`"`)
	}
	if !hasXMLNS {
		b.WriteString(` xmlns="http://www.w3.org/2000/svg"`)
	}
	b.WriteString(">")
	b.WriteString(parsed.InnerHTML)
	b.WriteString("</svg>")
	return b.String()
}

// isValidXMLAttributeName reports whether name is a safe XML attribute name.
//
// It accepts a conservative subset of the XML Name production: a name must be non-empty,
// begin with a letter, underscore, or colon (for namespaced names such as "xlink:href"),
// and contain only letters, digits, hyphen, underscore, full stop, or colon thereafter.
// Anything else is rejected so it is never emitted verbatim into the reconstructed
// markup.
//
// Takes name (string) which is the attribute name to validate.
//
// Returns bool which is true when the name is a valid XML attribute name.
func isValidXMLAttributeName(name string) bool {
	if name == "" {
		return false
	}
	for index, ch := range name {
		if isXMLNameStartChar(ch) {
			continue
		}
		if index > 0 && (ch == '-' || ch == '.' || (ch >= '0' && ch <= '9')) {
			continue
		}
		return false
	}
	return true
}

// isXMLNameStartChar reports whether ch is permitted at any position of an XML name in
// the conservative subset accepted by isValidXMLAttributeName: an ASCII letter,
// underscore, or colon.
//
// Takes ch (rune) which is the character to test.
//
// Returns bool which is true when ch is an accepted name character.
func isXMLNameStartChar(ch rune) bool {
	return (ch >= 'a' && ch <= 'z') ||
		(ch >= 'A' && ch <= 'Z') ||
		ch == '_' || ch == ':'
}

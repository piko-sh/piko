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
	"context"
	"fmt"
	"slices"

	qt "github.com/valyala/quicktemplate"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/logger/logger_domain"
)

// internalAttributeNames lists attributes used internally by Piko that should
// never appear in final rendered output. These are used for tracking during
// rendering but are not valid HTML attributes.
var internalAttributeNames = []string{
	"p-key",
	"partial",
	"p-ref",
}

// writeTextNode writes text content to the output.
//
// If the node has a direct writer with content, it uses that. Otherwise, it
// writes the static text content.
//
// Takes node (*ast_domain.TemplateNode) which contains the text to write.
// Takes qw (*qt.Writer) which receives the output.
func writeTextNode(node *ast_domain.TemplateNode, qw *qt.Writer) {
	if node.TextContentWriter != nil && node.TextContentWriter.Len() > 0 {
		writeDirectWriterParts(node.TextContentWriter, qw)
	} else {
		qw.N().S(node.TextContent)
	}
}

// writeCommentNode writes an HTML comment to the template output.
//
// Takes content (string) which specifies the text to include in the comment.
// Takes qw (*qt.Writer) which is the template writer for output.
func writeCommentNode(content string, qw *qt.Writer) {
	qw.N().Z(commentOpen)
	qw.N().S(content)
	qw.N().Z(commentClose)
}

// logCollectedDiagnostics writes warnings and errors from the render context
// to the log.
//
// Takes rctx (*renderContext) which holds the collected diagnostics to log.
func logCollectedDiagnostics(ctx context.Context, rctx *renderContext) {
	_, l := logger_domain.From(ctx, log)

	if len(rctx.diagnostics.Warnings) > 0 {
		for _, w := range rctx.diagnostics.Warnings {
			attrs := []logger_domain.Attr{logger_domain.String("location", w.Location)}
			for k, v := range w.Details {
				attrs = append(attrs, logger_domain.String(k, v))
			}
			l.Warn(w.Message, attrs...)
		}
	}

	if len(rctx.diagnostics.Errors) > 0 {
		for _, e := range rctx.diagnostics.Errors {
			attrs := []logger_domain.Attr{logger_domain.String("location", e.Location)}
			for k, v := range e.Details {
				attrs = append(attrs, logger_domain.String(k, v))
			}
			l.Error(e.Message, append(attrs, logger_domain.Error(e.Err))...)
		}
	}
}

// writeEventDirectives writes event directive attributes to the output. Go
// inlines this small helper for better performance.
//
// Takes events (map[string][]ast_domain.Directive) which holds the event
// directives to write, grouped by event name.
// Takes prefix ([]byte) which is the attribute prefix to write before each
// event name.
// Takes qw (*qt.Writer) which is the output writer.
func writeEventDirectives(events map[string][]ast_domain.Directive, prefix []byte, qw *qt.Writer) {
	for eventName, directives := range events {
		for i := range directives {
			qw.N().Z(prefix)
			qw.N().S(eventName)
			if directives[i].Modifier != "" {
				qw.N().Z(dot)
				qw.N().S(directives[i].Modifier)
			}
			qw.N().Z(equalsQuote)
			qw.N().S(directives[i].RawExpression)
			qw.N().Z(quote)
		}
	}
}

// writeDirectWriterParts writes all parts of a DirectWriter to the output
// buffer.
//
// Used by writeAttributeWriters and text content rendering for consistent
// output without memory allocation. String, EscapeString, Int, Uint, Float,
// and Bool parts do not allocate. WriterPartAny may allocate when it calls
// Stringer methods.
//
// Takes dw (*ast_domain.DirectWriter) which provides the parts to write.
// Takes qw (*qt.Writer) which receives the output.
func writeDirectWriterParts(dw *ast_domain.DirectWriter, qw *qt.Writer) {
	for i := range dw.Len() {
		part := dw.Part(i)
		if part != nil {
			writeWriterPart(part, qw)
		}
	}
}

// writerPartHandlers maps each WriterPartType to a handler function that writes
// the part's value to the quicktemplate writer. Indexed by the uint8 enum value
// for O(1) dispatch without branching.
var writerPartHandlers = [ast_domain.WriterPartEscapeBytes + 1]func(*ast_domain.WriterPart, *qt.Writer){
	ast_domain.WriterPartString:       func(p *ast_domain.WriterPart, qw *qt.Writer) { qw.N().S(p.StringValue) },
	ast_domain.WriterPartEscapeString: func(p *ast_domain.WriterPart, qw *qt.Writer) { qw.E().S(p.StringValue) },
	ast_domain.WriterPartInt:          func(p *ast_domain.WriterPart, qw *qt.Writer) { qw.N().DL(p.IntValue) },
	ast_domain.WriterPartUint:         func(p *ast_domain.WriterPart, qw *qt.Writer) { qw.N().DUL(p.UintValue) },
	ast_domain.WriterPartFloat:        func(p *ast_domain.WriterPart, qw *qt.Writer) { qw.N().F(p.FloatValue) },
	ast_domain.WriterPartBool:         func(p *ast_domain.WriterPart, qw *qt.Writer) { writePartBool(p.BoolValue, qw) },
	ast_domain.WriterPartAny:          func(p *ast_domain.WriterPart, qw *qt.Writer) { writePartAny(p.AnyValue, qw) },
	ast_domain.WriterPartFNVString:    writePartFNVString,
	ast_domain.WriterPartFNVFloat:     writePartFNVFloat,
	ast_domain.WriterPartFNVAny:       func(p *ast_domain.WriterPart, qw *qt.Writer) { writePartFNVAny(p.AnyValue, qw) },
	ast_domain.WriterPartBytes:        func(p *ast_domain.WriterPart, qw *qt.Writer) { qw.N().SZ(p.BytesValue) },
	ast_domain.WriterPartEscapeBytes:  func(p *ast_domain.WriterPart, qw *qt.Writer) { qw.E().SZ(p.BytesValue) },
}

// writePartFNVString writes the FNV-32 hash of a string
// value to the output.
//
// Takes p (*ast_domain.WriterPart) which contains the string
// value to hash.
// Takes qw (*qt.Writer) which receives the hashed output.
func writePartFNVString(p *ast_domain.WriterPart, qw *qt.Writer) {
	buffer := ast_domain.GetFNVStringBuf(p.StringValue)
	qw.N().SZ(buffer.Bytes())
	buffer.Release()
}

// writePartFNVFloat writes the FNV-32 hash of a float
// value to the output.
//
// Takes p (*ast_domain.WriterPart) which contains the float
// value to hash.
// Takes qw (*qt.Writer) which receives the hashed output.
func writePartFNVFloat(p *ast_domain.WriterPart, qw *qt.Writer) {
	buffer := ast_domain.GetFNVFloatBuf(p.FloatValue)
	qw.N().SZ(buffer.Bytes())
	buffer.Release()
}

// writeWriterPart writes a single DirectWriter part to the output.
//
// Dispatches through the writerPartHandlers array indexed by part type for
// branch-free dispatch.
//
// Takes part (*ast_domain.WriterPart) which contains the value to write.
// Takes qw (*qt.Writer) which receives the output.
func writeWriterPart(part *ast_domain.WriterPart, qw *qt.Writer) {
	if int(part.Type) < len(writerPartHandlers) {
		if handler := writerPartHandlers[part.Type]; handler != nil {
			handler(part, qw)
		}
	}
}

// writePartBool writes a boolean value as "true" or "false" to the template
// writer.
//
// Takes b (bool) which is the value to write.
// Takes qw (*qt.Writer) which is the output destination.
func writePartBool(b bool, qw *qt.Writer) {
	if b {
		qw.N().S("true")
	} else {
		qw.N().S("false")
	}
}

// writePartAny writes any value to a quicktemplate writer.
//
// When v is nil, returns without writing anything.
//
// Takes v (any) which is the value to write.
// Takes qw (*qt.Writer) which receives the output.
func writePartAny(v any, qw *qt.Writer) {
	if v == nil {
		return
	}
	if stringer, ok := v.(fmt.Stringer); ok {
		qw.N().S(stringer.String())
	} else {
		qw.N().S(fmt.Sprint(v))
	}
}

// writePartFNVAny writes the FNV hash of the given value to the template
// writer.
//
// Takes v (any) which is the value to hash.
// Takes qw (*qt.Writer) which receives the hashed output.
func writePartFNVAny(v any, qw *qt.Writer) {
	buffer := ast_domain.GetFNVAnyBuf(v)
	if buffer.Bytes() != nil {
		qw.N().SZ(buffer.Bytes())
	}
	buffer.Release()
}

// writeAttributeWriters writes all attribute writers to the output.
//
// Each attribute writer is rendered as ` name="value"` where the value is
// built from DirectWriter parts, so dynamic attributes are rendered without
// extra memory use and with smart HTML escaping.
//
// Takes writers ([]*ast_domain.DirectWriter) which contains the DirectWriters
// with Name fields to render as attributes.
// Takes qw (*qt.Writer) which is the output target for rendered content.
func writeAttributeWriters(writers []*ast_domain.DirectWriter, qw *qt.Writer) {
	writeAttributeWritersExcluding(writers, qw, "")
}

// writeAttributeWritersExcluding writes attribute writers to the output,
// skipping any with the given names. Used by renderers like piko:img and piko:svg
// that handle certain attributes themselves.
//
// Takes writers ([]*ast_domain.DirectWriter) which contains the attribute
// writers to render.
// Takes qw (*qt.Writer) which receives the rendered output.
// Takes excludeNames (...string) which lists attribute names to skip.
func writeAttributeWritersExcluding(writers []*ast_domain.DirectWriter, qw *qt.Writer, excludeNames ...string) {
	for _, dw := range writers {
		if dw == nil || dw.Len() == 0 || dw.Name == "" {
			continue
		}
		if shouldExcludeAttribute(dw.Name, excludeNames) {
			continue
		}
		qw.N().Z(space)
		qw.N().S(dw.Name)
		qw.N().Z(equalsQuote)
		writeDirectWriterParts(dw, qw)
		qw.N().Z(quote)
	}
}

// shouldExcludeAttribute checks if an attribute name is in the exclude list.
// The parser lowercases attribute names, so direct comparison is used.
//
// Takes name (string) which is the attribute name to check.
// Takes excludeNames ([]string) which contains names to exclude.
//
// Returns bool which is true if the name matches any excluded name.
func shouldExcludeAttribute(name string, excludeNames []string) bool {
	return slices.Contains(excludeNames, name)
}

// isInternalAttribute checks whether an attribute name is internal to Piko
// and should be excluded from rendered output.
//
// Takes name (string) which is the attribute name to check.
//
// Returns bool which is true if the attribute is internal, false otherwise.
func isInternalAttribute(name string) bool {
	return slices.Contains(internalAttributeNames, name)
}

// writeNodeAndFragmentAttributes writes node and fragment attributes to the
// output, merging them with clear priority rules.
//
// Node attributes take priority over fragment attributes with the same name.
// Attribute writers (dynamic bindings) take priority over both. Any static
// attribute that has a matching attribute writer is skipped to avoid writing
// the same attribute twice.
//
// In email mode (rctx.isEmailMode), internal attributes like p-key and partial
// are filtered out as they are only used for web rendering.
//
// Takes nodeAttrs ([]ast_domain.HTMLAttribute) which provides the main
// attributes from the node.
// Takes fragmentAttrs ([]ast_domain.HTMLAttribute) which provides fallback
// attributes from the fragment.
// Takes attributeWriters ([]*ast_domain.DirectWriter) which provides dynamic
// attribute bindings that replace static attributes.
// Takes qw (*qt.Writer) which is the output writer for the rendered HTML.
// Takes rctx (*renderContext) which provides the rendering mode; may be nil.
//
// Note: uses linear search rather than a map for small attribute counts
// (typically 3-8). Linear search is faster because it avoids hash work, has
// better cache use, and needs no memory allocation.
func writeNodeAndFragmentAttributes(nodeAttrs, fragmentAttrs []ast_domain.HTMLAttribute, attributeWriters []*ast_domain.DirectWriter, qw *qt.Writer, rctx *renderContext) {
	hasWriters := len(attributeWriters) > 0
	filterInternal := rctx != nil && rctx.isEmailMode

	writeNodeAttrs(nodeAttrs, attributeWriters, hasWriters, filterInternal, qw)
	writeFragmentAttrs(fragmentAttrs, nodeAttrs, attributeWriters, hasWriters, filterInternal, qw)
}

// writeNodeAttrs writes node-level attributes, skipping internal attributes in
// email mode and attributes that have a dynamic writer override.
//
// Takes attrs ([]ast_domain.HTMLAttribute) which provides the node attributes.
// Takes writers ([]*ast_domain.DirectWriter) which provides dynamic overrides.
// Takes hasWriters (bool) which indicates whether any dynamic writers exist.
// Takes filterInternal (bool) which indicates whether internal attributes
// should be filtered.
// Takes qw (*qt.Writer) which receives the rendered output.
func writeNodeAttrs(attrs []ast_domain.HTMLAttribute, writers []*ast_domain.DirectWriter, hasWriters, filterInternal bool, qw *qt.Writer) {
	for i := range attrs {
		attr := &attrs[i]
		if filterInternal && isInternalAttribute(attr.Name) {
			continue
		}
		if hasWriters && hasAttributeWriter(writers, attr.Name) {
			continue
		}
		qw.N().Z(space)
		qw.N().S(attr.Name)
		qw.N().Z(equalsQuote)
		qw.N().S(attr.Value)
		qw.N().Z(quote)
	}
}

// writeFragmentAttrs writes fragment-level attributes that are not already
// present as node attributes, skipping internal attributes in email mode and
// attributes that have a dynamic writer override.
//
// Takes attrs ([]ast_domain.HTMLAttribute) which provides the fragment
// attributes.
// Takes nodeAttrs ([]ast_domain.HTMLAttribute) which provides node attributes
// used to check for duplicates.
// Takes writers ([]*ast_domain.DirectWriter) which provides dynamic overrides.
// Takes hasWriters (bool) which indicates whether any dynamic writers exist.
// Takes filterInternal (bool) which indicates whether internal attributes
// should be filtered.
// Takes qw (*qt.Writer) which receives the rendered output.
func writeFragmentAttrs(attrs, nodeAttrs []ast_domain.HTMLAttribute, writers []*ast_domain.DirectWriter, hasWriters, filterInternal bool, qw *qt.Writer) {
	for i := range attrs {
		attr := &attrs[i]
		if filterInternal && isInternalAttribute(attr.Name) {
			continue
		}
		if hasAttrByName(nodeAttrs, attr.Name) {
			continue
		}
		if hasWriters && hasAttributeWriter(writers, attr.Name) {
			continue
		}
		qw.N().Z(space)
		qw.N().S(attr.Name)
		qw.N().Z(equalsQuote)
		qw.N().S(attr.Value)
		qw.N().Z(quote)
	}
}

// hasAttrByName checks if an attribute with the given name exists in the slice.
// For small slices (typically 3-8 attributes), linear search is faster than a
// map lookup because there is no hash calculation, better cache use, and no
// memory allocation.
//
// Takes attrs ([]ast_domain.HTMLAttribute) which is the slice to search.
// Takes name (string) which is the attribute name to find.
//
// Returns bool which is true if an attribute with the given name exists.
func hasAttrByName(attrs []ast_domain.HTMLAttribute, name string) bool {
	for i := range attrs {
		if attrs[i].Name == name {
			return true
		}
	}
	return false
}

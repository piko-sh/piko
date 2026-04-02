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

package llm_domain

import (
	"fmt"
	"io"
	"strings"
	"time"

	"piko.sh/piko/internal/llm/llm_dto"
)

// RequestDump captures the fully assembled state of a completion request after
// RAG resolution and context injection but before the provider call. It
// provides a structured, human-readable representation useful for debugging
// and testing RAG pipelines.
//
// Use [CompletionBuilder.DryRun] to obtain a RequestDump without executing the
// completion.
type RequestDump struct {
	// Timestamp is when the dump was captured.
	Timestamp time.Time

	// Model is the model identifier.
	Model string

	// Provider is the resolved provider name.
	Provider string

	// MaxTokens is the token limit, or nil if unset.
	MaxTokens *int

	// Temperature is the sampling temperature, or nil if unset.
	Temperature *float64

	// Messages is the full message list including injected RAG context.
	Messages []llm_dto.Message

	// Sources holds the raw vector search results used for RAG context.
	// Empty when RAG was not configured or returned no results.
	Sources []llm_dto.VectorSearchResult

	// Tools lists the tool definitions registered on the builder.
	Tools []llm_dto.ToolDefinition

	// OriginalQuery is the base query text before rewriting. Empty when RAG
	// is not configured or no query was available.
	OriginalQuery string

	// RewrittenQueries holds the queries produced by the rewriter. Empty when
	// no rewriter was configured or the rewriter returned no results.
	RewrittenQueries []string
}

// WriteTo writes a structured, human-readable representation of the request
// to w. The format uses header lines and delimited sections, similar in spirit
// to email .eml files but designed for LLM requests.
//
// Takes w (io.Writer) which receives the formatted output.
//
// Returns int64 which is the number of bytes written.
// Returns error when writing to w fails.
func (d *RequestDump) WriteTo(w io.Writer) (int64, error) {
	var buffer strings.Builder

	d.writeHeaders(&buffer)
	d.writeQueryRewriting(&buffer)
	d.writeMessages(&buffer)
	d.writeSources(&buffer)
	d.writeTools(&buffer)

	n, err := io.WriteString(w, buffer.String())
	return int64(n), err
}

// String returns the human-readable dump as a string.
//
// Returns string which contains the formatted request dump.
func (d *RequestDump) String() string {
	var buffer strings.Builder
	_, _ = d.WriteTo(&buffer)
	return buffer.String()
}

// writeHeaders writes the model, provider, and optional parameter headers.
//
// Takes buffer (*strings.Builder) which receives the formatted output.
func (d *RequestDump) writeHeaders(buffer *strings.Builder) {
	_, _ = fmt.Fprintf(buffer, "Model: %s\n", d.Model)
	_, _ = fmt.Fprintf(buffer, "Provider: %s\n", d.Provider)
	if d.MaxTokens != nil {
		_, _ = fmt.Fprintf(buffer, "MaxTokens: %d\n", *d.MaxTokens)
	}
	if d.Temperature != nil {
		_, _ = fmt.Fprintf(buffer, "Temperature: %.2f\n", *d.Temperature)
	}
	_, _ = fmt.Fprintf(buffer, "Timestamp: %s\n", d.Timestamp.Format(time.RFC3339))
}

// writeQueryRewriting writes the query rewriting section when queries were
// rewritten.
//
// Takes buffer (*strings.Builder) which receives the formatted output.
func (d *RequestDump) writeQueryRewriting(buffer *strings.Builder) {
	if len(d.RewrittenQueries) == 0 {
		return
	}
	_, _ = fmt.Fprint(buffer, "\n=== Query Rewriting ===\n\n")
	_, _ = fmt.Fprintf(buffer, "Original: %s\n\n", d.OriginalQuery)
	_, _ = fmt.Fprintf(buffer, "Rewritten (%d):\n", len(d.RewrittenQueries))
	for i, q := range d.RewrittenQueries {
		_, _ = fmt.Fprintf(buffer, "  %d. %s\n", i+1, q)
	}
}

// writeMessages writes the messages section.
//
// Takes buffer (*strings.Builder) which receives the formatted output.
func (d *RequestDump) writeMessages(buffer *strings.Builder) {
	_, _ = fmt.Fprintf(buffer, "\n=== Messages (%d) ===\n", len(d.Messages))
	for _, message := range d.Messages {
		_, _ = fmt.Fprintf(buffer, "\n--- %s ---\n", message.Role)
		if message.Content != "" {
			_, _ = buffer.WriteString(message.Content)
			_ = buffer.WriteByte('\n')
		}
	}
}

// writeSources writes the vector search sources section.
//
// Takes buffer (*strings.Builder) which receives the formatted output.
func (d *RequestDump) writeSources(buffer *strings.Builder) {
	if len(d.Sources) == 0 {
		return
	}
	_, _ = fmt.Fprintf(buffer, "\n=== Sources (%d) ===\n", len(d.Sources))
	for i, source := range d.Sources {
		_, _ = fmt.Fprintf(buffer, "\n--- [%d] score=%.4f id=%s ---\n", i+1, source.Score, source.ID)
		if heading := metaString(source.Metadata, "heading"); heading != "" {
			_, _ = fmt.Fprintf(buffer, "Heading: %s\n", heading)
		}
		if file := metaString(source.Metadata, "source"); file != "" {
			_, _ = fmt.Fprintf(buffer, "File: %s\n", file)
		}
		for k, v := range source.Metadata {
			if k == "heading" || k == "source" {
				continue
			}
			_, _ = fmt.Fprintf(buffer, "%s: %v\n", k, v)
		}
		_ = buffer.WriteByte('\n')
		_, _ = buffer.WriteString(source.Content)
		_ = buffer.WriteByte('\n')
	}
}

// writeTools writes the tools section.
//
// Takes buffer (*strings.Builder) which receives the formatted output.
func (d *RequestDump) writeTools(buffer *strings.Builder) {
	if len(d.Tools) == 0 {
		return
	}
	_, _ = fmt.Fprintf(buffer, "\n=== Tools (%d) ===\n", len(d.Tools))
	for _, tool := range d.Tools {
		_, _ = fmt.Fprintf(buffer, "\n--- %s ---\n", tool.Function.Name)
		if tool.Function.Description != nil && *tool.Function.Description != "" {
			_, _ = buffer.WriteString(*tool.Function.Description)
			_ = buffer.WriteByte('\n')
		}
	}
}

// metaString safely extracts a string value from a metadata map.
// This is a package-level helper also used by the markdown splitter.
//
// Takes meta (map[string]any) which is the metadata map to search.
// Takes key (string) which is the key to look up.
//
// Returns string which is the value if found, or an empty string otherwise.
func metaString(meta map[string]any, key string) string {
	if meta == nil {
		return ""
	}
	v, ok := meta[key]
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}

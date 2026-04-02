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
	"strings"

	"gopkg.in/yaml.v3"
)

// FrontmatterOption configures the behaviour of [ExtractFrontmatter].
type FrontmatterOption func(*frontmatterConfig)

// frontmatterConfig holds settings for extracting frontmatter fields.
type frontmatterConfig struct {
	// prefix is prepended to metadata keys when storing frontmatter values.
	prefix string

	// keys specifies the frontmatter keys to extract; nil extracts all keys.
	keys []string
}

// StripFrontmatter returns a TransformFunc that removes YAML frontmatter
// delimited by "---" from the beginning of document content.
//
// Returns TransformFunc which removes YAML frontmatter from documents.
func StripFrontmatter() TransformFunc {
	return func(document Document) Document {
		document.Content = stripFrontmatter(document.Content)
		return document
	}
}

// WithFrontmatterKeys restricts extraction to the named keys. By default all
// top-level frontmatter keys are extracted.
//
// Takes keys (...string) which specifies the frontmatter keys to extract.
//
// Returns FrontmatterOption which applies the key filter.
func WithFrontmatterKeys(keys ...string) FrontmatterOption {
	return func(c *frontmatterConfig) { c.keys = keys }
}

// WithFrontmatterPrefix prepends a string to each extracted metadata key.
// For example, WithFrontmatterPrefix("doc_") turns a frontmatter key "title"
// into the metadata key "doc_title".
//
// Takes prefix (string) which is prepended to each key.
//
// Returns FrontmatterOption which applies the prefix.
func WithFrontmatterPrefix(prefix string) FrontmatterOption {
	return func(c *frontmatterConfig) { c.prefix = prefix }
}

// ExtractFrontmatter returns a TransformFunc that parses YAML frontmatter,
// merges the extracted keys into the document's metadata, and strips the
// frontmatter from the content.
//
// When no frontmatter is found, the document is returned unchanged.
//
// Takes opts (...FrontmatterOption) which configures the extraction behaviour.
//
// Returns TransformFunc which processes documents to extract their frontmatter.
//
// By default all top-level string, numeric, and boolean values are extracted.
// Use [WithFrontmatterKeys] to restrict which keys are extracted and
// [WithFrontmatterPrefix] to namespace metadata keys.
func ExtractFrontmatter(opts ...FrontmatterOption) TransformFunc {
	var config frontmatterConfig
	for _, o := range opts {
		o(&config)
	}

	keySet := make(map[string]struct{}, len(config.keys))
	for _, k := range config.keys {
		keySet[k] = struct{}{}
	}

	return func(document Document) Document {
		raw, body, ok := extractRawFrontmatter(document.Content)
		if !ok {
			return document
		}

		var parsed map[string]any
		if err := yaml.Unmarshal([]byte(raw), &parsed); err != nil {
			return document
		}

		document.Metadata = mergeFrontmatterMetadata(document.Metadata, parsed, keySet, config.prefix)
		document.Content = body
		return document
	}
}

// PrependChunkContext returns a [TransformFunc] intended for use as a
// post-split transform. It reads the "doc_title" and "heading" metadata keys
// and prepends them to the chunk content so that the embedding model can use
// the surrounding context for better semantic matching.
//
// For example, a chunk with doc_title "Your First Page" and heading "Add a
// simple template" would have "Your First Page > Add a simple template\n\n"
// prepended to its content.
//
// Chunks that already start with a markdown heading (# ...) or have no
// contextual metadata are returned unchanged.
//
// Returns TransformFunc which prepends contextual metadata to chunk content.
func PrependChunkContext() TransformFunc {
	return func(document Document) Document {
		if document.Metadata == nil {
			return document
		}

		title, ok := document.Metadata["doc_title"].(string)
		if !ok {
			title = ""
		}

		heading, ok := document.Metadata["heading"].(string)
		if !ok {
			heading = ""
		}

		if title == "" && heading == "" {
			return document
		}

		trimmed := strings.TrimSpace(document.Content)
		if strings.HasPrefix(trimmed, "#") {
			return document
		}

		var prefix string
		switch {
		case title != "" && heading != "":
			prefix = fmt.Sprintf("%s > %s", title, heading)
		case title != "":
			prefix = title
		default:
			prefix = heading
		}

		document.Content = prefix + "\n\n" + document.Content
		return document
	}
}

// mergeFrontmatterMetadata merges parsed frontmatter values into the document
// metadata map, applying key filtering and prefixing.
//
// Takes metadata (map[string]any) which is the existing document metadata.
// Takes parsed (map[string]any) which contains the parsed frontmatter values.
// Takes keySet (map[string]struct{}) which restricts which keys to merge;
// an empty set means all keys are included.
// Takes prefix (string) which is prepended to each metadata key.
//
// Returns map[string]any which is the merged metadata map.
func mergeFrontmatterMetadata(metadata, parsed map[string]any, keySet map[string]struct{}, prefix string) map[string]any {
	if metadata == nil {
		metadata = make(map[string]any)
	}

	for k, v := range parsed {
		if len(keySet) > 0 {
			if _, ok := keySet[k]; !ok {
				continue
			}
		}
		metadata[prefix+k] = v
	}

	return metadata
}

// extractRawFrontmatter splits markdown content into raw YAML frontmatter and
// the remaining body.
//
// Takes content (string) which is the markdown text to parse.
//
// Returns raw (string) which is the extracted YAML frontmatter without
// delimiters.
// Returns body (string) which is the remaining content after the frontmatter.
// Returns ok (bool) which is false if no frontmatter delimiters are found.
func extractRawFrontmatter(content string) (raw, body string, ok bool) {
	if !strings.HasPrefix(content, "---") {
		return "", content, false
	}

	yamlBlock, after, found := strings.Cut(content[3:], "\n---")
	if !found {
		return "", content, false
	}

	return strings.TrimSpace(yamlBlock), strings.TrimLeft(after, "\n"), true
}

// stripFrontmatter removes YAML frontmatter from markdown content.
// Frontmatter is delimited by "---" at the start and a closing "\n---".
//
// Takes markdown (string) which is the content to process.
//
// Returns string which is the content with frontmatter removed.
func stripFrontmatter(markdown string) string {
	if !strings.HasPrefix(markdown, "---") {
		return markdown
	}

	_, after, found := strings.Cut(markdown[3:], "\n---")
	if !found {
		return markdown
	}

	return strings.TrimLeft(after, "\n")
}

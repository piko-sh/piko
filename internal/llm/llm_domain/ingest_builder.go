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
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"

	"piko.sh/piko/internal/logger/logger_domain"
)

// TransformFunc defines a document transformation applied before splitting and
// embedding. It can strip frontmatter, clean markup, or apply pre-processing.
type TransformFunc func(document Document) Document

// IngestBuilder provides a fluent API for document ingestion.
type IngestBuilder struct {
	// loader provides the document source for ingestion.
	loader LoaderPort

	// splitter splits documents into smaller chunks; nil means no splitting.
	splitter SplitterPort

	// service provides document embedding and storage operations.
	service *service

	// namespace identifies the vector store namespace for document storage.
	namespace string

	// transforms are applied in order to each document after loading.
	transforms []TransformFunc

	// postSplitTransforms are applied in order to each chunk after splitting.
	postSplitTransforms []TransformFunc
}

// NewIngestBuilder creates a new IngestBuilder.
//
// Takes service (*service) which provides the underlying service instance.
// Takes namespace (string) which specifies the namespace for ingestion.
//
// Returns *IngestBuilder which is configured and ready for use.
func NewIngestBuilder(service *service, namespace string) *IngestBuilder {
	return &IngestBuilder{
		service:   service,
		namespace: namespace,
	}
}

// FromFS sets a file system loader using the given glob patterns. Patterns
// prefixed with "**/" match recursively, while plain patterns match a single
// level.
//
// Takes fsys (fs.FS) which provides the file system to read from.
// Takes patterns (...string) which specify the glob patterns to match.
//
// Returns *IngestBuilder which allows for method chaining.
func (b *IngestBuilder) FromFS(fsys fs.FS, patterns ...string) *IngestBuilder {
	if hasRecursivePattern(patterns) {
		b.loader = NewRecursiveFSLoader(fsys, stripRecursivePrefix(patterns)...)
	} else {
		b.loader = NewFSLoader(fsys, patterns...)
	}
	return b
}

// FromDirectory configures file loading from a local directory using glob
// patterns. Patterns with "**/" prefix walk recursively; plain patterns match
// a single level.
//
// Takes path (string) which specifies the directory to load files from.
// Takes patterns (...string) which specifies the glob patterns to match.
//
// Returns *IngestBuilder which allows method chaining.
func (b *IngestBuilder) FromDirectory(path string, patterns ...string) *IngestBuilder {
	return b.FromFS(os.DirFS(path), patterns...)
}

// Transform appends a transformation function to the pipeline. Transforms are
// applied in order to each document after loading but before splitting and
// embedding.
//
// Takes transformFunction (TransformFunc) which transforms each document.
//
// Returns *IngestBuilder which allows method chaining.
func (b *IngestBuilder) Transform(transformFunction TransformFunc) *IngestBuilder {
	b.transforms = append(b.transforms, transformFunction)
	return b
}

// PostSplitTransform appends a transformation function that runs after
// splitting, useful for enriching chunks with metadata or filtering them out.
// Multiple calls chain additional transforms.
//
// Takes transformFunction (TransformFunc) which transforms each chunk
// after splitting.
//
// Returns *IngestBuilder which allows method chaining.
func (b *IngestBuilder) PostSplitTransform(transformFunction TransformFunc) *IngestBuilder {
	b.postSplitTransforms = append(b.postSplitTransforms, transformFunction)
	return b
}

// Loader sets the document loader for the ingestion.
//
// Takes loader (LoaderPort) which provides the document loading strategy.
//
// Returns *IngestBuilder which allows method chaining.
func (b *IngestBuilder) Loader(loader LoaderPort) *IngestBuilder {
	b.loader = loader
	return b
}

// Splitter sets the document splitter for the ingestion.
//
// Takes splitter (SplitterPort) which defines how documents are split.
//
// Returns *IngestBuilder which allows method chaining.
func (b *IngestBuilder) Splitter(splitter SplitterPort) *IngestBuilder {
	b.splitter = splitter
	return b
}

// Do executes the ingestion process.
//
// When the context is already cancelled or has exceeded its deadline, returns
// the context's error without performing any work.
//
// Returns error when the loader is not set, documents fail to load, or adding
// documents to the vector store fails.
func (b *IngestBuilder) Do(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("context cancelled before ingestion: %w", err)
	}

	ctx, l := logger_domain.From(ctx, log)

	if b.loader == nil {
		return errors.New("loader is required")
	}

	return l.RunInSpan(ctx, "IngestBuilder.Do", b.executeIngestion,
		logger_domain.String("namespace", b.namespace),
	)
}

// executeIngestion runs the load-transform-split-store pipeline within a
// tracing span.
//
// Takes spanCtx (context.Context) which carries the span and cancellation.
// Takes spanLog (logger_domain.Logger) which provides structured logging.
//
// Returns error when any pipeline stage fails.
func (b *IngestBuilder) executeIngestion(spanCtx context.Context, spanLog logger_domain.Logger) error {
	docs, err := b.loader.Load(spanCtx)
	if err != nil {
		return fmt.Errorf("loading documents: %w", err)
	}
	spanLog.Internal("Loaded documents", logger_domain.Int("count", len(docs)))

	if err := spanCtx.Err(); err != nil {
		return fmt.Errorf("context cancelled before applying transforms: %w", err)
	}

	for _, transformFunction := range b.transforms {
		for i := range docs {
			docs[i] = transformFunction(docs[i])
		}
	}

	finalDocuments := b.splitDocuments(docs, spanLog)

	if err := spanCtx.Err(); err != nil {
		return fmt.Errorf("context cancelled before adding documents to vector store: %w", err)
	}

	finalDocuments = b.applyPostSplitTransforms(finalDocuments, spanLog)

	if err := b.service.AddDocuments(spanCtx, b.namespace, finalDocuments); err != nil {
		return fmt.Errorf("adding documents to vector store: %w", err)
	}

	spanLog.Internal("Ingestion completed successfully", logger_domain.Int("final_count", len(finalDocuments)))
	return nil
}

// splitDocuments applies the splitter to loaded documents when configured.
//
// Takes docs ([]Document) which contains the loaded documents.
// Takes spanLog (logger_domain.Logger) which provides structured logging.
//
// Returns []Document which contains the split chunks or the original documents.
func (b *IngestBuilder) splitDocuments(docs []Document, spanLog logger_domain.Logger) []Document {
	if b.splitter == nil {
		return docs
	}

	var finalDocuments []Document
	for _, document := range docs {
		chunks := b.splitter.Split(document)
		finalDocuments = append(finalDocuments, chunks...)
	}
	spanLog.Internal("Split documents into chunks", logger_domain.Int("chunk_count", len(finalDocuments)))
	return finalDocuments
}

// applyPostSplitTransforms applies post-split transforms and filters out
// chunks with empty content.
//
// Takes docs ([]Document) which contains the chunks to transform.
// Takes spanLog (logger_domain.Logger) which provides structured logging.
//
// Returns []Document which contains the transformed and filtered chunks.
func (b *IngestBuilder) applyPostSplitTransforms(docs []Document, spanLog logger_domain.Logger) []Document {
	if len(b.postSplitTransforms) == 0 {
		return docs
	}

	for _, transformFunction := range b.postSplitTransforms {
		for i := range docs {
			docs[i] = transformFunction(docs[i])
		}
	}

	kept := docs[:0]
	for _, d := range docs {
		if d.Content != "" {
			kept = append(kept, d)
		}
	}
	if dropped := len(docs) - len(kept); dropped > 0 {
		spanLog.Internal("Filtered chunks after post-split transforms",
			logger_domain.Int("dropped", dropped),
			logger_domain.Int("remaining", len(kept)),
		)
	}
	return kept
}

// hasRecursivePattern reports whether any pattern uses the "**/" prefix.
//
// Takes patterns ([]string) which contains the patterns to check.
//
// Returns bool which is true if any pattern uses the "**/" prefix.
func hasRecursivePattern(patterns []string) bool {
	for _, p := range patterns {
		if strings.HasPrefix(p, "**/") {
			return true
		}
	}
	return false
}

// stripRecursivePrefix removes the "**/" prefix from patterns, converting
// "**/*.md" to "*.md" for use with RecursiveFSLoader which matches file names.
//
// Takes patterns ([]string) which contains the glob patterns to process.
//
// Returns []string which contains the patterns with the "**/" prefix removed.
func stripRecursivePrefix(patterns []string) []string {
	out := make([]string, len(patterns))
	for i, p := range patterns {
		out[i] = strings.TrimPrefix(p, "**/")
	}
	return out
}

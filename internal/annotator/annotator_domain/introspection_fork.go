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

package annotator_domain

import (
	"bytes"
	"context"
	"fmt"
	"maps"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/sfcparser"
)

// IntrospectionSnapshot holds an immutable component graph, virtual module, and type
// resolver for a build.
type IntrospectionSnapshot struct {
	// ComponentGraph holds the parsed components and their source contents.
	ComponentGraph *annotator_dto.ComponentGraph

	// VirtualModule holds the virtual components built from ComponentGraph.
	VirtualModule *annotator_dto.VirtualModule

	// TypeResolver resolves types against VirtualModule.
	TypeResolver *TypeResolver
}

// refreshMutableComponentData refreshes changed templates, styles, and translations
// without modifying the cached snapshot.
//
// Unchanged snapshots are returned directly; changed components are copied into a fork
// that shares their script data.
//
// Takes cached (*IntrospectionSnapshot) which is the published introspection state.
// Takes changedComponents (map[string]bool) which limits the refresh to these hashed
// names, or is nil to consider all components.
//
// Returns *IntrospectionSnapshot which is cached when nothing changed, or a fork holding
// the refreshed components.
// Returns error when reading or parsing a component file fails.
func (s *AnnotatorService) refreshMutableComponentData(
	ctx context.Context,
	cached *IntrospectionSnapshot,
	changedComponents map[string]bool,
) (*IntrospectionSnapshot, error) {
	componentGraph := cached.ComponentGraph
	candidateCount := len(componentGraph.Components)
	if changedComponents != nil {
		candidateCount = len(changedComponents)
	}
	ctx, span, l := log.Span(ctx, "refreshMutableComponentData",
		logger_domain.Int("component_count", candidateCount))
	defer span.End()

	refreshed := make(map[string]*annotator_dto.ParsedComponent)
	freshContents := make(map[string][]byte)

	for hashedName, component := range componentGraph.Components {
		if changedComponents != nil && !changedComponents[hashedName] {
			continue
		}
		sourcePath := component.SourcePath
		if sourcePath == "" {
			l.Warn("Component has no source path, skipping refresh",
				logger_domain.String("hashed_name", hashedName))
			continue
		}

		fileData, err := s.fsReader.ReadFile(ctx, sourcePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read component file '%s': %w", sourcePath, err)
		}

		if previous, known := componentGraph.AllSourceContents[sourcePath]; known && bytes.Equal(previous, fileData) {
			continue
		}

		refreshedComponent, err := reparseMutableComponentData(ctx, component, fileData)
		if err != nil {
			return nil, err
		}
		refreshed[hashedName] = refreshedComponent
		freshContents[sourcePath] = fileData

		l.Trace("Refreshed mutable data for component",
			logger_domain.String(logKeyPath, sourcePath),
			logger_domain.String("hashed_name", hashedName))
	}

	if len(refreshed) == 0 {
		l.Internal("No component content changed; reusing the cached introspection snapshot")
		return cached, nil
	}

	l.Internal("Refreshed mutable content into a forked introspection snapshot",
		logger_domain.Int("refreshed_count", len(refreshed)))
	return forkIntrospectionSnapshot(cached, refreshed, freshContents), nil
}

// reparseMutableComponentData parses the template, styles, and i18n blocks of a component
// file into a shallow copy of the cached component.
//
// Takes component (*annotator_dto.ParsedComponent) which is the cached component; it is
// not written to.
// Takes fileData ([]byte) which is the component file's current content.
//
// Returns *annotator_dto.ParsedComponent which is the refreshed copy.
// Returns error when the SFC or its i18n blocks cannot be parsed.
func reparseMutableComponentData(
	ctx context.Context,
	component *annotator_dto.ParsedComponent,
	fileData []byte,
) (*annotator_dto.ParsedComponent, error) {
	_, l := logger_domain.From(ctx, log)
	sourcePath := component.SourcePath

	sfcResult, err := sfcparser.Parse(fileData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse SFC for '%s': %w", sourcePath, err)
	}

	templateStartLocation := ast_domain.Location{
		Line:   sfcResult.TemplateContentLocation.Line,
		Column: sfcResult.TemplateContentLocation.Column,
		Offset: 0,
	}

	freshTemplate, err := parseTemplateBlock(ctx, sfcResult.Template, sourcePath, templateStartLocation)
	if err != nil {
		l.Warn("Template parse error during refresh (will be caught in annotation phase)",
			logger_domain.String(logKeyPath, sourcePath),
			logger_domain.Error(err))
	}

	freshTranslations, err := parseI18nBlocksInto(freshTemplate, sfcResult, sourcePath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse i18n blocks for '%s': %w", sourcePath, err)
	}

	refreshed := new(*component)
	refreshed.Template = freshTemplate
	refreshed.StyleBlocks = sfcResult.Styles
	refreshed.LocalTranslations = freshTranslations
	return refreshed, nil
}

// forkIntrospectionSnapshot replaces refreshed components in a copy of the cached
// snapshot.
//
// The fork copies mutable maps and components and shares the remaining immutable state.
//
// Takes cached (*IntrospectionSnapshot) which is the published snapshot to fork.
// Takes refreshed (map[string]*annotator_dto.ParsedComponent) which maps hashed names to
// refreshed components.
// Takes freshContents (map[string][]byte) which maps source paths to the file contents
// the refreshed components were parsed from.
//
// Returns *IntrospectionSnapshot which is the fork.
func forkIntrospectionSnapshot(
	cached *IntrospectionSnapshot,
	refreshed map[string]*annotator_dto.ParsedComponent,
	freshContents map[string][]byte,
) *IntrospectionSnapshot {
	graph := new(*cached.ComponentGraph)
	graph.Components = maps.Clone(cached.ComponentGraph.Components)
	maps.Copy(graph.Components, refreshed)
	graph.AllSourceContents = make(map[string][]byte, len(cached.ComponentGraph.AllSourceContents)+len(freshContents))
	maps.Copy(graph.AllSourceContents, cached.ComponentGraph.AllSourceContents)
	maps.Copy(graph.AllSourceContents, freshContents)

	fork := &IntrospectionSnapshot{
		ComponentGraph: graph,
		VirtualModule:  nil,
		TypeResolver:   cached.TypeResolver,
	}
	if cached.VirtualModule == nil {
		return fork
	}

	virtualModule := new(*cached.VirtualModule)
	virtualModule.ComponentsByHash = maps.Clone(cached.VirtualModule.ComponentsByHash)
	virtualModule.ComponentsByGoPath = maps.Clone(cached.VirtualModule.ComponentsByGoPath)
	if cached.VirtualModule.Graph == cached.ComponentGraph {
		virtualModule.Graph = graph
	}

	for hashedName, component := range refreshed {
		cachedVirtual, ok := virtualModule.ComponentsByHash[hashedName]
		if !ok || cachedVirtual == nil {
			continue
		}
		forkedVirtual := new(*cachedVirtual)
		forkedVirtual.Source = component
		virtualModule.ComponentsByHash[hashedName] = forkedVirtual
		if virtualModule.ComponentsByGoPath[cachedVirtual.CanonicalGoPackagePath] == cachedVirtual {
			virtualModule.ComponentsByGoPath[cachedVirtual.CanonicalGoPackagePath] = forkedVirtual
		}
	}

	fork.VirtualModule = virtualModule
	if cached.TypeResolver != nil {
		fork.TypeResolver = cached.TypeResolver.WithVirtualModule(virtualModule)
	}
	return fork
}

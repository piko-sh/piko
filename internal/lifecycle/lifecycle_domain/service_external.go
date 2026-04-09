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

// This file contains external component and asset directory resolution,
// walking, and seeding for the lifecycle service.

import (
	"context"
	"io/fs"
	"path/filepath"
	"strings"
	"sync"

	"piko.sh/piko/internal/goroutine"
	"piko.sh/piko/internal/logger/logger_domain"
)

// resolveExternalComponentDirs resolves unique ModulePath values from external
// component definitions to absolute filesystem directories using the module
// resolver.
//
// Returns map[string]string which maps absolute directory paths to their
// original module paths (needed for artefact ID computation during seeding).
func (ls *lifecycleService) resolveExternalComponentDirs(ctx context.Context) map[string]string {
	if ls.resolver == nil || len(ls.externalComponents) == 0 {
		return nil
	}

	ctx, l := logger_domain.From(ctx, log)

	seen := make(map[string]struct{})
	var modulePaths []string
	for _, definition := range ls.externalComponents {
		if definition.ModulePath == "" {
			continue
		}
		if _, ok := seen[definition.ModulePath]; !ok {
			seen[definition.ModulePath] = struct{}{}
			modulePaths = append(modulePaths, definition.ModulePath)
		}
	}

	result := make(map[string]string, len(modulePaths))
	for _, mp := range modulePaths {
		moduleBase, subpath, err := ls.resolver.FindModuleBoundary(ctx, mp)
		if err != nil {
			l.Warn("Failed to find module boundary for external component",
				logger_domain.String("module_path", mp),
				logger_domain.Error(err))
			continue
		}

		moduleDir, err := ls.resolver.GetModuleDir(ctx, moduleBase)
		if err != nil {
			l.Warn("Failed to resolve module directory for external component",
				logger_domain.String("module_base", moduleBase),
				logger_domain.Error(err))
			continue
		}

		absDir := ls.fs.Join(moduleDir, subpath)
		result[absDir] = mp
		l.Internal("Resolved external component directory",
			logger_domain.String("module_path", mp),
			logger_domain.String("resolved_dir", absDir))
	}

	return result
}

// seedExternalComponentFiles resolves external component module directories
// and seeds their .pkc files into the registry blob store using module-path-
// based artefact IDs.
//
// Takes limiter (chan struct{}) which controls concurrency.
func (ls *lifecycleService) seedExternalComponentFiles(ctx context.Context, limiter chan struct{}) {
	dirMap := ls.resolveExternalComponentDirs(ctx)
	for absDir, modulePath := range dirMap {
		if ctx.Err() != nil {
			return
		}
		ls.walkAndSeedExternalDir(ctx, absDir, modulePath, limiter)
	}

	assetDirMap := ls.resolveExternalAssetDirs(ctx)
	for absDir, artefactPrefix := range assetDirMap {
		if ctx.Err() != nil {
			return
		}
		ls.walkAndSeedExternalAssetDir(ctx, absDir, artefactPrefix, limiter)
	}
}

// walkAndSeedExternalDir walks a single external directory and seeds .pkc
// files into the registry with artefact IDs prefixed by the module path.
//
// Takes absDir (string) which is the absolute path to walk.
// Takes modulePath (string) which provides the artefact ID prefix.
// Takes limiter (chan struct{}) which controls concurrency.
func (ls *lifecycleService) walkAndSeedExternalDir(
	ctx context.Context,
	absDir, modulePath string,
	limiter chan struct{},
) {
	ctx, l := logger_domain.From(ctx, log)
	var wg sync.WaitGroup

	walkErr := ls.fs.WalkDir(absDir, func(absPath string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if d.IsDir() || !strings.HasSuffix(strings.ToLower(d.Name()), ".pkc") {
			return nil
		}

		relPath, relErr := ls.fs.Rel(absDir, absPath)
		if relErr != nil {
			return nil
		}
		relPathSlash := filepath.ToSlash(relPath)
		artefactID := modulePath + "/" + relPathSlash

		wg.Add(1)
		limiter <- struct{}{}
		ls.seedArtefactAsync(ctx, &wg, limiter, absPath, artefactID, relPathSlash, "component")

		return nil
	})

	wg.Wait()

	if walkErr != nil {
		l.Warn("Failed to walk external component directory",
			logger_domain.String("dir", absDir),
			logger_domain.Error(walkErr))
	}
}

// resolveExternalAssetDirs collects unique (moduleBase, assetPath) pairs from
// external component definitions and resolves them to absolute directories.
//
// Returns map[string]string which maps absolute asset directory paths to their
// artefact ID prefixes (e.g. "piko.sh/piko/lib/icons").
func (ls *lifecycleService) resolveExternalAssetDirs(ctx context.Context) map[string]string {
	if ls.resolver == nil || len(ls.externalComponents) == 0 {
		return nil
	}

	ctx, l := logger_domain.From(ctx, log)
	pairs := ls.collectExternalAssetPairs(ctx)

	result := make(map[string]string, len(pairs))
	for _, p := range pairs {
		moduleDir, err := ls.resolver.GetModuleDir(ctx, p.moduleBase)
		if err != nil {
			l.Warn("Failed to resolve module directory for external asset",
				logger_domain.String("module_base", p.moduleBase),
				logger_domain.Error(err))
			continue
		}
		absDir := ls.fs.Join(moduleDir, p.assetPath)
		artefactPrefix := p.moduleBase + "/" + filepath.ToSlash(p.assetPath)
		result[absDir] = artefactPrefix
		l.Internal("Resolved external asset directory",
			logger_domain.String("module_base", p.moduleBase),
			logger_domain.String("asset_path", p.assetPath),
			logger_domain.String("resolved_dir", absDir))
	}
	return result
}

// externalAssetPair holds a deduplicated module base and asset path pair.
type externalAssetPair struct {
	// moduleBase is the Go module path prefix for the external component.
	moduleBase string

	// assetPath is the relative path to the asset directory within the module.
	assetPath string
}

// collectExternalAssetPairs deduplicates (moduleBase, assetPath) pairs from
// external component definitions by resolving each module boundary.
//
// Takes ctx (context.Context) which carries tracing and cancellation.
//
// Returns []externalAssetPair which contains the unique pairs.
func (ls *lifecycleService) collectExternalAssetPairs(ctx context.Context) []externalAssetPair {
	type assetKey struct{ moduleBase, assetPath string }
	seen := make(map[assetKey]struct{})
	var pairs []externalAssetPair

	for _, definition := range ls.externalComponents {
		if definition.ModulePath == "" || len(definition.AssetPaths) == 0 {
			continue
		}
		moduleBase, _, err := ls.resolver.FindModuleBoundary(ctx, definition.ModulePath)
		if err != nil {
			continue
		}
		for _, ap := range definition.AssetPaths {
			key := assetKey{moduleBase: moduleBase, assetPath: ap}
			if _, ok := seen[key]; !ok {
				seen[key] = struct{}{}
				pairs = append(pairs, externalAssetPair{moduleBase: moduleBase, assetPath: ap})
			}
		}
	}

	return pairs
}

// walkAndSeedExternalAssetDir walks a single external asset directory and seeds
// all files into the registry with artefact IDs prefixed by artefactPrefix.
//
// Takes absDir (string) which is the absolute path to walk.
// Takes artefactPrefix (string) which provides the artefact ID prefix
// (e.g. "piko.sh/piko/lib/icons").
// Takes limiter (chan struct{}) which controls concurrency.
func (ls *lifecycleService) walkAndSeedExternalAssetDir(
	ctx context.Context,
	absDir, artefactPrefix string,
	limiter chan struct{},
) {
	ctx, l := logger_domain.From(ctx, log)
	var wg sync.WaitGroup

	walkErr := ls.fs.WalkDir(absDir, func(absPath string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}

		relPath, relErr := ls.fs.Rel(absDir, absPath)
		if relErr != nil {
			return nil
		}
		relPathSlash := filepath.ToSlash(relPath)
		artefactID := artefactPrefix + "/" + relPathSlash

		wg.Add(1)
		limiter <- struct{}{}
		ls.seedArtefactAsync(ctx, &wg, limiter, absPath, artefactID, relPathSlash, "asset")

		return nil
	})

	wg.Wait()

	if walkErr != nil {
		l.Warn("Failed to walk external asset directory",
			logger_domain.String("dir", absDir),
			logger_domain.Error(walkErr))
	}
}

// seedArtefactAsync opens, reads, and upserts a single file into the
// registry in a new goroutine. The caller must have already incremented
// wg and sent to limiter before calling this method.
//
// Takes ctx (context.Context) which is the operation context.
// Takes wg (*sync.WaitGroup) which tracks the goroutine's completion.
// Takes limiter (chan struct{}) which controls concurrency.
// Takes filePath (string) which is the absolute file path to read.
// Takes artefactID (string) which is the artefact identifier.
// Takes sourcePath (string) which is the source-relative path.
// Takes kind (string) which describes the artefact type for log
// messages (e.g. "component" or "asset").
func (ls *lifecycleService) seedArtefactAsync(
	ctx context.Context,
	wg *sync.WaitGroup,
	limiter chan struct{},
	filePath, artefactID, sourcePath, kind string,
) {
	ctx, l := logger_domain.From(ctx, log)
	go func() {
		defer wg.Done()
		defer goroutine.RecoverPanic(ctx, "lifecycle.seedArtefactAsync")
		defer func() { <-limiter }()

		file, openErr := ls.fs.Open(filePath)
		if openErr != nil {
			l.Error("Failed to read external "+kind+" file",
				logger_domain.String(fieldPath, filePath),
				logger_domain.Error(openErr))
			return
		}
		defer func() { _ = file.Close() }()

		profiles := GetProfilesForFile(artefactID, nil)
		normalisedID := NormaliseAssetArtefactID(artefactID)
		if _, upsertErr := ls.registryService.UpsertArtefact(ctx, normalisedID, sourcePath, file, "local_disk_cache", profiles); upsertErr != nil {
			l.Error("Failed to seed external "+kind+" artefact",
				logger_domain.String("artefact_id", artefactID),
				logger_domain.Error(upsertErr))
		}
	}()
}

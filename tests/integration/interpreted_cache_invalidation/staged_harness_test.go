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

package cache_invalidation_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/coordinator/coordinator_adapters"
	"piko.sh/piko/wdk/interp/interp_provider_piko"
	"piko.sh/piko/wdk/json"
)

type StagedTestSpec struct {
	Description string  `json:"description"`
	RequestURL  string  `json:"requestURL"`
	IsFragment  bool    `json:"isFragment,omitempty"`
	Stages      []Stage `json:"stages"`
}

type Stage struct {
	Stage                int      `json:"stage"`
	Description          string   `json:"description"`
	ExpectedGolden       string   `json:"expectedGolden"`
	ExpectChange         bool     `json:"expectChange,omitempty"`
	ExpectCacheHit       bool     `json:"expectCacheHit,omitempty"`
	DelayBeforeRenderMs  int      `json:"delayBeforeRenderMs,omitempty"`
	ExpectTier1Hit       *bool    `json:"expectTier1Hit,omitempty"`
	ExpectTier1Miss      *bool    `json:"expectTier1Miss,omitempty"`
	ExpectTier2Hit       *bool    `json:"expectTier2Hit,omitempty"`
	ExpectTier2Miss      *bool    `json:"expectTier2Miss,omitempty"`
	ExpectFastPath       *bool    `json:"expectFastPath,omitempty"`
	ExpectFullBuild      *bool    `json:"expectFullBuild,omitempty"`
	UseTargetedRebuild   bool     `json:"useTargetedRebuild,omitempty"`
	TargetedEntryPoints  []string `json:"targetedEntryPoints,omitempty"`
	ExpectAnnotatedCount *int     `json:"expectAnnotatedCount,omitempty"`
	ExpectGeneratedCount *int     `json:"expectGeneratedCount,omitempty"`
}

type stagedServerResult struct {
	server           *piko.SSRServer
	srcDir           string
	testCasePath     string
	cleanup          func()
	tier2Spy         *CacheSpy
	introspectionSpy *IntrospectionCacheSpy
}

func setupStagedServer(t *testing.T, tc testCase) stagedServerResult {
	t.Helper()

	absTestCasePath, err := filepath.Abs(tc.Path)
	require.NoError(t, err)

	origSrcDir := filepath.Join(absTestCasePath, "src")
	tmpDir := t.TempDir()
	tmpSrcDir := filepath.Join(tmpDir, "src")
	copyDirRecursive(t, origSrcDir, tmpSrcDir)

	fixGoModReplace(t, origSrcDir, tmpSrcDir)

	originalWd, err := os.Getwd()
	require.NoError(t, err)
	err = os.Chdir(tmpSrcDir)
	require.NoError(t, err)

	cleanup := func() {
		_ = os.Chdir(originalWd)
	}

	server := piko.New(
		piko.WithCSSReset(piko.WithCSSResetComplete()),
	)
	server.WithInterpreterProvider(interp_provider_piko.NewProvider())

	server.Configure(piko.PublicConfig{
		BaseDir:        ".",
		PagesSourceDir: "pages",
		WatchMode:      false,
	})

	err = server.Setup()
	require.NoError(t, err, "Failed to setup server for spy injection")

	ctx := server.Container.GetAppContext()
	cacheService, err := server.Container.GetCacheService()
	require.NoError(t, err, "Failed to get cache service for spy injection")

	realBuildCache, err := coordinator_adapters.NewBuildResultCache(ctx, cacheService)
	require.NoError(t, err, "Failed to create build result cache for spy")

	realIntrospectionCache, err := coordinator_adapters.NewIntrospectionCache(ctx, cacheService)
	require.NoError(t, err, "Failed to create introspection cache for spy")

	tier2Spy := NewCacheSpy(realBuildCache)
	introspectionSpy := NewIntrospectionCacheSpy(realIntrospectionCache)

	server.Container.SetCoordinatorCacheOverride(tier2Spy)
	server.Container.SetIntrospectionCacheOverride(introspectionSpy)

	return stagedServerResult{
		server:           server,
		srcDir:           tmpSrcDir,
		testCasePath:     absTestCasePath,
		cleanup:          cleanup,
		tier2Spy:         tier2Spy,
		introspectionSpy: introspectionSpy,
	}
}

func runStagedTestCase(t *testing.T, tc testCase) {

	resetGlobalStateForTestIsolation()

	spec := loadStagedTestSpec(t, tc)
	result := setupStagedServer(t, tc)
	defer result.cleanup()
	defer result.server.Close()

	var previousHTML []byte

	for _, stage := range spec.Stages {

		if stage.Stage > 0 {
			applyStageModifications(t, result.srcDir, stage.Stage)
		}

		if stage.DelayBeforeRenderMs > 0 {
			time.Sleep(time.Duration(stage.DelayBeforeRenderMs) * time.Millisecond)
		}

		var buildResult *annotator_dto.ProjectAnnotationResult

		if stage.UseTargetedRebuild {
			coordinatorService, coordErr := result.server.Container.GetCoordinatorService()
			require.NoError(t, coordErr, "Failed to get coordinator service for stage %d", stage.Stage)

			require.NoError(t, coordinatorService.Invalidate(context.Background()),
				"Failed to invalidate cache for stage %d", stage.Stage)

			result.tier2Spy.ResetStats()
			result.introspectionSpy.ResetStats()

			entryPoints := make([]annotator_dto.EntryPoint, len(stage.TargetedEntryPoints))
			for i, path := range stage.TargetedEntryPoints {
				entryPoints[i] = annotator_dto.EntryPoint{Path: path, IsPage: true}
			}

			var err error
			buildResult, err = coordinatorService.GetOrBuildProject(context.Background(), entryPoints)
			require.NoError(t, err, "Failed targeted rebuild for stage %d", stage.Stage)
		} else {

			result.server.PreBuildHook = func() {
				result.tier2Spy.ResetStats()
				result.introspectionSpy.ResetStats()
			}

			err := result.server.Generate(context.Background(), piko.RunModeDevInterpreted)
			require.NoError(t, err, "Failed to generate project for stage %d", stage.Stage)
		}

		tier1Stats := result.introspectionSpy.GetStats()
		tier2Stats := result.tier2Spy.GetStats()
		t.Logf("Stage %d cache stats - Tier1: gets=%d hits=%d misses=%d key=%q | Tier2: gets=%d hits=%d misses=%d key=%q",
			stage.Stage,
			tier1Stats.GetCount, tier1Stats.HitCount, tier1Stats.MissCount, tier1Stats.LastKey,
			tier2Stats.GetCount, tier2Stats.HitCount, tier2Stats.MissCount, tier2Stats.LastKey)

		assertCacheExpectations(t, stage, result.tier2Spy, result.introspectionSpy)

		if buildResult != nil {
			assertComponentCounts(t, stage, buildResult)
		}

		if stage.UseTargetedRebuild {
			continue
		}

		html := makeStagedRequest(t, result.server, spec.RequestURL)

		goldenPath := filepath.Join(result.testCasePath, "golden", stage.ExpectedGolden)
		assertStagedGoldenFile(t, goldenPath, html, "Stage %d HTML for %s", stage.Stage, tc.Name)

		if stage.Stage > 0 && stage.ExpectChange {
			assert.NotEqual(t, string(previousHTML), string(html),
				"Expected HTML to change at stage %d", stage.Stage)
		}

		previousHTML = html
	}
}

func assertCacheExpectations(t *testing.T, stage Stage, tier2Spy *CacheSpy, introspectionSpy *IntrospectionCacheSpy) {
	t.Helper()

	tier1Stats := introspectionSpy.GetStats()
	tier2Stats := tier2Spy.GetStats()

	prefix := fmt.Sprintf("Stage %d (%s)", stage.Stage, stage.Description)

	if stage.ExpectTier1Hit != nil {
		if *stage.ExpectTier1Hit {
			assert.Greater(t, tier1Stats.HitCount, 0,
				"%s: expected Tier 1 (introspection) cache HIT but got %d hits, %d misses",
				prefix, tier1Stats.HitCount, tier1Stats.MissCount)
		} else {
			assert.Equal(t, 0, tier1Stats.HitCount,
				"%s: expected no Tier 1 (introspection) cache hits but got %d",
				prefix, tier1Stats.HitCount)
		}
	}

	if stage.ExpectTier1Miss != nil {
		if *stage.ExpectTier1Miss {
			assert.Greater(t, tier1Stats.MissCount, 0,
				"%s: expected Tier 1 (introspection) cache MISS but got %d misses, %d hits",
				prefix, tier1Stats.MissCount, tier1Stats.HitCount)
		} else {
			assert.Equal(t, 0, tier1Stats.MissCount,
				"%s: expected no Tier 1 (introspection) cache misses but got %d",
				prefix, tier1Stats.MissCount)
		}
	}

	if stage.ExpectTier2Hit != nil {
		if *stage.ExpectTier2Hit {
			assert.Greater(t, tier2Stats.HitCount, 0,
				"%s: expected Tier 2 (build result) cache HIT but got %d hits, %d misses",
				prefix, tier2Stats.HitCount, tier2Stats.MissCount)
		} else {
			assert.Equal(t, 0, tier2Stats.HitCount,
				"%s: expected no Tier 2 (build result) cache hits but got %d",
				prefix, tier2Stats.HitCount)
		}
	}

	if stage.ExpectTier2Miss != nil {
		if *stage.ExpectTier2Miss {
			assert.Greater(t, tier2Stats.MissCount, 0,
				"%s: expected Tier 2 (build result) cache MISS but got %d misses, %d hits",
				prefix, tier2Stats.MissCount, tier2Stats.HitCount)
		} else {
			assert.Equal(t, 0, tier2Stats.MissCount,
				"%s: expected no Tier 2 (build result) cache misses but got %d",
				prefix, tier2Stats.MissCount)
		}
	}

	if stage.ExpectFastPath != nil && *stage.ExpectFastPath {
		assert.Greater(t, tier1Stats.HitCount, 0,
			"%s: expected fast path (Tier 1 hit) but got %d Tier 1 hits",
			prefix, tier1Stats.HitCount)
	}

	if stage.ExpectFullBuild != nil && *stage.ExpectFullBuild {
		assert.Greater(t, tier1Stats.MissCount, 0,
			"%s: expected full build (Tier 1 miss) but got %d Tier 1 misses",
			prefix, tier1Stats.MissCount)
	}
}

func makeStagedRequest(t *testing.T, server *piko.SSRServer, url string) []byte {
	t.Helper()
	handler := server.GetHandler()
	require.NotNil(t, handler, "GetHandler returned nil - daemon not built correctly")
	request := httptest.NewRequest("GET", url, nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	response := recorder.Result()
	defer func() { _ = response.Body.Close() }()

	htmlBytes, err := io.ReadAll(response.Body)
	require.NoError(t, err, "Failed to read response body")
	return htmlBytes
}

func applyStageModifications(t *testing.T, srcDir string, stageNum int) {
	t.Helper()

	suffix := fmt.Sprintf("_%d", stageNum)

	err := filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		if before, ok := strings.CutSuffix(path, suffix); ok {

			targetPath := before

			content, readErr := os.ReadFile(path)
			if readErr != nil {
				return fmt.Errorf("reading stage file %s: %w", path, readErr)
			}

			if writeErr := os.WriteFile(targetPath, content, 0644); writeErr != nil {
				return fmt.Errorf("writing target file %s: %w", targetPath, writeErr)
			}
		}
		return nil
	})

	require.NoError(t, err, "Failed to apply stage %d modifications", stageNum)
}

func loadStagedTestSpec(t *testing.T, tc testCase) StagedTestSpec {
	t.Helper()
	var spec StagedTestSpec
	specPath := filepath.Join(tc.Path, "testspec.json")
	specBytes, err := os.ReadFile(specPath)
	require.NoError(t, err, "Failed to read testspec.json for %s", tc.Name)

	err = json.Unmarshal(specBytes, &spec)
	require.NoError(t, err, "Failed to parse testspec.json for %s", tc.Name)
	return spec
}

func fixGoModReplace(t *testing.T, origDir, tmpDir string) {
	t.Helper()

	goModPath := filepath.Join(tmpDir, "go.mod")
	content, err := os.ReadFile(goModPath)
	if err != nil {
		return
	}

	lines := strings.Split(string(content), "\n")
	changed := false
	for i, line := range lines {

		if !strings.Contains(line, "=>") {
			continue
		}
		parts := strings.SplitN(line, "=>", 2)
		if len(parts) != 2 {
			continue
		}
		relPath := strings.TrimSpace(parts[1])
		if !strings.HasPrefix(relPath, ".") {
			continue
		}

		absPath := filepath.Join(origDir, relPath)
		absPath, resolveErr := filepath.Abs(absPath)
		if resolveErr != nil {
			continue
		}
		lines[i] = parts[0] + "=> " + absPath
		changed = true
	}

	if changed {
		err = os.WriteFile(goModPath, []byte(strings.Join(lines, "\n")), 0644)
		require.NoError(t, err, "Failed to rewrite go.mod replace directives")
	}
}

func copyDirRecursive(t *testing.T, src, dst string) {
	t.Helper()

	err := filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relPath, relErr := filepath.Rel(src, path)
		if relErr != nil {
			return relErr
		}
		targetPath := filepath.Join(dst, relPath)

		if d.IsDir() {
			return os.MkdirAll(targetPath, 0755)
		}

		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		return os.WriteFile(targetPath, content, 0644)
	})

	require.NoError(t, err, "Failed to copy directory %s to %s", src, dst)
}

func assertComponentCounts(t *testing.T, stage Stage, result *annotator_dto.ProjectAnnotationResult) {
	t.Helper()

	prefix := fmt.Sprintf("Stage %d (%s)", stage.Stage, stage.Description)

	if stage.ExpectAnnotatedCount != nil {
		assert.Equal(t, *stage.ExpectAnnotatedCount, result.AnnotatedComponentCount,
			"%s: expected %d annotated components, got %d",
			prefix, *stage.ExpectAnnotatedCount, result.AnnotatedComponentCount)
	}

	if stage.ExpectGeneratedCount != nil {
		assert.Equal(t, *stage.ExpectGeneratedCount, result.GeneratedArtefactCount,
			"%s: expected %d generated artefacts, got %d",
			prefix, *stage.ExpectGeneratedCount, result.GeneratedArtefactCount)
	}
}

func assertStagedGoldenFile(t *testing.T, goldenPath string, actualBytes []byte, msgAndArgs ...any) {
	t.Helper()

	normalisedActual := normaliseHTML(actualBytes)

	if *updateGolden {
		require.NoError(t, os.MkdirAll(filepath.Dir(goldenPath), 0755))
		require.NoError(t, os.WriteFile(goldenPath, normalisedActual, 0644))
	}
	expectedBytes, readErr := os.ReadFile(goldenPath)
	require.NoError(t, readErr, "Failed to read golden file %s. Run with -update flag to create it.", goldenPath)

	normalisedExpected := normaliseHTML(expectedBytes)

	if !bytes.Equal(normalisedExpected, normalisedActual) {
		t.Logf("--- EXPECTED (%s) ---\n%s\n--- ACTUAL (%s) ---\n%s\n",
			filepath.Base(goldenPath), string(normalisedExpected),
			filepath.Base(goldenPath), string(normalisedActual))
		assert.Fail(t, fmt.Sprintf("Golden file mismatch: %s. Run with -update if this change is intentional.", goldenPath), msgAndArgs...)
	}
}

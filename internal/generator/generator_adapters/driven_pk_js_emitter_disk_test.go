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

package generator_adapters

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/wdk/safedisk"
)

func TestDiskPKJSEmitter_EmitJS(t *testing.T) {
	t.Parallel()

	t.Run("empty source returns empty artefact ID", func(t *testing.T) {
		t.Parallel()
		tempDir := t.TempDir()
		emitter := NewDiskPKJSEmitter(tempDir, false)

		artefactID, err := emitter.EmitJS(context.Background(), "", "pages/test", "", false)

		require.NoError(t, err)
		assert.Empty(t, artefactID)
	})

	t.Run("transpiles and writes TypeScript to disk", func(t *testing.T) {
		t.Parallel()
		tempDir := t.TempDir()
		emitter := NewDiskPKJSEmitter(tempDir, false)

		source := `function greet(name: string): string { return "Hello " + name; }`
		artefactID, err := emitter.EmitJS(context.Background(), source, "pages/greeting", "", false)

		require.NoError(t, err)
		assert.Equal(t, "pk-js/pages/greeting.js", artefactID)

		expectedPath := filepath.Join(tempDir, "pk-js/pages/greeting.js")
		content, err := os.ReadFile(expectedPath)
		require.NoError(t, err)

		assert.NotContains(t, string(content), ": string")
		assert.Contains(t, string(content), "function greet")
	})

	t.Run("strips .pk extension from page path", func(t *testing.T) {
		t.Parallel()
		tempDir := t.TempDir()
		emitter := NewDiskPKJSEmitter(tempDir, false)

		source := `console.log("test");`
		artefactID, err := emitter.EmitJS(context.Background(), source, "pages/checkout.pk", "", false)

		require.NoError(t, err)
		assert.Equal(t, "pk-js/pages/checkout.js", artefactID)
	})

	t.Run("handles nested paths", func(t *testing.T) {
		t.Parallel()
		tempDir := t.TempDir()
		emitter := NewDiskPKJSEmitter(tempDir, false)

		source := `console.log("nested");`
		artefactID, err := emitter.EmitJS(context.Background(), source, "partials/widgets/counter", "", false)

		require.NoError(t, err)
		assert.Equal(t, "pk-js/partials/widgets/counter.js", artefactID)

		expectedPath := filepath.Join(tempDir, "pk-js/partials/widgets/counter.js")
		_, err = os.Stat(expectedPath)
		require.NoError(t, err)
	})

	t.Run("tracks written files", func(t *testing.T) {
		t.Parallel()
		tempDir := t.TempDir()
		emitter := NewDiskPKJSEmitter(tempDir, false)

		source := `console.log("track me");`
		artefactID, err := emitter.EmitJS(context.Background(), source, "pages/tracked", "", false)
		require.NoError(t, err)

		writtenPath := emitter.GetWrittenFilePath(artefactID)
		expectedPath := filepath.Join(tempDir, "pk-js/pages/tracked.js")
		assert.Equal(t, expectedPath, writtenPath)

		allFiles := emitter.GetAllWrittenFiles()
		assert.Len(t, allFiles, 1)
		assert.Equal(t, expectedPath, allFiles[artefactID])
	})

	t.Run("reset clears written files", func(t *testing.T) {
		t.Parallel()
		tempDir := t.TempDir()
		emitter := NewDiskPKJSEmitter(tempDir, false)

		source := `console.log("reset me");`
		_, err := emitter.EmitJS(context.Background(), source, "pages/reset", "", false)
		require.NoError(t, err)

		assert.Len(t, emitter.GetAllWrittenFiles(), 1)

		emitter.Reset()
		assert.Empty(t, emitter.GetAllWrittenFiles())
	})

	t.Run("returns error on syntax error", func(t *testing.T) {
		t.Parallel()
		tempDir := t.TempDir()
		emitter := NewDiskPKJSEmitter(tempDir, false)

		source := `function broken( { this is not valid }`
		_, err := emitter.EmitJS(context.Background(), source, "pages/broken", "", false)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "transpiling")
	})

	t.Run("minification works when enabled", func(t *testing.T) {
		t.Parallel()
		tempDir := t.TempDir()
		emitter := NewDiskPKJSEmitter(tempDir, true)

		source := `
		function greet(name) {
			return "Hello " + name;
		}
		`
		artefactID, err := emitter.EmitJS(context.Background(), source, "pages/minified", "", false)
		require.NoError(t, err)

		content, err := os.ReadFile(emitter.GetWrittenFilePath(artefactID))
		require.NoError(t, err)

		assert.NotContains(t, string(content), "\n\t\t")
	})
}

func TestDiskPKJSEmitter_EmitJS_Sandbox(t *testing.T) {
	t.Parallel()

	t.Run("writes files via sandbox", func(t *testing.T) {
		t.Parallel()
		sandbox := safedisk.NewMockSandbox("/output", safedisk.ModeReadWrite)
		defer sandbox.Close()

		emitter := NewDiskPKJSEmitter("/output", false, WithEmitterSandbox(sandbox))
		source := `console.log("hello");`
		artefactID, err := emitter.EmitJS(context.Background(), source, "pages/test", "", false)

		require.NoError(t, err)
		assert.Equal(t, "pk-js/pages/test.js", artefactID)
		assert.GreaterOrEqual(t, sandbox.CallCounts["MkdirAll"], 1)
		assert.GreaterOrEqual(t, sandbox.CallCounts["WriteFile"], 1)

		file := sandbox.GetFile("pk-js/pages/test.js")
		require.NotNil(t, file)
		assert.Contains(t, string(file.Data()), "hello")
	})

	t.Run("records written file path from sandbox root", func(t *testing.T) {
		t.Parallel()
		sandbox := safedisk.NewMockSandbox("/output", safedisk.ModeReadWrite)
		defer sandbox.Close()

		emitter := NewDiskPKJSEmitter("/output", false, WithEmitterSandbox(sandbox))
		source := `console.log("path");`
		artefactID, err := emitter.EmitJS(context.Background(), source, "pages/tracked", "", false)

		require.NoError(t, err)
		assert.Equal(t, "/output/pk-js/pages/tracked.js", emitter.GetWrittenFilePath(artefactID))
	})

	t.Run("MkdirAll error propagates", func(t *testing.T) {
		t.Parallel()
		sandbox := safedisk.NewMockSandbox("/output", safedisk.ModeReadWrite)
		defer sandbox.Close()
		sandbox.MkdirAllErr = errors.New("disk full")

		emitter := NewDiskPKJSEmitter("/output", false, WithEmitterSandbox(sandbox))
		_, err := emitter.EmitJS(context.Background(), `console.log("fail");`, "pages/fail", "", false)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "creating directory")
	})

	t.Run("WriteFile error propagates", func(t *testing.T) {
		t.Parallel()
		sandbox := safedisk.NewMockSandbox("/output", safedisk.ModeReadWrite)
		defer sandbox.Close()
		sandbox.WriteFileErr = errors.New("permission denied")

		emitter := NewDiskPKJSEmitter("/output", false, WithEmitterSandbox(sandbox))
		_, err := emitter.EmitJS(context.Background(), `console.log("fail");`, "pages/fail", "", false)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "writing PK JS")
	})
}

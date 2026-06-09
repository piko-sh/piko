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

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"piko.sh/piko/internal/annotator/annotator_dto"
)

func newStyleDepsTestService() *lifecycleService {
	return &lifecycleService{
		fs:          &MockFileSystem{},
		pathsConfig: LifecyclePathsConfig{BaseDir: "/proj"},
	}
}

func styleBuildResult(hashedName, sourcePath string, importedStyles ...string) *annotator_dto.ProjectAnnotationResult {
	return &annotator_dto.ProjectAnnotationResult{
		VirtualModule: &annotator_dto.VirtualModule{
			ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
				hashedName: {Source: &annotator_dto.ParsedComponent{SourcePath: sourcePath}},
			},
		},
		ComponentResults: map[string]*annotator_dto.AnnotationResult{
			hashedName: {ImportedStylePaths: importedStyles},
		},
	}
}

func TestProjectRelPath(t *testing.T) {
	t.Parallel()

	ls := newStyleDepsTestService()

	tests := []struct {
		name    string
		absPath string
		want    string
	}{
		{name: "inside root", absPath: "/proj/assets/x.css", want: "assets/x.css"},
		{name: "outside root", absPath: "/other/x.css", want: ""},
		{name: "dot-dot prefixed directory name stays local", absPath: "/proj/..cache/x.css", want: "..cache/x.css"},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, testCase.want, ls.projectRelPath(testCase.absPath))
		})
	}
}

func TestNormaliseStylePath(t *testing.T) {
	t.Parallel()

	ls := newStyleDepsTestService()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "absolute inside root", input: "/proj/lib/theme.css", want: "lib/theme.css"},
		{name: "absolute outside root", input: "/other/theme.css", want: ""},
		{name: "relative path is cleaned", input: "lib/./theme.css", want: "lib/theme.css"},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, testCase.want, ls.normaliseStylePath(testCase.input))
		})
	}
}

func TestComponentRelPath(t *testing.T) {
	t.Parallel()

	ls := newStyleDepsTestService()
	vm := &annotator_dto.VirtualModule{
		ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
			"hashA":    {Source: &annotator_dto.ParsedComponent{SourcePath: "/proj/pages/index.pk"}},
			"noSource": {Source: nil},
		},
	}

	assert.Equal(t, "pages/index.pk", ls.componentRelPath("hashA", vm))
	assert.Equal(t, "", ls.componentRelPath("missing", vm))
	assert.Equal(t, "", ls.componentRelPath("noSource", vm))
}

func TestUpdateStyleDepsFromBuildRoundTripsToImporters(t *testing.T) {
	t.Parallel()

	ls := newStyleDepsTestService()
	ls.updateStyleDepsFromBuild(styleBuildResult("hashA", "/proj/pages/index.pk", "/proj/lib/theme.css"))

	assert.Equal(t, []string{"pages/index.pk"}, ls.importersOfStyle("/proj/lib/theme.css"))
	assert.Equal(t, []string{"/proj/lib/theme.css"}, ls.allWatchedStyleFiles())
}

func TestUpdateStyleDepsFromBuildMergesAndDrops(t *testing.T) {
	t.Parallel()

	ls := newStyleDepsTestService()
	ls.updateStyleDepsFromBuild(styleBuildResult("hashA", "/proj/pages/index.pk", "/proj/lib/theme.css"))
	ls.updateStyleDepsFromBuild(styleBuildResult("hashB", "/proj/pages/about.pk", "/proj/lib/other.css"))

	assert.Equal(t, []string{"pages/index.pk"}, ls.importersOfStyle("/proj/lib/theme.css"))
	assert.Equal(t, []string{"pages/about.pk"}, ls.importersOfStyle("/proj/lib/other.css"))

	ls.updateStyleDepsFromBuild(styleBuildResult("hashA", "/proj/pages/index.pk"))
	assert.Nil(t, ls.importersOfStyle("/proj/lib/theme.css"))
}

func TestUpdateStyleDepsFromBuildIgnoresIncompleteResults(t *testing.T) {
	t.Parallel()

	ls := newStyleDepsTestService()
	ls.updateStyleDepsFromBuild(nil)
	ls.updateStyleDepsFromBuild(&annotator_dto.ProjectAnnotationResult{})

	assert.Nil(t, ls.allWatchedStyleFiles())
}

func TestAllWatchedStyleFilesDeDuplicatesUnion(t *testing.T) {
	t.Parallel()

	ls := newStyleDepsTestService()
	ls.updateStyleDepsFromBuild(styleBuildResult("hashA", "/proj/pages/index.pk", "/proj/lib/theme.css"))
	ls.updateStyleDepsFromBuild(styleBuildResult("hashB", "/proj/pages/about.pk", "/proj/lib/theme.css"))

	assert.Equal(t, []string{"/proj/lib/theme.css"}, ls.allWatchedStyleFiles())
}

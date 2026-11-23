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

package runner

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"

	"golang.org/x/mod/modfile"
	"piko.sh/piko/tests/fuzz/internal/casefile"
	"piko.sh/piko/tests/fuzz/internal/fixturego"
	"piko.sh/piko/tests/fuzz/internal/process"
)

const (
	// stagedFileMax bounds one file copied or read while staging an application, in bytes.
	stagedFileMax = 16 << 20

	// localModuleVersion is the placeholder version of locally replaced modules.
	localModuleVersion = "v0.0.0"

	// pikoModule is the Piko module path.
	pikoModule = "piko.sh/piko"

	// topologyModuleVersion is the major version of the topology fixture module.
	topologyModuleVersion = "v2.0.0"
)

var (
	// nativeMain is the formatted Go source staged in the generated project.
	nativeMain = fixturego.Fixed("hosts/native/main.go")

	// errFileOversize reports a staged or evidence file above its byte limit.
	errFileOversize = errors.New("file exceeds byte limit")
)

// stage creates the case's private application and Go workspace.
//
// Takes repo (string) which is the repository source directory.
// Takes work (string) which is the invocation-private working directory.
// Takes c (*casefile.Case) which holds the saved sources, choices and expected results.
//
// Returns app (string) which is the private application directory.
// Returns workspaceFile (string) which is the private Go workspace file.
// Returns err (error) which reports source staging or private module/workspace setup
// failure, or nil on success.
func stage(repo, work string, c *casefile.Case) (app, workspaceFile string, err error) {
	app = filepath.Join(work, "app")
	if err := writeCaseSources(app, c); err != nil {
		return "", "", err
	}
	workFile := process.Workspace(repo)
	workspace, appMod, err := loadStagingModules(repo, workFile)
	if err != nil {
		return "", "", err
	}
	uses, err := requireCaseModules(appMod, app, c)
	if err != nil {
		return "", "", err
	}
	if err := rebaseWorkspace(workspace, workFile, appMod, append(uses, &modfile.Use{Path: app})); err != nil {
		return "", "", err
	}
	if err := stageTargetModules(repo, work, app, c, workspace, appMod); err != nil {
		return "", "", err
	}
	workspaceFile = filepath.Join(work, "go.work")
	if err := writeStagedModules(repo, app, workFile, workspaceFile, workspace, appMod); err != nil {
		return "", "", err
	}
	return app, workspaceFile, nil
}

// writeCaseSources writes the case's source files into the private application.
//
// Takes app (string) which is the private application directory.
// Takes c (*casefile.Case) which holds the saved sources.
//
// Returns error which reports an unsafe source path or a write failure.
func writeCaseSources(app string, c *casefile.Case) error {
	if err := os.MkdirAll(app, privateDirectoryMode); err != nil {
		return err
	}
	for name, source := range c.Files {
		if !filepath.IsLocal(name) || filepath.Clean(name) != name {
			return fmt.Errorf("invalid source path %q", name)
		}
		destination := filepath.Join(app, name)
		if err := os.MkdirAll(filepath.Dir(destination), privateDirectoryMode); err != nil {
			return err
		}
		if err := os.WriteFile(destination, []byte(source), privateFileMode); err != nil {
			return err
		}
	}
	return os.WriteFile(filepath.Join(app, "piko_dependency.go"), []byte("package main\nimport _ \"piko.sh/piko\"\n"), privateFileMode)
}

// loadStagingModules parses the repository workspace and root module.
//
// Takes repo (string) which is the repository source directory.
// Takes workFile (string) which is the repository workspace file.
//
// Returns *modfile.WorkFile which is the parsed workspace.
// Returns *modfile.File which is the root module, renamed to the case module.
// Returns error which reports a read, parse or module edit failure.
func loadStagingModules(repo, workFile string) (*modfile.WorkFile, *modfile.File, error) {
	data, err := readBoundedFile(filepath.Dir(workFile), filepath.Base(workFile), stagedFileMax)
	if err != nil {
		return nil, nil, err
	}
	workspace, err := modfile.ParseWork(workFile, data, nil)
	if err != nil {
		return nil, nil, err
	}
	rootMod := filepath.Join(repo, "go.mod")
	data, err = readBoundedFile(repo, "go.mod", stagedFileMax)
	if err != nil {
		return nil, nil, err
	}
	appMod, err := modfile.Parse(rootMod, data, nil)
	if err != nil {
		return nil, nil, err
	}
	err = runSteps(
		func() error { return appMod.AddModuleStmt(casefile.Module) },
		func() error { return appMod.AddRequire(pikoModule, localModuleVersion) },
	)
	return workspace, appMod, err
}

// requireCaseModules adds the collection module requirements a case needs.
//
// Takes appMod (*modfile.File) which is the application module.
// Takes app (string) which is the private application directory.
// Takes c (*casefile.Case) which selects the collection module.
//
// Returns []*modfile.Use which lists the extra workspace modules.
// Returns error which reports a module edit failure.
func requireCaseModules(appMod *modfile.File, app string, c *casefile.Case) ([]*modfile.Use, error) {
	if !c.UsesCollections() {
		return nil, nil
	}
	source := filepath.Join(app, casefile.CollectionDir)
	err := runSteps(
		func() error {
			return appMod.AddRequire("piko.sh/piko/wdk/markdown/markdown_provider_goldmark", localModuleVersion)
		},
		func() error { return appMod.AddRequire(casefile.CollectionModule, localModuleVersion) },
		func() error { return appMod.AddReplace(casefile.CollectionModule, localModuleVersion, source, "") },
	)
	return []*modfile.Use{{Path: source}}, err
}

// rebaseWorkspace rewrites the workspace's uses and replacements as absolute paths and
// mirrors the replacements into the application module.
//
// Takes workspace (*modfile.WorkFile) which is the parsed repository workspace.
// Takes workFile (string) which is the repository workspace file.
// Takes appMod (*modfile.File) which is the application module.
// Takes extra ([]*modfile.Use) which lists the case modules: collections before the
// repository modules, and the application last.
//
// Returns error which reports a workspace or module edit failure.
func rebaseWorkspace(workspace *modfile.WorkFile, workFile string, appMod *modfile.File, extra []*modfile.Use) error {
	directory := filepath.Dir(workFile)
	application := extra[len(extra)-1]
	uses := slices.Clone(extra[:len(extra)-1])
	for _, use := range workspace.Use {
		uses = append(uses, &modfile.Use{Path: absolute(directory, use.Path)})
	}
	for _, use := range workspace.Use {
		if err := workspace.DropUse(use.Path); err != nil {
			return err
		}
	}
	workspace.Cleanup()
	workspace.SetUse(append(uses, application))
	for _, replacement := range workspace.Replace {
		newPath := replacement.New.Path
		if replacement.New.Version == "" {
			newPath = absolute(directory, newPath)
		}
		if err := workspace.AddReplace(replacement.Old.Path, replacement.Old.Version, newPath, replacement.New.Version); err != nil {
			return err
		}
		if err := appMod.AddReplace(replacement.Old.Path, replacement.Old.Version, newPath, replacement.New.Version); err != nil {
			return err
		}
	}
	return nil
}

// stageTargetModules adds the workflow and topology modules a target needs, and the local
// Piko replacement.
//
// Takes repo (string) which is the repository source directory.
// Takes work (string) which is the invocation-private working directory.
// Takes app (string) which is the private application directory.
// Takes c (*casefile.Case) which selects the target.
// Takes workspace (*modfile.WorkFile) which is the private workspace.
// Takes appMod (*modfile.File) which is the application module.
//
// Returns error which reports a staging or module edit failure.
func stageTargetModules(repo, work, app string, c *casefile.Case, workspace *modfile.WorkFile, appMod *modfile.File) error {
	if c.Target == targetFormWorkflow {
		if err := stageWorkflowModules(repo, appMod); err != nil {
			return err
		}
	}
	if c.Target == "go-topology" {
		moduleDirectories := filepath.Join(work, "topology-modules")
		if err := runSteps(
			func() error { return os.Rename(filepath.Join(app, "topology-modules"), moduleDirectories) },
			func() error { return appMod.AddRequire(casefile.TopologyModule, topologyModuleVersion) },
			func() error {
				return appMod.AddReplace(casefile.TopologyModule, topologyModuleVersion, filepath.Join(moduleDirectories, "module"), "")
			},
			func() error {
				return workspace.AddReplace(casefile.TopologyModule, topologyModuleVersion, filepath.Join(moduleDirectories, "workspace"), "")
			},
		); err != nil {
			return err
		}
	}
	if hasLocalPiko(appMod) {
		return nil
	}
	return appMod.AddReplace(pikoModule, localModuleVersion, repo, "")
}

// writeStagedModules writes the application module, workspace and their checksums.
//
// Takes repo (string) which is the repository source directory.
// Takes app (string) which is the private application directory.
// Takes workFile (string) which is the repository workspace file.
// Takes out (string) which is the private workspace file to write.
// Takes workspace (*modfile.WorkFile) which is the private workspace.
// Takes appMod (*modfile.File) which is the application module.
//
// Returns error which reports a formatting, write or copy failure.
func writeStagedModules(repo, app, workFile, out string, workspace *modfile.WorkFile, appMod *modfile.File) error {
	if err := writeMod(filepath.Join(app, "go.mod"), appMod); err != nil {
		return err
	}
	if err := copyIfPresent(repo, "go.sum", filepath.Join(app, "go.sum")); err != nil {
		return err
	}
	workspace.Cleanup()
	if err := os.WriteFile(out, modfile.Format(workspace.Syntax), privateFileMode); err != nil {
		return err
	}
	return copyIfPresent(filepath.Dir(workFile), filepath.Base(workFile)+".sum", out+".sum")
}

// runSteps runs fallible steps in order, stopping at the first failure.
//
// Takes steps (...func() error) which are the ordered steps.
//
// Returns error which is the first step failure, or nil when every step succeeds.
func runSteps(steps ...func() error) error {
	for _, step := range steps {
		if err := step(); err != nil {
			return err
		}
	}
	return nil
}

// absolute resolves a relative path against its original directory.
//
// Takes base (string) which is the original directory for relative paths.
// Takes path (string) which is the original absolute or relative filesystem path.
//
// Returns string which is the path resolved against its original directory.
func absolute(base, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(base, path)
}

// hasLocalPiko reports whether the module replaces Piko with local sources.
//
// Takes file (*modfile.File) which holds the parsed module definition.
//
// Returns bool which reports whether the module uses a local Piko replacement.
func hasLocalPiko(file *modfile.File) bool {
	for _, replacement := range file.Replace {
		if replacement.Old.Path == pikoModule && (replacement.Old.Version == "" || replacement.Old.Version == localModuleVersion) {
			return true
		}
	}
	return false
}

// copyIfPresent copies an optional file without changing its contents.
//
// Takes root (string) which is the directory the source file must stay inside.
// Takes from (string) which is the source file's path relative to root.
// Takes to (string) which is the destination file path.
//
// Returns error which reports file access, copying or byte-limit failure, or nil on
// success.
func copyIfPresent(root, from, to string) error {
	err := copyRequired(root, from, to)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return err
}

// copyRequired copies a required file of at most stagedFileMax bytes without changing its
// contents.
//
// Takes root (string) which is the directory the source file must stay inside.
// Takes from (string) which is the source file's path relative to root.
// Takes to (string) which is the destination file path.
//
// Returns error which reports file access, copying or byte-limit failure, or nil on
// success.
func copyRequired(root, from, to string) error {
	data, err := readBoundedFile(root, from, stagedFileMax)
	if err != nil {
		return err
	}
	return os.WriteFile(to, data, privateFileMode)
}

// readBoundedFile reads a whole file below root, refusing one larger than limit bytes.
//
// The file is opened through an os.Root, so name cannot escape root by a symbolic link or
// a parent-directory element.
//
// Takes root (string) which is the directory the file must stay inside.
// Takes name (string) which is the file's path relative to root.
// Takes limit (int) which is the largest accepted file size in bytes.
//
// Returns []byte which holds the file contents.
// Returns error which reports an unreadable file, or wraps errFileOversize when the file
// exceeds limit.
func readBoundedFile(root, name string, limit int) (data []byte, err error) {
	file, err := os.OpenInRoot(root, name)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, file.Close()) }()
	data, err = io.ReadAll(io.LimitReader(file, int64(limit)+1))
	if err != nil {
		return nil, err
	}
	if len(data) > limit {
		return nil, fmt.Errorf("%w: %s exceeds %d bytes", errFileOversize, filepath.Join(root, name), limit)
	}
	return data, nil
}

// writeMod writes a formatted private Go module file.
//
// Takes path (string) which is the private module file to write.
// Takes file (*modfile.File) which is the parsed module definition to format.
//
// Returns error which reports module formatting or file-write failure, or nil on success.
func writeMod(path string, file *modfile.File) error {
	file.Cleanup()
	data, err := file.Format()
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, privateFileMode)
}

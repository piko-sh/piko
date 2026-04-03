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

package templates

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"piko.sh/piko/plugins/agents"
	"piko.sh/piko/wdk/safedisk"
)

const (
	// currentDirPath represents the current directory path for use in destination
	// checks.
	currentDirPath = "."

	// dirPermissions is the file mode used when creating directories.
	dirPermissions = 0750

	// filePermissions is the file mode used when creating scaffold files.
	filePermissions = 0640
)

// ScaffoldData holds configuration for creating a new project from a template.
type ScaffoldData struct {
	// ProjectName is the name of the new Piko project to create.
	ProjectName string

	// ModuleName is the Go module path for the generated project.
	ModuleName string

	// DestinationPath is the target directory path; "." means the current
	// directory.
	DestinationPath string

	// PikoVersion is the resolved version of piko.sh/piko to use in the
	// generated go.mod. When empty, the template falls back to "v0.0.0".
	PikoVersion string

	// EnableInterpreted enables experimental interpreted mode support.
	EnableInterpreted bool

	// EnableAgents enables AI agent integration files (AGENTS.md, SKILL.md,
	// .claude-plugin/, .lsp.json, references/) for coding assistants.
	EnableAgents bool

	// EnableValidator enables struct validation via the go-playground/validator
	// wdk module.
	EnableValidator bool

	// EnableSonicJSON enables the high-performance Sonic JSON provider via
	// the piko.sh/piko/wdk/json/json_provider_sonic module.
	EnableSonicJSON bool
}

// CreateProject sets up a new Piko project at the given destination.
//
// Takes data (ScaffoldData) which contains the project name and destination
// path.
//
// Returns error when the destination already exists, the current folder is not
// empty, or any file operation fails.
func CreateProject(data ScaffoldData) error {
	sandboxDir, _ := resolveSandboxPaths(data.DestinationPath)
	factory, err := safedisk.NewFactory(safedisk.FactoryConfig{
		CWD:          sandboxDir,
		AllowedPaths: []string{data.DestinationPath},
		Enabled:      true,
	})
	if err != nil {
		return fmt.Errorf("could not create sandbox factory: %w", err)
	}

	if err := validateDestination(factory, data.DestinationPath); err != nil {
		return err
	}

	if err := createDirs(factory, data); err != nil {
		return err
	}

	if err := createReadmes(factory, data); err != nil {
		return err
	}

	if err := createConfigs(factory, data); err != nil {
		return err
	}

	if err := createIcons(factory, data); err != nil {
		return err
	}

	if err := createTemplateFiles(factory, data); err != nil {
		return err
	}

	return createAgents(data)
}

// CopyProjectAgents writes AGENTS.md and references/ to the given directory.
// Used by the scaffold wizard and `piko agents install` for project-level
// integration (Codex, Cursor, Copilot, Windsurf, and other AGENTS.md tools).
//
// Stale files in references/ are removed before copying so that renamed or
// deleted reference files do not linger across Piko upgrades.
//
// Takes destRoot (string) which is the directory to write agent files into.
//
// Returns error when a file cannot be read from the embed or written to disc.
func CopyProjectAgents(destRoot string) error {
	refsDir := filepath.Join(destRoot, "references")
	if err := os.RemoveAll(refsDir); err != nil {
		return fmt.Errorf("failed to clean references directory %s: %w", refsDir, err)
	}

	return copyAgentFiles(destRoot, func(path string) bool {
		if path == "AGENTS.md" {
			return true
		}
		if strings.HasPrefix(path, "references/") {
			return true
		}
		return false
	})
}

// CopyClaudeCodeSkill writes SKILL.md and references/ to the given directory.
// Used by `piko agents install` to install a personal-level Claude Code skill
// at ~/.claude/skills/piko/.
//
// The entire destination directory is removed before copying so that renamed
// or deleted files do not linger across Piko upgrades.
//
// Takes destDir (string) which is the directory to write skill files into.
//
// Returns error when a file cannot be read from the embed or written to disc.
func CopyClaudeCodeSkill(destDir string) error {
	if err := os.RemoveAll(destDir); err != nil {
		return fmt.Errorf("failed to clean skill directory %s: %w", destDir, err)
	}

	return copyAgentFiles(destDir, func(path string) bool {
		if path == "SKILL.md" {
			return true
		}
		if strings.HasPrefix(path, "references/") {
			return true
		}
		return false
	})
}

// validateDestination checks that the destination path is suitable for project
// creation.
//
// Takes factory (safedisk.Factory) which creates sandboxes for filesystem
// access.
// Takes destinationPath (string) which is the path to validate.
//
// Returns error when the sandbox cannot be created, the path cannot be checked,
// or the existing path is not suitable for project creation.
func validateDestination(factory safedisk.Factory, destinationPath string) error {
	sandboxDir, checkPath := resolveSandboxPaths(destinationPath)

	sandbox, err := factory.Create("scaffold-validate", sandboxDir, safedisk.ModeReadWrite)
	if err != nil {
		return fmt.Errorf("could not create sandbox for path '%s': %w", sandboxDir, err)
	}
	defer func() { _ = sandbox.Close() }()

	info, err := sandbox.Stat(checkPath)
	if err == nil {
		return validateExistingPath(sandbox, info, destinationPath)
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("could not stat destination path '%s': %w", destinationPath, err)
	}
	return nil
}

// resolveSandboxPaths determines the sandbox directory and check path for
// validation.
//
// Takes destinationPath (string) which specifies the target path to resolve.
//
// Returns sandboxDir (string) which is the directory to use as the sandbox.
// Returns checkPath (string) which is the path to validate within the sandbox.
func resolveSandboxPaths(destinationPath string) (sandboxDir, checkPath string) {
	if destinationPath == currentDirPath {
		cwd, err := os.Getwd()
		if err != nil {
			return currentDirPath, currentDirPath
		}
		return cwd, currentDirPath
	}
	return filepath.Dir(destinationPath), filepath.Base(destinationPath)
}

// validateExistingPath checks if an existing path is suitable for project
// creation.
//
// Takes sandbox (safedisk.Sandbox) which provides filesystem access.
// Takes info (os.FileInfo) which contains metadata about the existing path.
// Takes destinationPath (string) which specifies the target project location.
//
// Returns error when the path is not a directory, already exists as a named
// directory, or the current directory is not empty.
func validateExistingPath(sandbox safedisk.Sandbox, info os.FileInfo, destinationPath string) error {
	if !info.IsDir() {
		return fmt.Errorf("path '%s' already exists and is not a directory", destinationPath)
	}
	if destinationPath != currentDirPath {
		return fmt.Errorf("directory '%s' already exists", destinationPath)
	}
	entries, _ := sandbox.ReadDir(currentDirPath)
	if len(entries) > 0 {
		return errors.New("current directory is not empty")
	}
	return nil
}

// createDirs creates the standard project directory structure at the given
// path.
//
// Takes factory (safedisk.Factory) which creates sandboxes for filesystem
// access.
// Takes data (ScaffoldData) which provides the destination path and feature
// flags.
//
// Returns error when the sandbox cannot be created or a directory cannot be
// made.
func createDirs(factory safedisk.Factory, data ScaffoldData) error {
	dirs := []string{
		"actions/greeting",
		"cmd/generator",
		"cmd/main",
		"components",
		"pages",
		"partials",
		"pkg",
		"lib/icons",
		"dist",
		"e2e",
	}

	if data.EnableInterpreted {
		dirs = append(dirs, "internal/interpreted")
	}

	sandbox, err := factory.Create("scaffold-copy", data.DestinationPath, safedisk.ModeReadWrite)
	if err != nil {
		return fmt.Errorf("failed to create sandbox for destination: %w", err)
	}
	defer func() { _ = sandbox.Close() }()

	for _, directory := range dirs {
		if err := sandbox.MkdirAll(directory, dirPermissions); err != nil {
			fullPath := filepath.Join(data.DestinationPath, directory)
			return fmt.Errorf("failed to create directory %s: %w", fullPath, err)
		}
	}
	return nil
}

// createStaticFiles copies embedded files to a destination directory.
//
// Takes factory (safedisk.Factory) which creates sandboxes for filesystem
// access.
// Takes destinationPath (string) which specifies where files will be written.
// Takes sourceFS (embed.FS) which contains the embedded source files.
// Takes files (map[string]string) which maps destination paths to source paths.
//
// Returns error when the sandbox cannot be created, a source file cannot be
// read, or a destination file cannot be written.
func createStaticFiles(factory safedisk.Factory, destinationPath string, sourceFS embed.FS, files map[string]string) error {
	sandbox, err := factory.Create("scaffold-static", destinationPath, safedisk.ModeReadWrite)
	if err != nil {
		return fmt.Errorf("failed to create sandbox for destination: %w", err)
	}
	defer func() { _ = sandbox.Close() }()

	for destination, source := range files {
		content, err := fs.ReadFile(sourceFS, source)
		if err != nil {
			return fmt.Errorf("failed to read embedded file %s: %w", source, err)
		}
		if err := sandbox.WriteFile(destination, content, filePermissions); err != nil {
			destPath := filepath.Join(destinationPath, destination)
			return fmt.Errorf("failed to write file %s: %w", destPath, err)
		}
	}
	return nil
}

// createReadmes creates README files for all standard project folders.
//
// Takes factory (safedisk.Factory) which creates sandboxes for filesystem
// access.
// Takes data (ScaffoldData) which provides template values and the output path.
//
// Returns error when any file creation or template processing fails.
func createReadmes(factory safedisk.Factory, data ScaffoldData) error {
	staticFiles := map[string]string{
		"actions/README.md":    "readmes/actions.md",
		"cmd/README.md":        "readmes/cmd.md",
		"components/README.md": "readmes/components.md",
		"pages/README.md":      "readmes/pages.md",
		"partials/README.md":   "readmes/partials.md",
		"e2e/README.md":        "readmes/e2e.md",
	}
	if err := createStaticFiles(factory, data.DestinationPath, ReadmesFS, staticFiles); err != nil {
		return err
	}

	destination := filepath.Join(data.DestinationPath, "README.md")
	if err := createFromTemplate(factory, destination, "readmes/main_readme.md.tmpl", ReadmesFS, data); err != nil {
		return err
	}

	if data.EnableInterpreted {
		interpretedReadmeDest := filepath.Join(data.DestinationPath, "internal", "interpreted", "README.md")
		if err := createFromTemplate(factory, interpretedReadmeDest, "readmes/internal_interpreted.md.tmpl", ReadmesFS, data); err != nil {
			return err
		}
	}

	return nil
}

// createConfigs copies configuration files to the given path.
//
// Takes factory (safedisk.Factory) which creates sandboxes for filesystem
// access.
// Takes data (ScaffoldData) which provides the destination path and feature
// flags.
//
// Returns error when the files cannot be created.
func createConfigs(factory safedisk.Factory, data ScaffoldData) error {
	files := map[string]string{
		"config.json":   "configs/config.json",
		"Dockerfile":    "configs/Dockerfile",
		".dockerignore": "configs/.dockerignore",
		".air.toml":     "configs/.air.toml",
		".gitignore":    "configs/.gitignore",
	}

	if data.EnableInterpreted {
		files[".air.interpreted.toml"] = "configs/.air.interpreted.toml"
	}

	return createStaticFiles(factory, data.DestinationPath, ConfigsFS, files)
}

// createIcons copies SVG icon files to the project's lib/icons directory.
//
// Takes factory (safedisk.Factory) which creates sandboxes for filesystem
// access.
// Takes data (ScaffoldData) which provides the destination path.
//
// Returns error when the files cannot be created.
func createIcons(factory safedisk.Factory, data ScaffoldData) error {
	files := map[string]string{
		"lib/icons/piko-mark.svg":   "icons/piko-mark.svg",
		"lib/icons/bolt.svg":        "icons/bolt.svg",
		"lib/icons/shield.svg":      "icons/shield.svg",
		"lib/icons/puzzle.svg":      "icons/puzzle.svg",
		"lib/icons/zap.svg":         "icons/zap.svg",
		"lib/icons/arrow-right.svg": "icons/arrow-right.svg",
	}
	return createStaticFiles(factory, data.DestinationPath, IconsFS, files)
}

// createTemplateFiles creates the project scaffold by rendering templates.
//
// Takes factory (safedisk.Factory) which creates sandboxes for filesystem
// access.
// Takes data (ScaffoldData) which contains the project settings.
//
// Returns error when a template cannot be created.
func createTemplateFiles(factory safedisk.Factory, data ScaffoldData) error {
	templates := []struct {
		fs           embed.FS
		destPath     string
		templateName string
	}{
		{destPath: "go.mod", templateName: "go/go.mod.tmpl", fs: GoTmplFS},
		{destPath: "go.work", templateName: "go/go.work.tmpl", fs: GoTmplFS},
		{destPath: "cmd/generator/main.go", templateName: "go/generator.go.tmpl", fs: GoTmplFS},
		{destPath: "dist/generated.go", templateName: "go/generated.go.tmpl", fs: GoTmplFS},
		{destPath: "cmd/main/main.go", templateName: "go/main.go.tmpl", fs: GoTmplFS},
		{destPath: "actions/greeting/print.go", templateName: "go/print.go.tmpl", fs: GoTmplFS},
		{destPath: "actions/greeting/submit.go", templateName: "go/submit.go.tmpl", fs: GoTmplFS},
		{destPath: "pages/index.pk", templateName: "piko/index.pk.tmpl", fs: PikoTmplFS},
		{destPath: "pages/!404.pk", templateName: "piko/!404.pk.tmpl", fs: PikoTmplFS},
		{destPath: "pages/index_test.go", templateName: "go/index_test.go.tmpl", fs: GoTmplFS},
		{destPath: "partials/layout.pk", templateName: "piko/layout.pk.tmpl", fs: PikoTmplFS},
		{destPath: "partials/feature-card.pk", templateName: "piko/feature-card.pk.tmpl", fs: PikoTmplFS},
		{destPath: "e2e/e2e_test.go", templateName: "e2e/e2e_test.go.tmpl", fs: E2ETmplFS},
		{destPath: "e2e/homepage_test.go", templateName: "e2e/homepage_test.go.tmpl", fs: E2ETmplFS},
	}

	if data.EnableInterpreted {
		templates = append(templates, struct {
			fs           embed.FS
			destPath     string
			templateName string
		}{destPath: "internal/interpreted/provider.go", templateName: "go/provider.go.tmpl", fs: GoTmplFS})
	}

	for _, t := range templates {
		destination := filepath.Join(data.DestinationPath, t.destPath)
		if err := createFromTemplate(factory, destination, t.templateName, t.fs, data); err != nil {
			return err
		}
	}
	return nil
}

// createFromTemplate creates a file at destPath by executing a named template.
//
// Takes factory (safedisk.Factory) which creates sandboxes for filesystem
// access.
// Takes destPath (string) which is the path where the file will be created.
// Takes templateName (string) which is the template file to parse from
// sourceFS.
// Takes sourceFS (embed.FS) which contains the embedded template files.
// Takes data (ScaffoldData) which provides the values for template execution.
//
// Returns error when the template cannot be parsed, the file cannot be created,
// or template execution fails.
func createFromTemplate(factory safedisk.Factory, destPath, templateName string, sourceFS embed.FS, data ScaffoldData) error {
	parsedTemplate, err := template.New(filepath.Base(templateName)).ParseFS(sourceFS, templateName)
	if err != nil {
		return fmt.Errorf("failed to parse template %s: %w", templateName, err)
	}

	parentDir := filepath.Dir(destPath)
	fileName := filepath.Base(destPath)

	sandbox, err := factory.Create("scaffold-build", parentDir, safedisk.ModeReadWrite)
	if err != nil {
		return fmt.Errorf("failed to create sandbox for %s: %w", parentDir, err)
	}
	defer func() { _ = sandbox.Close() }()

	file, err := sandbox.Create(fileName)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", destPath, err)
	}
	defer func() { _ = file.Close() }()

	if err := parsedTemplate.Execute(file, data); err != nil {
		return fmt.Errorf("failed to execute template for %s: %w", destPath, err)
	}
	return nil
}

// createAgents copies project-level AI agent files into the project when
// enabled. Only AGENTS.md and references/ are written  - Claude Code-specific
// files are installed globally via `piko agents install`.
//
// Takes data (ScaffoldData) which provides the destination path and feature
// flags.
//
// Returns error when the embedded files cannot be copied.
func createAgents(data ScaffoldData) error {
	if !data.EnableAgents {
		return nil
	}
	return CopyProjectAgents(data.DestinationPath)
}

// copyAgentFiles walks the embedded agents filesystem and copies files
// matching the include filter to destRoot, preserving directory structure.
// Parent directories are created on demand.
//
// Takes destRoot (string) which is the target directory.
// Takes include (func(string) bool) which returns true for paths to copy.
//
// Returns error when a file cannot be read or written.
func copyAgentFiles(destRoot string, include func(path string) bool) error {
	return fs.WalkDir(agents.FS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if path == "." || d.IsDir() {
			return nil
		}

		if strings.HasSuffix(path, ".go") {
			return nil
		}

		if !include(path) {
			return nil
		}

		destPath := filepath.Join(destRoot, path)

		content, err := agents.FS.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read embedded agent file %s: %w", path, err)
		}

		if err := os.MkdirAll(filepath.Dir(destPath), dirPermissions); err != nil {
			return fmt.Errorf("failed to create directory for %s: %w", destPath, err)
		}

		if err := os.WriteFile(destPath, content, filePermissions); err != nil {
			return fmt.Errorf("failed to write agent file %s: %w", destPath, err)
		}

		return nil
	})
}

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

package campaign

import (
	"bytes"
	"context"
	"crypto/sha256"
	"debug/buildinfo"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"io/fs"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"slices"
	"strings"
	"time"

	"golang.org/x/mod/modfile"
	"piko.sh/json"
	"piko.sh/piko/tests/fuzz/internal/process"
)

const (
	// provenanceBytes bounds all changed-source bytes read for one provenance receipt.
	provenanceBytes = 64 << 20

	// maxManifest bounds one source-identity manifest read.
	maxManifest = 2 << 20

	// maxDiagnosticBytes bounds retained command diagnostics and checkout marker reads.
	maxDiagnosticBytes = 4096

	// provenanceTimeout bounds one provenance receipt.
	provenanceTimeout = 30 * time.Second

	// identityCommandTimeout bounds one provenance command.
	identityCommandTimeout = 8 * time.Second

	// goModName is the module manifest's file name.
	goModName = "go.mod"

	// goTool is the Go toolchain executable.
	goTool = "go"

	// toolchainModule is the linked-module key that records the Go toolchain version.
	toolchainModule = "go"

	// linkedBinaries identifies module resolution taken from linked worker binaries.
	linkedBinaries = "linked-binaries"

	// statusPrefixBytes is the length of a porcelain git status code and its separator.
	statusPrefixBytes = 3

	// minStatusEntryBytes is the shortest porcelain git status entry with a path.
	minStatusEntryBytes = statusPrefixBytes + 1

	// moduleDigestBytes is the number of digest bytes kept in a module set identity.
	moduleDigestBytes = 12

	// moduleFieldSeparator separates the path, version and sum of a module record.
	moduleFieldSeparator = " "
)

var (
	// errProvenanceTimeout marks a provenance receipt that exceeded its deadline.
	errProvenanceTimeout = errors.New("provenance receipt timeout")

	// errIdentityCommandTimeout marks a provenance command that exceeded its deadline.
	errIdentityCommandTimeout = errors.New("provenance command timeout")
)

// moduleListing is one module record from go list -m -json.
type moduleListing struct {
	// Replace describes the module's replacement, if any.
	Replace *moduleReplacement

	// Path is the module path.
	Path string

	// Version is the selected module version.
	Version string

	// Dir is the module's source directory.
	Dir string

	// Sum is the module checksum.
	Sum string

	// Main reports whether the module is a workspace main module.
	Main bool
}

// moduleReplacement is a module replacement from go list -m -json.
type moduleReplacement struct {
	// Path is the replacement module path.
	Path string

	// Version is the replacement version, or empty for a local directory.
	Version string

	// Dir is the replacement's source directory.
	Dir string

	// Sum is the replacement checksum.
	Sum string
}

// SourceRevision identifies a local checkout.
//
// A partial scan records its reason and omits the digest rather than assigning an
// identity to unread inputs.
type SourceRevision struct {
	// Directory is the local checkout directory.
	Directory string

	// Revision identifies the checkout's commit.
	Revision string `json:",omitempty"`

	// DirtySHA256 identifies changed source bytes when the scan succeeds.
	DirtySHA256 string `json:",omitempty"`

	// Problem records why source identity could not be established.
	Problem string `json:",omitempty"`
}

// Provenance separates versions linked into the supervisor from the workspace used to
// resolve worker builds.
//
// Local overrides carry their own source receipts.
type Provenance struct {
	// WorkerModules records module versions linked into each worker binary.
	WorkerModules map[string]map[string]string `json:",omitempty"`

	// Resolved holds the resolved workspace module versions.
	Resolved map[string]string

	// Linked holds module versions linked into the supervisor.
	Linked map[string]string

	// LocalModules maps local replacement modules to their source directories.
	LocalModules map[string]string

	// ModuleFiles identifies module and checksum file contents.
	ModuleFiles map[string]string

	// Sources holds local checkout source identities.
	Sources map[string]SourceRevision

	// ResolutionSource identifies the source of module-resolution evidence.
	ResolutionSource string

	// ResolutionProblem records a workspace-resolution failure.
	ResolutionProblem string `json:",omitempty"`

	// Go identifies the toolchain.
	Go string

	// Platform identifies the target platform.
	Platform string

	// Workspace is the workspace file path.
	Workspace string

	// WorkspaceSHA256 identifies the workspace file contents.
	WorkspaceSHA256 string
}

// limitedWriter fails the producing command if it exceeds the bound, retaining only a
// diagnostic prefix. git status is NUL-delimited and cannot be truncated.
type limitedWriter struct {
	// buffer retains the bounded command output.
	buffer bytes.Buffer

	// limit is the maximum retained or admitted byte count.
	limit int
}

// Write accepts output up to the configured retention bound.
//
// Takes data ([]byte) which holds subprocess output bytes.
//
// Returns int which is the number of accepted bytes.
// Returns error which reports that the bounded output buffer would overflow, or nil when
// all bytes fit.
func (w *limitedWriter) Write(data []byte) (int, error) {
	if len(data) > w.limit-w.buffer.Len() {
		return 0, fmt.Errorf("provenance command output exceeds %d bytes", w.limit)
	}
	return w.buffer.Write(data)
}

// recordWorker keeps the versions in the binary Go actually linked.
//
// Takes binary (string) which is the worker executable path.
//
// Returns error which reports an unreadable worker binary or invalid linked-module
// receipt, or nil on success.
func (p *Provenance) recordWorker(binary string) error {
	info, err := buildinfo.ReadFile(binary)
	if err != nil {
		return err
	}
	if p.WorkerModules == nil {
		p.WorkerModules = map[string]map[string]string{}
	}
	modules := map[string]string{toolchainModule: info.GoVersion}
	for _, dep := range info.Deps {
		modules[dep.Path] = linkedVersion(dep)
	}
	key, err := p.workerKey(filepath.Base(binary), modules)
	if err != nil {
		return err
	}
	p.WorkerModules[key] = modules

	if p.ResolutionSource == linkedBinaries || (p.ResolutionSource == "" && p.ResolutionProblem != "") {
		p.resolveFromLinked(modules)
	}
	return nil
}

// workerKey names a worker's linked-module receipt.
//
// Generated app or resumed workers can link a different module set. Each distinct receipt
// is retained while repeated cases stay compact.
//
// Takes name (string) which is the worker binary's base name.
// Takes modules (map[string]string) which maps linked modules to their versions.
//
// Returns string which is the base name, qualified by a module digest when it differs
// from the receipt already recorded under that name.
// Returns error which reports a digest write failure, or nil on success.
func (p *Provenance) workerKey(name string, modules map[string]string) (string, error) {
	previous, existing := p.WorkerModules[name]
	if !existing || maps.Equal(previous, modules) {
		return name, nil
	}
	digest := sha256.New()
	for _, module := range slices.Sorted(maps.Keys(modules)) {
		if _, err := fmt.Fprintf(digest, "%q:%q\n", module, modules[module]); err != nil {
			return "", err
		}
	}
	return fmt.Sprintf("%s@%x", name, digest.Sum(nil)[:moduleDigestBytes]), nil
}

// resolveFromLinked records linked worker modules as the fallback module resolution.
//
// Takes modules (map[string]string) which maps linked modules to their versions.
func (p *Provenance) resolveFromLinked(modules map[string]string) {
	p.ResolutionSource = linkedBinaries
	if p.Resolved == nil {
		p.Resolved = map[string]string{}
		maps.Copy(p.Resolved, p.Linked)
	}
	for name, value := range modules {
		if name == toolchainModule {
			continue
		}
		if previous, exists := p.Resolved[name]; exists && previous != value {
			p.Resolved[name] = "multiple linked selections (see Linked and WorkerModules)"
		} else {
			p.Resolved[name] = value
		}
	}
}

// recordToolchain records the Go toolchain version and the supervisor's linked modules.
//
// Takes repo (string) which is the repository source directory.
//
// Returns error which reports a failed go version command, or nil on success.
func (p *Provenance) recordToolchain(ctx context.Context, repo string) error {
	version, err := identityCommand(ctx, repo, goTool, "version")
	if err != nil {
		return err
	}
	p.Go = strings.TrimSpace(string(version))
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, dep := range info.Deps {
			p.Linked[dep.Path] = linkedVersion(dep)
		}
	}
	return nil
}

// recordGraph records offline workspace module resolution, falling back to linked
// modules.
//
// Takes repo (string) which is the repository source directory.
// Takes work (string) which is the workspace file path.
// Takes data ([]byte) which holds the workspace file bytes.
//
// Returns []string which lists locally replaced module directories found by resolution.
func (p *Provenance) recordGraph(ctx context.Context, repo, work string, data []byte) []string {
	graph, extras, resolveErr := resolveGraph(ctx, repo, work, data)
	p.Resolved = graph
	p.ResolutionSource = "workspace-graph"
	if resolveErr != nil {
		p.ResolutionSource = linkedBinaries
		p.Resolved = maps.Clone(p.Linked)
		p.ResolutionProblem = strings.SplitN(resolveErr.Error(), "\n", 2)[0]
	}
	return extras
}

// recordModule records one local module's path, manifest digest and source identity.
//
// Takes dir (string) which is the local module directory.
// Takes remaining (*int64) which tracks the shared source-hashing byte budget.
//
// Returns error which reports an unreadable or invalid manifest or checkout, or nil on
// success.
func (p *Provenance) recordModule(ctx context.Context, dir string, remaining *int64) error {
	data, err := smallRead(dir, goModName, maxManifest)
	if err != nil {
		return err
	}
	name := modfile.ModulePath(data)
	if name == "" {
		return fmt.Errorf("missing module path in %s/go.mod", dir)
	}
	p.LocalModules[dir] = name
	p.ModuleFiles[dir] = fmt.Sprintf("%x", sha256.Sum256(data))
	root, git, err := moduleSourceRoot(ctx, dir)
	if err != nil {
		return err
	}
	if _, found := p.Sources[root]; !found {
		p.Sources[root] = sourceRevision(ctx, root, git, remaining)
	}
	return nil
}

// linkedVersion formats a linked module's version and replacement.
//
// Takes dep (*debug.Module) which is the linked module metadata.
//
// Returns string which is the linked version including its replacement identity.
func linkedVersion(dep *debug.Module) string {
	value := strings.TrimSpace(dep.Version + moduleFieldSeparator + dep.Sum)
	if replacement := dep.Replace; replacement != nil {
		value = strings.TrimSpace(value + " => " + strings.TrimSpace(replacement.Path+moduleFieldSeparator+replacement.Version+moduleFieldSeparator+replacement.Sum))
	}
	return value
}

// RecordProvenance records local source and module identities without fetching
// dependencies.
//
// Its subprocesses never fetch modules or refresh the Git index. Hashing changed sources
// shares one byte budget.
//
// Takes repo (string) which is the repository source directory.
// Takes extraLocalModules (...string) which lists additional local source roots to
// identify.
//
// Returns Provenance which holds the collected toolchain, module and source identities.
// Returns error which reports cancellation or an unreadable required source identity, or
// nil on success.
func RecordProvenance(ctx context.Context, repo string, extraLocalModules ...string) (Provenance, error) {
	p := newProvenance()
	ctx, cancel := context.WithTimeoutCause(ctx, provenanceTimeout, errProvenanceTimeout)
	defer cancel()
	if err := p.recordToolchain(ctx, repo); err != nil {
		return p, err
	}
	work, err := filepath.Abs(process.Workspace(repo))
	if err != nil {
		return p, err
	}
	p.Workspace = work
	data, err := smallRead(filepath.Dir(work), filepath.Base(work), maxManifest)
	if err != nil {
		return p, err
	}
	p.WorkspaceSHA256 = fmt.Sprintf("%x", sha256.Sum256(data))
	parsed, err := modfile.ParseWork(work, data, nil)
	if err != nil {
		return p, err
	}
	modules := workspaceModules(work, parsed, append([]string{repo}, extraLocalModules...))
	for _, dir := range p.recordGraph(ctx, repo, work, data) {
		addModule(modules, work, dir)
	}
	dirs := slices.Sorted(maps.Keys(modules))

	for _, name := range slices.Clone(dirs) {
		replacements, err := moduleReplacements(name, modules)
		if err != nil {
			return p, err
		}
		dirs = append(dirs, replacements...)
	}
	remaining := int64(provenanceBytes)
	for _, dir := range dirs {
		if err := ctx.Err(); err != nil {
			return p, err
		}
		if err := p.recordModule(ctx, dir, &remaining); err != nil {
			return p, err
		}
	}
	return p, nil
}

// newProvenance creates an empty receipt for the running platform.
//
// Returns Provenance which holds the platform and empty identity maps.
func newProvenance() Provenance {
	return Provenance{
		WorkerModules:     nil,
		Resolved:          nil,
		ResolutionSource:  "",
		ResolutionProblem: "",
		Go:                "",
		Platform:          runtime.GOOS + "/" + runtime.GOARCH,
		Workspace:         "",
		WorkspaceSHA256:   "",
		Linked:            map[string]string{},
		LocalModules:      map[string]string{},
		ModuleFiles:       map[string]string{},
		Sources:           map[string]SourceRevision{},
	}
}

// workspaceModules collects the local module directories a workspace selects.
//
// Takes work (string) which is the workspace file path.
// Takes parsed (*modfile.WorkFile) which is the parsed workspace.
// Takes roots ([]string) which lists further local module directories to include.
//
// Returns map[string]bool which holds the cleaned absolute module directories.
func workspaceModules(work string, parsed *modfile.WorkFile, roots []string) map[string]bool {
	modules := map[string]bool{}
	for _, dir := range roots {
		addModule(modules, work, dir)
	}
	for _, use := range parsed.Use {
		addModule(modules, work, use.Path)
	}
	for _, replacement := range parsed.Replace {
		if replacement.New.Version == "" {
			addModule(modules, work, replacement.New.Path)
		}
	}
	return modules
}

// addModule adds one module directory, resolved against the workspace directory.
//
// Takes modules (map[string]bool) which receives the cleaned absolute directory.
// Takes work (string) which is the workspace file path.
// Takes dir (string) which is the module directory, absolute or workspace-relative.
func addModule(modules map[string]bool, work, dir string) {
	if !filepath.IsAbs(dir) {
		dir = filepath.Join(filepath.Dir(work), dir)
	}
	modules[filepath.Clean(dir)] = true
}

// moduleReplacements finds a module's existing local replacements not yet recorded.
//
// Takes name (string) which is the module directory.
// Takes modules (map[string]bool) which holds the recorded module directories and
// receives each found replacement.
//
// Returns []string which lists the newly found replacement directories.
// Returns error which reports an unreadable or invalid manifest or replacement, or nil on
// success.
func moduleReplacements(name string, modules map[string]bool) ([]string, error) {
	data, err := smallRead(name, goModName, maxManifest)
	if err != nil {
		return nil, err
	}
	module, err := modfile.Parse(goModName, data, nil)
	if err != nil {
		return nil, err
	}
	var found []string
	for _, replacement := range module.Replace {
		if replacement.New.Version != "" {
			continue
		}
		local := replacement.New.Path
		if !filepath.IsAbs(local) {
			local = filepath.Join(name, local)
		}
		local = filepath.Clean(local)

		if modules[local] {
			continue
		}
		_, err := os.Stat(filepath.Join(local, goModName))
		switch {
		case err == nil:
			modules[local] = true
			found = append(found, local)
		case !errors.Is(err, fs.ErrNotExist):
			return nil, fmt.Errorf("inspecting replacement %s: %w", local, err)
		}
	}
	return found, nil
}

// moduleSourceRoot selects the directory whose sources identify a module.
//
// Git status omits ignored modules, so those trees and modules inside the campaign state
// directory are hashed directly, keeping a local replacement inside a build or scratch
// directory replayable.
//
// Takes dir (string) which is the local module directory.
//
// Returns string which is the directory to fingerprint.
// Returns bool which reports whether the directory is fingerprinted through Git.
// Returns error which reports a failed git check-ignore, or nil on success.
func moduleSourceRoot(ctx context.Context, dir string) (string, bool, error) {
	root, git := checkoutRoot(dir)
	if !git || root == dir {
		return root, git, nil
	}
	_, err := identityCommand(ctx, dir, "git", "check-ignore", "-q", "--", goModName)
	if err == nil {
		root, git = dir, false
	} else if exit, ok := errors.AsType[*exec.ExitError](err); !ok || exit.ExitCode() != 1 {
		return root, git, err
	}
	if state := os.Getenv("PIKO_FUZZ_STATE"); state != "" {
		if relative, err := filepath.Rel(state, dir); err == nil && filepath.IsLocal(relative) {
			root, git = dir, false
		}
	}
	return root, git, nil
}

// smallRead reads a file below root subject to a byte limit.
//
// The file is opened through an os.Root, so name cannot escape root by a symbolic link or
// a parent-directory element.
//
// Takes root (string) which is the directory the file must stay inside.
// Takes relative (string) which is the file's path relative to root.
// Takes limit (int64) which is the maximum file size in bytes.
//
// Returns []byte which holds the file contents.
// Returns error which reports file access or a byte-limit violation, or nil on success.
func smallRead(root, relative string, limit int64) (data []byte, err error) {
	name := filepath.Join(root, relative)
	file, err := os.OpenInRoot(root, relative)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("closing %s: %w", name, closeErr)
		}
	}()
	data, err = io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", name, err)
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("input exceeds %d bytes: %s", limit, name)
	}
	return data, nil
}

// identityCommand runs a bounded local command for provenance evidence.
//
// Takes dir (string) which is the command working directory.
// Takes tool (string) which is the local executable to invoke.
// Takes args (...string) which supplies the command arguments.
//
// Returns []byte which holds the bounded command output.
// Returns error which reports command execution or bounded-output failure, or nil on
// success.
func identityCommand(ctx context.Context, dir, tool string, args ...string) ([]byte, error) {
	return identityCommandEnv(ctx, dir, nil, tool, args...)
}

// identityCommandEnv runs a bounded provenance command with the selected environment.
//
// Takes dir (string) which is the command working directory.
// Takes env ([]string) which adds or overrides inherited environment entries.
// Takes tool (string) which is the local executable to invoke.
// Takes args (...string) which supplies the local command arguments.
//
// Returns []byte which holds the bounded command output.
// Returns error which reports command execution or bounded-output failure, or nil on
// success.
func identityCommandEnv(ctx context.Context, dir string, env []string, tool string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeoutCause(ctx, identityCommandTimeout, errIdentityCommandTimeout)
	defer cancel()
	command := exec.CommandContext(ctx, tool, args...) //nolint:gosec // G204,G702: tool is the fixed go or git executable and the harness builds every argument
	command.Dir = dir
	command.Env = append(os.Environ(), "GIT_OPTIONAL_LOCKS=0", "GOPROXY=off", "GOTOOLCHAIN=local")
	output := limitedWriter{buffer: bytes.Buffer{}, limit: maxManifest}
	diagnostics := limitedWriter{buffer: bytes.Buffer{}, limit: maxDiagnosticBytes}
	command.Env = append(command.Env, env...)
	command.Stdout, command.Stderr = &output, &diagnostics
	if err := command.Run(); err != nil {
		if cause := context.Cause(ctx); cause != nil {
			err = errors.Join(err, cause)
		}
		return nil, fmt.Errorf("%s %v: %w %s", tool, args, err, diagnostics.buffer.String())
	}
	return output.buffer.Bytes(), nil
}

// checkoutRoot finds the enclosing Git checkout.
//
// Takes dir (string) which is the source directory to locate.
//
// Returns string which is the checkout root, or the original directory when no checkout
// exists.
// Returns bool which reports whether a Git checkout was found.
func checkoutRoot(dir string) (string, bool) {
	resolved, err := filepath.EvalSymlinks(dir)
	if err == nil {
		dir = resolved
	}
	for parent := dir; ; parent = filepath.Dir(parent) {
		if gitMarker(filepath.Join(parent, ".git")) {
			return parent, true
		}
		if filepath.Dir(parent) == parent {
			return dir, false
		}
	}
}

// gitMarker reports whether a path is a Git directory or a gitdir link file.
//
// Takes marker (string) which is the candidate .git path.
//
// Returns bool which reports whether the marker identifies a Git checkout.
func gitMarker(marker string) bool {
	info, err := os.Stat(marker)
	if err != nil {
		return false
	}
	if info.IsDir() {
		_, err := os.Stat(filepath.Join(marker, "HEAD"))
		return err == nil
	}
	if !info.Mode().IsRegular() {
		return false
	}
	data, err := smallRead(filepath.Dir(marker), filepath.Base(marker), maxDiagnosticBytes)
	return err == nil && bytes.HasPrefix(data, []byte("gitdir: "))
}

// sourceRevision records a checkout's revision and bounded changed-source identity.
//
// Takes root (string) which is the local source directory to fingerprint.
// Takes git (bool) which reports whether the source directory belongs to a Git checkout.
// Takes remaining (*int64) which tracks the shared source-hashing byte budget.
//
// Returns SourceRevision which holds the commit and source digest, with any hashing
// problem.
func sourceRevision(ctx context.Context, root string, git bool, remaining *int64) SourceRevision {
	revision := SourceRevision{Directory: root, Revision: "", DirtySHA256: "", Problem: ""}
	digest := sha256.New()
	tree, err := os.OpenRoot(root)
	if err != nil {
		revision.Problem = err.Error()
		return revision
	}
	defer func() { _ = tree.Close() }()
	if git {
		revision.Revision, err = hashGitChanges(ctx, digest, root, tree, remaining)
	} else {
		err = hashTree(ctx, digest, tree, remaining)
	}
	if err != nil {
		revision.Problem = err.Error()
	} else {
		revision.DirtySHA256 = hex.EncodeToString(digest.Sum(nil))
	}
	return revision
}

// hashGitChanges adds a checkout's changed and untracked sources to the identity.
//
// Takes digest (hash.Hash) which receives the ordered source identity bytes.
// Takes root (string) which is the checkout directory.
// Takes tree (*os.Root) which confines reads to the checkout.
// Takes remaining (*int64) which tracks the shared source-hashing byte budget.
//
// Returns string which is the checkout's commit, or empty when unreadable.
// Returns error which reports a git, status or hashing failure, or nil on success.
func hashGitChanges(ctx context.Context, digest hash.Hash, root string, tree *os.Root, remaining *int64) (string, error) {
	head, err := identityCommand(ctx, root, "git", "rev-parse", "HEAD")
	if err != nil {
		return "", err
	}
	revision := strings.TrimSpace(string(head))
	listing, err := identityCommand(ctx, root, "git", statusArguments(root)...)
	if err != nil {
		return revision, err
	}
	for line := range bytes.SplitSeq(listing, []byte{0}) {
		if len(line) == 0 {
			continue
		}
		if len(line) < minStatusEntryBytes || !filepath.IsLocal(string(line[statusPrefixBytes:])) {
			return revision, errors.New("invalid git status entry")
		}
		if _, err := digest.Write(line[:statusPrefixBytes]); err != nil {
			return revision, err
		}
		if err := hashPath(ctx, digest, tree, string(line[statusPrefixBytes:]), remaining); err != nil {
			return revision, err
		}
	}
	return revision, nil
}

// hashTree adds every file below a directory to the source identity.
//
// Takes digest (hash.Hash) which receives the ordered source identity bytes.
// Takes tree (*os.Root) which confines the walk to the directory.
// Takes remaining (*int64) which tracks the shared source-hashing byte budget.
//
// Returns error which reports a walk, cancellation or hashing failure, or nil on success.
func hashTree(ctx context.Context, digest hash.Hash, tree *os.Root, remaining *int64) error {
	return fs.WalkDir(tree.FS(), ".", func(name string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		return hashPath(ctx, digest, tree, filepath.FromSlash(name), remaining)
	})
}

// hashPath adds a file's name and contents to the source identity.
//
// Takes digest (hash.Hash) which receives the ordered source identity bytes.
// Takes tree (*os.Root) which confines reads to the local source directory.
// Takes name (string) which is the source path relative to the source directory.
// Takes remaining (*int64) which tracks the shared source-hashing byte budget.
//
// Returns error which reports cancellation, an unreadable source or exhaustion of the
// shared byte budget, or nil on success.
func hashPath(ctx context.Context, digest hash.Hash, tree *os.Root, name string, remaining *int64) (err error) {
	if err := ctx.Err(); err != nil {
		return err
	}
	info, err := tree.Lstat(name)
	if errors.Is(err, fs.ErrNotExist) {
		_, err = fmt.Fprintf(digest, "%q deleted\n", name)
		return err
	}
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(digest, "%q %v %d\n", name, info.Mode(), info.Size()); err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		target, err := tree.Readlink(name)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(digest, "%q\n", target)
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("unsupported source entry %q", name)
	}
	if info.Size() > *remaining {
		return fmt.Errorf("dirty source content exceeds the %d byte receipt budget", provenanceBytes)
	}
	file, err := tree.Open(name)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("closing %s: %w", name, closeErr)
		}
	}()
	copied, err := io.Copy(digest, io.LimitReader(file, info.Size()+1))
	*remaining -= copied
	if copied != info.Size() {
		return fmt.Errorf("source changed while hashing %q", name)
	}
	return err
}

// resolveGraph runs module selection against a disposable copy of the workspace.
//
// Takes repo (string) which is the repository source directory.
// Takes work (string) which is the original go.work file path.
// Takes data ([]byte) which holds the original go.work file bytes.
//
// Returns map[string]string which maps module paths to selected version and replacement
// identities.
// Returns []string which maps locally replaced modules to their source directories.
// Returns error which reports offline workspace resolution or malformed module-graph
// output, or nil on success.
func resolveGraph(ctx context.Context, repo, work string, data []byte) (map[string]string, []string, error) {
	directory, err := os.MkdirTemp("", "piko-fuzz-provenance-")
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = os.RemoveAll(directory) }()
	copied, err := writeResolutionWorkspace(work, data, directory)
	if err != nil {
		return nil, nil, err
	}
	listing, err := identityCommandEnv(ctx, repo, []string{"GOWORK=" + copied}, goTool, "list", "-mod=readonly", "-m", "-json", "all")
	if err != nil {
		return nil, nil, err
	}
	return parseModuleListings(listing)
}

// writeResolutionWorkspace writes a copy of the workspace with absolute local paths.
//
// Takes work (string) which is the original go.work file path.
// Takes data ([]byte) which holds the original go.work file bytes.
// Takes directory (string) which is the disposable directory receiving the copy.
//
// Returns string which is the copied go.work path.
// Returns error which reports an invalid workspace or a failed write, or nil on success.
func writeResolutionWorkspace(work string, data []byte, directory string) (string, error) {
	workFile, err := modfile.ParseWork(work, data, nil)
	if err != nil {
		return "", err
	}
	base := filepath.Dir(work)
	absolute := func(name string) string {
		if filepath.IsAbs(name) {
			return name
		}
		return filepath.Join(base, name)
	}
	uses := make([]*modfile.Use, 0, len(workFile.Use))
	for _, use := range workFile.Use {
		uses = append(uses, &modfile.Use{Path: absolute(use.Path), ModulePath: use.ModulePath, Syntax: nil})
	}
	for _, use := range workFile.Use {
		if err := workFile.DropUse(use.Path); err != nil {
			return "", err
		}
	}
	workFile.Cleanup()
	workFile.SetUse(uses)
	for _, replacement := range workFile.Replace {
		if replacement.New.Version != "" {
			continue
		}
		if err := workFile.AddReplace(replacement.Old.Path, replacement.Old.Version, absolute(replacement.New.Path), ""); err != nil {
			return "", err
		}
	}
	workFile.Cleanup()
	copied := filepath.Join(directory, "go.work")
	if err := os.WriteFile(copied, modfile.Format(workFile.Syntax), filePermissions); err != nil {
		return "", err
	}
	return copied, copyWorkspaceSums(work, copied)
}

// copyWorkspaceSums copies the workspace checksum file beside the copied workspace.
//
// Takes work (string) which is the original go.work file path.
// Takes copied (string) which is the copied go.work file path.
//
// Returns error which reports an unreadable or unwritable checksum file, or nil when it
// is copied or absent.
func copyWorkspaceSums(work, copied string) error {
	sums, err := smallRead(filepath.Dir(work), filepath.Base(work)+".sum", maxManifest)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	return os.WriteFile(copied+".sum", sums, filePermissions)
}

// statusArguments builds the git status arguments that list a checkout's changes.
//
// The campaign state directory is excluded when it lies inside the checkout.
//
// Takes root (string) which is the checkout directory.
//
// Returns []string which holds the git arguments.
func statusArguments(root string) []string {
	args := []string{
		"-c", "core.fsmonitor=false", "status", "--porcelain=v1", "-z", "--untracked-files=all", "--no-renames",
		"--ignore-submodules=all", "--", ".",
	}
	if state := os.Getenv("PIKO_FUZZ_STATE"); state != "" {
		if relative, err := filepath.Rel(root, state); err == nil && filepath.IsLocal(relative) && relative != "." {
			args = append(args, ":(exclude,top,literal)"+filepath.ToSlash(relative))
		}
	}
	return args
}

// parseModuleListings decodes go list -m -json output into selected module versions.
//
// Takes data ([]byte) which holds the concatenated module listings.
//
// Returns map[string]string which maps module paths to selected version and replacement
// identities.
// Returns []string which lists locally replaced module directories.
// Returns error which reports malformed listing output, or nil on success.
func parseModuleListings(data []byte) (map[string]string, []string, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	versions := map[string]string{}
	var dirs []string
	for {
		var listing moduleListing
		err := decoder.Decode(&listing)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, nil, err
		}
		choice := strings.TrimSpace(listing.Version + moduleFieldSeparator + listing.Sum)
		if listing.Main {
			choice = listing.Dir
		}
		if replacement := listing.Replace; replacement != nil {
			if replacement.Version == "" && replacement.Dir != "" {
				dirs = append(dirs, replacement.Dir)
				choice += " => " + replacement.Dir
			} else {
				choice += " => " + strings.TrimSpace(replacement.Path+moduleFieldSeparator+replacement.Version+moduleFieldSeparator+replacement.Sum)
			}
		}
		versions[listing.Path] = choice
	}
	return versions, dirs, nil
}

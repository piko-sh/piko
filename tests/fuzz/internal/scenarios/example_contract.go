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

package scenarios

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"hash"
	"io/fs"
	"path/filepath"
	"slices"

	"golang.org/x/mod/modfile"
	"piko.sh/json"
	"piko.sh/piko/tests/fuzz/internal/casefile"
	"piko.sh/piko/tests/fuzz/internal/process"
	"piko.sh/piko/tests/fuzz/internal/publiccontracts"
	"piko.sh/safedisk"
)

const (
	// maxExampleFiles bounds the authored files staged for one example.
	maxExampleFiles = 512

	// maxExampleBytes bounds the authored bytes staged for one example.
	maxExampleBytes = 12 << 20

	// exampleSpecFile is the example's browser test specification file name.
	exampleSpecFile = "testspec.json"

	// workspaceName is the private Go workspace file name.
	workspaceName = "go.work"
)

var (
	// errExampleTooLarge reports an example beyond the staging limits.
	errExampleTooLarge = errors.New("example exceeds staging limits")

	// errExampleSymlink reports an authored symbolic link in an example.
	errExampleSymlink = errors.New("unexpected authored symlink")

	// errExampleEntryPoint reports a cmd/main without exactly one piko.New call.
	errExampleEntryPoint = errors.New("example must contain exactly one piko.New call in cmd/main")

	// errExampleUnverified reports an example run without independent assertions.
	errExampleUnverified = errors.New("missing independent assertion")
)

// exampleCopier stages one example's authored files and digests them.
type exampleCopier struct {
	// source is the read-only sandbox over the authored example sources.
	source safedisk.Sandbox

	// app is the read-write sandbox over the private application directory.
	app safedisk.Sandbox

	// hasher digests every staged file.
	hasher hash.Hash

	// name is the example name used in limit errors.
	name string

	// total counts the staged bytes.
	total int64

	// count counts the staged files.
	count int
}

// visit stages one walked entry into the private application directory.
//
// Takes relative (string) which is the entry path relative to the example sources.
// Takes entry (fs.DirEntry) which describes the walked entry.
// Takes err (error) which is the walk error for the entry.
//
// Returns error which reports an authored symbolic link, an example beyond the staging
// limits or a copy failure, or filepath.SkipDir for a generated directory.
func (c *exampleCopier) visit(relative string, entry fs.DirEntry, err error) error {
	if err != nil {
		return err
	}
	if entry.IsDir() {
		switch entry.Name() {
		case ".piko", "dist", "bin", "node_modules", ".git":
			return filepath.SkipDir
		}
		return c.app.MkdirAll(relative, privateDirectoryMode)
	}
	if entry.Type()&fs.ModeSymlink != 0 {
		return fmt.Errorf("%w: %s", errExampleSymlink, relative)
	}
	if !entry.Type().IsRegular() || relative == "go.work" || relative == "go.work.sum" {
		return nil
	}
	info, err := entry.Info()
	if err != nil {
		return err
	}
	c.total += info.Size()
	c.count++
	if c.count > maxExampleFiles || c.total > maxExampleBytes {
		return fmt.Errorf("%w: %s", errExampleTooLarge, c.name)
	}
	contents, err := c.source.ReadFile(relative)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(c.hasher, "%d:%s:%d:", len(relative), relative, len(contents)); err != nil {
		return fmt.Errorf("digest %s: %w", relative, err)
	}
	if _, err := c.hasher.Write(contents); err != nil {
		return fmt.Errorf("digest %s: %w", relative, err)
	}
	return c.app.WriteFile(relative, contents, privateFileMode)
}

// runExample runs an authored example contract in a private build.
//
// Takes repository (string) which is the repository source directory.
// Takes work (string) which is the invocation-private working directory.
// Takes fuzzCase (*casefile.Case) which holds the saved sources, choices and expected
// results.
//
// Returns publiccontracts.Receipt which holds selected example assertion evidence,
// including partial evidence on failure.
// Returns error which reports contract setup, execution or a missing positive assertion
// receipt, or nil on success.
func runExample(ctx context.Context, repository, work string, fuzzCase *casefile.Case) (publiccontracts.Receipt, error) {
	name := publiccontracts.LocalExamples[fuzzCase.Index%uint64(len(publiccontracts.LocalExamples))]
	receipt := publiccontracts.Receipt{
		Contract:           "example/" + name,
		SourceSHA256:       "",
		Steps:              0,
		Assertions:         0,
		HTTPStatusVerified: false,
		BrowserVisited:     false,
	}
	source := filepath.Join(repository, "examples", "scenarios", name, "src")
	app := filepath.Join(work, "src")
	digest, err := stageExample(source, app, work, name)
	if err != nil {
		return receipt, err
	}
	receipt.SourceSHA256 = digest
	if err := buildExample(ctx, repository, source, app, work, name); err != nil {
		return receipt, err
	}
	test := SuiteTest{
		Package:  "tests/integration/e2e_browser",
		Source:   "tests/fuzz/internal/scenarios/testdata/examples.go",
		TestName: "TestFuzzExampleContract",
		Files:    nil,
	}
	err = RunSuite(ctx, repository, work, fuzzCase, test)
	return exampleReceipt(work, name, receipt, err)
}

// stageExample copies the authored example and its test specification into private
// directories.
//
// Takes source (string) which is the authored example source directory.
// Takes app (string) which is the private application directory.
// Takes work (string) which is the invocation-private working directory.
// Takes name (string) which is the example name.
//
// Returns string which is the hex digest of the staged sources and specification.
// Returns error which reports a staging failure.
func stageExample(source, app, work, name string) (string, error) {
	hasher := sha256.New()
	if err := copyExampleSources(source, app, name, hasher); err != nil {
		return "", fmt.Errorf("example/%s: stage sources: %w", name, err)
	}
	spec, err := readFileIn(filepath.Dir(source), exampleSpecFile)
	if err != nil {
		return "", fmt.Errorf("example/%s: read test specification: %w", name, err)
	}
	if _, err := hasher.Write(slices.Concat([]byte("spec:"), spec)); err != nil {
		return "", fmt.Errorf("example/%s: digest test specification: %w", name, err)
	}
	digest := fmt.Sprintf("%x", hasher.Sum(nil))
	if err := writeFileIn(work, exampleSpecFile, spec); err != nil {
		return "", fmt.Errorf("example/%s: stage test specification: %w", name, err)
	}
	return digest, nil
}

// copyExampleSources copies an example's authored files through sandboxes rooted at the
// source and application directories.
//
// Takes source (string) which is the authored example source directory.
// Takes app (string) which is the private application directory.
// Takes name (string) which is the example name.
// Takes hasher (hash.Hash) which digests every staged file.
//
// Returns error which reports a sandbox, walk or copy failure.
func copyExampleSources(source, app, name string, hasher hash.Hash) (err error) {
	sourceSandbox, err := safedisk.NewSandbox(source, safedisk.ModeReadOnly)
	if err != nil {
		return fmt.Errorf("open %s: %w", source, err)
	}
	defer func() { err = errors.Join(err, sourceSandbox.Close()) }()
	appSandbox, err := safedisk.NewSandbox(app, safedisk.ModeReadWrite)
	if err != nil {
		return fmt.Errorf("open %s: %w", app, err)
	}
	defer func() { err = errors.Join(err, appSandbox.Close()) }()
	copier := exampleCopier{source: sourceSandbox, app: appSandbox, hasher: hasher, name: name, total: 0, count: 0}
	return sourceSandbox.WalkDir(".", copier.visit)
}

// buildExample generates and builds the staged example against a private workspace.
//
// Takes repository (string) which is the repository source directory.
// Takes source (string) which is the authored example source directory.
// Takes app (string) which is the private application directory.
// Takes work (string) which is the invocation-private working directory.
// Takes name (string) which is the example name.
//
// Returns error which reports a workspace, listener, generation or build failure.
func buildExample(ctx context.Context, repository, source, app, work, name string) error {
	gowork, err := exampleWorkspace(repository, source, app, work)
	if err != nil {
		return fmt.Errorf("example/%s: prepare workspace: %w", name, err)
	}
	environment := []string{
		"GOWORK=" + gowork, "GOPROXY=off", "GOSUMDB=off", "GOTOOLCHAIN=local",
		"HTTP_PROXY=http://127.0.0.1:1", "HTTPS_PROXY=http://127.0.0.1:1", "NO_PROXY=localhost,127.0.0.1",
	}
	steps := [][]string{
		{"run", "-mod=readonly", "./cmd/generator", "all"},
		{"build", "-mod=readonly", "-o", filepath.Join(work, "example-server"), "./cmd/main"},
	}
	for _, arguments := range steps {
		if arguments[0] == "build" {
			if err := configureExampleListener(app); err != nil {
				return fmt.Errorf("example/%s: configure listener: %w", name, err)
			}
		}
		output, err := process.Run(ctx, app, environment, "go", arguments...)
		if err != nil {
			return fmt.Errorf("example/%s/%s: %w\n%s", name, arguments[0], err, output)
		}
	}
	return nil
}

// exampleReceipt merges the browser suite's assertion counts into the receipt.
//
// Takes work (string) which is the invocation-private working directory.
// Takes name (string) which is the example name.
// Takes receipt (publiccontracts.Receipt) which holds the evidence gathered so far.
// Takes runErr (error) which is the browser suite's result.
//
// Returns publiccontracts.Receipt which holds the merged evidence, including partial
// evidence on failure.
// Returns error which reports the suite failure, unreadable counts or a missing positive
// assertion receipt.
func exampleReceipt(work, name string, receipt publiccontracts.Receipt, runErr error) (publiccontracts.Receipt, error) {
	raw, readErr := readFileIn(work, "example-counts.json")
	if readErr == nil {
		var counts publiccontracts.Receipt
		if decodeErr := json.Unmarshal(raw, &counts); decodeErr != nil {
			return receipt, errors.Join(runErr, fmt.Errorf("example/%s: decode assertion counts: %w", name, decodeErr))
		}
		receipt.Steps, receipt.Assertions = counts.Steps, counts.Assertions
		receipt.HTTPStatusVerified, receipt.BrowserVisited = counts.HTTPStatusVerified, counts.BrowserVisited
	}
	if runErr != nil {
		return receipt, runErr
	}
	if readErr != nil {
		return receipt, fmt.Errorf("example/%s: read assertion counts: %w", name, readErr)
	}
	if receipt.Assertions == 0 || !receipt.HTTPStatusVerified || !receipt.BrowserVisited {
		return receipt, fmt.Errorf("example/%s/%w", name, errExampleUnverified)
	}
	return receipt, nil
}

// exampleWorkspace rebases the example's local replacements into a private workspace.
//
// Takes repository (string) which is the repository source directory.
// Takes source (string) which is the original example source directory.
// Takes app (string) which is the private generated application directory.
// Takes work (string) which is the invocation-private working directory.
//
// Returns string which is the rebased private Go workspace file.
// Returns error which reports invalid local replacements or private-workspace file
// failure, or nil on success.
func exampleWorkspace(repository, source, app, work string) (string, error) {
	if err := rebaseModule(source, app); err != nil {
		return "", err
	}
	workspaceFile := process.Workspace(repository)
	workspaceDirectory := filepath.Dir(workspaceFile)
	data, err := readFileIn(workspaceDirectory, filepath.Base(workspaceFile))
	if err != nil {
		return "", fmt.Errorf("read workspace: %w", err)
	}
	workspace, err := modfile.ParseWork(workspaceFile, data, nil)
	if err != nil {
		return "", fmt.Errorf("parse workspace: %w", err)
	}
	if err := rebaseWorkspace(workspace, workspaceDirectory, app); err != nil {
		return "", err
	}
	if err := writeFileIn(work, workspaceName, modfile.Format(workspace.Syntax)); err != nil {
		return "", fmt.Errorf("write workspace: %w", err)
	}
	sums, err := readFileIn(workspaceDirectory, filepath.Base(workspaceFile)+".sum")
	if errors.Is(err, fs.ErrNotExist) {
		return filepath.Join(work, workspaceName), nil
	}
	if err != nil {
		return "", fmt.Errorf("read workspace sums: %w", err)
	}
	if err := writeFileIn(work, workspaceName+".sum", sums); err != nil {
		return "", fmt.Errorf("write workspace sums: %w", err)
	}
	return filepath.Join(work, workspaceName), nil
}

// rebaseModule rewrites the staged go.mod so local replacements resolve against the
// authored source directory.
//
// Takes source (string) which is the original example source directory.
// Takes app (string) which is the private generated application directory.
//
// Returns error which reports an unreadable, invalid or unwritable go.mod.
func rebaseModule(source, app string) error {
	moduleFile := filepath.Join(app, "go.mod")
	data, err := readFileIn(app, "go.mod")
	if err != nil {
		return fmt.Errorf("read go.mod: %w", err)
	}
	module, err := modfile.Parse(moduleFile, data, nil)
	if err != nil {
		return fmt.Errorf("parse go.mod: %w", err)
	}
	for _, replacement := range module.Replace {
		if replacement.New.Version != "" {
			continue
		}
		if err := module.AddReplace(replacement.Old.Path, replacement.Old.Version, resolvePath(source, replacement.New.Path), ""); err != nil {
			return fmt.Errorf("rebase replacement %s: %w", replacement.Old.Path, err)
		}
	}
	output, err := module.Format()
	if err != nil {
		return fmt.Errorf("format go.mod: %w", err)
	}
	if err := writeFileIn(app, "go.mod", output); err != nil {
		return fmt.Errorf("write go.mod: %w", err)
	}
	return nil
}

// rebaseWorkspace makes the private application the first workspace module and resolves
// every relative use and local replacement against the original workspace directory.
//
// Takes workspace (*modfile.WorkFile) which is the parsed repository workspace.
// Takes base (string) which is the original workspace directory.
// Takes app (string) which is the private generated application directory.
//
// Returns error which reports a use or replacement that cannot be rewritten.
func rebaseWorkspace(workspace *modfile.WorkFile, base, app string) error {
	uses := []*modfile.Use{{Path: app}}
	for _, use := range workspace.Use {
		uses = append(uses, &modfile.Use{Path: resolvePath(base, use.Path)})
		if err := workspace.DropUse(use.Path); err != nil {
			return fmt.Errorf("drop workspace use %s: %w", use.Path, err)
		}
	}
	workspace.Cleanup()
	workspace.SetUse(uses)
	for _, replacement := range workspace.Replace {
		if replacement.New.Version != "" {
			continue
		}
		if err := workspace.AddReplace(replacement.Old.Path, replacement.Old.Version, resolvePath(base, replacement.New.Path), ""); err != nil {
			return fmt.Errorf("rebase workspace replacement %s: %w", replacement.Old.Path, err)
		}
	}
	workspace.Cleanup()
	return nil
}

// configureExampleListener selects a private loopback listener in the example build.
//
// Takes app (string) which is the private generated application directory.
//
// Returns error which reports an unsupported example entry point or source-rewrite
// failure, or nil on success.
func configureExampleListener(app string) error {
	mainFile := filepath.Join("cmd", "main", "main.go")
	data, err := readFileIn(app, mainFile)
	if err != nil {
		return fmt.Errorf("read cmd/main: %w", err)
	}
	if bytes.Count(data, []byte("piko.New(")) != 1 {
		return errExampleEntryPoint
	}
	data = bytes.Replace(data, []byte("piko.New("), []byte("newLocalExample("), 1)
	if err = writeFileIn(app, mainFile, data); err != nil {
		return fmt.Errorf("write cmd/main: %w", err)
	}
	const shim = `package main
import("os";"strconv";"piko.sh/piko")
func newLocalExample(options ...piko.Option)*piko.SSRServer{
	port,err:=strconv.Atoi(os.Getenv("PIKO_TEST_PORT"));if err!=nil||port<=0{panic("private listener port missing")}
	options=append(options,piko.WithPort(port),piko.WithBindAddress("127.0.0.1"),piko.WithAutoNextPort(false),
		piko.WithHealthEnabled(false),
		piko.WithMonitoring(piko.WithMonitoringAddress("127.0.0.1:0"),piko.WithMonitoringBindAddress("127.0.0.1")))
	return piko.New(options...)
}
`
	return writeFileIn(app, filepath.Join("cmd", "main", "piko_fuzz_listener.go"), []byte(shim))
}

// resolvePath resolves a possibly relative path against base.
//
// Takes base (string) which is the directory relative paths resolve against.
// Takes path (string) which is the path to resolve.
//
// Returns string which is path when absolute, or path joined to base.
func resolvePath(base, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(base, path)
}

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

package modules_provider_filesystem

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"strings"

	"piko.sh/piko/wdk/modules"
	"piko.sh/piko/wdk/safedisk"
)

const (
	// fileExtension is the suffix every bundle file under the root must carry. Allows the
	// provider to ignore stray non-bundle content (READMEs, hidden files) that hosts may
	// drop in the same tree.
	fileExtension = ".pkbundle"

	// bundleDirPermissions is the permission mode applied when MkdirAll creates the
	// per-bundle storage directories.
	bundleDirPermissions = 0o750

	// bundleFilePermissions is the permission mode applied when the envelope file is written
	// through the temp-then-rename helper.
	bundleFilePermissions = 0o600

	// defaultMaxBundleBytes is the default ceiling on the size of one bundle file. It sits
	// well above any realistic module so only a corrupt or hostile file reaches it.
	defaultMaxBundleBytes int64 = 512 << 20
)

var (
	// errBundleTooLarge reports a bundle file larger than the configured read limit.
	errBundleTooLarge = errors.New("modules_provider_filesystem: bundle file exceeds the size limit")
)

// Option configures a Provider.
type Option func(*Provider)

// Provider serves bundles from a directory tree on the local filesystem. Construct with
// New; safe for concurrent Resolve.
type Provider struct {
	// sandbox confines every file operation under the configured root. Paths passed to its
	// methods are interpreted relative to that root, so the provider cannot read or write
	// outside its own directory tree.
	sandbox safedisk.Sandbox

	// maxBundleBytes is the largest bundle file Resolve reads; a larger file is rejected
	// rather than truncated.
	maxBundleBytes int64

	// ownsSandbox reports whether the provider created the sandbox and so closes it in
	// Close.
	ownsSandbox bool
}

// WithMaxBundleBytes sets the largest bundle file Resolve will read.
//
// Takes limit (int64) which is the size ceiling in bytes; values below one keep the
// default.
//
// Returns Option which applies the limit.
func WithMaxBundleBytes(limit int64) Option {
	return func(p *Provider) {
		if limit > 0 {
			p.maxBundleBytes = limit
		}
	}
}

// New constructs a Provider rooted at the given directory, creating the directory when it
// does not exist yet. Resolve returns modules.ErrModuleNotFound for any path that doesn't
// resolve to a readable file at lookup time.
//
// All I/O is routed through a safedisk sandbox rooted at the directory, which the
// provider owns and releases in Close. Callers that need to inject a pre-built sandbox
// (for tests, capability gating, or a custom storage backend) should use NewWithSandbox
// instead.
//
// Takes root (string) which is the absolute or relative directory path.
// Takes options (...Option) which configure read limits.
//
// Returns *Provider which is a sandbox-backed module provider implementing
// modules.ModuleProvider.
// Returns error when the root path is empty or rejected by safedisk.
func New(root string, options ...Option) (*Provider, error) {
	sandbox, err := safedisk.NewSandbox(root, safedisk.ModeReadWrite)
	if err != nil {
		return nil, fmt.Errorf("modules_provider_filesystem: building sandbox for %q: %w", root, err)
	}
	provider := NewWithSandbox(sandbox, options...)
	provider.ownsSandbox = true
	return provider, nil
}

// NewWithSandbox constructs a Provider that delegates every file operation to the given
// sandbox.
//
// Useful for tests and for hosts that build sandboxes through a capability-gated factory.
// The caller keeps ownership of the sandbox.
//
// Takes sandbox (safedisk.Sandbox) which provides the rooted file system view; must be
// writable for Write to succeed.
// Takes options (...Option) which configure read limits.
//
// Returns *Provider implementing modules.ModuleProvider.
func NewWithSandbox(sandbox safedisk.Sandbox, options ...Option) *Provider {
	provider := &Provider{
		sandbox:        sandbox,
		maxBundleBytes: defaultMaxBundleBytes,
		ownsSandbox:    false,
	}
	for _, option := range options {
		option(provider)
	}
	return provider
}

// Close releases the sandbox when the provider created it in New. A provider built with
// NewWithSandbox leaves the caller's sandbox open.
//
// Returns error when closing the owned sandbox fails.
func (p *Provider) Close() error {
	if !p.ownsSandbox {
		return nil
	}
	if err := p.sandbox.Close(); err != nil {
		return fmt.Errorf("modules_provider_filesystem: closing sandbox: %w", err)
	}
	return nil
}

// Write persists a bundle under the root.
//
// Replaces any existing entry for the same (Path, Version). Atomic via temp + rename;
// callers do not need to quiesce concurrent readers.
//
// Takes bundle (*modules.ModuleBundle) which must include a Ref with a non-empty Path.
// Empty Version is permitted and stored as "latest".
//
// Returns error when validation, marshalling, or filesystem I/O fails.
func (p *Provider) Write(bundle *modules.ModuleBundle) error {
	if err := bundle.Validate(); err != nil {
		return err
	}
	envelope, err := MarshalEnvelope(bundle)
	if err != nil {
		return err
	}
	folder, err := encodeModuleFolder(bundle.Descriptor.Ref.Path)
	if err != nil {
		return err
	}
	if err := p.sandbox.MkdirAll(folder, bundleDirPermissions); err != nil {
		return fmt.Errorf("modules_provider_filesystem: creating %s: %w", folder, err)
	}
	target := filepath.Join(folder, fileName(bundle.Descriptor.Ref.Version))
	if err := p.sandbox.WriteFileAtomic(target, envelope, bundleFilePermissions); err != nil {
		return fmt.Errorf("modules_provider_filesystem: writing %s: %w", target, err)
	}
	return nil
}

// Resolve implements modules.ModuleProvider.
//
// Reads the bundle file for (ref.Path, ref.Version), parses it, and yields the bundle.
// The interpreter verifies the ref's Pin by checking the bundle against the pin when the
// module is loaded. A missing file maps to modules.ErrModuleNotFound; a file larger than
// the configured limit is rejected; all other errors propagate unchanged.
//
// Takes ctx (context.Context) which is checked for cancellation; I/O itself is
// synchronous.
// Takes ref (modules.ModuleRef) which identifies the bundle.
//
// Returns *modules.ModuleBundle which is the parsed bundle.
// Returns error when ctx is cancelled, the file is missing or too large, or the envelope
// fails to parse.
func (p *Provider) Resolve(ctx context.Context, ref modules.ModuleRef) (*modules.ModuleBundle, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if ref.Path == "" {
		return nil, modules.ErrModuleNotFound
	}
	folder, err := encodeModuleFolder(ref.Path)
	if err != nil {
		return nil, err
	}
	target := filepath.Join(folder, fileName(ref.Version))
	data, err := p.readBundleFile(target)
	if err != nil {
		return nil, err
	}
	return UnmarshalEnvelope(data)
}

// readBundleFile reads one bundle file through the sandbox, refusing files larger than
// the configured limit.
//
// Takes target (string) which is the sandbox-relative path of the bundle file.
//
// Returns []byte which holds the complete file.
// Returns error which wraps modules.ErrModuleNotFound when the file is missing,
// errBundleTooLarge when it exceeds the limit, or the underlying I/O failure.
func (p *Provider) readBundleFile(target string) ([]byte, error) {
	handle, err := p.sandbox.Open(target)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, modules.ErrModuleNotFound
		}
		return nil, fmt.Errorf("modules_provider_filesystem: opening %s: %w", target, err)
	}
	defer func() { _ = handle.Close() }()

	data, err := io.ReadAll(io.LimitReader(handle, p.maxBundleBytes+1))
	if err != nil {
		return nil, fmt.Errorf("modules_provider_filesystem: reading %s: %w", target, err)
	}
	if int64(len(data)) > p.maxBundleBytes {
		return nil, fmt.Errorf("%w: %s is larger than %d bytes", errBundleTooLarge, target, p.maxBundleBytes)
	}
	return data, nil
}

// encodeModuleFolder converts a module path into its on-disk folder name (relative to the
// sandbox root). Encodes "/" as "__" so the filesystem treats the path as a single
// directory name and we get the same shape on case-sensitive (Linux) and case-insensitive
// (macOS/Windows) filesystems.
//
// Rejects paths containing ".." segments to defend against directory traversal from
// untrusted callers.
//
// Takes modulePath (string) which is the module's canonical identifier.
//
// Returns string which is the relative folder name to use under the sandbox root.
// Returns error when modulePath is empty or contains a traversal segment.
func encodeModuleFolder(modulePath string) (string, error) {
	clean := filepath.ToSlash(modulePath)
	if clean == "" || strings.Contains(clean, "..") {
		return "", fmt.Errorf("modules_provider_filesystem: invalid module path %q", modulePath)
	}
	return strings.ReplaceAll(clean, "/", "__"), nil
}

// fileName returns the bundle file's basename for a given module version.
//
// Empty version becomes "latest" so providers can ship a floating-pointer-style mode for
// tests and dev.
//
// Takes version (string) which is the module's version string.
//
// Returns string which is the basename including extension.
func fileName(version string) string {
	if version == "" {
		return "latest" + fileExtension
	}
	return version + fileExtension
}

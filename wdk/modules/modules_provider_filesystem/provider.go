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
)

// Provider serves bundles from a directory tree on the local filesystem. Construct with
// New; safe for concurrent Resolve.
type Provider struct {
	// sandbox confines every file operation under the configured root. Paths passed to its
	// methods are interpreted relative to that root, so the provider cannot read or write
	// outside its own directory tree.
	sandbox safedisk.Sandbox
}

// New constructs a Provider rooted at the given directory. The directory does not need to
// exist when New is called; Resolve returns modules.ErrModuleNotFound for any path that
// doesn't resolve to a readable file at lookup time.
//
// Internally wraps the root in a safedisk.NewNoOpSandbox so all I/O is routed through the
// sandbox API. Callers that need to inject a pre-built sandbox (for tests, capability
// gating, or a custom storage backend) should use NewWithSandbox instead.
//
// Takes root (string) which is the absolute or relative directory path.
//
// Returns *Provider which is a sandbox-backed module provider implementing
// modules.ModuleProvider.
// Returns error when the root path is empty or rejected by safedisk.
func New(root string) (*Provider, error) {
	sandbox, err := safedisk.NewNoOpSandbox(root, safedisk.ModeReadWrite)
	if err != nil {
		return nil, fmt.Errorf("modules_provider_filesystem: building sandbox for %q: %w", root, err)
	}
	return NewWithSandbox(sandbox), nil
}

// NewWithSandbox constructs a Provider that delegates every file operation to the given
// sandbox. Useful for tests and for hosts that build sandboxes through a capability-gated
// factory.
//
// Takes sandbox (safedisk.Sandbox) which provides the rooted file system view; must be
// writable for Write to succeed.
//
// Returns *Provider implementing modules.ModuleProvider.
func NewWithSandbox(sandbox safedisk.Sandbox) *Provider {
	return &Provider{sandbox: sandbox}
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
// The ref's Pin is NOT verified here; callers (or piko's loader) check integrity via
// modules.ModuleBundle.VerifyAgainstRef. A missing file maps to
// modules.ErrModuleNotFound; all other errors propagate unchanged.
//
// Takes ctx (context.Context) which is checked for cancellation; I/O itself is
// synchronous.
// Takes ref (modules.ModuleRef) which identifies the bundle.
//
// Returns *modules.ModuleBundle which is the parsed bundle.
// Returns error when ctx is cancelled, the file is missing, or the envelope fails to
// parse.
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
	data, err := p.sandbox.ReadFile(target)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, modules.ErrModuleNotFound
		}
		return nil, fmt.Errorf("modules_provider_filesystem: reading %s: %w", target, err)
	}
	bundle, err := UnmarshalEnvelope(data)
	if err != nil {
		return nil, err
	}
	return bundle, nil
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

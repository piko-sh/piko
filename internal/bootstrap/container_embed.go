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

package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/generator/generator_adapters"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/safedisk"
)

// EmbedScope selects how much of the runtime payload generation copies into the embed
// package a piko_embed-tagged build carries.
type EmbedScope int

const (
	// EmbedAll embeds the whole runtime payload: source and derived blobs, storage, and the
	// registry snapshot. It is the default.
	EmbedAll EmbedScope = iota

	// EmbedSourceOnly is reserved for embedding only source blobs, leaving derived variants
	// to be regenerated at runtime.
	EmbedSourceOnly
)

const (
	// embedPikoDirPermission is the mode for the .piko directory ensured before an embed
	// emit.
	embedPikoDirPermission os.FileMode = 0o750
)

var (
	// ErrEmbedSourceOnlyUnimplemented is returned when generation is asked for the reserved
	// source-only scope, which generation rejects rather than partially honouring.
	ErrEmbedSourceOnlyUnimplemented = errors.New(
		"EmbedSourceOnly is not yet implemented; use EmbedAll or build with --no-embed")
)

// EmitEmbeddedRuntime copies the runtime payload from .piko into dist/embed/piko and
// writes the tag-gated dist/embed_gen.go, so a subsequent build with the piko_embed tag
// produces a self-contained binary. Generation invokes it after the persistence snapshot
// flush, because the registry snapshot must exist before the copy.
//
// A project with no .piko directory emits only the marker and the generated file, keeping
// the embed directive resolvable and the build green.
//
// Returns error when the scope is unsupported, a sandbox cannot be made, or the emit
// fails.
func (c *Container) EmitEmbeddedRuntime(ctx context.Context) error {
	switch c.embedScope {
	case EmbedAll:
	case EmbedSourceOnly:
		return ErrEmbedSourceOnlyUnimplemented
	default:
		return fmt.Errorf("unknown embed scope %d; use EmbedAll or build with --no-embed", c.embedScope)
	}

	ctx, l := logger_domain.From(ctx, log)
	baseDir := deref(c.serverConfig.Paths.BaseDir, ".")

	factory, err := c.GetSandboxFactory()
	if err != nil {
		return fmt.Errorf("getting sandbox factory for the embed payload: %w", err)
	}

	pikoDir := filepath.Join(baseDir, config.PikoInternalPath)
	if err := os.MkdirAll(pikoDir, embedPikoDirPermission); err != nil {
		return fmt.Errorf("ensuring .piko exists for the embed payload: %w", err)
	}
	source, err := factory.Create("embed-source", pikoDir, safedisk.ModeReadOnly)
	if err != nil {
		return fmt.Errorf("creating the embed source sandbox: %w", err)
	}
	defer func() { _ = source.Close() }()

	destination, err := factory.Create("embed-output", filepath.Join(baseDir, distDirName), safedisk.ModeReadWrite)
	if err != nil {
		return fmt.Errorf("creating the embed output sandbox: %w", err)
	}
	defer func() { _ = destination.Close() }()

	var opts []generator_adapters.EmbedEmitterOption
	if c.dbProvider != nil && c.dbProvider.IsSecondary() {
		snapshotSource, cleanup, err := c.stageRegistrySnapshot(ctx)
		if err != nil {
			return err
		}
		defer cleanup()
		opts = append(opts, generator_adapters.WithSnapshotSource(snapshotSource))
	}

	l.Internal("Emitting the embedded runtime payload into dist/embed")
	return generator_adapters.NewDrivenEmbedEmitter(source, destination, opts...).Emit(ctx)
}

// stageRegistrySnapshot writes an in-memory registry to a private directory for
// embedding.
//
// Returns safedisk.Sandbox which holds wal/persistence/registry/snapshot.piko.
// Returns func() which closes the sandbox and removes the directory.
// Returns error when the directory, snapshot, or sandbox cannot be created.
func (c *Container) stageRegistrySnapshot(ctx context.Context) (safedisk.Sandbox, func(), error) {
	staging, err := os.MkdirTemp("", "piko-registry-")
	if err != nil {
		return nil, nil, fmt.Errorf("creating a staging directory for the registry snapshot: %w", err)
	}
	removeStaging := func() { _ = os.RemoveAll(staging) }

	if err := c.dbProvider.WriteRegistrySnapshot(ctx, filepath.Join(staging, "wal", "persistence")); err != nil {
		removeStaging()
		return nil, nil, fmt.Errorf("staging the registry snapshot: %w", err)
	}
	sandbox, err := safedisk.NewSandbox(staging, safedisk.ModeReadOnly)
	if err != nil {
		removeStaging()
		return nil, nil, fmt.Errorf("creating the registry snapshot sandbox: %w", err)
	}
	return sandbox, func() {
		_ = sandbox.Close()
		removeStaging()
	}, nil
}

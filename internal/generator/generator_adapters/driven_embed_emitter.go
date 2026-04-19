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
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"strings"

	"piko.sh/piko/internal/generator/generator_dto"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/safedisk"
)

const (
	// embedDirName is the directory under dist/ holding the copied runtime payload.
	embedDirName = "embed"

	// embedPikoDirName is the directory under embedDirName mirroring the .piko root.
	embedPikoDirName = "embed/piko"

	// embedMarkerName is a marker file written into the payload so the go:embed directive
	// always resolves, even for a project with no blobs, storage, or snapshot.
	embedMarkerName = "embed/piko/.pikoembed"

	// embedGenFileName is the generated Go file that embeds and registers the payload.
	embedGenFileName = "embed_gen.go"

	// embedRegistrySnapshotPath is the registry snapshot's path inside .piko, the single
	// file the embedded runtime loads its artefact metadata from.
	embedRegistrySnapshotPath = "wal/persistence/registry/snapshot.piko"

	// embedDirPermission is the mode for directories created inside the payload.
	embedDirPermission = 0o750

	// embedFilePermission is the mode for files created inside the payload.
	embedFilePermission = 0o640

	// copyBufferSize is the chunk size used by the cancellable payload copy loop.
	copyBufferSize = 32 * 1024
)

var (
	// embedPayloadTrees are the .piko subtrees the runtime reads from an embedded
	// filesystem.
	embedPayloadTrees = []string{"blobs", "storage"}
)

// DrivenEmbedEmitter copies the runtime payload out of .piko into dist/embed/piko and
// emits the tag-gated dist/embed_gen.go that embeds and registers it, which is what makes
// a piko_embed-tagged build self-contained.
type DrivenEmbedEmitter struct {
	// source is a read-only sandbox rooted at the project's .piko directory.
	source safedisk.Sandbox

	// destination is a writable sandbox rooted at the project's dist directory.
	destination safedisk.Sandbox
}

// NewDrivenEmbedEmitter creates an embed emitter over the given sandboxes.
//
// Takes source (safedisk.Sandbox) which is rooted at the project's .piko directory.
// Takes destination (safedisk.Sandbox) which is rooted at the project's dist directory.
//
// Returns *DrivenEmbedEmitter ready to emit.
func NewDrivenEmbedEmitter(source, destination safedisk.Sandbox) *DrivenEmbedEmitter {
	return &DrivenEmbedEmitter{source: source, destination: destination}
}

// Emit rebuilds dist/embed/piko from the current .piko payload and writes
// dist/embed_gen.go.
//
// The previous payload directory is removed first, so a rerun never accumulates stale
// blobs. The copy takes only the allow-listed subtrees (blobs, storage) and the registry
// snapshot; a marker file guarantees the embed directive resolves even when all three are
// absent.
//
// Returns error when the payload cannot be copied or the generated file cannot be
// written.
func (e *DrivenEmbedEmitter) Emit(ctx context.Context) error {
	ctx, l := logger_domain.From(ctx, log)

	copied, err := e.writePayload(ctx)
	if err != nil {
		e.cleanupPartialPayload(ctx)
		return err
	}

	l.Internal("Emitted embedded runtime payload",
		logger_domain.Int("files", copied))
	return nil
}

// writePayload clears any previous payload, copies the allow-listed subtrees and the
// registry snapshot into dist/embed/piko, writes the marker, and only then writes
// dist/embed_gen.go, so the generated file never references a half-copied tree. Returns
// the count copied.
//
// Returns int which is the number of payload files copied so far, even on failure.
// Returns error when any clear, copy, or write step fails.
func (e *DrivenEmbedEmitter) writePayload(ctx context.Context) (int, error) {
	if err := e.destination.RemoveAll(embedDirName); err != nil {
		return 0, fmt.Errorf("clearing previous embed payload: %w", err)
	}
	if err := e.destination.MkdirAll(embedPikoDirName, embedDirPermission); err != nil {
		return 0, fmt.Errorf("creating embed payload directory: %w", err)
	}

	copied := 0
	for _, tree := range embedPayloadTrees {
		treeCopied, err := e.copyTree(ctx, tree)
		if err != nil {
			return copied, err
		}
		copied += treeCopied
	}

	snapshotCopied, err := e.copyFileIfPresent(ctx, embedRegistrySnapshotPath)
	if err != nil {
		return copied, err
	}
	if snapshotCopied {
		copied++
	}

	marker := []byte("Embedded runtime payload. Generated by Piko; do not edit.\n")
	if err := e.destination.WriteFile(embedMarkerName, marker, embedFilePermission); err != nil {
		return copied, fmt.Errorf("writing embed marker: %w", err)
	}

	writer := NewFSWriter(e.destination)
	if err := writer.WriteFile(ctx, embedGenFileName, []byte(GenerateEmbedGenFile())); err != nil {
		return copied, fmt.Errorf("writing %s: %w", embedGenFileName, err)
	}

	return copied, nil
}

// cleanupPartialPayload removes a half-written payload after a failed emit.
//
// It deletes the embed directory and any stale generated file. Without it a later
// piko_embed build could embed a truncated tree or a generated file that outlived its
// payload. Removal failures are logged and never returned, so they cannot mask the
// original emit error.
func (e *DrivenEmbedEmitter) cleanupPartialPayload(ctx context.Context) {
	_, l := logger_domain.From(ctx, log)
	if err := e.destination.RemoveAll(embedDirName); err != nil {
		l.Warn("Failed to remove the partial embed payload after an emit error",
			logger_domain.Error(err))
	}
	if err := e.destination.RemoveAll(embedGenFileName); err != nil {
		l.Warn("Failed to remove the stale generated embed file after an emit error",
			logger_domain.Error(err))
	}
}

// copyTree streams every regular file under the named .piko subtree into the payload.
//
// Relative paths are preserved. An absent subtree is not an error, because a project may
// have no blobs or no storage; absence is decided by stat-ing the subtree root first.
// Once the root exists, any walk failure propagates, including a file vanishing mid-walk,
// so a partial tree is never mistaken for an absent one and shipped as a truncated
// payload.
//
// Takes tree (string) which is the subtree name relative to the .piko root.
//
// Returns int which is the number of files copied, even on failure.
// Returns error when the subtree cannot be inspected, read, or written.
func (e *DrivenEmbedEmitter) copyTree(ctx context.Context, tree string) (int, error) {
	ctx, l := logger_domain.From(ctx, log)
	if _, err := e.source.Stat(tree); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			l.Internal("Embed payload subtree absent, skipping", logger_domain.String("tree", tree))
			return 0, nil
		}
		return 0, fmt.Errorf("inspecting .piko/%s for the embed payload: %w", tree, err)
	}

	copied := 0
	walkErr := e.source.WalkDir(tree, func(entryPath string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		if entry.IsDir() {
			return e.destination.MkdirAll(path.Join(embedPikoDirName, entryPath), embedDirPermission)
		}
		if !entry.Type().IsRegular() {
			return nil
		}
		if copyErr := e.copyFile(ctx, entryPath); copyErr != nil {
			return copyErr
		}
		copied++
		return nil
	})
	if walkErr != nil {
		return copied, fmt.Errorf("copying .piko/%s into the embed payload: %w", tree, walkErr)
	}
	return copied, nil
}

// copyFileIfPresent copies one file from the .piko root into the payload when it exists.
//
// Takes relativePath (string) which is the file's path relative to the .piko root.
//
// Returns bool which is true when the file existed and was copied.
// Returns error when the copy fails for a reason other than absence.
func (e *DrivenEmbedEmitter) copyFileIfPresent(ctx context.Context, relativePath string) (bool, error) {
	if err := e.copyFile(ctx, relativePath); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("copying .piko/%s into the embed payload: %w", relativePath, err)
	}
	return true, nil
}

// copyFile streams one file from the source sandbox into the payload.
//
// The file is written at the same relative path, creating parent directories as needed.
// The write-side close error is folded into the return so a flush failure at close (for
// example ENOSPC) surfaces as a truncated blob rather than being discarded, and the copy
// is cancellable through ctx.
//
// Takes relativePath (string) which is the file's path relative to the .piko root.
//
// Returns error when the read, directory creation, write, or close fails.
func (e *DrivenEmbedEmitter) copyFile(ctx context.Context, relativePath string) (err error) {
	sourceFile, err := e.source.Open(relativePath)
	if err != nil {
		return err
	}
	defer func() { _ = sourceFile.Close() }()

	destinationPath := path.Join(embedPikoDirName, relativePath)
	if mkdirErr := e.destination.MkdirAll(path.Dir(destinationPath), embedDirPermission); mkdirErr != nil {
		return mkdirErr
	}

	destinationFile, err := e.destination.OpenFile(destinationPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, embedFilePermission)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := destinationFile.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	return copyStreamWithContext(ctx, destinationFile, sourceFile)
}

// GenerateEmbedGenFile returns the source of dist/embed_gen.go: a piko_embed-tagged file
// that embeds the compiled manifest and the copied payload, and registers both with the
// runtime in init(). The embed directives are slash-separated literals, never
// filepath-joined, since an embed pattern is always forward-slash regardless of platform.
//
// Returns string which is the complete Go source of the generated file.
func GenerateEmbedGenFile() string {
	var builder strings.Builder
	builder.WriteString(generator_dto.EmbedBuildConstraint)
	builder.WriteString(`// Code generated by Piko. DO NOT EDIT.
// This file embeds the runtime payload (registry snapshot, blobs, storage, and the compiled
// manifest) and registers it with the runtime, so a production build made with the
// piko_embed tag serves entirely from the binary.

package dist

import (
	"context"
	"embed"
	"io/fs"

	pikoruntime "piko.sh/piko/wdk/runtime"
)

//go:embed manifest.bin
var embeddedManifest []byte

//go:embed all:embed/piko
var embeddedPikoTree embed.FS

func init() {
	sub, err := fs.Sub(embeddedPikoTree, "embed/piko")
	if err != nil {
		panic("piko: embedded runtime payload is malformed: " + err.Error())
	}
	pikoruntime.RegisterEmbeddedRuntime(context.Background(), sub, embeddedManifest)
}
`)
	return builder.String()
}

// copyStreamWithContext copies source into destination in bounded chunks, checking ctx
// before each read so a large payload file copy is cancellable, returning any write or
// read error.
//
// Takes destination (io.Writer) which receives the copied bytes.
// Takes source (io.Reader) which supplies the bytes to copy.
//
// Returns error when ctx is cancelled, a read fails, or a write fails.
func copyStreamWithContext(ctx context.Context, destination io.Writer, source io.Reader) error {
	buffer := make([]byte, copyBufferSize)
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		readCount, readErr := source.Read(buffer)
		if readCount > 0 {
			if _, writeErr := destination.Write(buffer[:readCount]); writeErr != nil {
				return writeErr
			}
		}
		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				return nil
			}
			return readErr
		}
	}
}

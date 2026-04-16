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

package orchestrator_adapters

import (
	"context"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"errors"
	"fmt"
	"hash"
	"io"
	"path"
	"strings"
	"sync"
	"time"

	"go.opentelemetry.io/otel/trace"

	"piko.sh/piko/internal/capabilities/capabilities_domain"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/registry/registry_dto"
)

// payloadKeyArtefactID is the payload key for artefact identifiers.
const payloadKeyArtefactID = "artefactID"

var (
	_ orchestrator_domain.TaskExecutor = (*compilerExecutor)(nil)

	// compilerSHA256Pool reuses SHA-256 hash.Hash instances to reduce allocation pressure.
	compilerSHA256Pool = sync.Pool{New: func() any { return sha256.New() }}

	// compilerSHA384Pool reuses SHA-384 hash.Hash instances to reduce allocation pressure.
	compilerSHA384Pool = sync.Pool{New: func() any { return sha512.New384() }}
)

// ExecutorNameArtefactCompiler is the executor name for artefact compilation
// tasks.
const ExecutorNameArtefactCompiler = "artefact.compiler"

// readCounter wraps an io.Reader to track the number of bytes read.
type readCounter struct {
	io.Reader

	// Count is the total number of bytes read so far.
	Count int64
}

// Read reads up to len(p) bytes into p and updates the byte count.
//
// Takes p ([]byte) which is the buffer to read data into.
//
// Returns int which is the number of bytes read.
// Returns error when the underlying reader fails.
func (rc *readCounter) Read(p []byte) (int, error) {
	n, err := rc.Reader.Read(p)
	rc.Count += int64(n)
	return n, err
}

// compilerPayload holds the parsed message data for a compilation task.
type compilerPayload struct {
	// ArtefactID is the unique identifier of the artefact to compile.
	ArtefactID string

	// SourceVariantID is the unique identifier of the source variant to compile.
	SourceVariantID string

	// DesiredProfileName is the name of the target profile to compile.
	DesiredProfileName string

	// CapabilityToRun is the name of the capability to execute.
	CapabilityToRun string

	// CapabilityParams holds the parameters passed to the capability when it runs.
	CapabilityParams map[string]string

	// TaskID is the unique identifier for this compilation task.
	TaskID string
}

// compilerExecutor runs artefact compilation tasks.
// It implements the orchestrator_domain.TaskExecutor interface.
type compilerExecutor struct {
	// registryService provides access to artefact data and blob storage.
	registryService registry_domain.RegistryService

	// capabilityService runs capability actions on artefact streams.
	capabilityService capabilities_domain.CapabilityService
}

// Execute runs the artefact compilation task with the given payload.
//
// Takes payload (map[string]any) which contains the task configuration
// including artefact ID, source variant, and capability to run.
//
// Returns map[string]any which contains the compilation result with the new
// variant details.
// Returns error when the payload is invalid, task metadata cannot be fetched,
// source data is unavailable, capability execution fails, or variant
// registration fails.
func (e *compilerExecutor) Execute(ctx context.Context, payload map[string]any) (map[string]any, error) {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "compilerExecutor.Execute",
		logger_domain.String("executor", ExecutorNameArtefactCompiler),
	)
	defer span.End()

	startTime := time.Now()

	p, err := e.parsePayload(ctx, payload)
	if err != nil {
		l.ReportError(span, err, "Invalid payload for compilerExecutor")
		ExecutorCompilationErrorCount.Add(ctx, 1)
		return nil, fmt.Errorf("invalid payload for compilerExecutor: %w", err)
	}

	ctx = e.enrichContext(ctx, l, p)

	return e.runCompilation(ctx, span, startTime, p)
}

// enrichContext attaches task-specific fields to the logger and returns the
// enriched context.
//
// Takes ctx (context.Context) which carries the current context.
// Takes l (logger_domain.Logger) which is the current logger to enrich.
// Takes p (*compilerPayload) which provides the task fields to attach.
//
// Returns context.Context which carries the enriched logger.
func (*compilerExecutor) enrichContext(ctx context.Context, l logger_domain.Logger, p *compilerPayload) context.Context {
	l = l.With(
		logger_domain.String(payloadKeyArtefactID, p.ArtefactID),
		logger_domain.String("capability", p.CapabilityToRun),
		logger_domain.String("profile", p.DesiredProfileName),
		logger_domain.String("sourceVariantID", p.SourceVariantID),
		logger_domain.String("taskID", p.TaskID),
	)
	return logger_domain.WithLogger(ctx, l)
}

// runCompilation executes the full compilation pipeline: fetch metadata, read
// source, run capability, store output, and register the variant.
//
// Takes ctx (context.Context) which carries tracing and cancellation.
// Takes span (trace.Span) which is the parent tracing span.
// Takes startTime (time.Time) which marks when compilation started.
// Takes p (*compilerPayload) which holds the compilation task details.
//
// Returns map[string]any which contains the compilation result.
// Returns error when any pipeline step fails.
func (e *compilerExecutor) runCompilation(ctx context.Context, span trace.Span, startTime time.Time, p *compilerPayload) (map[string]any, error) {
	_, l := logger_domain.From(ctx, log)

	artefact, sourceVariant, desiredProfile, err := e.fetchTaskMetadata(ctx, p)
	if err != nil {
		l.ReportError(span, err, "Failed to get task metadata")
		ExecutorCompilationErrorCount.Add(ctx, 1)
		return nil, fmt.Errorf("fetching task metadata for artefact %q: %w", p.ArtefactID, err)
	}

	sourceStream, err := e.fetchSourceStream(ctx, sourceVariant)
	if err != nil {
		l.ReportError(span, err, "Failed to get source variant stream")
		ExecutorCompilationErrorCount.Add(ctx, 1)
		return nil, fmt.Errorf("failed to get source data for variant '%s': %w", sourceVariant.VariantID, err)
	}
	defer func() { _ = sourceStream.Close() }()

	outputStream, err := e.executeCapability(ctx, p, sourceStream)
	if err != nil {
		return nil, e.handleCapabilityError(ctx, span, l, p, err)
	}

	if closer, ok := outputStream.(io.ReadCloser); ok {
		defer func() { _ = closer.Close() }()
	}

	newVariant, err := e.storeAndCreateVariant(ctx, p, artefact, sourceVariant, desiredProfile, outputStream)
	if err != nil {
		return nil, wrapFatalIfNeeded(err, fmt.Errorf("storing and creating variant for artefact %q: %w", p.ArtefactID, err))
	}

	if newVariant.SizeBytes <= 0 {
		l.Trace("Skipping variant registration: capability produced empty output",
			logger_domain.String(payloadKeyArtefactID, artefact.ID),
			logger_domain.String("variantID", newVariant.VariantID),
		)
		return recordCompilationSuccess(ctx, startTime, &newVariant)
	}

	if err = e.registerVariant(ctx, artefact.ID, &newVariant); err != nil {
		l.ReportError(span, err, "Failed to add new variant to registry")
		return nil, fmt.Errorf("failed to add variant '%s': %w", newVariant.VariantID, err)
	}

	return recordCompilationSuccess(ctx, startTime, &newVariant)
}

// handleCapabilityError records the capability error metric and wraps the
// error, marking it as fatal when appropriate.
//
// Takes ctx (context.Context) which carries tracing and metrics context.
// Takes span (trace.Span) which is the parent tracing span.
// Takes l (logger_domain.Logger) which logs the failure.
// Takes p (*compilerPayload) which provides the capability name for the error
// message.
// Takes err (error) which is the original capability error.
//
// Returns error which is the wrapped (and optionally fatal) error.
func (*compilerExecutor) handleCapabilityError(ctx context.Context, span trace.Span, l logger_domain.Logger, p *compilerPayload, err error) error {
	l.ReportError(span, err, "Capability execution failed")
	ExecutorCompilationErrorCount.Add(ctx, 1)
	capErr := fmt.Errorf("capability '%s' failed: %w", p.CapabilityToRun, err)
	return wrapFatalIfNeeded(err, capErr)
}

// parsePayload parses and validates the compiler payload.
//
// Takes ctx (context.Context) which carries tracing spans and cancellation.
// Takes payload (map[string]any) which contains the raw payload data to parse.
//
// Returns *compilerPayload which is the parsed and validated payload.
// Returns error when parsing fails.
func (*compilerExecutor) parsePayload(
	ctx context.Context,
	payload map[string]any,
) (*compilerPayload, error) {
	ctx, l := logger_domain.From(ctx, log)
	l.Trace("Parsing compiler payload")

	var p *compilerPayload
	err := l.RunInSpan(ctx, "ParseCompilerPayload", func(_ context.Context, _ logger_domain.Logger) error {
		var parseErr error
		p, parseErr = parseCompilerPayload(payload)
		return parseErr
	})

	return p, err
}

// fetchTaskMetadata gets the artefact, source variant, and desired profile
// for a task.
//
// Takes ctx (context.Context) which carries tracing spans and cancellation.
// Takes p (*compilerPayload) which holds the task details to look up.
//
// Returns *registry_dto.ArtefactMeta which is the artefact metadata.
// Returns *registry_dto.Variant which is the source variant.
// Returns *registry_dto.DesiredProfile which is the target profile.
// Returns error when the metadata lookup fails.
func (e *compilerExecutor) fetchTaskMetadata(
	ctx context.Context,
	p *compilerPayload,
) (*registry_dto.ArtefactMeta, *registry_dto.Variant, *registry_dto.DesiredProfile, error) {
	ctx, l := logger_domain.From(ctx, log)
	l.Trace("Fetching artefact metadata and source stream")

	var artefact *registry_dto.ArtefactMeta
	var sourceVariant *registry_dto.Variant
	var desiredProfile *registry_dto.DesiredProfile

	err := l.RunInSpan(ctx, "GetTaskMetadata", func(ctx context.Context, _ logger_domain.Logger) error {
		var metaErr error
		artefact, sourceVariant, desiredProfile, metaErr = e.getTaskMetadata(ctx, p)
		return metaErr
	})

	return artefact, sourceVariant, desiredProfile, err
}

// fetchSourceStream retrieves the source data stream for a variant.
//
// Takes ctx (context.Context) which carries tracing spans and cancellation.
// Takes sourceVariant (*registry_dto.Variant) which identifies the variant to
// fetch.
//
// Returns io.ReadCloser which provides the source data stream.
// Returns error when the variant data cannot be retrieved.
func (e *compilerExecutor) fetchSourceStream(
	ctx context.Context,
	sourceVariant *registry_dto.Variant,
) (io.ReadCloser, error) {
	ctx, l := logger_domain.From(ctx, log)
	var sourceStream io.ReadCloser

	err := l.RunInSpan(ctx, "GetVariantData", func(ctx context.Context, _ logger_domain.Logger) error {
		var streamErr error
		sourceStream, streamErr = e.registryService.GetVariantData(ctx, sourceVariant)
		return streamErr
	})

	return sourceStream, err
}

// executeCapability runs the capability transformation on the source stream.
//
// Takes ctx (context.Context) which carries tracing spans and cancellation.
// Takes p (*compilerPayload) which contains the capability and its parameters.
// Takes sourceStream (io.Reader) which provides the input data to transform.
//
// Returns io.Reader which provides the transformed output stream.
// Returns error when the capability execution fails.
func (e *compilerExecutor) executeCapability(
	ctx context.Context,
	p *compilerPayload,
	sourceStream io.Reader,
) (io.Reader, error) {
	ctx, l := logger_domain.From(ctx, log)
	l.Trace("Executing capability in streaming mode",
		logger_domain.Int("paramCount", len(p.CapabilityParams)))

	var outputStream io.Reader
	capabilityStartTime := time.Now()

	err := l.RunInSpan(ctx, "ExecuteCapability", func(ctx context.Context, _ logger_domain.Logger) error {
		var execErr error
		outputStream, execErr = e.capabilityService.Execute(ctx, p.CapabilityToRun, sourceStream, p.CapabilityParams)
		return execErr
	})

	ExecutorCapabilityExecutionDuration.Record(ctx, float64(time.Since(capabilityStartTime).Milliseconds()))

	return outputStream, err
}

// storeAndCreateVariant stores the output stream and creates a variant record.
//
// Takes ctx (context.Context) which carries tracing spans and cancellation.
// Takes p (*compilerPayload) which contains the compilation request details.
// Takes artefact (*registry_dto.ArtefactMeta) which provides artefact metadata.
// Takes sourceVariant (*registry_dto.Variant) which is the parent variant used
// as input to the capability.
// Takes desiredProfile (*registry_dto.DesiredProfile) which specifies the
// target profile and resulting tags.
// Takes outputStream (io.Reader) which provides the compiled output to store.
//
// Returns registry_dto.Variant which contains the created variant record with
// storage details and metadata.
// Returns error when the blob store cannot be retrieved or storage fails.
func (e *compilerExecutor) storeAndCreateVariant(
	ctx context.Context,
	p *compilerPayload,
	artefact *registry_dto.ArtefactMeta,
	sourceVariant *registry_dto.Variant,
	desiredProfile *registry_dto.DesiredProfile,
	outputStream io.Reader,
) (registry_dto.Variant, error) {
	ctx, l := logger_domain.From(ctx, log)
	storageBackendID, _ := desiredProfile.ResultingTags.GetByName("storageBackendId")
	blobStore, err := e.getBlobStore(ctx, storageBackendID)
	if err != nil {
		l.ReportError(nil, err, "Failed to get blob store")
		return registry_dto.Variant{}, fmt.Errorf("getting blob store for backend %q: %w", storageBackendID, err)
	}

	result, err := e.streamToStorage(ctx, p, artefact, desiredProfile, blobStore, outputStream)
	if err != nil {
		return registry_dto.Variant{}, fmt.Errorf("streaming output to storage: %w", err)
	}

	if desiredProfile.ResultingTags.Get(registry_dto.TagContentEncoding) != "" && sourceVariant.SRIHash != "" {
		result.sriHash = sourceVariant.SRIHash
	}

	var tags registry_dto.Tags
	for k, v := range desiredProfile.ResultingTags.All() {
		tags.SetByName(k, v)
	}
	tags.Set(registry_dto.TagEtag, fmt.Sprintf(`"%x"`, result.hash))

	mimeType, _ := desiredProfile.ResultingTags.GetByName("mimeType")
	return registry_dto.Variant{
		VariantID:        p.DesiredProfileName,
		StorageBackendID: storageBackendID,
		StorageKey:       result.storageKey,
		MimeType:         mimeType,
		SizeBytes:        result.size,
		ContentHash:      fmt.Sprintf("%x", result.hash),
		SRIHash:          result.sriHash,
		MetadataTags:     tags,
		CreatedAt:        time.Now().UTC(),
		Status:           registry_dto.VariantStatusReady,
		Chunks:           nil,
	}, nil
}

// getBlobStore gets the blob store for a given storage backend.
//
// Takes ctx (context.Context) which carries tracing spans and cancellation.
// Takes storageBackendID (string) which identifies the storage backend.
//
// Returns registry_domain.BlobStore which gives access to blob storage.
// Returns error when the blob store cannot be found.
func (e *compilerExecutor) getBlobStore(
	ctx context.Context,
	storageBackendID string,
) (registry_domain.BlobStore, error) {
	ctx, l := logger_domain.From(ctx, log)
	l.Trace("Streaming capability output to temporary storage while hashing")

	var blobStore registry_domain.BlobStore
	err := l.RunInSpan(ctx, "GetBlobStore", func(_ context.Context, _ logger_domain.Logger) error {
		var blobErr error
		blobStore, blobErr = e.registryService.GetBlobStore(storageBackendID)
		return blobErr
	})

	return blobStore, err
}

// streamToStorageResult holds the output of streaming processed content to
// blob storage.
type streamToStorageResult struct {
	// sriHash is the SRI integrity hash in "sha384-<base64>" format.
	sriHash string

	// storageKey is the final blob storage path for the written content.
	storageKey string

	// hash is the raw SHA-256 digest of the stored content.
	hash []byte

	// size is the number of bytes written to storage.
	size int64
}

// acquireCompilerHashers retrieves and resets SHA-256 and SHA-384 hashers from
// their respective pools.
//
// Returns hash.Hash which is the reset SHA-256 hasher.
// Returns hash.Hash which is the reset SHA-384 hasher.
// Returns error when a pool returns an unexpected type.
func acquireCompilerHashers() (sha256Hasher hash.Hash, sha384Hasher hash.Hash, err error) {
	var ok bool
	sha256Hasher, ok = compilerSHA256Pool.Get().(hash.Hash)
	if !ok {
		return nil, nil, errors.New("compilerSHA256Pool returned unexpected type")
	}
	sha384Hasher, ok = compilerSHA384Pool.Get().(hash.Hash)
	if !ok {
		compilerSHA256Pool.Put(sha256Hasher)
		return nil, nil, errors.New("compilerSHA384Pool returned unexpected type")
	}
	sha256Hasher.Reset()
	sha384Hasher.Reset()
	return sha256Hasher, sha384Hasher, nil
}

// streamToStorage writes the output stream to blob storage and returns the
// hash, size, and final storage key.
//
// Takes ctx (context.Context) which carries tracing spans and cancellation.
// Takes p (*compilerPayload) which contains the task ID for temporary storage.
// Takes artefact (*registry_dto.ArtefactMeta) which provides the source path
// for building the final key.
// Takes desiredProfile (*registry_dto.DesiredProfile) which specifies the
// output file extension.
// Takes blobStore (registry_domain.BlobStore) which is the storage backend.
// Takes outputStream (io.Reader) which provides the data to store.
//
// Returns streamToStorageResult which holds the hash, SRI hash, size, and
// storage key of the stored content.
// Returns error when the blob cannot be written or renamed.
func (*compilerExecutor) streamToStorage(
	ctx context.Context,
	p *compilerPayload,
	artefact *registry_dto.ArtefactMeta,
	desiredProfile *registry_dto.DesiredProfile,
	blobStore registry_domain.BlobStore,
	outputStream io.Reader,
) (streamToStorageResult, error) {
	ctx, l := logger_domain.From(ctx, log)
	tempStorageKey := path.Join("tmp", p.TaskID)

	sha256Hasher, sha384Hasher, err := acquireCompilerHashers()
	if err != nil {
		return streamToStorageResult{}, err
	}

	hashingReader := io.TeeReader(outputStream, io.MultiWriter(sha256Hasher, sha384Hasher))
	counter := &readCounter{Reader: hashingReader, Count: 0}

	err = l.RunInSpan(ctx, "PutBlob", func(ctx context.Context, _ logger_domain.Logger) error {
		return blobStore.Put(ctx, tempStorageKey, counter)
	})

	if err != nil {
		compilerSHA256Pool.Put(sha256Hasher)
		compilerSHA384Pool.Put(sha384Hasher)
		_ = blobStore.Delete(ctx, tempStorageKey)
		l.ReportError(nil, err, "Failed to write temporary blob")
		return streamToStorageResult{}, fmt.Errorf("failed to write temporary blob: %w", err)
	}

	finalHash := sha256Hasher.Sum(nil)
	sriHash := "sha384-" + base64.StdEncoding.EncodeToString(sha384Hasher.Sum(nil))
	compilerSHA256Pool.Put(sha256Hasher)
	compilerSHA384Pool.Put(sha384Hasher)

	finalSize := counter.Count
	l.Trace("Finalised stream processing",
		logger_domain.Int64("size", finalSize),
		logger_domain.String("hash", fmt.Sprintf("%x", finalHash[:8])))

	outputExtension, _ := desiredProfile.ResultingTags.GetByName("fileExtension")
	shortHash := fmt.Sprintf("%x", finalHash[:8])
	basePath := strings.TrimSuffix(artefact.SourcePath, path.Ext(artefact.SourcePath))
	finalStorageKey := path.Join("generated", fmt.Sprintf("%s_%s%s", basePath, shortHash, outputExtension))

	l.Trace("Renaming temporary blob to final destination",
		logger_domain.String("from", tempStorageKey),
		logger_domain.String("to", finalStorageKey))

	err = l.RunInSpan(ctx, "RenameBlob", func(ctx context.Context, _ logger_domain.Logger) error {
		return blobStore.Rename(ctx, tempStorageKey, finalStorageKey)
	})

	if err != nil {
		_ = blobStore.Delete(ctx, tempStorageKey)
		l.ReportError(nil, err, "Failed to rename blob")
		return streamToStorageResult{}, fmt.Errorf("failed to rename blob: %w", err)
	}

	return streamToStorageResult{
		hash:       finalHash,
		sriHash:    sriHash,
		size:       finalSize,
		storageKey: finalStorageKey,
	}, nil
}

// registerVariant adds a new variant to an artefact in the registry.
//
// Takes ctx (context.Context) which carries tracing spans and cancellation.
// Takes artefactID (string) which identifies the target artefact.
// Takes newVariant (registry_dto.Variant) which specifies the variant to add.
//
// Returns error when the registry service fails to add the variant.
func (e *compilerExecutor) registerVariant(
	ctx context.Context,
	artefactID string,
	newVariant *registry_dto.Variant,
) error {
	ctx, l := logger_domain.From(ctx, log)
	l.Trace("Adding new variant to artefact",
		logger_domain.String("variantID", newVariant.VariantID),
		logger_domain.Int64("sizeBytes", newVariant.SizeBytes))

	return l.RunInSpan(ctx, "AddVariant", func(ctx context.Context, _ logger_domain.Logger) error {
		_, addErr := e.registryService.AddVariant(ctx, artefactID, newVariant)
		return addErr
	})
}

// getTaskMetadata fetches the artefact, source variant, and desired profile
// for a compilation task.
//
// Takes p (*compilerPayload) which holds the artefact ID, source variant ID,
// and desired profile name to look up.
//
// Returns *registry_dto.ArtefactMeta which is the full artefact metadata.
// Returns *registry_dto.Variant which is the source variant to compile from.
// Returns *registry_dto.DesiredProfile which is the target profile for output.
// Returns error when the artefact cannot be found, the source variant does not
// exist, or the desired profile is not set.
func (e *compilerExecutor) getTaskMetadata(ctx context.Context, p *compilerPayload) (*registry_dto.ArtefactMeta, *registry_dto.Variant, *registry_dto.DesiredProfile, error) {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "compilerExecutor.getTaskMetadata",
		logger_domain.String(payloadKeyArtefactID, p.ArtefactID),
		logger_domain.String("sourceVariantID", p.SourceVariantID),
		logger_domain.String("desiredProfileName", p.DesiredProfileName),
	)
	defer span.End()

	var artefact *registry_dto.ArtefactMeta
	var err error

	err = l.RunInSpan(ctx, "GetArtefact", func(ctx context.Context, _ logger_domain.Logger) error {
		var getErr error
		artefact, getErr = e.registryService.GetArtefact(ctx, p.ArtefactID)
		return getErr
	})

	if err != nil {
		l.ReportError(span, err, "Failed to get artefact")
		return nil, nil, nil, fmt.Errorf("failed to get artefact '%s': %w", p.ArtefactID, err)
	}

	sourceVariant := findVariantByID(artefact.ActualVariants, p.SourceVariantID)
	if sourceVariant == nil {
		err = fmt.Errorf("source variant '%s' not found for artefact '%s'", p.SourceVariantID, p.ArtefactID)
		l.ReportError(span, err, "Source variant not found")
		return nil, nil, nil, err
	}

	desiredProfile, ok := artefact.GetProfile(p.DesiredProfileName)
	if !ok {
		err = fmt.Errorf("desired profile '%s' not found in artefact '%s'", p.DesiredProfileName, p.ArtefactID)
		l.ReportError(span, err, "Desired profile not found")
		return nil, nil, nil, err
	}

	l.Trace("Metadata retrieved successfully")
	return artefact, sourceVariant, &desiredProfile, nil
}

// NewCompilerExecutor creates a new compilerExecutor with the given registry
// and capability services.
//
// Takes registry (registry_domain.RegistryService) which provides access to
// the registry for compiler lookup.
// Takes capabilities (capabilities_domain.CapabilityService) which provides
// capability checking for compilation tasks.
//
// Returns orchestrator_domain.TaskExecutor which executes compilation tasks.
func NewCompilerExecutor(
	registry registry_domain.RegistryService,
	capabilities capabilities_domain.CapabilityService,
) orchestrator_domain.TaskExecutor {
	return &compilerExecutor{
		registryService:   registry,
		capabilityService: capabilities,
	}
}

// wrapFatalIfNeeded wraps the given error as a fatal orchestrator error when
// the cause is a fatal capability error.
//
// Takes cause (error) which is the original error to check for fatality.
// Takes wrapped (error) which is the contextual error to wrap.
//
// Returns error which is either a fatal orchestrator error or the wrapped
// error unchanged.
func wrapFatalIfNeeded(cause, wrapped error) error {
	if capabilities_domain.IsFatalError(cause) {
		return orchestrator_domain.NewFatalError(wrapped)
	}
	return wrapped
}

// recordCompilationSuccess logs a successful compilation and returns the
// result.
//
// Takes ctx (context.Context) which carries tracing spans and cancellation.
// Takes startTime (time.Time) which marks when compilation started.
// Takes newVariant (registry_dto.Variant) which holds the compiled output.
//
// Returns map[string]any which contains the success status and variant details.
// Returns error when recording fails.
func recordCompilationSuccess(
	ctx context.Context,
	startTime time.Time,
	newVariant *registry_dto.Variant,
) (map[string]any, error) {
	ctx, l := logger_domain.From(ctx, log)
	duration := time.Since(startTime)
	ExecutorCompilationDuration.Record(ctx, float64(duration.Milliseconds()))

	l.Trace("Compilation successful",
		logger_domain.Int64("durationMs", duration.Milliseconds()),
		logger_domain.Int64("outputSizeBytes", newVariant.SizeBytes),
		logger_domain.String("status", "SUCCESS"))

	return map[string]any{
		"status":     "SUCCESS",
		"variantId":  newVariant.VariantID,
		"storageKey": newVariant.StorageKey,
		"sizeBytes":  newVariant.SizeBytes,
	}, nil
}

// parseCompilerPayload extracts and validates compiler task fields from a map.
//
// Takes payload (map[string]any) which contains the raw compiler task data.
//
// Returns *compilerPayload which holds the validated compiler task fields.
// Returns error when a required field is missing or has the wrong type.
func parseCompilerPayload(payload map[string]any) (*compilerPayload, error) {
	var p compilerPayload
	var err error

	p.ArtefactID, err = getString(payload, payloadKeyArtefactID)
	if err != nil {
		return nil, fmt.Errorf("parsing artefactID: %w", err)
	}
	p.SourceVariantID, err = getString(payload, "sourceVariantID")
	if err != nil {
		return nil, fmt.Errorf("parsing sourceVariantID: %w", err)
	}
	p.DesiredProfileName, err = getString(payload, "desiredProfileName")
	if err != nil {
		return nil, fmt.Errorf("parsing desiredProfileName: %w", err)
	}
	p.CapabilityToRun, err = getString(payload, "capabilityToRun")
	if err != nil {
		return nil, fmt.Errorf("parsing capabilityToRun: %w", err)
	}
	p.TaskID, err = getString(payload, "taskID")
	if err != nil {
		return nil, fmt.Errorf("parsing taskID: %w", err)
	}

	p.CapabilityParams, err = parseCapabilityParams(payload)
	if err != nil {
		return nil, fmt.Errorf("parsing capability parameters: %w", err)
	}

	return &p, nil
}

// parseCapabilityParams extracts and converts capability parameters from a
// payload map.
//
// When the payload lacks a capabilityParams key or the value is nil, returns
// an empty map. Non-string values within a map[string]any are skipped without
// warning.
//
// Takes payload (map[string]any) which contains the raw payload to parse.
//
// Returns map[string]string which contains the extracted string parameters.
// Returns error when capabilityParams has an invalid type.
func parseCapabilityParams(payload map[string]any) (map[string]string, error) {
	rawParams, exists := payload["capabilityParams"]
	if !exists || rawParams == nil {
		return make(map[string]string), nil
	}

	if params, ok := rawParams.(map[string]string); ok {
		return params, nil
	}

	interfaceMap, ok := rawParams.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("payload 'capabilityParams' has invalid type; expected map[string]string or map[string]any, got %T with value %+v", rawParams, rawParams)
	}

	result := make(map[string]string)
	for k, v := range interfaceMap {
		if vString, isString := v.(string); isString {
			result[k] = vString
		}
	}
	return result, nil
}

// getString retrieves a string value from a map by its key.
//
// Takes payload (map[string]any) which holds the data to search.
// Takes key (string) which is the map key to look up.
//
// Returns string which is the value found at the given key.
// Returns error when the key is missing, the value is not a string, or the
// value is empty.
func getString(payload map[string]any, key string) (string, error) {
	value, ok := payload[key]
	if !ok {
		return "", fmt.Errorf("payload missing required key '%s'", key)
	}
	stringValue, ok := value.(string)
	if !ok || stringValue == "" {
		return "", fmt.Errorf("payload key '%s' is invalid or empty", key)
	}
	return stringValue, nil
}

// findVariantByID searches for a variant with the given ID in a slice.
//
// Takes variants ([]registry_dto.Variant) which is the slice to search.
// Takes id (string) which is the variant ID to find.
//
// Returns *registry_dto.Variant which points to the matching variant, or nil
// if not found.
func findVariantByID(variants []registry_dto.Variant, id string) *registry_dto.Variant {
	for i := range variants {
		if variants[i].VariantID == id {
			return &variants[i]
		}
	}
	return nil
}

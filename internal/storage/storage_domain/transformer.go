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

package storage_domain

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"slices"

	"piko.sh/piko/internal/json"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/provider/provider_domain"
	"piko.sh/piko/internal/storage/storage_dto"
)

const (
	// metadataKeyTransformers is the metadata key used to store the transformer
	// chain applied to an object. The value is a JSON array of transformer names
	// in the order they were applied.
	metadataKeyTransformers = "x-piko-transformers"
)

var _ StorageProviderPort = (*TransformerWrapper)(nil)

// StreamTransformerPort defines the interface for stream transformers.
// Transformers wrap an io.Reader to apply on-the-fly transformations like
// compression or encryption.
type StreamTransformerPort interface {
	// Name returns the unique name of this transformer (e.g. "zstd",
	// "aes-256-gcm").
	Name() string

	// Type returns the transformer's category, such as compression, encryption,
	// or custom.
	//
	// Returns storage_dto.TransformerType which indicates the kind of transformer.
	Type() storage_dto.TransformerType

	// Priority returns the execution order for this filter. Lower values run first
	// during write operations.
	//
	// Returns int which indicates priority. Recommended ranges: 100-199 for
	// compression, 200-299 for encryption, 300+ for custom filters.
	Priority() int

	// Transform wraps a reader to apply the forward transformation such as
	// compression or encryption. It is used during upload operations.
	//
	// Takes input (io.Reader) which provides the data to transform.
	// Takes options (any) which is sourced from
	// TransformConfig.TransformerOptions[Name()].
	//
	// Returns io.Reader which provides the transformed data stream.
	// Returns error when the transformation cannot be applied.
	Transform(ctx context.Context, input io.Reader, options any) (io.Reader, error)

	// Reverse wraps a reader to apply the reverse transformation, such as
	// decompress or decrypt. It is used during download operations.
	//
	// Takes input (io.Reader) which provides the data to transform.
	// Takes options (any) which is sourced from
	// TransformConfig.TransformerOptions[Name()].
	//
	// Returns io.Reader which provides the reverse-transformed data.
	// Returns error when the transformation cannot be applied.
	Reverse(ctx context.Context, input io.Reader, options any) (io.Reader, error)
}

// TransformerRegistry holds a set of named transformers for stream processing.
type TransformerRegistry struct {
	// transformers maps transformer names to their implementations.
	transformers map[string]StreamTransformerPort
}

// NewTransformerRegistry creates a new, empty transformer registry.
//
// Returns *TransformerRegistry which is ready to have transformers added.
func NewTransformerRegistry() *TransformerRegistry {
	return &TransformerRegistry{
		transformers: make(map[string]StreamTransformerPort),
	}
}

// Register adds a new transformer to the registry.
//
// Takes transformer (StreamTransformerPort) which is the transformer to add.
//
// Returns error when transformer is nil, has an empty name, or a transformer
// with the same name is already registered.
func (r *TransformerRegistry) Register(transformer StreamTransformerPort) error {
	if transformer == nil {
		return errTransformerNil
	}
	name := transformer.Name()
	if name == "" {
		return errTransformerNameEmpty
	}
	if _, exists := r.transformers[name]; exists {
		return fmt.Errorf("transformer '%s' is already registered", name)
	}
	r.transformers[name] = transformer
	return nil
}

// Get retrieves a transformer by its registered name.
//
// Takes name (string) which is the registered transformer name to look up.
//
// Returns StreamTransformerPort which is the resolved transformer.
// Returns error when no transformer with the given name is registered.
func (r *TransformerRegistry) Get(name string) (StreamTransformerPort, error) {
	transformer, ok := r.transformers[name]
	if !ok {
		return nil, fmt.Errorf("transformer '%s' not found", name)
	}
	return transformer, nil
}

// Has checks if a transformer with the given name is registered.
//
// Takes name (string) which is the transformer name to look up.
//
// Returns bool which is true if the transformer exists, false otherwise.
func (r *TransformerRegistry) Has(name string) bool {
	_, ok := r.transformers[name]
	return ok
}

// GetNames returns a sorted list of all registered transformer names.
//
// Returns []string which contains the names in alphabetical order.
func (r *TransformerRegistry) GetNames() []string {
	names := make([]string, 0, len(r.transformers))
	for name := range r.transformers {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

// TransformerChain represents an ordered sequence of transformers to be applied
// to a stream.
type TransformerChain struct {
	// config holds transformer options keyed by transformer name.
	config *storage_dto.TransformConfig

	// transformers holds the ordered list of transformers to apply.
	transformers []StreamTransformerPort
}

// NewTransformerChain creates and sorts a new transformer chain based on a
// configuration.
//
// When config is nil, returns an empty but valid chain.
//
// Takes registry (*TransformerRegistry) which provides available transformers.
// Takes config (*storage_dto.TransformConfig) which specifies which
// transformers to enable.
//
// Returns *TransformerChain which contains the sorted transformers ready for
// use.
// Returns error when the registry is nil or a transformer cannot be resolved.
func NewTransformerChain(registry *TransformerRegistry, config *storage_dto.TransformConfig) (*TransformerChain, error) {
	if registry == nil {
		return nil, errors.New("transformer registry cannot be nil")
	}
	if config == nil {
		return &TransformerChain{}, nil
	}

	chain := &TransformerChain{
		transformers: make([]StreamTransformerPort, 0, len(config.EnabledTransformers)),
		config:       config,
	}

	for _, name := range config.EnabledTransformers {
		transformer, err := registry.Get(name)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve transformer '%s': %w", name, err)
		}
		chain.transformers = append(chain.transformers, transformer)
	}

	slices.SortFunc(chain.transformers, func(a, b StreamTransformerPort) int {
		return cmp.Compare(a.Priority(), b.Priority())
	})

	return chain, nil
}

// IsEmpty returns true if the chain contains no transformers.
//
// Returns bool which indicates whether the chain is empty.
func (c *TransformerChain) IsEmpty() bool {
	return len(c.transformers) == 0
}

// Transform applies all transformers in forward, priority order for uploads.
// Data flows from original through each transformer in sequence to storage.
//
// Takes input (io.Reader) which provides the original data to transform.
//
// Returns io.Reader which provides the fully transformed data stream.
// Returns error when any transformer in the chain fails.
func (c *TransformerChain) Transform(ctx context.Context, input io.Reader) (io.Reader, error) {
	if c.IsEmpty() {
		return input, nil
	}

	ctx, l := logger_domain.From(ctx, log)
	current := input
	for _, transformer := range c.transformers {
		options := c.config.TransformerOptions[transformer.Name()]
		transformed, err := transformer.Transform(ctx, current, options)
		if err != nil {
			return nil, fmt.Errorf("transformer '%s' failed: %w", transformer.Name(), err)
		}
		current = transformed
		l.Trace("Applied transformer", logger_domain.String(logFieldTransformer, transformer.Name()), logger_domain.Int(logFieldPriority, transformer.Priority()))
	}
	return current, nil
}

// Reverse applies all transformers in reverse priority order for downloads.
// Data flows from storage through each transformer in descending priority,
// ending with the original format.
//
// Takes input (io.Reader) which provides the data to reverse-transform.
//
// Returns io.Reader which yields the fully reverse-transformed data stream.
// Returns error when any transformer in the chain fails to reverse.
func (c *TransformerChain) Reverse(ctx context.Context, input io.Reader) (io.Reader, error) {
	if c.IsEmpty() {
		return input, nil
	}

	ctx, l := logger_domain.From(ctx, log)
	current := input
	for i := len(c.transformers) - 1; i >= 0; i-- {
		transformer := c.transformers[i]
		options := c.config.TransformerOptions[transformer.Name()]
		reversed, err := transformer.Reverse(ctx, current, options)
		if err != nil {
			return nil, fmt.Errorf("transformer '%s' reverse failed: %w", transformer.Name(), err)
		}
		current = reversed
		l.Trace("Reversed transformer", logger_domain.String(logFieldTransformer, transformer.Name()), logger_domain.Int(logFieldPriority, transformer.Priority()))
	}
	return current, nil
}

// TransformerWrapper decorates a StorageProviderPort to add transparent,
// on-the-fly data transformation such as compression or encryption.
// It implements StorageProviderPort.
type TransformerWrapper struct {
	// provider is the underlying storage provider that handles actual data
	// operations.
	provider StorageProviderPort

	// registry holds the available transformers used to create chains.
	registry *TransformerRegistry

	// defaultConfig is the fallback transform configuration when none is provided.
	defaultConfig *storage_dto.TransformConfig

	// providerName identifies the storage provider for log messages.
	providerName string
}

// NewTransformerWrapper creates a new transformer wrapper for a given provider.
//
// Takes provider (StorageProviderPort) which handles storage operations.
// Takes registry (*TransformerRegistry) which contains available transformers.
// Takes defaultConfig (*storage_dto.TransformConfig) which specifies default
// transformation settings.
// Takes providerName (string) which identifies the storage provider.
//
// Returns *TransformerWrapper which wraps the provider with transformation
// capabilities.
func NewTransformerWrapper(provider StorageProviderPort, registry *TransformerRegistry, defaultConfig *storage_dto.TransformConfig, providerName string) *TransformerWrapper {
	return &TransformerWrapper{
		provider:      provider,
		registry:      registry,
		defaultConfig: defaultConfig,
		providerName:  providerName,
	}
}

// Unwrap returns the underlying storage provider.
//
// Returns StorageProviderPort which is the wrapped provider.
func (tw *TransformerWrapper) Unwrap() StorageProviderPort {
	return tw.provider
}

// GetProviderType forwards the provider type query through the wrapper chain.
//
// Returns string which is the type from the inner provider, or "unknown" if
// no layer implements ProviderMetadata.
func (tw *TransformerWrapper) GetProviderType() string {
	if meta, ok := tw.provider.(provider_domain.ProviderMetadata); ok {
		return meta.GetProviderType()
	}
	return "unknown"
}

// GetProviderMetadata forwards the metadata query through the wrapper chain.
//
// Returns map[string]any which is the metadata from the inner provider, or nil
// if no layer implements ProviderMetadata.
func (tw *TransformerWrapper) GetProviderMetadata() map[string]any {
	if meta, ok := tw.provider.(provider_domain.ProviderMetadata); ok {
		return meta.GetProviderMetadata()
	}
	return nil
}

// Put intercepts the Put operation, applying the configured transformations
// to the data stream before passing it to the underlying storage provider.
// It automatically stores the transformer chain in object metadata for
// automatic reversal on Get.
//
// Takes params (*storage_dto.PutParams) which specifies the object key, reader,
// and optional transform configuration.
//
// Returns error when creating the transformer chain fails, transformation
// fails, or the underlying provider returns an error.
func (tw *TransformerWrapper) Put(ctx context.Context, params *storage_dto.PutParams) error {
	ctx, l := logger_domain.From(ctx, log)
	config := tw.resolveConfig(params.TransformConfig)
	if config == nil || len(config.EnabledTransformers) == 0 {
		return tw.provider.Put(ctx, params)
	}

	chain, err := NewTransformerChain(tw.registry, config)
	if err != nil {
		return fmt.Errorf("failed to create transformer chain for Put: %w", err)
	}
	if chain.IsEmpty() {
		return tw.provider.Put(ctx, params)
	}

	l.Trace("Applying transformations before upload",
		logger_domain.String(logFieldProvider, tw.providerName),
		logger_domain.String(logFieldKey, params.Key),
		logger_domain.Strings(LogFieldTransformers, config.EnabledTransformers))

	transformedReader, err := chain.Transform(ctx, params.Reader)
	if err != nil {
		return fmt.Errorf("transformation failed for key '%s': %w", params.Key, err)
	}

	modifiedParams := *params
	modifiedParams.Reader = transformedReader
	modifiedParams.Size = -1

	if modifiedParams.Metadata == nil {
		modifiedParams.Metadata = make(map[string]string)
	}

	transformerNames := make([]string, len(chain.transformers))
	for i, t := range chain.transformers {
		transformerNames[i] = t.Name()
	}
	metadataJSON, err := json.Marshal(transformerNames)
	if err != nil {
		return fmt.Errorf("failed to encode transformer metadata: %w", err)
	}
	modifiedParams.Metadata[metadataKeyTransformers] = string(metadataJSON)

	l.Trace("Storing transformer metadata",
		logger_domain.String(logFieldKey, params.Key),
		logger_domain.String("metadata", string(metadataJSON)))

	return tw.provider.Put(ctx, &modifiedParams)
}

// Get intercepts the Get operation, first retrieving the raw data from the
// underlying provider, then applying the reverse transformations before
// returning the stream to the caller. It automatically detects and applies
// transformers from object metadata if available.
//
// Takes params (storage_dto.GetParams) which specifies what object to retrieve.
//
// Returns io.ReadCloser which provides the transformed data stream.
// Returns error when the underlying provider fails or transformation fails.
func (tw *TransformerWrapper) Get(ctx context.Context, params storage_dto.GetParams) (io.ReadCloser, error) {
	config := tw.retrieveTransformerConfig(ctx, params)
	if config == nil {
		config = tw.resolveConfig(params.TransformConfig)
	}

	rawReader, err := tw.provider.Get(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("getting object %q through transformer wrapper: %w", params.Key, err)
	}

	if config == nil || len(config.EnabledTransformers) == 0 {
		return rawReader, nil
	}

	return tw.applyReverseTransformations(ctx, params, rawReader, config)
}

// transformedReadCloser wraps a transformed reader and its cleanup resources.
// It implements io.ReadCloser, ensuring the original reader and any background
// goroutines are properly released when closed.
type transformedReadCloser struct {
	// reader holds the transformed content for reading.
	reader io.Reader

	// closer is the underlying io.Closer to call when closing.
	closer io.Closer

	// cancel is called on Close to release context resources; may be nil.
	cancel context.CancelCauseFunc
}

// Read reads data from the transformed reader.
//
// Takes p ([]byte) which is the buffer to read data into.
//
// Returns n (int) which is the number of bytes read.
// Returns err (error) when the underlying reader returns an error.
func (r *transformedReadCloser) Read(p []byte) (n int, err error) {
	return r.reader.Read(p)
}

// Close closes the transformed reader and releases underlying resources.
//
// Returns error when the underlying closer fails.
func (r *transformedReadCloser) Close() error {
	if r.cancel != nil {
		r.cancel(errors.New("transformation reader closed"))
	}
	return r.closer.Close()
}

// Stat passes the call directly to the underlying provider without
// transformation.
//
// Takes params (storage_dto.GetParams) which specifies the object to query.
//
// Returns *ObjectInfo which contains metadata about the requested object.
// Returns error when the underlying provider fails.
func (tw *TransformerWrapper) Stat(ctx context.Context, params storage_dto.GetParams) (*ObjectInfo, error) {
	info, err := tw.provider.Stat(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("statting object %q through transformer wrapper: %w", params.Key, err)
	}
	return info, nil
}

// Copy passes the call directly to the underlying provider without
// transformation.
//
// Takes srcRepo (string) which identifies the source repository.
// Takes srcKey (string) which identifies the source object.
// Takes dstKey (string) which specifies the destination object key.
//
// Returns error when the underlying provider fails to copy.
func (tw *TransformerWrapper) Copy(ctx context.Context, srcRepo string, srcKey, dstKey string) error {
	if err := tw.provider.Copy(ctx, srcRepo, srcKey, dstKey); err != nil {
		return fmt.Errorf("copying object %q to %q through transformer wrapper: %w", srcKey, dstKey, err)
	}
	return nil
}

// CopyToAnotherRepository passes the call directly to the underlying provider.
//
// Takes srcRepo (string) which specifies the source repository name.
// Takes srcKey (string) which specifies the key of the item to copy.
// Takes dstRepo (string) which specifies the destination repository name.
// Takes dstKey (string) which specifies the key for the copied item.
//
// Returns error when the underlying provider fails to copy the item.
func (tw *TransformerWrapper) CopyToAnotherRepository(ctx context.Context, srcRepo string, srcKey string, dstRepo string, dstKey string) error {
	if err := tw.provider.CopyToAnotherRepository(ctx, srcRepo, srcKey, dstRepo, dstKey); err != nil {
		return fmt.Errorf("copying object %q to repository %q through transformer wrapper: %w", srcKey, dstRepo, err)
	}
	return nil
}

// Remove passes the call directly to the underlying provider.
//
// Takes params (storage_dto.GetParams) which identifies the items to remove.
//
// Returns error when the underlying provider fails to remove the items.
func (tw *TransformerWrapper) Remove(ctx context.Context, params storage_dto.GetParams) error {
	if err := tw.provider.Remove(ctx, params); err != nil {
		return fmt.Errorf("removing object %q through transformer wrapper: %w", params.Key, err)
	}
	return nil
}

// GetHash passes the call directly to the underlying provider.
//
// Takes params (storage_dto.GetParams) which specifies what to retrieve.
//
// Returns string which is the hash value from the provider.
// Returns error when the underlying provider fails.
func (tw *TransformerWrapper) GetHash(ctx context.Context, params storage_dto.GetParams) (string, error) {
	hash, err := tw.provider.GetHash(ctx, params)
	if err != nil {
		return "", fmt.Errorf("getting hash for %q through transformer wrapper: %w", params.Key, err)
	}
	return hash, nil
}

// PresignURL passes the call directly to the underlying provider.
//
// Takes params (storage_dto.PresignParams) which specifies the presigning
// options.
//
// Returns string which is the presigned URL for the requested resource.
// Returns error when the underlying provider fails to generate the URL.
func (tw *TransformerWrapper) PresignURL(ctx context.Context, params storage_dto.PresignParams) (string, error) {
	url, err := tw.provider.PresignURL(ctx, params)
	if err != nil {
		return "", fmt.Errorf("generating presigned URL for %q through transformer wrapper: %w", params.Key, err)
	}
	return url, nil
}

// PresignDownloadURL passes the call directly to the underlying provider.
//
// Takes params (storage_dto.PresignDownloadParams) which specifies the download
// options.
//
// Returns string which is the presigned URL for downloading.
// Returns error when the underlying provider fails to generate the URL.
func (tw *TransformerWrapper) PresignDownloadURL(ctx context.Context, params storage_dto.PresignDownloadParams) (string, error) {
	url, err := tw.provider.PresignDownloadURL(ctx, params)
	if err != nil {
		return "", fmt.Errorf("generating presigned download URL for %q through transformer wrapper: %w", params.Key, err)
	}
	return url, nil
}

// Close passes the call directly to the underlying provider.
//
// Takes ctx (context.Context) which carries cancellation and tracing
// for the close operation.
//
// Returns error when the underlying provider fails to close.
func (tw *TransformerWrapper) Close(ctx context.Context) error {
	if err := tw.provider.Close(ctx); err != nil {
		return fmt.Errorf("closing provider through transformer wrapper: %w", err)
	}
	return nil
}

// SupportsMultipart passes the call directly to the underlying provider.
//
// Returns bool which indicates whether the provider supports multipart.
func (tw *TransformerWrapper) SupportsMultipart() bool {
	return tw.provider.SupportsMultipart()
}

// PutMany applies transformations to each object in the batch.
//
// Takes params (*storage_dto.PutManyParams) which contains the objects to
// upload.
//
// Returns *storage_dto.BatchResult which contains the upload results for each
// object.
// Returns error when the transformer chain cannot be created or when any
// object transformation fails.
//
// Transformers are applied per-object, not to the entire batch.
func (tw *TransformerWrapper) PutMany(ctx context.Context, params *storage_dto.PutManyParams) (*storage_dto.BatchResult, error) {
	ctx, l := logger_domain.From(ctx, log)
	config := tw.resolveConfig(params.TransformConfig)

	if config == nil || len(config.EnabledTransformers) == 0 {
		return tw.provider.PutMany(ctx, params)
	}

	chain, err := NewTransformerChain(tw.registry, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create transformer chain for PutMany: %w", err)
	}

	if chain.IsEmpty() {
		return tw.provider.PutMany(ctx, params)
	}

	l.Trace("Applying transformations to batch upload",
		logger_domain.String(logFieldProvider, tw.providerName),
		logger_domain.Int("object_count", len(params.Objects)),
		logger_domain.Strings(LogFieldTransformers, config.EnabledTransformers))

	transformedParams := *params
	transformedParams.Objects = make([]storage_dto.PutObjectSpec, len(params.Objects))

	for i, storageObject := range params.Objects {
		transformedReader, err := chain.Transform(ctx, storageObject.Reader)
		if err != nil {
			return nil, fmt.Errorf("transformation failed for object %d (key: %s): %w", i, storageObject.Key, err)
		}

		transformedParams.Objects[i] = storage_dto.PutObjectSpec{
			Key:         storageObject.Key,
			Reader:      transformedReader,
			ContentType: storageObject.ContentType,
			Size:        -1,
		}
	}

	return tw.provider.PutMany(ctx, &transformedParams)
}

// RemoveMany passes through to the underlying provider without transformation,
// as deletions do not need transformation.
//
// Takes params (storage_dto.RemoveManyParams) which specifies items to remove.
//
// Returns *storage_dto.BatchResult which contains the outcome of the removal.
// Returns error when the underlying provider fails.
func (tw *TransformerWrapper) RemoveMany(ctx context.Context, params storage_dto.RemoveManyParams) (*storage_dto.BatchResult, error) {
	result, err := tw.provider.RemoveMany(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("removing %d objects through transformer wrapper: %w", len(params.Keys), err)
	}
	return result, nil
}

// SupportsBatchOperations passes through to the underlying provider.
//
// Returns bool which is true if the provider supports batch operations.
func (tw *TransformerWrapper) SupportsBatchOperations() bool {
	return tw.provider.SupportsBatchOperations()
}

// SupportsRetry passes through to the underlying provider.
//
// Returns bool which is true if the provider supports retry operations.
func (tw *TransformerWrapper) SupportsRetry() bool {
	return tw.provider.SupportsRetry()
}

// SupportsCircuitBreaking passes through to the underlying provider.
//
// Returns bool which indicates whether the provider supports circuit breaking.
func (tw *TransformerWrapper) SupportsCircuitBreaking() bool {
	return tw.provider.SupportsCircuitBreaking()
}

// SupportsRateLimiting passes through to the underlying provider.
//
// Returns bool which is true if the provider supports rate limiting.
func (tw *TransformerWrapper) SupportsRateLimiting() bool {
	return tw.provider.SupportsRateLimiting()
}

// SupportsPresignedURLs passes through to the underlying provider.
//
// Returns bool which is true if the provider supports native presigned URLs.
func (tw *TransformerWrapper) SupportsPresignedURLs() bool {
	return tw.provider.SupportsPresignedURLs()
}

// Rename renames a blob from oldKey to newKey. Transformers do not affect
// rename operations.
//
// Takes repo (string) which identifies the target repository.
// Takes oldKey (string) which specifies the current blob key.
// Takes newKey (string) which specifies the desired new blob key.
//
// Returns error when the underlying provider fails to rename the blob.
func (tw *TransformerWrapper) Rename(ctx context.Context, repo string, oldKey, newKey string) error {
	if err := tw.provider.Rename(ctx, repo, oldKey, newKey); err != nil {
		return fmt.Errorf("renaming object %q to %q through transformer wrapper: %w", oldKey, newKey, err)
	}
	return nil
}

// Exists checks if a blob exists. Transformers do not affect exists
// operations.
//
// Takes params (storage_dto.GetParams) which specifies the blob to check.
//
// Returns bool which indicates whether the blob exists.
// Returns error when the existence check fails.
func (tw *TransformerWrapper) Exists(ctx context.Context, params storage_dto.GetParams) (bool, error) {
	exists, err := tw.provider.Exists(ctx, params)
	if err != nil {
		return false, fmt.Errorf("checking existence of %q through transformer wrapper: %w", params.Key, err)
	}
	return exists, nil
}

// retrieveTransformerConfig fetches transformer settings from object metadata.
//
// Takes params (storage_dto.GetParams) which specifies the object to query.
//
// Returns *storage_dto.TransformConfig which contains the transformer chain,
// or nil if no metadata is found or if parsing fails.
func (tw *TransformerWrapper) retrieveTransformerConfig(
	ctx context.Context, params storage_dto.GetParams,
) *storage_dto.TransformConfig {
	ctx, l := logger_domain.From(ctx, log)
	objInfo, statErr := tw.provider.Stat(ctx, params)
	if statErr != nil || objInfo.Metadata == nil {
		return nil
	}

	metadataJSON, ok := objInfo.Metadata[metadataKeyTransformers]
	if !ok || metadataJSON == "" {
		return nil
	}

	var transformerNames []string
	if err := json.UnmarshalString(metadataJSON, &transformerNames); err != nil {
		l.Warn("Failed to parse transformer metadata, falling back to TransformConfig",
			logger_domain.String(logFieldKey, params.Key),
			logger_domain.Error(err))
		return nil
	}

	if len(transformerNames) == 0 {
		return nil
	}

	l.Trace("Using transformer chain from object metadata",
		logger_domain.String(logFieldKey, params.Key),
		logger_domain.Strings(LogFieldTransformers, transformerNames))

	return &storage_dto.TransformConfig{
		EnabledTransformers: transformerNames,
		TransformerOptions:  make(map[string]any),
	}
}

// applyReverseTransformations creates and applies a transformer chain to
// reverse transformations.
//
// Takes params (storage_dto.GetParams) which provides the key and other
// retrieval parameters.
// Takes rawReader (io.ReadCloser) which supplies the transformed data to
// reverse.
// Takes config (*storage_dto.TransformConfig) which specifies which
// transformations to reverse.
//
// Returns io.ReadCloser which provides the reversed data stream.
// Returns error when the transformer chain cannot be created or reversal fails.
func (tw *TransformerWrapper) applyReverseTransformations(
	ctx context.Context, params storage_dto.GetParams,
	rawReader io.ReadCloser, config *storage_dto.TransformConfig,
) (io.ReadCloser, error) {
	ctx, l := logger_domain.From(ctx, log)
	chain, err := NewTransformerChain(tw.registry, config)
	if err != nil {
		if closeErr := rawReader.Close(); closeErr != nil {
			l.Warn("closing raw reader after transformer chain creation failure", logger_domain.Error(closeErr))
		}
		return nil, fmt.Errorf("failed to create transformer chain for Get: %w", err)
	}

	if chain.IsEmpty() {
		return rawReader, nil
	}

	l.Trace("Reversing transformations after download",
		logger_domain.String(logFieldProvider, tw.providerName),
		logger_domain.String(logFieldKey, params.Key),
		logger_domain.Strings(LogFieldTransformers, config.EnabledTransformers))

	transformCtx, cancel := context.WithCancelCause(ctx)

	reversedReader, err := chain.Reverse(transformCtx, rawReader)
	if err != nil {
		cancel(fmt.Errorf("transformation failed: %w", err))
		if closeErr := rawReader.Close(); closeErr != nil {
			l.Warn("closing raw reader after reverse transformation failure", logger_domain.Error(closeErr))
		}
		return nil, fmt.Errorf("reverse transformation failed for key '%s': %w", params.Key, err)
	}

	return &transformedReadCloser{
		reader: reversedReader,
		closer: rawReader,
		cancel: cancel,
	}, nil
}

// resolveConfig determines the final transform config to use, prioritising the
// operation-specific config over the wrapper's default config.
//
// Takes operationConfig (*storage_dto.TransformConfig) which specifies the
// operation-specific config, or nil to use the default.
//
// Returns *storage_dto.TransformConfig which is the resolved config to use.
func (tw *TransformerWrapper) resolveConfig(operationConfig *storage_dto.TransformConfig) *storage_dto.TransformConfig {
	if operationConfig != nil {
		return operationConfig
	}
	return tw.defaultConfig
}

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

package daemon_domain

// This file implements on-demand variant generation for responsive images. When
// a srcset URL like /_piko/assets/{id}?v=image_w240_webp is requested but the
// variant doesn't exist, this service generates it lazily on first request.

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"path"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"go.opentelemetry.io/otel/trace"
	"piko.sh/piko/internal/capabilities/capabilities_domain"
	"piko.sh/piko/internal/capabilities/capabilities_dto"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/registry/registry_dto"
	"piko.sh/piko/wdk/clock"
)

// OnDemandVariantGenerator creates image variants when they are first
// requested, enabling lazy transformation where variants are only made
// when needed.
type OnDemandVariantGenerator interface {
	// GenerateVariant generates a variant for an artefact based on the profile
	// name.
	//
	// Takes artefact (*registry_dto.ArtefactMeta) which is the source artefact.
	// Takes profileName (string) which identifies the variant profile to use.
	//
	// Returns *registry_dto.Variant which is the newly created variant, or nil if
	// the profile name is invalid or generation fails.
	// Returns error when variant generation fails.
	GenerateVariant(ctx context.Context, artefact *registry_dto.ArtefactMeta, profileName string) (*registry_dto.Variant, error)

	// ParseProfileName parses a profile name like "image_w240_webp" and returns
	// the parsed settings.
	//
	// Takes profileName (string) which is the profile name to parse.
	//
	// Returns *ParsedImageProfile which contains the parsed settings, or nil if
	// the profile name is not valid.
	ParseProfileName(profileName string) *ParsedImageProfile
}

// ParsedImageProfile holds the settings parsed from a profile name. It is
// returned by ParseProfileName and passed to image transformation functions.
type ParsedImageProfile struct {
	// Format specifies the output image format (e.g. "webp", "jpeg").
	Format string

	// Width is the target image width in pixels.
	Width int

	// Quality is the compression quality level; 0 uses the default.
	Quality int
}

// onDemandVariantGeneratorImpl implements the OnDemandVariantGenerator
// interface for creating image variants when they are requested.
type onDemandVariantGeneratorImpl struct {
	// registryService provides access to artefacts, variants, and blob storage.
	registryService registry_domain.RegistryService

	// capabilityService runs image transformation tasks.
	capabilityService capabilities_domain.CapabilityService

	// clock provides the current time for timestamps.
	clock clock.Clock

	// inProgress maps variant keys to mutexes for generation tasks in progress.
	inProgress map[string]*sync.Mutex

	// config holds settings for on-demand variant generation.
	config OnDemandGeneratorConfig

	// inProgressMutex guards access to the inProgress map.
	inProgressMutex sync.Mutex
}

const (
	// defaultMaxWidth is the largest image width in pixels for on-demand
	// generation.
	defaultMaxWidth = 4096

	// defaultMinWidth is the smallest allowed width in pixels for generated
	// images.
	defaultMinWidth = 1

	// defaultImageQuality is the default JPEG quality as a percentage.
	defaultImageQuality = 80

	// defaultStorageBackend is the storage backend used when none is set.
	defaultStorageBackend = "local_disk_cache"
)

// OnDemandGeneratorConfig holds settings for the on-demand image generator.
type OnDemandGeneratorConfig struct {
	// Clock provides the time source for cache expiry; nil uses the real clock.
	Clock clock.Clock

	// StorageBackendID identifies the storage backend used for storing variants.
	StorageBackendID string

	// AllowedFormats lists the image formats that may be generated.
	AllowedFormats []string

	// MaxWidth is the maximum allowed image width in pixels.
	MaxWidth int

	// MinWidth is the minimum allowed width in pixels; widths below this are
	// rejected.
	MinWidth int

	// DefaultQuality is the image quality used when not specified in the
	// profile name; range is 1-100.
	DefaultQuality int
}

// profileNameRegex matches profile names like "image_w240_webp" or
// "image_w1024_jpeg".
var profileNameRegex = regexp.MustCompile(`^image_w(\d+)_([a-z]+)$`)

// ParseProfileName parses a profile name and validates it against security
// constraints.
//
// Takes profileName (string) which specifies the profile name to parse.
//
// Returns *ParsedImageProfile which contains the parsed width, format, and
// quality, or nil when the profile name is invalid or fails validation.
func (g *onDemandVariantGeneratorImpl) ParseProfileName(profileName string) *ParsedImageProfile {
	matches := profileNameRegex.FindStringSubmatch(profileName)
	if matches == nil {
		return nil
	}

	width, err := strconv.Atoi(matches[1])
	if err != nil {
		return nil
	}

	format := matches[2]

	if width < g.config.MinWidth || width > g.config.MaxWidth {
		return nil
	}

	if !g.isAllowedFormat(format) {
		return nil
	}

	return &ParsedImageProfile{
		Width:   width,
		Format:  format,
		Quality: g.config.DefaultQuality,
	}
}

// GenerateVariant generates a variant on-demand.
//
// Takes artefact (*registry_dto.ArtefactMeta) which specifies the source
// artefact to generate a variant from.
// Takes profileName (string) which identifies the transformation profile to
// apply.
//
// Returns *registry_dto.Variant which is the generated variant metadata.
// Returns error when the profile name is invalid or generation fails.
//
// Safe for concurrent use. Uses per-variant locking to prevent duplicate
// generation of the same variant.
func (g *onDemandVariantGeneratorImpl) GenerateVariant(
	ctx context.Context,
	artefact *registry_dto.ArtefactMeta,
	profileName string,
) (*registry_dto.Variant, error) {
	ctx, span, l := log.Span(ctx, "OnDemandVariantGenerator.GenerateVariant",
		logger_domain.String("artefactID", artefact.ID),
		logger_domain.String("profileName", profileName),
	)
	defer span.End()

	profile := g.ParseProfileName(profileName)
	if profile == nil {
		l.Warn("Invalid profile name requested", logger_domain.String("profileName", profileName))
		return nil, fmt.Errorf("invalid or disallowed profile name: %s", profileName)
	}

	variantKey := fmt.Sprintf("%s:%s", artefact.ID, profileName)
	variantMutex := g.getOrCreateVariantMutex(variantKey)
	variantMutex.Lock()
	defer func() {
		variantMutex.Unlock()
		g.cleanupVariantMutex(variantKey)
	}()

	if existing := g.checkExistingVariant(ctx, artefact.ID, profileName); existing != nil {
		return existing, nil
	}

	l.Trace("Generating variant on-demand",
		logger_domain.Int("width", profile.Width),
		logger_domain.String("format", profile.Format),
		logger_domain.Int("quality", profile.Quality),
	)

	newVariant, err := g.executeVariantGeneration(ctx, artefact, profileName, profile, span)
	if err != nil {
		return nil, fmt.Errorf("executing variant generation for profile %q: %w", profileName, err)
	}

	l.Trace("Variant generated successfully",
		logger_domain.String("variantID", newVariant.VariantID),
		logger_domain.Int64("sizeBytes", newVariant.SizeBytes),
	)

	return newVariant, nil
}

// isAllowedFormat checks if the format is in the allowed list.
//
// Takes format (string) which is the format name to check.
//
// Returns bool which is true if the format matches any allowed format.
func (g *onDemandVariantGeneratorImpl) isAllowedFormat(format string) bool {
	for _, allowed := range g.config.AllowedFormats {
		if strings.EqualFold(format, allowed) {
			return true
		}
	}
	return false
}

// checkExistingVariant checks if the variant was created while waiting for
// the lock.
//
// Takes artefactID (string) which identifies the artefact to check.
// Takes profileName (string) which specifies the variant profile to look for.
//
// Returns *registry_dto.Variant which is the existing variant if found, or nil
// if not found or on error.
func (g *onDemandVariantGeneratorImpl) checkExistingVariant(ctx context.Context, artefactID, profileName string) *registry_dto.Variant {
	ctx, l := logger_domain.From(ctx, log)
	refreshedArtefact, err := g.registryService.GetArtefact(ctx, artefactID)
	if err != nil || refreshedArtefact == nil {
		return nil
	}

	for i := range refreshedArtefact.ActualVariants {
		if refreshedArtefact.ActualVariants[i].VariantID == profileName {
			l.Trace("Variant was created by another request while waiting")
			return &refreshedArtefact.ActualVariants[i]
		}
	}
	return nil
}

// executeVariantGeneration performs the actual variant generation pipeline.
//
// Takes artefact (*registry_dto.ArtefactMeta) which identifies the source
// artefact to transform.
// Takes profileName (string) which specifies the name of the image profile.
// Takes profile (*ParsedImageProfile) which defines the transformation rules.
// Takes span (trace.Span) which provides tracing context.
//
// Returns *registry_dto.Variant which is the newly generated and stored
// variant.
// Returns error when the source variant is not found, image transformation
// fails, or storage fails.
func (g *onDemandVariantGeneratorImpl) executeVariantGeneration(
	ctx context.Context,
	artefact *registry_dto.ArtefactMeta,
	profileName string,
	profile *ParsedImageProfile,
	span trace.Span,
) (*registry_dto.Variant, error) {
	ctx, l := logger_domain.From(ctx, log)
	sourceVariant := g.findSourceVariant(artefact)
	if sourceVariant == nil {
		err := fmt.Errorf("source variant not found for artefact %s", artefact.ID)
		l.ReportError(span, err, "Source variant not found")
		return nil, err
	}

	outputStream, err := g.transformImage(ctx, sourceVariant, profile, span)
	if err != nil {
		return nil, fmt.Errorf("transforming image: %w", err)
	}
	if closer, ok := outputStream.(io.Closer); ok {
		defer func() { _ = closer.Close() }()
	}

	newVariant, err := g.storeVariant(ctx, artefact, profileName, profile, outputStream)
	if err != nil {
		l.ReportError(span, err, "Failed to store generated variant")
		return nil, fmt.Errorf("storing generated variant: %w", err)
	}

	return newVariant, nil
}

// transformImage executes the image transformation capability.
//
// Takes sourceVariant (*registry_dto.Variant) which identifies the source
// image to transform.
// Takes profile (*ParsedImageProfile) which specifies the target dimensions,
// format, and quality.
// Takes span (trace.Span) which provides tracing context for error reporting.
//
// Returns io.Reader which streams the transformed image data.
// Returns error when the source data cannot be retrieved or transformation
// fails.
func (g *onDemandVariantGeneratorImpl) transformImage(
	ctx context.Context,
	sourceVariant *registry_dto.Variant,
	profile *ParsedImageProfile,
	span trace.Span,
) (io.Reader, error) {
	ctx, l := logger_domain.From(ctx, log)
	sourceStream, err := g.registryService.GetVariantData(ctx, sourceVariant)
	if err != nil {
		l.ReportError(span, err, "Failed to get source variant data")
		return nil, fmt.Errorf("failed to get source data: %w", err)
	}

	capabilityParams := capabilities_domain.CapabilityParams{
		"width":   strconv.Itoa(profile.Width),
		"format":  profile.Format,
		"quality": strconv.Itoa(profile.Quality),
	}

	outputStream, err := g.capabilityService.Execute(ctx, string(capabilities_dto.CapabilityImageTransform), sourceStream, capabilityParams)
	if err != nil {
		_ = sourceStream.Close()
		l.ReportError(span, err, "Image transform capability failed")
		return nil, fmt.Errorf("image transform failed: %w", err)
	}

	_ = sourceStream.Close()

	return outputStream, nil
}

// findSourceVariant finds the source variant for an artefact.
//
// Takes artefact (*registry_dto.ArtefactMeta) which contains the variants to
// search.
//
// Returns *registry_dto.Variant which is the source variant, or nil if not
// found.
func (*onDemandVariantGeneratorImpl) findSourceVariant(artefact *registry_dto.ArtefactMeta) *registry_dto.Variant {
	for i := range artefact.ActualVariants {
		if artefact.ActualVariants[i].VariantID == "source" {
			return &artefact.ActualVariants[i]
		}
	}
	return nil
}

// storeVariant stores the generated variant data and adds it to the registry.
//
// Takes artefact (*registry_dto.ArtefactMeta) which provides the source
// artefact metadata.
// Takes profileName (string) which identifies the image profile to apply.
// Takes profile (*ParsedImageProfile) which contains the parsed image
// transformation settings.
// Takes outputStream (io.Reader) which provides the transformed image data.
//
// Returns *registry_dto.Variant which is the newly created variant record.
// Returns error when the blob store is unavailable, the transformation
// produces zero bytes, or the variant cannot be saved to the registry.
func (g *onDemandVariantGeneratorImpl) storeVariant(
	ctx context.Context,
	artefact *registry_dto.ArtefactMeta,
	profileName string,
	profile *ParsedImageProfile,
	outputStream io.Reader,
) (*registry_dto.Variant, error) {
	ctx, span, l := log.Span(ctx, "OnDemandVariantGenerator.storeVariant",
		logger_domain.String("artefactID", artefact.ID),
		logger_domain.String("profileName", profileName),
	)
	defer span.End()

	blobStore, err := g.registryService.GetBlobStore(g.config.StorageBackendID)
	if err != nil {
		l.ReportError(span, err, "Failed to get blob store")
		return nil, fmt.Errorf("failed to get blob store: %w", err)
	}

	tempKey := g.generateTempKey(artefact.ID, profileName)
	finalHash, byteCount, err := g.writeToBlobStore(ctx, blobStore, tempKey, outputStream, span)
	if err != nil {
		return nil, fmt.Errorf("writing variant to blob store: %w", err)
	}

	if byteCount == 0 {
		_ = blobStore.Delete(ctx, tempKey)
		err := fmt.Errorf("image transformation produced zero bytes for artefact %s profile %s", artefact.ID, profileName)
		l.ReportError(span, err, "Empty image output")
		return nil, err
	}

	finalStorageKey := g.generateFinalStorageKey(artefact.SourcePath, finalHash, profile.Format)
	if err := g.renameBlobToFinal(ctx, blobStore, tempKey, finalStorageKey, span); err != nil {
		return nil, fmt.Errorf("renaming blob to final storage key: %w", err)
	}

	newVariant := g.buildVariantRecord(profileName, finalStorageKey, profile, finalHash, byteCount)
	return g.addVariantToRegistry(ctx, blobStore, artefact.ID, finalStorageKey, &newVariant, span)
}

// generateTempKey creates a temporary storage key for the variant.
//
// Takes artefactID (string) which identifies the source artefact.
// Takes profileName (string) which specifies the variant profile.
//
// Returns string which is the temporary path combining the artefact ID,
// profile name, and current timestamp.
func (g *onDemandVariantGeneratorImpl) generateTempKey(artefactID, profileName string) string {
	return path.Join("tmp", fmt.Sprintf("ondemand_%s_%s_%d", artefactID, profileName, g.clock.Now().UnixNano()))
}

// writeToBlobStore writes the output stream to the blob store and returns
// hash and size.
//
// Takes blobStore (registry_domain.BlobStore) which provides blob storage.
// Takes tempKey (string) which specifies the temporary storage key.
// Takes outputStream (io.Reader) which provides the data to write.
// Takes span (trace.Span) which provides tracing context.
//
// Returns []byte which is the SHA-256 hash of the written data.
// Returns int64 which is the number of bytes written.
// Returns error when the blob cannot be written to the store.
func (*onDemandVariantGeneratorImpl) writeToBlobStore(
	ctx context.Context,
	blobStore registry_domain.BlobStore,
	tempKey string,
	outputStream io.Reader,
	span trace.Span,
) ([]byte, int64, error) {
	ctx, l := logger_domain.From(ctx, log)
	hasher := sha256.New()
	var byteCount int64
	countingReader := &countingHashReader{reader: outputStream, hasher: hasher, byteCount: &byteCount}

	if err := blobStore.Put(ctx, tempKey, countingReader); err != nil {
		_ = blobStore.Delete(ctx, tempKey)
		l.ReportError(span, err, "Failed to write blob")
		return nil, 0, fmt.Errorf("failed to write blob: %w", err)
	}

	return hasher.Sum(nil), byteCount, nil
}

// generateFinalStorageKey creates the final storage key using the hash.
//
// Takes sourcePath (string) which is the original file path.
// Takes finalHash ([]byte) which is the hash used to create a unique key.
// Takes format (string) which specifies the output format for the extension.
//
// Returns string which is the generated storage key with path, hash, and
// extension.
func (g *onDemandVariantGeneratorImpl) generateFinalStorageKey(sourcePath string, finalHash []byte, format string) string {
	shortHash := fmt.Sprintf("%x", finalHash[:8])
	extension := g.getExtensionForFormat(format)
	basePath := strings.TrimSuffix(sourcePath, path.Ext(sourcePath))
	return path.Join("generated", fmt.Sprintf("%s_%s%s", basePath, shortHash, extension))
}

// renameBlobToFinal moves the blob from temp to final location.
//
// Takes blobStore (registry_domain.BlobStore) which provides blob operations.
// Takes tempKey (string) which is the temporary blob location.
// Takes finalKey (string) which is the target blob location.
// Takes span (trace.Span) which provides tracing context.
//
// Returns error when the rename fails. Deletes the temp blob on failure.
func (*onDemandVariantGeneratorImpl) renameBlobToFinal(
	ctx context.Context,
	blobStore registry_domain.BlobStore,
	tempKey, finalKey string,
	span trace.Span,
) error {
	ctx, l := logger_domain.From(ctx, log)
	if err := blobStore.Rename(ctx, tempKey, finalKey); err != nil {
		_ = blobStore.Delete(ctx, tempKey)
		l.ReportError(span, err, "Failed to rename blob")
		return fmt.Errorf("failed to rename blob: %w", err)
	}
	return nil
}

// buildVariantRecord creates the variant metadata record.
//
// Takes profileName (string) which identifies the image profile.
// Takes storageKey (string) which specifies the storage location.
// Takes profile (*ParsedImageProfile) which provides format settings.
// Takes finalHash ([]byte) which contains the content hash.
// Takes byteCount (int64) which specifies the file size in bytes.
//
// Returns registry_dto.Variant which contains the complete variant metadata.
func (g *onDemandVariantGeneratorImpl) buildVariantRecord(
	profileName, storageKey string,
	profile *ParsedImageProfile,
	finalHash []byte,
	byteCount int64,
) registry_dto.Variant {
	extension := g.getExtensionForFormat(profile.Format)

	var tags registry_dto.Tags
	tags.Set(registry_dto.TagEtag, fmt.Sprintf(`"%x"`, finalHash))
	tags.Set(registry_dto.TagType, "image-variant")
	tags.SetByName("storageBackendId", g.config.StorageBackendID)
	tags.SetByName("fileExtension", extension)

	return registry_dto.Variant{
		MetadataTags:     tags,
		CreatedAt:        g.clock.Now().UTC(),
		VariantID:        profileName,
		StorageBackendID: g.config.StorageBackendID,
		StorageKey:       storageKey,
		MimeType:         g.getMimeTypeForFormat(profile.Format),
		Status:           registry_dto.VariantStatusReady,
		ContentHash:      fmt.Sprintf("%x", finalHash),
		Chunks:           nil,
		SizeBytes:        byteCount,
	}
}

// addVariantToRegistry adds the variant to the registry and handles cleanup
// on failure.
//
// Takes blobStore (registry_domain.BlobStore) which provides blob storage for
// cleanup on failure.
// Takes artefactID (string) which identifies the parent artefact.
// Takes storageKey (string) which identifies the blob to delete on failure.
// Takes variant (*registry_dto.Variant) which contains the variant to add.
// Takes span (trace.Span) which provides tracing context for error reporting.
//
// Returns *registry_dto.Variant which is the added variant on success.
// Returns error when adding the variant to the registry fails.
func (g *onDemandVariantGeneratorImpl) addVariantToRegistry(
	ctx context.Context,
	blobStore registry_domain.BlobStore,
	artefactID, storageKey string,
	variant *registry_dto.Variant,
	span trace.Span,
) (*registry_dto.Variant, error) {
	ctx, l := logger_domain.From(ctx, log)
	_, err := g.registryService.AddVariant(ctx, artefactID, variant)
	if err != nil {
		_ = blobStore.Delete(ctx, storageKey)
		l.ReportError(span, err, "Failed to add variant to registry")
		return nil, fmt.Errorf("failed to add variant: %w", err)
	}

	return variant, nil
}

// getExtensionForFormat returns the file extension for a given image format.
//
// Takes format (string) which specifies the image format name (e.g. "webp",
// "jpeg", "png", "avif").
//
// Returns string which is the file extension with a leading dot, or ".img" if
// the format is not recognised.
func (*onDemandVariantGeneratorImpl) getExtensionForFormat(format string) string {
	switch strings.ToLower(format) {
	case "webp":
		return ".webp"
	case "jpeg", "jpg":
		return ".jpeg"
	case "png":
		return ".png"
	case "avif":
		return ".avif"
	default:
		return ".img"
	}
}

// getMimeTypeForFormat returns the MIME type for a format.
//
// Takes format (string) which specifies the image format name.
//
// Returns string which is the corresponding MIME type, or
// "application/octet-stream" for unknown formats.
func (*onDemandVariantGeneratorImpl) getMimeTypeForFormat(format string) string {
	switch strings.ToLower(format) {
	case "webp":
		return "image/webp"
	case "jpeg", "jpg":
		return "image/jpeg"
	case "png":
		return "image/png"
	case "avif":
		return "image/avif"
	default:
		return "application/octet-stream"
	}
}

// getOrCreateVariantMutex gets or creates a mutex for a specific variant key.
//
// Takes key (string) which identifies the variant to lock.
//
// Returns *sync.Mutex which is the mutex for the given key.
func (g *onDemandVariantGeneratorImpl) getOrCreateVariantMutex(key string) *sync.Mutex {
	g.inProgressMutex.Lock()
	defer g.inProgressMutex.Unlock()

	if mu, exists := g.inProgress[key]; exists {
		return mu
	}

	mu := &sync.Mutex{}
	g.inProgress[key] = mu
	return mu
}

// cleanupVariantMutex removes the mutex for a variant key if no longer needed.
//
// Takes key (string) which identifies the variant mutex to remove.
func (g *onDemandVariantGeneratorImpl) cleanupVariantMutex(key string) {
	g.inProgressMutex.Lock()
	defer g.inProgressMutex.Unlock()
	delete(g.inProgress, key)
}

// countingHashReader wraps a reader to count bytes and compute a hash.
// It implements io.Reader.
type countingHashReader struct {
	// reader is the source from which bytes are read.
	reader io.Reader

	// hasher computes a hash from the bytes read.
	hasher io.Writer

	// byteCount tracks the total bytes read from the reader.
	byteCount *int64
}

// Read implements io.Reader, counting bytes and writing them to the hasher.
//
// Takes p ([]byte) which is the buffer to read data into.
//
// Returns n (int) which is the number of bytes read.
// Returns err (error) which signals the end of the stream or a read
// failure.
func (r *countingHashReader) Read(p []byte) (n int, err error) {
	n, err = r.reader.Read(p)
	if n > 0 {
		*r.byteCount += int64(n)
		_, _ = r.hasher.Write(p[:n])
	}
	return n, err
}

// DefaultOnDemandGeneratorConfig returns the default settings for image
// generation.
//
// Returns OnDemandGeneratorConfig which contains default values for allowed
// formats, size limits, and quality settings.
func DefaultOnDemandGeneratorConfig() OnDemandGeneratorConfig {
	return OnDemandGeneratorConfig{
		Clock:            nil,
		StorageBackendID: defaultStorageBackend,
		AllowedFormats:   []string{"webp", "jpeg", "jpg", "png", "avif"},
		MaxWidth:         defaultMaxWidth,
		MinWidth:         defaultMinWidth,
		DefaultQuality:   defaultImageQuality,
	}
}

// NewOnDemandVariantGenerator creates a new on-demand variant generator.
//
// Takes registryService (RegistryService) which provides access to the registry.
// Takes capabilityService (CapabilityService) which provides capability lookups.
// Takes config (OnDemandGeneratorConfig) which sets the generator options.
// If config.Clock is nil, a real clock is used.
//
// Returns OnDemandVariantGenerator which is ready for concurrent use.
func NewOnDemandVariantGenerator(
	registryService registry_domain.RegistryService,
	capabilityService capabilities_domain.CapabilityService,
	config OnDemandGeneratorConfig,
) OnDemandVariantGenerator {
	clk := config.Clock
	if clk == nil {
		clk = clock.RealClock()
	}

	return &onDemandVariantGeneratorImpl{
		registryService:   registryService,
		capabilityService: capabilityService,
		clock:             clk,
		inProgress:        make(map[string]*sync.Mutex),
		config:            config,
		inProgressMutex:   sync.Mutex{},
	}
}

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

package asset_resolver

import (
	"context"
	"fmt"
	"io"

	"go.opentelemetry.io/otel/trace"
	"piko.sh/piko/internal/email/email_domain"
	"piko.sh/piko/internal/email/email_dto"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/registry/registry_dto"
)

// adapter implements email_domain.AssetResolverPort by delegating to the
// registry service. It fetches assets, applies the requested transformation
// profile, and returns them as ready-to-embed email attachments with CID set.
type adapter struct {
	// registryService fetches artefact metadata and variant data from the registry.
	registryService registry_domain.RegistryService
}

var _ email_domain.AssetResolverPort = (*adapter)(nil)

// ResolveAsset fetches and transforms a single asset based on the request.
//
// Takes request (*email_dto.EmailAssetRequest) which specifies the asset to
// resolve and any transformation options.
//
// Returns *email_dto.Attachment which contains the resolved asset ready for
// email attachment.
// Returns error when the artefact cannot be fetched, no suitable variant is
// available, or the variant content cannot be retrieved.
func (a *adapter) ResolveAsset(ctx context.Context, request *email_dto.EmailAssetRequest) (*email_dto.Attachment, error) {
	ctx, l := logger_domain.From(ctx, log)
	spanAttrs := buildSpanAttributes(request)
	ctx, span, l := l.Span(ctx, "AssetResolver.ResolveAsset", spanAttrs...)
	defer span.End()

	artefact, err := a.fetchArtefact(ctx, span, request.SourcePath)
	if err != nil {
		return nil, fmt.Errorf("fetching artefact for email asset: %w", err)
	}

	matchingVariant, fallbackUsed, err := a.selectVariant(artefact.ActualVariants, request)
	if err != nil {
		l.ReportError(span, err, "No variants available")
		return nil, fmt.Errorf("selecting variant for email asset: %w", err)
	}

	logVariantSelection(ctx, matchingVariant, fallbackUsed, request.Profile)

	content, err := a.fetchVariantContent(ctx, span, matchingVariant, request)
	if err != nil {
		return nil, fmt.Errorf("fetching variant content for email asset: %w", err)
	}

	attachment := buildAttachment(request.SourcePath, matchingVariant, content, request.CID)

	l.Trace("Successfully resolved email asset",
		logger_domain.String("filename", attachment.Filename),
		logger_domain.String("cid", request.CID),
		logger_domain.Int("size_bytes", len(content)),
	)

	return attachment, nil
}

// ResolveAssets resolves multiple assets in batch.
//
// Takes requests ([]*email_dto.EmailAssetRequest) which specifies the assets
// to resolve.
//
// Returns []*email_dto.Attachment which contains the successfully resolved
// attachments.
// Returns []error which contains one error per request, or nil for successes.
func (a *adapter) ResolveAssets(ctx context.Context, requests []*email_dto.EmailAssetRequest) ([]*email_dto.Attachment, []error) {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "AssetResolver.ResolveAssets",
		logger_domain.Int("request_count", len(requests)),
	)
	defer span.End()

	attachments := make([]*email_dto.Attachment, 0, len(requests))
	errors := make([]error, len(requests))

	for i, request := range requests {
		attachment, err := a.ResolveAsset(ctx, request)
		if err != nil {
			errors[i] = err
			l.Warn("Failed to resolve asset in batch",
				logger_domain.String("source_path", request.SourcePath),
				logger_domain.String("profile", request.Profile),
				logger_domain.Error(err),
			)
			continue
		}
		attachments = append(attachments, attachment)
		errors[i] = nil
	}

	successCount := len(attachments)
	failureCount := len(requests) - successCount

	l.Trace("Batch asset resolution completed",
		logger_domain.Int("success_count", successCount),
		logger_domain.Int("failure_count", failureCount),
	)

	return attachments, errors
}

// fetchArtefact gets the artefact metadata from the registry.
//
// Takes span (trace.Span) which is used for error reporting.
// Takes sourcePath (string) which identifies the artefact to fetch.
//
// Returns *registry_dto.ArtefactMeta which contains the artefact metadata.
// Returns error when the artefact cannot be fetched or is not found.
func (a *adapter) fetchArtefact(
	ctx context.Context,
	span trace.Span,
	sourcePath string,
) (*registry_dto.ArtefactMeta, error) {
	ctx, l := logger_domain.From(ctx, log)
	artefact, err := a.registryService.GetArtefact(ctx, sourcePath)
	if err != nil {
		l.ReportError(span, err, "Failed to fetch artefact from registry")
		return nil, fmt.Errorf("fetching artefact '%s': %w", sourcePath, err)
	}

	if artefact == nil {
		err := fmt.Errorf("fetching artefact for email asset: artefact not found at %q", sourcePath)
		l.ReportError(span, err, "Artefact not found in registry")
		return nil, err
	}

	return artefact, nil
}

// selectVariant finds the best matching variant using the fallback strategy.
//
// Takes variants ([]registry_dto.Variant) which contains the available variants
// to choose from.
// Takes request (*email_dto.EmailAssetRequest) which specifies the desired
// profile, width, and density.
//
// Returns *registry_dto.Variant which is the best matching variant.
// Returns string which describes the fallback used, or empty if exact match.
// Returns error when no usable variants are available.
func (a *adapter) selectVariant(
	variants []registry_dto.Variant,
	request *email_dto.EmailAssetRequest,
) (*registry_dto.Variant, string, error) {
	matchingVariant, fallbackUsed := a.findBestMatchingVariant(variants, request)

	if matchingVariant == nil {
		err := fmt.Errorf("no usable variants found for asset '%s' (tried profile, source, and any available variant)", request.SourcePath)
		return nil, "", err
	}

	return matchingVariant, fallbackUsed, nil
}

// fetchVariantContent gets the binary content of a variant from the blob store.
//
// Takes span (trace.Span) which records tracing data for the operation.
// Takes variant (*registry_dto.Variant) which identifies the variant to fetch.
// Takes request (*email_dto.EmailAssetRequest) which provides the asset details.
//
// Returns []byte which contains the raw binary content of the variant.
// Returns error when the variant data cannot be fetched or read.
func (a *adapter) fetchVariantContent(
	ctx context.Context,
	span trace.Span,
	variant *registry_dto.Variant,
	request *email_dto.EmailAssetRequest,
) ([]byte, error) {
	ctx, l := logger_domain.From(ctx, log)
	dataReader, err := a.registryService.GetVariantData(ctx, variant)
	if err != nil {
		l.ReportError(span, err, "Failed to fetch variant data from blob store")
		return nil, fmt.Errorf("fetching variant data for '%s' (profile '%s'): %w", request.SourcePath, request.Profile, err)
	}
	defer func() { _ = dataReader.Close() }()

	content, err := io.ReadAll(dataReader)
	if err != nil {
		l.ReportError(span, err, "Failed to read variant data")
		return nil, fmt.Errorf("reading variant data for '%s': %w", request.SourcePath, err)
	}

	l.Trace("Successfully fetched asset data", logger_domain.Int("content_size_bytes", len(content)))

	return content, nil
}

// findBestMatchingVariant selects the most suitable variant for an email asset
// request using a step-by-step fallback strategy.
//
// Takes variants ([]registry_dto.Variant) which contains the available image
// variants to choose from.
// Takes request (*email_dto.EmailAssetRequest) which specifies the desired
// profile, width, and density.
//
// Returns *registry_dto.Variant which is the best matching variant, or nil if
// variants is empty.
// Returns string which describes the fallback used, or empty if an exact match
// is found.
//
// Fallback strategy:
// 1. Exact match: profile + width + density (if all specified).
// 2. Profile + width: ignore density mismatch.
// 3. Profile only: ignore width and density mismatches.
// 4. Width + density: ignore profile (use any variant with matching size).
// 5. Source variant: the original unmodified file.
// 6. Any available variant: last resort (e.g., compressed versions).
func (*adapter) findBestMatchingVariant(
	variants []registry_dto.Variant,
	request *email_dto.EmailAssetRequest,
) (*registry_dto.Variant, string) {
	if len(variants) == 0 {
		return nil, ""
	}

	if variant := findExactMatch(variants, request); variant != nil {
		return variant, ""
	}

	if variant := findProfileAndWidthMatch(variants, request); variant != nil {
		return variant, fmt.Sprintf("profile and width match, but density '%s' not available", request.Density)
	}

	if variant := findProfileOnlyMatch(variants, request); variant != nil {
		return variant, "profile matches, but requested width/density not available"
	}

	if variant := findDimensionMatch(variants, request); variant != nil {
		return variant, fmt.Sprintf("width/density match, but profile '%s' not configured", request.Profile)
	}

	if variant := findSourceVariant(variants); variant != nil {
		return variant, "using source variant (no matching responsive variants found)"
	}

	return &variants[0], "using first available variant (no source or matching variants found)"
}

// New creates a new asset resolver adapter that wraps the registry service.
//
// Takes registryService (registry_domain.RegistryService) which provides
// the underlying registry operations for asset resolution.
//
// Returns email_domain.AssetResolverPort which is the configured adapter
// ready for use.
func New(registryService registry_domain.RegistryService) email_domain.AssetResolverPort {
	return &adapter{
		registryService: registryService,
	}
}

// buildSpanAttributes creates tracing span attributes for an asset request.
//
// Takes request (*email_dto.EmailAssetRequest) which holds the asset details
// to convert into span attributes.
//
// Returns []logger_domain.Attr which holds the span attributes for tracing.
func buildSpanAttributes(request *email_dto.EmailAssetRequest) []logger_domain.Attr {
	spanAttrs := []logger_domain.Attr{
		logger_domain.String("source_path", request.SourcePath),
		logger_domain.String("profile", request.Profile),
		logger_domain.String("cid", request.CID),
	}

	if request.Width > 0 {
		spanAttrs = append(spanAttrs, logger_domain.Int("width", request.Width))
	}
	if request.Density != "" {
		spanAttrs = append(spanAttrs, logger_domain.String("density", request.Density))
	}

	return spanAttrs
}

// logVariantSelection logs details about the chosen variant and any fallback
// that was used.
//
// Takes ctx (context.Context) which carries the logger and tracing data.
// Takes variant (*registry_dto.Variant) which is the chosen variant to log.
// Takes fallbackUsed (string) which explains why a fallback was needed, or is
// empty if no fallback was used.
// Takes requestedProfile (string) which is the profile that was first asked
// for.
func logVariantSelection(
	ctx context.Context,
	variant *registry_dto.Variant,
	fallbackUsed string,
	requestedProfile string,
) {
	ctx, l := logger_domain.From(ctx, log)
	if fallbackUsed != "" {
		l.Warn("Using fallback variant for email asset",
			logger_domain.String("fallback_reason", fallbackUsed),
			logger_domain.String("requested_profile", requestedProfile),
			logger_domain.String("selected_variant", variant.VariantID),
		)
	}

	l.Trace("Found matching variant",
		logger_domain.String("storage_key", variant.StorageKey),
		logger_domain.String("mime_type", variant.MimeType),
		logger_domain.Int64("size_bytes", variant.SizeBytes),
	)
}

// buildAttachment creates an email attachment from the given variant data.
//
// Takes sourcePath (string) which is the path used to get the filename.
// Takes variant (*registry_dto.Variant) which provides the MIME type.
// Takes content ([]byte) which holds the attachment data.
// Takes cid (string) which sets the Content-ID for inline references.
//
// Returns *email_dto.Attachment which is the built attachment.
func buildAttachment(
	sourcePath string,
	variant *registry_dto.Variant,
	content []byte,
	cid string,
) *email_dto.Attachment {
	filename := extractFilename(sourcePath)

	return &email_dto.Attachment{
		Filename:  filename,
		MIMEType:  variant.MimeType,
		Content:   content,
		ContentID: cid,
	}
}

// extractFilename gets the filename from a source path.
// For example, "assets/images/logo.png" returns "logo.png".
//
// Takes sourcePath (string) which is the full path to get the filename from.
//
// Returns string which is the filename after the last slash, or the full path
// if no slash is found.
func extractFilename(sourcePath string) string {
	filename := sourcePath
	for i := len(filename) - 1; i >= 0; i-- {
		if filename[i] == '/' {
			filename = filename[i+1:]
			break
		}
	}
	return filename
}

// findExactMatch searches for a variant that matches profile, width, and
// density exactly.
//
// Takes variants ([]registry_dto.Variant) which is the list of variants to
// search through.
// Takes request (*email_dto.EmailAssetRequest) which specifies the profile,
// width, and density to match.
//
// Returns *registry_dto.Variant which is the first matching variant, or nil
// if no exact match is found.
func findExactMatch(variants []registry_dto.Variant, request *email_dto.EmailAssetRequest) *registry_dto.Variant {
	for i := range variants {
		v := &variants[i]
		if variantMatchesProfile(v, request.Profile) &&
			variantMatchesWidth(v, request.Width) &&
			variantMatchesDensity(v, request.Density) {
			return v
		}
	}
	return nil
}

// findProfileAndWidthMatch finds a variant that matches the profile and width,
// ignoring density. This only applies when density was requested.
//
// Takes variants ([]registry_dto.Variant) which provides the list of variants
// to search.
// Takes request (*email_dto.EmailAssetRequest) which specifies the profile and
// width to match.
//
// Returns *registry_dto.Variant which is the first matching variant, or nil if
// no match is found or density was not requested.
func findProfileAndWidthMatch(variants []registry_dto.Variant, request *email_dto.EmailAssetRequest) *registry_dto.Variant {
	if request.Density == "" {
		return nil
	}

	for i := range variants {
		v := &variants[i]
		if variantMatchesProfile(v, request.Profile) && variantMatchesWidth(v, request.Width) {
			return v
		}
	}
	return nil
}

// findProfileOnlyMatch searches for a variant that matches only the profile,
// without checking dimensions. This is used as a fallback when width or density
// was requested but no exact match was found.
//
// Takes variants ([]registry_dto.Variant) which contains the available variants
// to search through.
// Takes request (*email_dto.EmailAssetRequest) which specifies the profile and
// dimension criteria.
//
// Returns *registry_dto.Variant which is the first matching variant, or nil if
// no match is found or no dimensions were requested.
func findProfileOnlyMatch(variants []registry_dto.Variant, request *email_dto.EmailAssetRequest) *registry_dto.Variant {
	if request.Width == 0 && request.Density == "" {
		return nil
	}

	for i := range variants {
		v := &variants[i]
		if variantMatchesProfile(v, request.Profile) {
			return v
		}
	}
	return nil
}

// findDimensionMatch searches for a variant that matches the requested width
// and density, without checking the profile. This only runs if a profile was
// originally requested.
//
// Takes variants ([]registry_dto.Variant) which contains the image variants
// to search through.
// Takes request (*email_dto.EmailAssetRequest) which specifies the desired
// width, density, and profile values.
//
// Returns *registry_dto.Variant which is the first matching variant, or nil
// if no match is found or if no profile was requested.
func findDimensionMatch(variants []registry_dto.Variant, request *email_dto.EmailAssetRequest) *registry_dto.Variant {
	if request.Profile == "" {
		return nil
	}

	for i := range variants {
		v := &variants[i]
		if variantMatchesWidth(v, request.Width) && variantMatchesDensity(v, request.Density) {
			return v
		}
	}
	return nil
}

// findSourceVariant searches for the source variant in a list of variants.
//
// Takes variants ([]registry_dto.Variant) which is the list of variants to
// search through.
//
// Returns *registry_dto.Variant which is the source variant, or nil if not
// found.
func findSourceVariant(variants []registry_dto.Variant) *registry_dto.Variant {
	for i := range variants {
		v := &variants[i]
		if v.VariantID == "source" {
			return v
		}
	}
	return nil
}

// variantMatchesProfile checks if a variant's ID matches the given profile.
//
// Takes variant (*registry_dto.Variant) which is the variant to check.
// Takes profile (string) which is the profile name to match.
//
// Returns bool which is true if the variant's ID equals the profile.
func variantMatchesProfile(variant *registry_dto.Variant, profile string) bool {
	return variant.VariantID == profile
}

// variantMatchesWidth checks if a variant's width matches the requested width.
//
// Takes variant (*registry_dto.Variant) which provides the variant to check.
// Takes width (int) which specifies the requested width in pixels.
//
// Returns bool which is true if width is zero or the variant's width tag
// matches the requested width in either "Npx" or "N" format.
func variantMatchesWidth(variant *registry_dto.Variant, width int) bool {
	if width == 0 {
		return true
	}

	variantWidth := variant.MetadataTags.Get(registry_dto.TagWidth)
	if variantWidth == "" {
		return false
	}

	requestedWidthPx := fmt.Sprintf("%dpx", width)
	requestedWidthRaw := fmt.Sprintf("%d", width)

	return variantWidth == requestedWidthPx || variantWidth == requestedWidthRaw
}

// variantMatchesDensity checks if a variant's density matches the requested
// density.
//
// Takes variant (*registry_dto.Variant) which is the variant to check.
// Takes density (string) which is the requested density to match against.
//
// Returns bool which is true if density is empty or matches the variant's
// density tag.
func variantMatchesDensity(variant *registry_dto.Variant, density string) bool {
	if density == "" {
		return true
	}

	return variant.MetadataTags.Get(registry_dto.TagDensity) == density
}

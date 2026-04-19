---
title: Media API
description: Image and video processing with transformations, responsive variants, low-quality image placeholders, and streaming transcoding.
nav:
  sidebar:
    section: "reference"
    subsection: "services"
    order: 170
---

# Media API

Piko's media service transforms images and videos. It resizes, re-encodes, builds responsive variant sets, generates low-quality placeholders, transcodes video, and emits HLS or DASH streams. The service is provider-agnostic. The default image backend is libvips and the default video backend is ffmpeg via astiav. For task recipes see the [assets how-to](../how-to/assets.md). Source of truth: [`wdk/media/facade.go`](https://github.com/piko-sh/piko/blob/master/wdk/media/facade.go).

## Service

| Function | Returns |
|---|---|
| `media.NewService(defaultTransformerName string) ImageService` | Constructs a new image service. |
| `media.GetDefaultService() (ImageService, error)` | Returns the bootstrap-configured service. |
| `media.GetImageDimensions(ctx, reader)` | Width and height of an image. |

## Transform builder

```go
func NewTransformBuilder(service ImageService, input io.Reader) *TransformBuilder
func NewTransformBuilderFromDefault(input io.Reader) (*TransformBuilder, error)
```

`TransformBuilder` is a fluent wrapper around `ImageService.TransformStream`. Each setter mutates the underlying `TransformationSpec` and returns the builder. `Do` executes the spec (returning a result), and `DoToWriter` streams directly to an `io.Writer`.

### Sizing

| Method | Effect |
|---|---|
| `.Width(px int)` | Set target width. `0` preserves aspect ratio from the height. |
| `.Height(px int)` | Set target height. `0` preserves aspect ratio from the width. |
| `.Size(width, height int)` | Set both axes at once. |
| `.MaxWidth(px int)` | Set width and clear height (scale to fit width). |
| `.MaxHeight(px int)` | Set height and clear width (scale to fit height). |
| `.AspectRatio(ratio string)` | Force an aspect ratio (`"16:9"`, `"4:3"`, `"1:1"`). |
| `.WithoutEnlargement()` | Prevent scaling beyond the source's natural size. |

### Fit

| Method | Equivalent to |
|---|---|
| `.Fit(mode FitMode)` | Set the fit mode explicitly. |
| `.Cover()` | `Fit(FitCover)`. Fills the box, cropping excess. |
| `.Contain()` | `Fit(FitContain)`. Letterboxes to keep the whole image. |
| `.Fill()` | `Fit(FitFill)`. Stretches to exact dimensions. |
| `.Inside()` | `Fit(FitInside)`. Never exceeds bounds. |
| `.Outside()` | `Fit(FitOutside)`. At least covers bounds. |

### Output and modifiers

| Method | Effect |
|---|---|
| `.Format(format string)` | Output format (`"jpeg"`, `"png"`, `"webp"`, `"avif"`, `"gif"`). |
| `.Quality(q int)` | Compression quality `1`-`100`. |
| `.Background(hex string)` | Letterbox or transparency-fill colour (`"#RRGGBB"`). |
| `.Provider(name string)` | Use a specific provider instead of the service default. |
| `.WithModifier(key, value string)` | Add a provider-specific modifier. |
| `.Blur(sigma float64)` | Convenience: sets the `blur` modifier. |
| `.Greyscale()` | Convenience: sets the `greyscale` modifier. |

### Variants and specs

| Method | Effect |
|---|---|
| `.WithPredefinedVariants(map[string]TransformationSpec)` | Register named variant lookups for `UseVariant`. |
| `.UseVariant(name string)` | Replace the current spec with a registered variant. |
| `.FromSpec(spec TransformationSpec)` | Replace the current spec with the supplied one. |
| `.Spec() TransformationSpec` | Return the current spec without executing. |

### Terminals

| Method | Effect |
|---|---|
| `.Do(ctx context.Context)` | Execute and return `*TransformedImageResult`. |
| `.DoToWriter(ctx context.Context, w io.Writer) error` | Execute and stream the output to `w`. |

### Responsive and placeholder generation

`ImageService` calls produce responsive variants and low-quality placeholders, not `TransformBuilder` methods.

```go
service.GenerateResponsiveVariants(ctx, input, baseSpec) // reads baseSpec.Responsive *ResponsiveSpec
service.GeneratePlaceholder(ctx, input, baseSpec)        // reads baseSpec.Placeholder *PlaceholderSpec
```

`TransformationSpec` carries optional `Responsive *ResponsiveSpec` and `Placeholder *PlaceholderSpec` fields that those methods consume.

## Config and spec builders

```go
Image() *ImageConfigBuilder
Variant() *VariantBuilder
GetPredefinedVariants() map[string]TransformationSpec
GetVariantSpec(name string) (TransformationSpec, bool)
```

## Types

### Images

| Type | Purpose |
|---|---|
| `ImageService` | Manages image providers. |
| `ImageTransformerPort` | Interface a backend implements. |
| `ImageServiceConfig` | Service defaults. |
| `TransformationSpec` | Named transformation. |
| `FitMode` | Crop or letterbox behaviour. |
| `ResponsiveSpec` | Generates multi-width variants. |
| `ResponsiveVariant` | One variant in a responsive set. |
| `PlaceholderSpec` | Low-quality image placeholder. |
| `ImageConfig` | Per-request settings. |
| `ImageConfigBuilder`, `VariantBuilder`, `TransformBuilder` | Fluent builders. |
| `TransformedImageResult` | Output blob plus metadata. |

### Videos

| Type | Purpose |
|---|---|
| `VideoService` | Manages video providers. |
| `VideoTranscoderPort` | Interface a backend implements. |
| `StreamingTranscoderPort` | Streaming variant. |
| `VideoServiceConfig` | Service defaults. |
| `TranscodeSpec` | Output format and encoding. |
| `ThumbnailSpec` | Frame extraction spec. |
| `VideoCapabilities` | Declares provider support. |
| `HLSSpec`, `HLSResult`, `HLSVariant`, `HLSSegment` | HLS streaming. |
| `DASHSpec`, `DASHResult` | DASH streaming. |
| `TranscodeResult` | Output metadata. |

## Fit modes

| Constant | Meaning |
|---|---|
| `FitCover` | Fill the box, cropping as needed. |
| `FitContain` | Letterbox: the whole image is visible. |
| `FitFill` | Stretch to the exact box without preserving aspect. |
| `FitInside` | Preserve aspect, do not exceed bounds. |
| `FitOutside` | Preserve aspect, cover bounds. |

## Errors

`ErrUnsupportedCodec`, `ErrUnsupportedFormat`, `ErrInvalidResolution`, `ErrInvalidBitrate`, `ErrInvalidFramerate`, `ErrDurationExceedsLimit`, `ErrFileSizeExceedsLimit`, `ErrResolutionExceedsLimit`, `ErrTranscodingFailed`, `ErrInvalidStream`, `ErrContextCancelled`, `ErrTimeout`, `ErrResourceExhausted`, `ErrInvalidHLSSpec`, `ErrSegmentationFailed`.

## Providers

| Sub-package | Backend |
|---|---|
| `image_provider_vips` | libvips (high performance). |
| `image_provider_imaging` | Pure-Go imaging library. |
| `video_provider_astiav` | ffmpeg via astiav. |

## Bootstrap options

| Option | Purpose |
|---|---|
| `piko.WithImageProvider(name, provider)` | Registers an image provider. |
| `piko.WithDefaultImageProvider(name)` | Marks default. |
| `piko.WithImage(cfg)` | Registers per-request image defaults. |
| `piko.WithImageService(service)` | Registers a fully configured service. |
| `piko.WithVideoProvider(name, provider)` | Registers a video provider. |
| `piko.WithDefaultVideoProvider(name)` | Marks default. |
| `piko.WithVideoService(service)` | Registers a fully configured service. |

## See also

- [How to assets](../how-to/assets.md) for responsive-image recipes.

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

Fluent methods: `.Size(w, h)`, `.Fit(mode)`, `.Format(fmt)`, `.Quality(q)`, `.Variant(name)`, `.WithVariant(spec)`, `.Responsive(spec)`, `.Placeholder(spec)`, `.Do(ctx)`.

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

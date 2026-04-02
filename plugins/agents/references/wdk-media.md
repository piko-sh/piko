# WDK Media

Use this guide when adding image processing, video transcoding, or media transformations.

## Overview

Media services are optional. When no providers are registered, Piko components degrade gracefully - `piko:img` renders basic `<img>` tags, `piko:video` renders basic `<video>` tags.

## Image processing

### Providers

| Provider | Package | Requirements | Production ready |
|----------|---------|--------------|-----------------|
| imaging | `image_provider_imaging` | None (pure Go) | Development only |
| vips | `image_provider_vips` | libvips (CGO) | Yes |
| mock | `image_provider_mock` | None | Testing |

**Warning**: The pure Go imaging provider uses lossless WebP (VP8L) which consumes 100-500MB per image. Use vips for production.

### Setup

```go
import (
    "piko.sh/piko"
    "piko.sh/piko/wdk/media/image_provider_vips"
)

provider, err := image_provider_vips.NewProvider(image_provider_vips.Config{
    ConcurrencyLevel: 4,
})
defer provider.Close()

app := piko.New(
    piko.WithImageProvider("vips", provider),
)
```

System requirements: `apt-get install libvips-dev` (Ubuntu) or `brew install vips` (macOS).

### Transform builder

```go
import "piko.sh/piko/wdk/media"

builder, err := media.NewTransformBuilderFromDefault(reader)

result, err := builder.
    Size(200, 200).
    Format("webp").
    Quality(80).
    Cover().
    Do(ctx)
defer result.Body.Close()
```

### Image configuration with variants

```go
config, err := media.Image().
    Provider("vips", vipsProvider).
    MaxFileSizeMB(50).
    MaxDimensions(8192, 8192).
    WithVariant("thumb", media.Variant().
        Size(200, 200).Format("webp").Quality(80).Cover().Build()).
    WithVariant("preview", media.Variant().
        MaxWidth(800).Format("webp").Quality(85).Contain().Build()).
    Build()

app := piko.New(piko.WithImage(config))
```

### Fit modes

| Mode | Behaviour |
|------|-----------|
| `cover` | Fill dimensions, crop excess |
| `contain` | Fit within dimensions, letterbox if needed (default) |
| `fill` | Stretch to exact dimensions |
| `inside` | Resize to be ≤ dimensions |
| `outside` | Resize to be ≥ dimensions |

## Video processing

### Providers

| Provider | Package | Requirements |
|----------|---------|--------------|
| astiav | `video_provider_astiav` | FFmpeg libs (CGO), `ffmpeg` build tag |
| mock | `video_provider_mock` | None |

Build with: `go build -tags ffmpeg ./...`

### Setup

```go
import "piko.sh/piko/wdk/media/video_provider_astiav"

provider, err := video_provider_astiav.NewProvider(video_provider_astiav.Config{
    MaxConcurrentTranscodes: 4,
    EnableHWAccel:           true,
})
```

### Transcoding

```go
spec := media.TranscodeSpec{
    Codec:    "h264",
    Width:    1920,
    Height:   1080,
    Bitrate:  5_000_000,
    Preset:   "medium",
    CRF:      intPtr(23),
}

outputReader, err := provider.Transcode(ctx, inputReader, spec)
```

### Thumbnail extraction

```go
spec := media.NewThumbnailSpec()
spec.Timestamp, _ = media.ParseThumbnailTime("1:30")
spec.Width = 640
spec.Format = "jpeg"

thumbnail, err := provider.ExtractThumbnail(ctx, videoReader, spec)
```

## LLM mistake checklist

- Using the pure Go imaging provider in production (memory-intensive, lossless WebP only)
- Forgetting `defer provider.Close()` for vips (leaks libvips resources)
- Not adding `-tags ffmpeg` build tag for video processing
- Forgetting to install system dependencies (libvips-dev or FFmpeg)
- Setting quality above 100 or below 1

## Related

- `references/wdk-data.md` - storing processed media in object storage

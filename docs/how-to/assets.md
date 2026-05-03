---
title: How to configure asset transformations
description: Register an image or video provider, declare reusable variants, and reference them from templates.
nav:
  sidebar:
    section: "how-to"
    subsection: "operations"
    order: 50
---

# How to configure asset transformations

Piko transforms images and videos through pluggable providers. Piko ships libvips (`wdk/media/image_provider_vips`) and astiav (`wdk/media/video_provider_astiav`) implementations, plus mock providers for tests. A project registers one or more providers, optionally declares predefined variants, and references them from the `<piko:img>` and `<piko:video>` tags in templates. See the [bootstrap options reference](../reference/bootstrap-options.md) for the full option surface.

## Register an image provider

Use the `media.Image()` builder to compose the image configuration. The builder accepts one or more providers and produces an `*ImageConfig` for `piko.WithImage`:

```go
import (
    "piko.sh/piko"
    "piko.sh/piko/wdk/media"
    "piko.sh/piko/wdk/media/image_provider_vips"
)

vipsProvider, err := image_provider_vips.NewProvider(image_provider_vips.Config{})
if err != nil {
    log.Fatal(err)
}

imageConfig, err := media.Image().
    Provider("vips", vipsProvider).
    MaxFileSizeMB(50).
    DefaultQuality(85).
    Build()
if err != nil {
    log.Fatal(err)
}

ssr := piko.New(
    piko.WithImage(imageConfig),
)
```

The `vips` build tag (`go build -tags vips`) gates the libvips provider, which links against the system `libvips` library. For a pure-Go alternative use `image_provider_imaging`. For tests, `image_provider_mock` returns deterministic output without performing any work.

## Declare reusable variants

A variant is a named `TransformationSpec`. The builder produces specs you can register up front and reference by name from template tags.

```go
imageConfig, err := media.Image().
    Provider("vips", vipsProvider).
    WithVariant("thumb", media.Variant().
        Size(200, 200).
        Format("webp").
        Quality(80).
        Cover().
        Build(),
    ).
    WithVariant("hero", media.Variant().
        MaxWidth(1920).
        Format("avif").
        Quality(82).
        Build(),
    ).
    Build()
```

`media.Variant()` exposes `Size`, `MaxWidth`, `MaxHeight`, `Format`, `Quality`, `Fit`/`Cover`/`Contain`/`Fill`/`Inside`/`Outside`, `Background`, `AspectRatio`, and modifier helpers (`Blur`, `Greyscale`). See `wdk/media/facade.go` for the full list.

To start with sensible defaults plus Piko's predefined variants (`thumb_100`, `thumb_200`, `thumb_400`, `preview_800`, `lqip`), call `FromDefaults()`:

```go
imageConfig, _ := media.Image().
    FromDefaults().
    Provider("vips", vipsProvider).
    Build()
```

## Use an image in a template

Declare the profile in `piko.config.json` (or the wizard YAML) under `assets.image.profiles`. The `profile=` attribute on `<piko:img>` resolves against this map at compile time. `media.Image().WithVariant(...)` registrations are for runtime callers of the media service, not for the `profile=` attribute.

```json
{
  "assets": {
    "image": {
      "profiles": {
        "thumb": [
          {
            "capability": "image-transform",
            "params": {
              "width": "200",
              "height": "200",
              "format": "webp",
              "quality": "80",
              "fit": "cover"
            }
          }
        ]
      }
    }
  }
}
```

```piko
<template>
  <piko:img src="/assets/hero.jpg" profile="thumb" alt="A hero image" />
</template>
```

See [about configuration](../explanation/about-configuration.md) for the config layering and [bootstrap options reference](../reference/bootstrap-options.md) for the runtime media API.

## Serve videos

Register a video transcoder the same way through `WithVideoProvider`:

```go
import (
    "piko.sh/piko/wdk/media/video_provider_astiav"
)

videoProvider, err := video_provider_astiav.NewProvider(video_provider_astiav.Config{})
if err != nil {
    log.Fatal(err)
}

ssr := piko.New(
    piko.WithVideoProvider("astiav", videoProvider),
    piko.WithDefaultVideoProvider("astiav"),
)
```

Astiav links against the system FFmpeg libraries behind a build tag. Reference videos from templates using `<piko:video>`:

```piko
<template>
  <piko:video src="/assets/intro.mp4" />
</template>
```

## See also

- [Bootstrap options reference](../reference/bootstrap-options.md) for `WithImage`, `WithImageProvider`, `WithVideoProvider`, and related options.
- `wdk/media/facade.go` for the full image and variant builder API.
- `wdk/media/image_provider_vips`, `image_provider_imaging`, `image_provider_mock` for shipped image providers.
- `wdk/media/video_provider_astiav`, `video_provider_mock` for shipped video providers.

---
title: How to configure asset transformations
description: Define image and video transformation profiles, generate responsive variants, and tune caching.
nav:
  sidebar:
    section: "how-to"
    subsection: "operations"
    order: 50
---

# How to configure asset transformations

Piko transforms images and videos at build time and on demand using pluggable providers. A project declares transformation profiles (breakpoints, densities, formats) and the framework serves the right variant per request. This guide covers the common setup. See the [bootstrap options reference](../reference/bootstrap-options.md) for `WithImageProvider` and related options.

## Register an image provider

Use a concrete backend (libvips, imaginary, a custom HTTP transformer) through `WithImageProvider`:

```go
import (
    "piko.sh/piko"
    "piko.sh/piko/adapters/image/vips"
)

ssr := piko.New(
    piko.WithImageProvider("vips", vips.NewProvider(vips.Config{
        MaxWidth:  4096,
        MaxHeight: 4096,
    })),
    piko.WithDefaultImageProvider("vips"),
)
```

The provider's transform operations are what `<img>` directives in templates call into.

## Declare transformation profiles

Transformation profiles describe the variants Piko generates for each source image. Pass them via `WithImage`:

```go
import "piko.sh/piko/internal/image/image_domain"

ssr := piko.New(
    piko.WithImage(&image_domain.ImageConfig{
        Breakpoints: []int{320, 640, 960, 1280, 1920},
        Densities:   []int{1, 2, 3},
        Formats:     []string{"avif", "webp", "jpeg"},
        Quality:     82,
    }),
)
```

Piko emits a `<picture>` element with one `<source>` per format and a `srcset` per breakpoint. The browser picks the best variant for its viewport and pixel density.

## Use an image in a template

The `<piko:image>` tag handles the transformation pipeline:

```piko
<template>
  <piko:image
    src="/assets/hero.jpg"
    alt="A hero image"
    sizes="(max-width: 768px) 100vw, 768px"
  />
</template>
```

The framework generates the appropriate `srcset` values, fills in `<source>` elements for each declared format, and applies lazy loading by default. Specify `eager="true"` for above-the-fold images that should not wait for viewport intersection.

## Add a Low-Quality Image Placeholder

Enable a Low-Quality Image Placeholder to render a blurred thumbnail while the full image loads:

```go
piko.WithImage(&image_domain.ImageConfig{
    // ...
    LQIPEnabled: true,
    LQIPSize:    16, // pixels; kept tiny for a compact inline data URI
})
```

The placeholder renders as an inline `data:` URI, so it adds no network round-trip.

## Serve videos

Piko supports video the same way through `WithVideoProvider`:

```go
import "piko.sh/piko/adapters/video/ffmpeg"

ssr := piko.New(
    piko.WithVideoProvider("ffmpeg", ffmpeg.NewProvider()),
    piko.WithDefaultVideoProvider("ffmpeg"),
)
```

The `<piko:video>` tag generates `<source>` elements per codec.

## Override per-image behaviour

A project can opt a specific image out of transformation:

```piko
<piko:image src="/assets/logo.svg" alt="Logo" raw />
```

`raw` skips the transform pipeline and serves the source file unmodified. Use for vector graphics and hand-optimised images.

## Serving from a CDN

The asset URLs Piko emits are relative by default. Front them with a CDN by setting the public base URL in `piko.yaml`:

```yaml
assets:
  publicBaseURL: "https://cdn.example.com"
```

The pipeline prefixes every asset URL with the configured base.

## See also

- [Bootstrap options reference](../reference/bootstrap-options.md) for the full image, video, and asset configuration surface.
- Integration tests: [`tests/integration/asset_pipeline/`](https://github.com/piko-sh/piko/tree/master/tests/integration/asset_pipeline) exercises every transformation.

---
title: How to transform images
description: Resize, re-encode, generate responsive variants, and emit low-quality placeholders with the media service.
nav:
  sidebar:
    section: "how-to"
    subsection: "services"
    order: 90
---

# How to transform images

This guide covers resizing, re-encoding, building responsive variant sets, and generating Low-Quality Image Placeholder with Piko's media service. See the [media reference](../reference/media-api.md) for the full API.

## Register a backend

```go
package main

import (
    "piko.sh/piko"
    "piko.sh/piko/wdk/media/image_provider_vips"
)

func main() {
    ssr := piko.New(
        piko.WithImageProvider("vips", image_provider_vips.New()),
        piko.WithDefaultImageProvider("vips"),
    )
    ssr.Run()
}
```

Use `image_provider_imaging.New()` for a pure-Go backend when libvips is unavailable.

## Resize a single image

```go
import "piko.sh/piko/wdk/media"

func resize(ctx context.Context, in io.Reader) (*media.TransformedImageResult, error) {
    builder, err := media.NewTransformBuilderFromDefault(in)
    if err != nil {
        return nil, err
    }
    return builder.Size(1200, 0).Fit(media.FitInside).Format("webp").Quality(80).Do(ctx)
}
```

A zero dimension preserves aspect ratio. `FitInside` keeps the image within the box.

## Build a responsive set

```go
func responsive(ctx context.Context, in io.Reader) ([]*media.TransformedImageResult, error) {
    builder, err := media.NewTransformBuilderFromDefault(in)
    if err != nil {
        return nil, err
    }

    spec := media.ResponsiveSpec{
        Variants: []media.ResponsiveVariant{
            {Width: 320, Format: "webp"},
            {Width: 640, Format: "webp"},
            {Width: 1280, Format: "webp"},
        },
    }

    return builder.Responsive(spec).DoAll(ctx)
}
```

Pair the outputs with a `<picture>` element in templates for browser-selected resolutions.

## Emit a low-quality placeholder

```go
spec := media.PlaceholderSpec{Width: 32, Quality: 30, Blur: 2.0}
placeholder, err := builder.Placeholder(spec).Do(ctx)
```

Render the base64-encoded placeholder inline, and let the full-size image load over the top.

## Use predefined variants

```go
for name := range media.GetPredefinedVariants() {
    fmt.Println(name)
}
```

Reference a named variant with `.Variant(name)` to reuse a transformation spec across pages.

## See also

- [Media API reference](../reference/media-api.md) for the full API.
- [How to assets](assets.md) for integrating transformations into the asset pipeline.
- [How to file storage](file-storage.md) for storing the transformed outputs.

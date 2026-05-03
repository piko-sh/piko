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
    vipsProvider, err := image_provider_vips.NewProvider(image_provider_vips.Config{})
    if err != nil {
        panic(err)
    }
    ssr := piko.New(
        piko.WithImageProvider("vips", vipsProvider),
        piko.WithDefaultImageProvider("vips"),
    )
    if err := ssr.Run(piko.RunModeProd); err != nil {
        panic(err)
    }
}
```

For a pure-Go backend when libvips is unavailable, use `image_provider_imaging.NewProvider(image_provider_imaging.Config{})` from `piko.sh/piko/wdk/media/image_provider_imaging`.

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

The service produces responsive variants, not the transform builder. Pass a `TransformationSpec` with the `Responsive` field set.

```go
func responsive(ctx context.Context, in io.Reader) ([]media.ResponsiveVariant, error) {
    service, err := media.GetDefaultService()
    if err != nil {
        return nil, err
    }

    spec := media.DefaultTransformationSpec()
    spec.Format = "webp"
    spec.Responsive = &media.ResponsiveSpec{
        Screens:   map[string]int{"sm": 640, "md": 768, "lg": 1024, "xl": 1280},
        Sizes:     "100vw sm:50vw md:33vw lg:25vw",
        Densities: []string{"x1", "x2"},
    }

    return service.GenerateResponsiveVariants(ctx, in, spec)
}
```

`Screens` maps breakpoint names to pixel widths, `Sizes` is a viewport-style descriptor, and `Densities` lists the pixel-density multipliers to emit. Pair the outputs with a `<picture>` element or a `srcset` attribute (each variant exposes a ready-made `SrcsetEntry`).

## Emit a low-quality placeholder

The placeholder builder is also service-level. Set `Placeholder.Enabled = true` and tune the size, quality, and blur:

```go
spec := media.DefaultTransformationSpec()
spec.Placeholder = &media.PlaceholderSpec{
    Enabled:   true,
    Width:     20,
    Quality:   10,
    BlurSigma: 5.0,
}

service, err := media.GetDefaultService()
if err != nil {
    return err
}

dataURL, err := service.GeneratePlaceholder(ctx, in, spec)
```

`GeneratePlaceholder` returns a base64 data URL ready to drop into a `src` attribute. Render it inline and let the full-size image load over the top.

## Use predefined variants

```go
for name := range media.GetPredefinedVariants() {
    fmt.Println(name)
}
```

Reference a named variant with `.UseVariant(name)` on a transform builder to reuse a transformation spec across pages.

## See also

- [Media API reference](../reference/media-api.md) for the full API.
- [How to assets](assets.md) for integrating transformations into the asset pipeline.
- [How to file storage](file-storage.md) for storing the transformed outputs.

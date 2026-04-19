---
title: PDF API
description: PDF rendering from Piko templates with metadata, watermarks, PDF/A compliance, and post-processing transformers.
nav:
  sidebar:
    section: "reference"
    subsection: "services"
    order: 150
---

# PDF API

Piko renders PDFs from PK templates using a fluent builder. The render pipeline supports metadata, watermarks, PDF/A conformance, accessibility tagging, custom page labels, custom fonts, and post-processing transformers for encryption and signing. For the design rationale see [about PDF](../explanation/about-pdf.md). For task recipes see [how to PDF generation](../how-to/pdf-generation.md). Source of truth: [`wdk/pdf/facade.go`](https://github.com/piko-sh/piko/blob/master/wdk/pdf/facade.go).

## Service

| Function | Returns |
|---|---|
| `pdf.GetDefaultService() (Service, error)` | Returns the bootstrap-configured service. |
| `pdf.NewRenderBuilder(service) (*RenderBuilder, error)` | Constructs a render builder bound to the given service. |
| `pdf.NewRenderBuilderFromDefault() (*RenderBuilder, error)` | Shortcut for the default service. |

## Builder

The fluent `RenderBuilder` composes a single render operation. Chain `With*` setters, then call `Do(ctx)` to execute.

| Method group | Methods |
|---|---|
| Template binding | `Template(path)`, `Request(r)`, `Props(propsStruct)` |
| Output structure | `PageConfig(cfg)`, `Fonts([]FontEntry)`, `PageLabels([]PageLabelRange)` |
| Metadata | `Metadata(meta)`, `ViewerPreferences(prefs)` |
| Watermarks | `Watermark(cfg)` |
| Compliance | `PdfA(cfg)`, `Accessibility(opts)` |
| Post-processing | `Transform(cfg)` |
| Execution | `Do(ctx)` (returns `*Result`, `error`) |

## Types

| Type | Purpose |
|---|---|
| `Service` | Orchestrates the render pipeline. |
| `Result` | Rendered bytes (`Bytes []byte`) plus `PageCount`. |
| `Config` | Full render configuration. |
| `Metadata` | PDF info dictionary (title, author, subject, keywords, creator, producer, creation and modification dates). |
| `ViewerPreferences` | Controls reader display flags (full-screen mode, hide-menubar, page layout). |
| `WatermarkConfig` | Text, opacity, rotation, font, placement. |
| `PdfAConfig` | Target level and embedded ICC profile. |
| `PdfALevel` | Enum (`PdfA2B`, `PdfA2U`, `PdfA2A`). |
| `PageLabelRange` | Custom labelling for a range of pages. |
| `PageLabelStyle` | Enum (`LabelDecimal`, `LabelRomanUpper`, `LabelRomanLower`, `LabelAlphaUpper`, `LabelAlphaLower`, `LabelNone`). |
| `PageConfig` | Dimensions and margins. |
| `FontEntry` | Embedded custom font. |
| `PainterConfig` | Low-level painter tuning. |

## Constants

| Group | Values |
|---|---|
| PDF/A level | `PdfA2B` (basic), `PdfA2U` (Unicode), `PdfA2A` (accessible, auto-enables tagged PDF) |
| Page label style | `LabelDecimal`, `LabelRomanUpper`, `LabelRomanLower`, `LabelAlphaUpper`, `LabelAlphaLower`, `LabelNone` |
| Page size | `PageA4`, `PageA3`, `PageLetter` |
| Transformer type | `TransformerContent`, `TransformerCompliance`, `TransformerDelivery`, `TransformerSecurity`, `TransformerCompression` |
| Compression algorithm | `CompressZstd` |

## Post-processing transformers

Transformers run after the initial render pass. Register them through a `TransformerRegistry`, then attach the registry to the builder or via bootstrap. Each transformer has a paired option struct that tunes its behaviour.

| Constructor | Purpose |
|---|---|
| `NewEncryptTransformer()` | AES-256 encryption of the output PDF. |
| `NewPadesSignTransformer()` | PAdES digital signature. |

### Option structs

| Struct | Applies to | Key fields |
|---|---|---|
| `EncryptionOptions` | Encryption transformer | Owner password, user password, permission flags (print, copy, modify, annotate). |
| `PadesSignOptions` | PAdES signing transformer | Signer certificate chain, private key, timestamp authority URL, signature reason and location. |
| `WatermarkOptions` | Watermark transformer | Text, font, size, colour, opacity, rotation, horizontal/vertical alignment. |
| `PdfAOptions` | PDF/A transformer | Target level (`PdfA2B`, `PdfA2U`, `PdfA2A`), embedded ICC profile, output intent. |
| `PdfUAOptions` | PDF/UA transformer | Tagged PDF settings for accessibility conformance. |
| `LineariseOptions` | Linearisation transformer | Fast web view layout. |
| `ObjStmOptions` | Object-stream transformer | Object-stream packing thresholds. |
| `FlattenOptions` | Flattening transformer | Form-field flattening, annotation flattening. |
| `RedactionOptions` | Redaction transformer | Page regions to redact (`[]RedactionRegion`), fill colour. |
| `CompressOptions` | Compression transformer | Algorithm (`CompressZstd`), level. |

`RedactionRegion` defines a rectangular area by page index and bounding box coordinates.

### Registry

```go
func NewTransformerRegistry() *TransformerRegistry
```

Methods on `*TransformerRegistry`: `Register(transformer TransformerPort)`, `Lookup(typ TransformerType, name string) (TransformerPort, bool)`, `Types() []TransformerType`.

### Transformer interface

| Type | Purpose |
|---|---|
| `TransformerPort` | Interface. Method: `Transform(ctx, input []byte, options any) ([]byte, error)`. |
| `TransformerType` | Enum of transformer categories. |
| `TransformConfig` | Bundle of transformer selections passed to the builder. |

## Rendering SVG inside a PDF

`SVGWriterPort` renders SVG content as native PDF vector commands. `SVGDataPort` provides raw SVG markup. Users who need custom SVG-to-PDF conversion implement one of these ports and plug it into the painter configuration.

## Bootstrap

PDF does not expose a dedicated `With*` option. Piko constructs the default service from the PDF manifest generated by `piko generate` when a `pdfs/` directory is present in the project. Custom transformers register through the `TransformerRegistry`:

```go
registry := pdf.NewTransformerRegistry()
registry.Register(pdf.NewEncryptTransformer())
registry.Register(pdf.NewPadesSignTransformer())
```

Pass the registry to the builder via `Transform(cfg)`.

## See also

- [About PDF](../explanation/about-pdf.md) for why PDFs sit on the same template substrate and when to reach for PDF/A or digital signatures.
- [How to PDF generation](../how-to/pdf-generation.md) for rendering a PK template, adding metadata, watermarks, and encryption.
- [Scenario 027: PDF invoice](../showcase/027-pdf-invoice.md) for a runnable example.

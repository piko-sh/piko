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

Piko renders PDF output from PK templates using a fluent builder. The render pipeline supports metadata, watermarks, PDF/A conformance, accessibility tagging via `TaggedPDF()`, and custom page labels. It also supports additional CSS stylesheets and font-size and line-height overrides. The pipeline also supports native SVG-to-PDF vector rendering and post-processing transformers for encryption and signing. For the design rationale see [about PDF](../explanation/about-pdf.md). For task recipes see [how to PDF generation](../how-to/pdf-generation.md). Source of truth: [`wdk/pdf/facade.go`](https://github.com/piko-sh/piko/blob/master/wdk/pdf/facade.go).

## Service

| Function | Returns |
|---|---|
| `pdf.GetDefaultService() (Service, error)` | Returns the bootstrap-configured service. |
| `pdf.NewRenderBuilder(service) (*RenderBuilder, error)` | Constructs a render builder bound to the given service. |
| `pdf.NewRenderBuilderFromDefault() (*RenderBuilder, error)` | Shortcut for the default service. |

## Builder

The fluent `RenderBuilder` composes a single render operation. Chain setters, then call `Do(ctx)` to execute.

| Method group | Methods |
|---|---|
| Template binding | `Template(path)`, `Request(r)`, `Props(propsStruct)` |
| Output structure | `Page(cfg PageConfig)`, `PageLabels(ranges ...PageLabelRange)`, `FontSize(size float64)`, `LineHeight(height float64)` |
| Stylesheets | `Stylesheet(css string)` (callable multiple times) |
| Metadata | `Metadata(meta Metadata)`, `ViewerPreferences(prefs ViewerPreferences)` |
| Watermarks | `Watermark(text string)`, `WatermarkConfig(cfg WatermarkConfig)` |
| Compliance | `PdfA(level PdfALevel)`, `TaggedPDF()` |
| SVG rendering | `SVGWriter(writer SVGWriterPort, data SVGDataPort)` |
| Post-processing | `Transformations(registry *TransformerRegistry, config TransformConfig)` |
| Execution | `Do(ctx) (*Result, error)` |

`Watermark(text)` is the convenience form using default styling (60pt, light grey, 45 degrees). Use `WatermarkConfig` for full control. `PdfA(PdfA2A)` enables tagged-PDF accessibility output automatically. The builder has no `Fonts` setter - the service configures fonts at construction time, not per render.

## Types

| Type | Purpose |
|---|---|
| `Service` | Orchestrates the render pipeline. |
| `Result` | Rendered output. Fields: `Content []byte` (the PDF bytes), `PageCount int` (number of pages), and `LayoutDump string` (debug-only human-readable serialisation of the layout box tree, empty when not requested). |
| `Config` | Full render configuration. |
| `Metadata` | PDF info dictionary (title, author, subject, keywords, creator, producer, creation and modification dates). |
| `ViewerPreferences` | Controls reader display flags (full-screen mode, hide-menubar, page layout). |
| `WatermarkConfig` | Builder-level watermark settings (text, opacity, rotation, placement) used by `WatermarkConfig(cfg)`. |
| `PdfAConfig` | Builder-level PDF/A target level. |
| `PdfALevel` | Enum (`PdfA2B`, `PdfA2U`, `PdfA2A`). |
| `PageLabelRange` | Custom labelling for a range of pages. |
| `PageLabelStyle` | Enum (`LabelDecimal`, `LabelRomanUpper`, `LabelRomanLower`, `LabelAlphaUpper`, `LabelAlphaLower`, `LabelNone`). |
| `PageConfig` | Page dimensions and margins (alias of `layouter_dto.PageConfig`). |
| `FontEntry` | Font embedding descriptor used at service construction (alias of `layouter_dto.FontEntry`). |
| `PainterConfig` | Low-level painter tuning. |
| `SVGWriterPort` | Renders SVG markup as native PDF vector commands. |
| `SVGDataPort` | Provides raw SVG markup for a given source. |
| `TransformerRegistry` | Holds available PDF post-processing transformers. |
| `TransformerPort` | Interface implemented by post-processing transformers. |
| `TransformerType` | Enum of transformer categories. |
| `TransformConfig` | Specifies which post-processing transformers to apply. |

## Constants

| Group | Values |
|---|---|
| PDF/A level | `PdfA2B` (basic), `PdfA2U` (Unicode), `PdfA2A` (accessible, auto-enables tagged PDF) |
| Page label style | `LabelDecimal`, `LabelRomanUpper`, `LabelRomanLower`, `LabelAlphaUpper`, `LabelAlphaLower`, `LabelNone` |
| Page size | `PageA4`, `PageA3`, `PageLetter` |
| Transformer type | `TransformerContent`, `TransformerCompliance`, `TransformerDelivery`, `TransformerSecurity`, `TransformerCompression` |
| Compression algorithm | `CompressZstd` |

## Post-processing transformers

Transformers run after the initial render pass. Register them through a `TransformerRegistry`, then attach the registry to the builder via `Transformations(registry, config)`. Each transformer has a paired option struct that tunes its behaviour.

| Constructor | Purpose |
|---|---|
| `NewEncryptTransformer(opts ...EncryptOption) TransformerPort` | AES-256 CBC encryption of the output PDF. Pass `WithEncryptRandomSource(r)` to override the cryptographic randomness source. Production callers can omit options. |
| `NewPadesSignTransformer() TransformerPort` | PAdES digital signature. |

### Option structs

Transformer options live in `pdfwriter_dto`, and the `wdk/pdf` facade aliases them. The fields below match the source structs exactly.

#### `EncryptionOptions`

| Field | Type | Purpose |
|---|---|---|
| `Algorithm` | `string` | Encryption algorithm: `"aes-256"`, `"aes-128"`, or `"rc4-128"`. |
| `OwnerPassword` | `string` | Owner (permissions) password. |
| `UserPassword` | `string` | User (open) password. |
| `Permissions` | `uint32` | PDF permission flags bitmask (PDF spec table 22). Use the bitmask form instead of separate booleans. |

#### `PadesSignOptions`

| Field | Type | Purpose |
|---|---|---|
| `PrivateKey` | `crypto.Signer` | Signing key. |
| `Level` | `string` | PAdES conformance level: `"b-b"`, `"b-t"`, `"b-lt"`, or `"b-lta"`. |
| `TimestampURL` | `string` | `Time Stamping Authority` URL (required for `"b-t"` and above). |
| `Reason` | `string` | Stated reason for signing. |
| `Location` | `string` | Stated location of signing. |
| `ContactInfo` | `string` | Contact information for the signer. |
| `CertificateChain` | `[][]byte` | Signing certificate chain in DER encoding, ordered from end-entity to root. |
| `OCSPResponses` | `[][]byte` | Pre-fetched OCSP responses for long-term validation (`"b-lt"` and above). |
| `CRLs` | `[][]byte` | Pre-fetched CRLs for long-term validation (`"b-lt"` and above). |

#### `WatermarkOptions`

| Field | Type | Purpose |
|---|---|---|
| `Text` | `string` | Watermark text. Empty when using image-only watermarks. |
| `ImageData` | `[]byte` | Raw image bytes for an image watermark. Nil for text-only watermarks. |
| `Pages` | `[]int` | Page indices the watermark applies to. Empty means all pages. |
| `FontSize` | `float64` | Font size in points for text watermarks. |
| `ColourR` | `float64` | Red channel in `[0, 1]`. |
| `ColourG` | `float64` | Green channel in `[0, 1]`. |
| `ColourB` | `float64` | Blue channel in `[0, 1]`. |
| `Angle` | `float64` | Rotation angle in degrees. |
| `Opacity` | `float64` | Opacity in `[0, 1]`. |

#### `PdfAOptions`

| Field | Type | Purpose |
|---|---|---|
| `Level` | `string` | PDF/A conformance level. Allowed values: `"1b"`, `"2b"`, `"3b"`. |

The string `Level` is the transformer-side configuration. The builder's `PdfA(level PdfALevel)` setter takes the typed `PdfALevel` constants (`PdfA2B`, `PdfA2U`, `PdfA2A`) instead.

#### `PdfUAOptions`

Empty struct. The PDF/UA transformer takes no configuration. Passing it enables PDF/UA enhancement.

#### `LineariseOptions`

Empty struct. The linearisation transformer takes no configuration.

#### `ObjStmOptions`

Empty struct. The object-stream compression transformer takes no configuration.

#### `FlattenOptions`

| Field | Type | Purpose |
|---|---|---|
| `FormFields` | `bool` | Flatten interactive form fields into static content. |
| `Annotations` | `bool` | Flatten annotations into page content. |
| `Transparency` | `bool` | Flatten transparency groups. |

#### `RedactionOptions`

| Field | Type | Purpose |
|---|---|---|
| `TextPatterns` | `[]string` | Regular expression patterns to match and redact. |
| `Regions` | `[]RedactionRegion` | Page-specific rectangular regions to redact. |
| `StripMetadata` | `bool` | Strip document metadata (author, title, etc.). |

#### `CompressOptions`

| Field | Type | Purpose |
|---|---|---|
| `Algorithm` | `CompressAlgorithm` | Compression algorithm. Defaults to `CompressZstd` when empty. |

#### `RedactionRegion`

| Field | Type | Purpose |
|---|---|---|
| `Page` | `int` | Zero-based page index. |
| `X` | `float64` | Left edge in points. |
| `Y` | `float64` | Bottom edge in points. |
| `Width` | `float64` | Width in points. |
| `Height` | `float64` | Height in points. |

### Registry

```go
func NewTransformerRegistry() *TransformerRegistry
```

`*TransformerRegistry` is an alias for `*pdfwriter_domain.PdfTransformerRegistry`. It exposes four methods:

| Method | Purpose |
|---|---|
| `Register(transformer TransformerPort) error` | Adds a transformer. Returns an error when the transformer is nil, has an empty `Name()`, or another transformer already holds the same name. |
| `Get(name string) (TransformerPort, error)` | Looks up a transformer by name. Returns an error wrapping `ErrTransformerNotFound` when no transformer holds that name. |
| `Has(name string) bool` | Reports whether the registry holds a transformer with the given name. |
| `GetNames() []string` | Returns all registered transformer names in alphabetical order. |

### Transformer interface

`TransformerPort` is the interface implementations satisfy:

```go
type TransformerPort interface {
    Name() string
    Type() pdfwriter_dto.TransformerType
    Priority() int
    Transform(ctx context.Context, pdf []byte, options any) ([]byte, error)
}
```

| Method | Purpose |
|---|---|
| `Name()` | Unique identifier (for example `"watermark"`, `"aes-256"`, `"pades-b-b"`, `"linearise"`). |
| `Type()` | Category used for grouping and chain validation. |
| `Priority()` | Execution order. Lower values run first. Recommended ranges: 100-199 for content, 200-299 for compliance, 300-399 for delivery, 400-499 for security. |
| `Transform(ctx, pdf, options)` | Apply the transformation. `TransformConfig.TransformerOptions[Name()]` provides the `options` argument. |

`TransformConfig` is the bundle passed to the builder through `Transformations(registry, config)`. It has two fields:

| Field | Type | Purpose |
|---|---|---|
| `EnabledTransformers` | `[]string` | Names of transformers to apply. The chain sorts them by priority; this list is a declaration of intent, not an ordering directive. |
| `TransformerOptions` | `map[string]any` | Maps transformer names to their option structs. Each transformer casts the entry to its expected concrete type. |

`pdfwriter_dto.DefaultTransformConfig()` returns a `TransformConfig` with no transformers enabled and an empty options map.

## Rendering SVG inside a PDF

`SVGWriterPort` renders SVG content as native PDF vector commands. `SVGDataPort` provides raw SVG markup. Users who need custom SVG-to-PDF conversion implement one of these ports and plug it into the painter configuration.

## Bootstrap

PDF does not expose a dedicated `With*` option. Piko constructs the default service from the PDF manifest produced by the project's scaffolded generator (`cmd/generator/main.go`) when a `pdfs/` directory is present in the project. Custom transformers register through the `TransformerRegistry`:

```go
registry := pdf.NewTransformerRegistry()
if err := registry.Register(pdf.NewEncryptTransformer()); err != nil {
    return err
}
if err := registry.Register(pdf.NewPadesSignTransformer()); err != nil {
    return err
}
```

Attach the registry and selection config to the builder via `Transformations(registry, config)`. The `config` is a `TransformConfig` describing which transformers to enable and their options.

## See also

- [About PDF](../explanation/about-pdf.md) for why PDFs sit on the same template substrate and when to reach for PDF/A or digital signatures.
- [How to PDF generation](../how-to/pdf-generation.md) for rendering a PK template, adding metadata, watermarks, and encryption.
- [Scenario 027: PDF invoice](../../examples/scenarios/027_pdf_invoice/) for a runnable example.

---
title: How to generate PDFs
description: Render a PK template to PDF and stream it back to the browser.
nav:
  sidebar:
    section: "how-to"
    subsection: "operations"
    order: 70
---

# How to generate PDFs

Piko renders PDFs from PK templates using the same template engine that renders HTML. For the full API surface see [PDF API reference](../reference/pdf-api.md). For the rationale see [about PDF](../explanation/about-pdf.md). The integration tests under [`tests/integration/pdf/`](https://github.com/piko-sh/piko/tree/master/tests/integration/pdf) exercise the pipeline, and [Scenario 027: PDF invoice](../../examples/scenarios/027_pdf_invoice/) provides the runnable walkthrough.

## Write a PDF-shaped template

PDF templates use the same PK syntax as HTML pages, with four conventions that make layout predictable:

- A single root element (typically `<main>` or `<article>`) so the PDF has one logical page body.
- Explicit widths and paddings on containers; the renderer honours CSS dimensions.
- Page breaks via `page-break-before`, `page-break-after`, and `page-break-inside` CSS properties.
- Print-oriented typography: specify point sizes, avoid viewport-relative units (`vw`, `vh`, `vmin`, `vmax`).

```piko
<template>
    <main class="invoice">
        <header>
            <h1>Invoice {{ state.Invoice.Number }}</h1>
            <p>{{ state.Invoice.Date }}</p>
        </header>

        <section class="lines">
            <table>
                <thead>
                    <tr><th>Description</th><th>Amount</th></tr>
                </thead>
                <tbody>
                    <tr p-for="line in state.Invoice.Lines" p-key="line.ID">
                        <td>{{ line.Description }}</td>
                        <td>{{ line.Amount }}</td>
                    </tr>
                </tbody>
            </table>
        </section>

        <footer class="total">
            Total: {{ state.Invoice.Total }}
        </footer>
    </main>
</template>

<script type="application/x-go">
package main

import (
    "piko.sh/piko"
)

type Invoice struct {
    Number string
    Date   string
    Lines  []InvoiceLine
    Total  string
}

type InvoiceLine struct {
    ID          int64
    Description string
    Amount      string
}

type Response struct {
    Invoice Invoice
}

func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    invoice, err := loadInvoice(r.PathParam("id"))
    if err != nil {
        return Response{}, piko.Metadata{Status: 404}, nil
    }
    return Response{Invoice: invoice}, piko.Metadata{}, nil
}
</script>

<style>
    @page { size: A4; margin: 20mm; }
    .invoice { font-family: "DejaVu Sans", sans-serif; font-size: 11pt; }
    .lines { margin: 2rem 0; }
    .lines table { width: 100%; border-collapse: collapse; }
    .lines th, .lines td { padding: 0.5rem; border-bottom: 1px solid #ccc; }
    .total { text-align: right; font-weight: bold; font-size: 12pt; }
</style>
```

## Return the PDF from an action

Stream the PDF bytes back through an action so the browser prompts a download:

```go
package invoices

import (
    "bytes"

    "piko.sh/piko"
)

type DownloadInput struct {
    ID string `json:"id" validate:"required"`
}

type DownloadAction struct {
    piko.ActionMetadata
}

type DownloadResponse struct {
    PDF      string `json:"pdf"`      // base64-encoded bytes
    Filename string `json:"filename"`
}

func (a *DownloadAction) Call(input DownloadInput) (DownloadResponse, error) {
    pdfBytes, err := renderInvoicePDF(a.Ctx(), input.ID)
    if err != nil {
        return DownloadResponse{}, piko.NewError("could not render invoice", err)
    }

    return DownloadResponse{
        PDF:      base64.StdEncoding.EncodeToString(pdfBytes),
        Filename: "invoice-" + input.ID + ".pdf",
    }, nil
}
```

Actions return JSON-serialisable values. To pipe bytes back to the browser, base64-encode the PDF in the response and decode it on the client to trigger a download. See [`examples/scenarios/027_pdf_invoice/src/actions/invoice/generate.go`](https://github.com/piko-sh/piko/tree/master/examples/scenarios/027_pdf_invoice/src/actions/invoice/generate.go) for a runnable end-to-end example.

## Store the PDF via the storage provider

Long-lived PDFs (audit logs, invoices stored for years) belong in an object store, not rendered on every request. Use `storage.GetDefaultService()` to persist and later fetch:

```go
import (
    "bytes"
    "time"

    "piko.sh/piko/wdk/storage"
)

service, err := storage.GetDefaultService()
if err != nil {
    return err
}

key := "invoices/" + invoice.ID + ".pdf"

builder, err := storage.NewUploadBuilderFromDefault(bytes.NewReader(pdfBytes))
if err != nil {
    return err
}
if err := builder.
    Key(key).
    ContentType("application/pdf").
    Size(int64(len(pdfBytes))).
    Do(a.Ctx()); err != nil {
    return err
}

url, err := service.GeneratePresignedDownloadURL(a.Ctx(), storage.StorageNameDefault, storage.PresignDownloadParams{
    Repository: storage.StorageRepositoryDefault,
    Key:        key,
    ContentType: "application/pdf",
    ExpiresIn:  10 * time.Minute,
})
if err != nil {
    return err
}

a.Response().AddHelper("redirect", url)
```

## Render through the fluent builder

For finer control over the render pipeline, use `pdf.NewRenderBuilderFromDefault` directly:

```go
import "piko.sh/piko/wdk/pdf"

builder, err := pdf.NewRenderBuilderFromDefault()
if err != nil {
    return err
}

result, err := builder.
    Template("invoice").
    Props(invoiceProps).
    Metadata(pdf.Metadata{
        Title:   "Invoice " + invoiceID,
        Author:  "Acme Ltd.",
        Subject: "Invoice",
        Creator: "Piko",
    }).
    Page(pdf.PageA4).
    Do(ctx)
if err != nil {
    return err
}

// Use result.Content ([]byte) and result.PageCount (int).
```

Piko predefines `pdf.PageA4`, `pdf.PageA3`, and `pdf.PageLetter` as `PageConfig` values. The builder takes a template name, props, metadata, page config, watermark, PDF/A settings, page labels, and a post-processing transformer chain.

## Set metadata

PDF info dictionary fields appear in a reader's document properties pane:

```go
builder.Metadata(pdf.Metadata{
    Title:    "Annual report 2026",
    Author:   "Finance team",
    Subject:  "Statutory filing",
    Keywords: "annual,report,2026,statutory",
    Creator:  "Piko",
    Producer: "Piko PDF engine",
})
```

Set `ViewerPreferences` to control how readers open the document (full-screen mode, page layout, hide menubar).

## Add a watermark

For a quick text-only watermark, pass the string:

```go
builder.Watermark("DRAFT")
```

For finer control, use `WatermarkConfig`:

```go
builder.WatermarkConfig(pdf.WatermarkConfig{
    Text:     "DRAFT",
    Opacity:  0.2,
    Angle:    45,
    FontSize: 120,
})
```

The watermark stamps every page. Diagonal and semi-transparent rendering is the default.

## Produce a PDF/A archive-ready file

PDF/A-2b is the most common archival target. Embed the output intent and enable tagged PDF:

```go
builder.PdfA(pdf.PdfA2B)
```

`PdfA(level)` accepts a `PdfALevel` directly (`pdf.PdfA2B`, `pdf.PdfA2U`, `pdf.PdfA2A`). A default sRGB intent ships with the level. Configure custom ICC profiles through the `PdfA` transformer (see [PDF API reference](../reference/pdf-api.md)).

For accessibility-conformant PDF/A (tagged headings, reading order, alt text on figures), target `pdf.PdfA2A` instead. `PdfA2A` auto-enables the tagged-PDF pipeline, so the template must provide semantic markup (use `<h1>`, `<ol>`, `alt` attributes).

## Encrypt the output

Register the encryption transformer and supply `EncryptionOptions` through the transformer's name in the `TransformConfig`:

```go
import "piko.sh/piko/wdk/pdf"

registry := pdf.NewTransformerRegistry()
registry.Register(pdf.NewEncryptTransformer())

builder.Transformations(registry, pdf.TransformConfig{
    EnabledTransformers: []string{"pdf-encrypt"},
    TransformerOptions: map[string]any{
        "pdf-encrypt": pdf.EncryptionOptions{
            Algorithm:     "aes-256",
            OwnerPassword: ownerPw,
            UserPassword:  userPw,
            Permissions:   pdfPermissionPrint,
        },
    },
})
```

`Permissions` is a 32-bit bitmask following PDF specification table 22. Combine the bits for the operations you want to allow, such as printing, copying, modifying, or annotating. `OwnerPassword` unlocks all permissions regardless of the bitmask. `UserPassword` opens the PDF with the permission flags applied. Leave it empty to produce a document that opens without a prompt but still enforces the permission flags.

## Sign the PDF

PAdES (`PDF Advanced Electronic Signatures`) signs the document with a certificate. Register the signing transformer:

```go
import "piko.sh/piko/wdk/pdf"

chain, signer, err := loadSignerChain() // [][]byte certificate chain + crypto.Signer key
if err != nil {
    return err
}

registry := pdf.NewTransformerRegistry()
registry.Register(pdf.NewPadesSignTransformer())

builder.Transformations(registry, pdf.TransformConfig{
    EnabledTransformers: []string{"pades-sign"},
    TransformerOptions: map[string]any{
        "pades-sign": pdf.PadesSignOptions{
            PrivateKey:       signer,
            CertificateChain: chain,
            Level:            "b-t",
            TimestampURL:     "https://tsa.example.com",
            Reason:           "Invoice authorisation",
            Location:         "London, UK",
        },
    },
})
```

`CertificateChain` is the `DER` encoded certificate chain ordered from end-entity to root. `PrivateKey` must satisfy `crypto.Signer`. The `Level` field selects the PAdES conformance level (`"b-b"`, `"b-t"`, `"b-lt"`, `"b-lta"`). Levels above `b-b` need a working `TimestampURL` to anchor the signature to an independent trusted time, which matters for long-term validity.

## Redact regions

For documents with private information to strip before sharing (credit-card numbers, names), the redaction transformer blanks out rectangular regions:

```go
builder.Transformations(registry, pdf.TransformConfig{
    EnabledTransformers: []string{"redaction"},
    TransformerOptions: map[string]any{
        "redaction": pdf.RedactionOptions{
            Regions: []pdf.RedactionRegion{
                {Page: 0, X: 120, Y: 600, Width: 200, Height: 20},
                {Page: 1, X: 120, Y: 400, Width: 200, Height: 20},
            },
        },
    },
})
```

Page indices are zero-based. Redaction runs after the render and burns the fill colour over the content. The original text is no longer recoverable from the PDF.

## Debugging PDF layout

- Render the same template as HTML first. If the HTML layout is wrong, the PDF layout is too.
- Inspect the rendered HTML with the browser's print preview (`Ctrl+P`) to see how print CSS applies.
- Reduce problems to a minimal template; layout bugs often come from complex flex or grid trees that render differently in the PDF engine.
- For PDF/A validation, open the output in a dedicated verifier (veraPDF, `Adobe Acrobat Preflight`).

## See also

- [PDF API reference](../reference/pdf-api.md) for the complete method, option, and transformer surface.
- [About PDF](../explanation/about-pdf.md) for when to reach for PDF/A, digital signatures, or redaction.
- [Scenario 027: PDF invoice](../../examples/scenarios/027_pdf_invoice/) for the runnable walkthrough.
- [How to email templates](email-templates.md) for the adjacent template-rendering pattern.
- Integration tests: [`tests/integration/pdf/`](https://github.com/piko-sh/piko/tree/master/tests/integration/pdf).

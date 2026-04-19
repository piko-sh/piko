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

Piko renders PDFs from PK templates using the same template engine that renders HTML. For the full API surface see [PDF API reference](../reference/pdf-api.md). For the rationale see [about PDF](../explanation/about-pdf.md). The integration tests under [`tests/integration/pdf/`](https://github.com/piko-sh/piko/tree/master/tests/integration/pdf) exercise the pipeline, and [Scenario 027: PDF invoice](../showcase/027-pdf-invoice.md) provides the runnable walkthrough.

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

func (a *DownloadAction) Call(input DownloadInput) (piko.StreamResponse, error) {
    pdf, err := renderInvoicePDF(a.Ctx(), input.ID)
    if err != nil {
        return piko.StreamResponse{}, piko.NewError("could not render invoice", err)
    }

    return piko.StreamResponse{
        ContentType: "application/pdf",
        Filename:    "invoice-" + input.ID + ".pdf",
        Body:        bytes.NewReader(pdf),
    }, nil
}
```

The framework sets `Content-Disposition: attachment; filename=...` automatically when `Filename` is non-empty. Omit `Filename` to serve inline.

## Store the PDF via the storage provider

Long-lived PDFs (audit logs, invoices stored for years) belong in an object store, not rendered on every request. Use `piko.GetStorageService()` to persist and later fetch:

```go
svc := piko.GetStorageService()

key := "invoices/" + invoice.ID + ".pdf"
if err := svc.Put(a.Ctx(), key, pdfBytes); err != nil {
    return err
}

url, err := svc.PresignGet(a.Ctx(), key, 10*time.Minute)
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
    PageConfig(pdf.PageConfig{Size: pdf.PageA4, Margins: pdf.Margins{Top: 20, Right: 20, Bottom: 20, Left: 20}}).
    Do(ctx)
if err != nil {
    return err
}

// result.Bytes, result.PageCount
```

The builder accepts template name, props, metadata, page config, watermark, PDF/A settings, page labels, custom fonts, and transformer config.

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

```go
builder.Watermark(pdf.WatermarkConfig{
    Text:     "DRAFT",
    Opacity:  0.2,
    Rotation: 45,
    FontSize: 120,
})
```

The watermark stamps every page. Diagonal and semi-transparent rendering is the default. Override through `WatermarkConfig`.

## Produce a PDF/A archive-ready file

PDF/A-2b is the most common archival target. Embed the output intent and enable tagged PDF:

```go
builder.PdfA(pdf.PdfAConfig{
    Level: pdf.PdfA2B,
    // Default sRGB intent is bundled. Pass your own ICC profile for custom intents.
})
```

For accessibility-conformant PDF/A (tagged headings, reading order, alt text on figures), target `pdf.PdfA2A` instead. `PdfA2A` auto-enables the tagged-PDF pipeline, so the template must provide semantic markup (use `<h1>`, `<ol>`, `alt` attributes).

## Encrypt the output

Register the encryption transformer and pass `EncryptionOptions`:

```go
import "piko.sh/piko/wdk/pdf"

registry := pdf.NewTransformerRegistry()
registry.Register(pdf.NewEncryptTransformer())

builder.Transform(pdf.TransformConfig{
    Registry: registry,
    Encryption: &pdf.EncryptionOptions{
        OwnerPassword: ownerPw,
        UserPassword:  userPw,
        AllowPrint:    true,
        AllowCopy:     false,
        AllowModify:   false,
        AllowAnnotate: false,
    },
})
```

`OwnerPassword` unlocks all permissions. `UserPassword` opens the PDF with the permission flags above. Leave `UserPassword` empty to produce a document that opens without a prompt but still enforces the permission flags.

## Sign the PDF

PAdES (`PDF Advanced Electronic Signatures`) signs the document with a certificate. Register the signing transformer:

```go
import (
    "piko.sh/piko/wdk/pdf"
    "piko.sh/piko/wdk/pdf/pdf_transform_pades"
)

cert, key, err := loadSignerCertificate() // X.509 + private key
if err != nil {
    return err
}

registry := pdf.NewTransformerRegistry()
registry.Register(pdf.NewPadesSignTransformer())

builder.Transform(pdf.TransformConfig{
    Registry: registry,
    Sign: &pdf.PadesSignOptions{
        Certificate:         cert,
        PrivateKey:          key,
        TimestampAuthority:  "https://tsa.example.com",
        SignatureReason:     "Invoice authorisation",
        SignatureLocation:   "London, UK",
    },
})
```

A timestamp authority anchors the signature to an independent trusted time, which matters for long-term validity.

## Redact regions

For documents with private information to strip before sharing (credit-card numbers, names), the redaction transformer blanks out rectangular regions:

```go
builder.Transform(pdf.TransformConfig{
    Registry: registry,
    Redaction: &pdf.RedactionOptions{
        Regions: []pdf.RedactionRegion{
            {Page: 1, X: 120, Y: 600, Width: 200, Height: 20},
            {Page: 2, X: 120, Y: 400, Width: 200, Height: 20},
        },
    },
})
```

Redaction runs after the render and burns the fill colour over the content. The original text is no longer recoverable from the PDF.

## Debugging PDF layout

- Render the same template as HTML first. If the HTML layout is wrong, the PDF layout is too.
- Inspect the rendered HTML with the browser's print preview (`Ctrl+P`) to see how print CSS applies.
- Reduce problems to a minimal template; layout bugs often come from complex flex or grid trees that render differently in the PDF engine.
- For PDF/A validation, open the output in a dedicated verifier (veraPDF, `Adobe Acrobat Preflight`).

## See also

- [PDF API reference](../reference/pdf-api.md) for the complete method, option, and transformer surface.
- [About PDF](../explanation/about-pdf.md) for when to reach for PDF/A, digital signatures, or redaction.
- [Scenario 027: PDF invoice](../showcase/027-pdf-invoice.md) for the runnable walkthrough.
- [How to email templates](email-templates.md) for the adjacent template-rendering pattern.
- Integration tests: [`tests/integration/pdf/`](https://github.com/piko-sh/piko/tree/master/tests/integration/pdf).

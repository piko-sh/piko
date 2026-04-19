---
title: "027: PDF invoice"
description: Generate a downloadable PDF invoice from a PK-rendered HTML template.
nav:
  sidebar:
    section: "showcase"
    subsection: "examples"
    order: 470
---

# 027: PDF invoice

A Piko project that renders an invoice as HTML from a PK template, converts it to PDF server-side, and streams the PDF back to the browser.

## What this demonstrates

- A PK template dedicated to PDF rendering (A4 print dimensions, explicit page breaks).
- Server-side HTML-to-PDF conversion.
- A server action that returns a `StreamResponse` with `application/pdf` content type.
- Storing the generated PDF via the storage provider for later retrieval.

## Project structure

```text
src/
  main.go               Bootstrap with storage provider.
  pages/
    invoices/
      {id}.pk           Preview page (HTML rendering).
  actions/
    invoices/
      download.go       Generates the PDF and returns a stream response.
  templates/
    invoice.pk          The PDF layout template.
```

## How to run this example

From the Piko repository root:

```bash
cd examples/scenarios/027_pdf_invoice/src/
go mod tidy
air
```

## See also

- [Server actions reference](../reference/server-actions.md).
- [Runnable source](https://github.com/piko-sh/piko/tree/master/examples/scenarios/027_pdf_invoice).

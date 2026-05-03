---
title: About PDF rendering
description: Why PDFs sit on the same template substrate, when to reach for PDF/A or PDF/UA, and the rationale for signing and encryption.
nav:
  sidebar:
    section: "explanation"
    subsection: "operations"
    order: 80
---

# About PDF rendering

PDFs are a different medium from HTML. They have a fixed page size, paginated flow, and a lifecycle that extends past the browser tab. The file gets saved, shared, printed, archived. The content inside a PDF is the same content that appears in a web page. Headings, paragraphs, tables, figures, conditional sections driven by data. Piko's PDF engine treats that similarity as the design axis. PDFs render from the same PK templates that serve HTML pages. This page explains why.

## Templates over SDKs

The alternative would be a programmatic PDF SDK. An API that draws rectangles, places text, measures fonts. Java's iText and Go's `gofpdf` work this way. The cost is imperative code that describes every layout decision. A template abstraction (HTML + CSS) is declarative and already familiar.

Piko chose templates. The renderer compiles the PK template, runs the CSS engine, builds a box tree, paginates, and paints. CSS properties for page breaks (`page-break-before`, `page-break-after`, `page-break-inside`) map cleanly onto the paginated output. `@page` rules control page size and margins. The whole lifecycle, from template authoring to paginated output, is familiar to anyone who has styled a print sheet in a browser.

The trade-off is that not every PDF feature maps onto CSS. Precise coordinate drawing, multi-column interleaving, and some advanced typographic controls need special handling. Piko supports these through SVG embedding (`SVGWriterPort`) and a low-level painter config, but they are escape hatches, not the default path.

## Reuse of template syntax and i18n

The template model brings one big benefit. Everything the HTML side of Piko has also works for PDFs. Interpolation with `{{ }}`. Conditional rendering with `p-if`. Loops with `p-for`. Partials. The i18n system. A team that writes a localised email template writes a localised PDF template the same way.

This matters for the common case of transactional PDFs. An invoice, a receipt, a contract, a shipping label. They have the same needs as a web page in most respects. Dynamic content, localisation, per-tenant branding. Reusing the template substrate means one mental model.

## When to reach for PDF/A

PDF/A is an archival variant of PDF. It enforces a set of constraints. All fonts embedded, no external references, restricted colour spaces, declared output intent. The intent is that a PDF/A document renders identically in 2026, 2056, and 2086.

Not every PDF needs to be PDF/A. A shipping label that lives for a week does not. An invoice that has to satisfy a seven-year retention requirement does. A contract that has to remain valid for decades does.

Piko targets PDF/A-2b for basic archival conformance. PDF/A-2u covers Unicode-preserving conformance. PDF/A-2a covers accessible, tagged conformance. The targets matter because downstream archival systems (records-management software, governmental submission portals) require a specific level.

The cost of PDF/A is bundle size and rendering cost. Every font has to ship embedded. Every image has to use a conforming colour space. A PDF/A-2a also requires a tagged structure (semantic markup for headings, lists, figures). Tagging is not free either. The template has to use semantic HTML (`<h1>`, `<ol>`, `<figure>`) instead of styled `<div>` blocks. The accessibility win is real, and so is the discipline cost.

## PDF/UA for accessible documents

PDF/UA (Universal Accessibility) is the accessibility variant of PDF. It enforces tagged structure, alternative text on images, reading order, and machine-readable metadata. Screen readers use the tags to navigate the document the same way they use HTML headings to navigate a web page.

PDF/UA and PDF/A-2a overlap significantly because both require tagged structure. A document can be both. The difference is that PDF/A-2a prioritises archival stability (self-contained, renders the same forever) and PDF/UA prioritises accessibility (machine-readable structure for assistive tech). A PDF/A-2a document is not automatically PDF/UA-conformant, but getting to PDF/UA from PDF/A-2a is short.

Piko's tagged-PDF pipeline handles both. Templates that use semantic HTML and set `alt` attributes on images pass most of the conformance checks automatically. Edge cases (complex table layouts, decorative images) may need manual tagging hints.

## Signing and timestamping

A signed PDF carries a cryptographic attestation of who signed it and when. Two standards exist. PAdES is the European long-term-validity standard. PKCS #7 is the older baseline. Piko ships PAdES because it is the stricter superset.

A signature makes sense when the document's author matters and must be provable. Contracts, audit reports, regulatory filings. An invoice rarely needs a signature. An audit log signed by the finance team has real use.

Timestamping, via an independent timestamp authority, anchors the signature to a trusted time. Without a timestamp, a signature is only as good as the signer's clock. With a timestamp, a verifier can prove the signature exists no later than the stamp, which matters for regulatory deadlines and long-term validity. Long-term PAdES (PAdES-LTV) extends this further. It periodically re-timestamps the archive to survive the eventual expiration of the signer's certificate.

The library does not mandate any of these. Applications pick the conformance level that matches the regulator's demands.

## Encryption

PDF encryption protects a document at rest. AES-256 encrypts the document, and a password unlocks it. Conforming PDF readers enforce the permission flags (print, copy, modify, annotate), but the flags do not constitute security against a determined adversary. A reader that ignores the permissions can choose not to enforce them. Encryption is therefore useful as a defence-in-depth mechanism, not as a sole protection.

The right use case is a PDF that lives in a shared system where the author wants to bound who can read it. Invoices emailed to a customer on an untrusted network. Medical reports stored on a shared drive. The encryption raises the bar against casual access. It does not defend against a motivated adversary who has the file and enough time.

## Redaction

Redaction removes content from a document so it does not survive into the rendered PDF stream. Piko's redaction transformer offers two modes that suit different inputs.

**Region redaction** takes coordinates. The caller names a page area, and the transformer paints a filled black rectangle over that area in the content stream. The rectangle overlays anything previously drawn there. This mode is the right tool when the caller knows exactly where the sensitive text sits in the rendered output. It also fits when the sensitive content is an image instead of searchable text.

**Text-pattern redaction** takes a pattern (a literal string or regular expression). The transformer walks the decoded content streams and replaces matched text with spaces. The substitution preserves the layout (one space per redacted character) so the surrounding text does not reflow. This mode is the right tool when the caller knows the sensitive value but not its coordinates, for example a customer's name or account number.

Both modes operate on the PDF content stream instead of a viewer overlay. An overlay (a black rectangle annotation, say) would still leave the original characters in the stream, where a script that extracts text from the stream would see them. The redaction transformer changes the stream itself. The transformer can also strip metadata (`/Info` from the trailer and `/Metadata` from the catalogue) so document properties do not leak the redacted content.

The use case covers documents that need to reach wider audiences with specific fields removed. Legal filings, freedom-of-information responses, internal memos going to wider audiences.

## When a PDF is the wrong choice

A PDF is heavy. It ships a full document's worth of structure, fonts, and images. For ephemeral content (a status update, a one-paragraph receipt), plain HTML or even plain text does the job better.

A PDF also carries a default assumption. The reader opens it separately from the website. That assumption is often wrong. A "download PDF" link on a help page treats the PDF as an archival copy when the reader wanted a printable copy of the web page. The browser's print-to-PDF already handles that case.

Reach for a PDF when the document has a life of its own. It gets saved, forwarded, archived. Reach for HTML when the document is content on a web page.

## See also

- [PDF API reference](../reference/pdf-api.md) for the render pipeline and transformer surface.
- [How to PDF generation](../how-to/pdf-generation.md) for templating, metadata, watermarks, PDF/A, encryption, and signing.
- [About email rendering](about-email-rendering.md) for the related template-to-non-HTML rendering pattern.

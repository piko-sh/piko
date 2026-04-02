// Copyright 2026 PolitePixels Limited
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This project stands against fascism, authoritarianism, and all forms of
// oppression. We built this to empower people, not to enable those who would
// strip others of their rights and dignity.

package layouter_domain

// Unit conversion constants. All conversions target points (1 point = 1/72
// inch).
const (
	// PixelsToPoints converts CSS pixels to points. CSS defines 1px =
	// 1/96 inch, and 1pt = 1/72 inch, so 1px = 72/96 = 0.75pt.
	PixelsToPoints = 0.75

	// InchesToPoints converts inches to points (1in = 72pt).
	InchesToPoints = 72.0

	// CentimetresToPoints converts centimetres to points (1cm = 28.3465pt).
	CentimetresToPoints = 28.3465

	// MillimetresToPoints converts millimetres to points (1mm = 2.83465pt).
	MillimetresToPoints = 2.83465

	// PicasToPoints converts picas to points (1pc = 12pt).
	PicasToPoints = 12.0
)

// UserAgentStylesheet is the default CSS applied to all documents before any
// author styles. It defines the default display values for HTML elements per
// the CSS specification.
const UserAgentStylesheet = `
html, address, blockquote, body, dd, div, dl, dt, fieldset, form,
frame, frameset, h1, h2, h3, h4, h5, h6, hr, noframes, ol, p, ul,
centre, dir, menu, pre, article, aside, details, dialog,
figcaption, figure, footer, header, hgroup, main, nav, section, summary {
	display: block;
}

li {
	display: list-item;
}

head, script, style, link, meta, title, base, option {
	display: none;
}

table {
	display: table;
	border-collapse: separate;
	border-spacing: 2px;
}

thead {
	display: table-header-group;
}

tbody {
	display: table-row-group;
}

tfoot {
	display: table-footer-group;
}

tr {
	display: table-row;
}

td, th {
	display: table-cell;
	padding: 1px;
}

caption {
	display: table-caption;
}

col {
	display: table-column;
}

colgroup {
	display: table-column-group;
}

h1 {
	font-size: 2em;
	font-weight: bold;
	margin-top: 0.67em;
	margin-bottom: 0.67em;
}

h2 {
	font-size: 1.5em;
	font-weight: bold;
	margin-top: 0.83em;
	margin-bottom: 0.83em;
}

h3 {
	font-size: 1.17em;
	font-weight: bold;
	margin-top: 1em;
	margin-bottom: 1em;
}

h4 {
	font-weight: bold;
	margin-top: 1.33em;
	margin-bottom: 1.33em;
}

h5 {
	font-size: 0.83em;
	font-weight: bold;
	margin-top: 1.67em;
	margin-bottom: 1.67em;
}

h6 {
	font-size: 0.67em;
	font-weight: bold;
	margin-top: 2.33em;
	margin-bottom: 2.33em;
}

p {
	margin-top: 1em;
	margin-bottom: 1em;
}

strong, b {
	font-weight: bold;
}

em, i {
	font-style: italic;
}

a {
	color: blue;
	text-decoration: underline;
}

ul, ol {
	padding-left: 40px;
	margin-top: 1em;
	margin-bottom: 1em;
}

pre {
	white-space: pre;
	font-family: monospace;
}

code, kbd, samp, tt {
	font-family: monospace;
}

hr {
	border-top-style: solid;
	border-top-width: 1px;
	margin-top: 0.5em;
	margin-bottom: 0.5em;
}

img, svg, video {
	display: inline;
}

input, textarea, select, button {
	display: inline;
}

sub {
	font-size: 0.83em;
	vertical-align: sub;
}

sup {
	font-size: 0.83em;
	vertical-align: super;
}
`

// InheritableProperties lists the CSS properties that are inherited by child
// elements when not explicitly set.
var InheritableProperties = map[string]bool{
	"color":                true,
	"font-family":          true,
	"font-size":            true,
	"font-style":           true,
	"font-weight":          true,
	"letter-spacing":       true,
	"line-height":          true,
	"text-align":           true,
	"text-decoration":      true,
	"text-decoration-line": true,
	"text-indent":          true,
	"text-transform":       true,
	"visibility":           true,
	"white-space":          true,
	"word-break":           true,
	"word-spacing":         true,
	"list-style-type":      true,
	"list-style-position":  true,
	"direction":            true,
	"hyphens":              true,
	"overflow-wrap":        true,
	"tab-size":             true,
	"text-shadow":          true,
	"orphans":              true,
	"widows":               true,
}

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

package generate

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"piko.sh/piko/tests/fuzz/internal/casefile"
)

const (
	// catalogueModel names the typed catalogue model package import path.
	catalogueModel = casefile.Module + "/pkg/catalog-item/v2"

	// catalogueData names the catalogue data package import path.
	catalogueData = casefile.Module + "/pkg/shelf"

	// catalogueTypes holds the catalogue's named type declarations.
	catalogueTypes = `package listing
type Code string
func (c Code) String() string { return string(c) }
func (c Code) Path() string { return "/product/"+string(c) }
type Picture struct { URL string; Description string }
type Detail struct { Caption string }
type Product struct { Code Code; Label string; Detail *Detail; Images []Picture }
`

	// catalogueLoader holds the catalogue content loader source.
	catalogueLoader = `package shelf
import (
 "io/fs"
 "strings"
 item "` + catalogueModel + `"

)

func Items() ([]item.Product,error) {
 b,err:=fs.ReadFile(Files,"items.txt");if err!=nil{return nil,err}
 lines:=strings.Split(strings.TrimSpace(string(b)),"\n")
 rows:=make([]item.Product,0,len(lines))
 for _,line:=range lines {
  parts:=strings.Split(line,"|")
  row:=item.Product{Code:item.Code(parts[0]),Label:parts[1]}
  if parts[2]=="full" {
   row.Detail=&item.Detail{Caption:"Featured "+parts[1]}
   row.Images=[]item.Picture{{URL:"/_piko/assets/example.org/pikofuzzcase/lib/catalog-one.svg",Description:parts[1]+" front"},` +
		`{URL:"/_piko/assets/example.org/pikofuzzcase/lib/catalog-two.svg",Description:parts[1]+" back"}}
  }
  rows=append(rows,row)
 }
 return rows,nil
}
`
	// catalogueCard holds the catalogue card component source.
	catalogueCard = `<template><article>
<a data-fuzz-element="buy" :href="props.Item.Code.Path()" :aria-label="state.BuyName" :data-sku="props.Item.Code">Buy</a>
<strong p-if="props.Item.Detail != nil" data-fuzz-element="feature" p-text="props.Item.Detail.Caption"></strong>
<img p-for="image in props.Item.Images" p-key="image.URL" data-fuzz-element="photo" :src="image.URL" :alt="image.Description">
</article></template>
<script type="application/x-go">
package main
import("piko.sh/piko"; item "` + catalogueModel + `")
type Props struct{ Item item.Product }
type Response struct{ BuyName string }
func Render(r *piko.RequestData,props Props)(Response,piko.Metadata,error){
 return Response{BuyName:"Buy "+props.Item.Label},piko.Metadata{},nil
}
</script>`
	// catalogueRebuild names the catalogue rebuild target.
	catalogueRebuild = "catalog-rebuild"
	// catalogueNarrow is the item count of a narrow catalogue.
	catalogueNarrow = 4
	// catalogueWide is the item count of a wide catalogue.
	catalogueWide = 8
	// catalogueFixedLandmarks is the number of catalogue landmarks before the item cards.
	catalogueFixedLandmarks = 5
	// headingTag is the HTML tag of a page's top-level heading.
	headingTag = "h1"
	// catalogueUnavailable is the catalogue not-found page's heading and title.
	catalogueUnavailable = "Catalog item unavailable"
	// catalogueLayoutTemplate holds the catalogue layout partial's template.
	catalogueLayoutTemplate = `<section><header data-fuzz-element="layout" :data-theme="props.Theme" p-text="props.Featured.Label"></header><piko:slot></piko:slot></section>`
	// catalogueLayoutScript holds the catalogue layout partial's script.
	catalogueLayoutScript = `type Props struct{ Featured item.Product; Theme string }
func Render(r *piko.RequestData,props Props)(piko.NoResponse,piko.Metadata,error){return piko.NoResponse{},piko.Metadata{},nil}`
	// catalogueBranchScript holds each branch panel's script.
	catalogueBranchScript = `type Props struct {Item item.Product}
func Render(r *piko.RequestData,props Props)(piko.NoResponse,piko.Metadata,error){return piko.NoResponse{},piko.Metadata{},nil}`
	// catalogueListingOpen opens the catalogue page's layout, heading and search form.
	catalogueListingOpen = `<piko:partial is="layout" :server.featured="state.Items[0]" :server.theme="state.Theme"><main>` +
		`<h1 data-fuzz-element="heading" :data-theme="state.Theme" p-text="state.Heading"></h1>` +
		`<form data-fuzz-element="search" action="/catalog" method="get" aria-label="Find items"><label for="filter">Search catalog</label>` +
		`<input data-fuzz-element="filter" id="filter" name="q" type="search"><button data-fuzz-element="submit" type="submit">Search</button></form>`
	// catalogueListingScript holds the catalogue page's script.
	catalogueListingScript = `type Response struct{ Items []item.Product; Heading string; Theme string }
func Render(r *piko.RequestData,props piko.NoProps)(Response,piko.Metadata,error){
 rows,err:=records.Items();if err!=nil{return Response{},piko.Metadata{},err}
 theme:="default";if cookie,err:=r.Cookie("catalog_theme");err==nil{theme=cookie.Value}
 title:="Catalog "+theme
 return Response{Items:rows,Heading:title,Theme:theme},piko.Metadata{Title:title, Description:"Listings for "+theme,Language:"en-GB",` +
		`CanonicalURL:"https://catalog.example/catalog",OGTags:[]piko.OGTag{{Property:"og:title",Content:title}},` +
		`TwitterCards:[]piko.MetaTag{{Name:"twitter:card",Content:"summary"}}},nil
}`
	// catalogueSummaryTemplate holds the catalogue summary page's template.
	catalogueSummaryTemplate = `<a data-fuzz-element="summary" :href="state.Item.Code.Path()" p-text="state.Item.Label"></a>`
	// catalogueSummaryScript holds the catalogue summary page's script.
	catalogueSummaryScript = `type Response struct{Item item.Product}
func Render(r *piko.RequestData,props piko.NoProps)(Response,piko.Metadata,error){
 rows,err:=records.Items();if err!=nil{return Response{},piko.Metadata{},err}
 return Response{Item:rows[0]},piko.Metadata{Title:"Catalog summary"},nil
}`
	// catalogueDraftScript holds the draft page's development-only script.
	catalogueDraftScript = `func Render(r *piko.RequestData,props piko.NoProps)(piko.NoResponse,piko.Metadata,error){` +
		`if !piko.IsDevelopmentMode(r){return piko.NoResponse{},piko.Metadata{},piko.NotFound("product","draft")};` +
		`return piko.NoResponse{},piko.Metadata{Title:"Draft preview"},nil}`
	// catalogueMissingScript holds the catalogue not-found page's script.
	catalogueMissingScript = `func Render(r *piko.RequestData,props piko.NoProps)(piko.NoResponse,piko.Metadata,error){` +
		`return piko.NoResponse{},piko.Metadata{Title:"Catalog item unavailable"},nil}`
)

// cataloguePage assembles a generated catalogue page.
//
// Takes template (string) which holds the template markup.
// Takes imports (string) which holds the source import declarations.
// Takes body (string) which holds the page's Go script declarations.
//
// Returns string which is the assembled catalogue page source.
func cataloguePage(template, imports, body string) string {
	return "<template>" + template + "</template>\n<script type=\"application/x-go\">\npackage main\nimport (\"piko.sh/piko\"\n" + imports + "\n)\n" + body + "\n</script>\n"
}

// catalogueElement models one marked semantic landmark.
//
// Takes key (string) which identifies the semantic landmark.
// Takes tag (string) which is the landmark's HTML tag.
// Takes name (string) which is the landmark's semantic text or accessible name.
// Takes attributes (...string) which lists semantic attribute name and value pairs; an
// unpaired trailing name is ignored.
//
// Returns casefile.PageElement which holds the marked semantic landmark.
func catalogueElement(key, tag, name string, attributes ...string) casefile.PageElement {
	element := casefile.PageElement{Key: key, Tag: tag, Name: name, Attributes: nil}
	if len(attributes) > 1 {
		element.Attributes = map[string]string{}
		for pair := range len(attributes) / 2 {
			element.Attributes[attributes[2*pair]] = attributes[2*pair+1]
		}
	}
	return element
}

// catalogueHead models the catalogue page's selected head fields.
//
// Takes title (string) which is the generated page title.
// Takes theme (string) which is the generated page theme.
//
// Returns map[string][]string which holds selected head fields and their duplicate
// values.
func catalogueHead(title, theme string) map[string][]string {
	return map[string][]string{
		"lang": {"en-GB"}, "title": {title}, "canonical": {"https://catalog.example/catalog"},
		"description": {"Listings for " + theme}, "og:title": {title}, "twitter:card": {"summary"},
	}
}

// addTypedCatalogue adds source and expected observations for typed catalogue routes.
//
// Takes c (*casefile.Case) which receives generated sources and expected results.
// Takes p (Project) which holds the drawn application model.
func addTypedCatalogue(c *casefile.Case, p Project) {
	width := catalogueNarrow
	if (c.Seed^c.Index)&1 != 0 {
		width = catalogueWide
	}
	c.Features = append(c.Features, fmt.Sprintf("catalog-width-%d", width), "catalog-typed-props", "catalog-mode-filesystem", "catalog-actor-revisit", "catalog-draft-policy")
	addCatalogueShelf(c)
	c.Files["partials/catalog/card.pk"] = catalogueCard
	c.Files["partials/catalog/layout.pk"] = cataloguePage(catalogueLayoutTemplate, fmt.Sprintf("item %q", catalogueModel), catalogueLayoutScript)
	cards := addCatalogueListing(c, p, width)
	page := c.Files["pages/main.pk"]
	page = strings.Replace(page, `<main id="case-root">`, `<main id="case-root"><a data-fuzz-element="catalog" href="/catalog" aria-label="Browse catalog">Catalog</a>`, 1)
	page = strings.Replace(page, "piko.Metadata{Title:title}", "piko.Metadata{Title:title,CanonicalURL:"+strconv.Quote("https://catalog.example/main")+"}", 1)
	c.Files["pages/main.pk"] = page
	c.Expected.Semantic = responseContract(map[string][]string{"canonical": {"https://catalog.example/main"}}, catalogueElement("catalog", "a", "Browse catalog", "href", "/catalog"))
	addCatalogueVisits(c, p, cards)
	if c.Target == catalogueRebuild {
		c.Files["pages/catalog-summary.pk"] = cataloguePage(catalogueSummaryTemplate, fmt.Sprintf("records %q\nitem %q", catalogueData, catalogueModel), catalogueSummaryScript)
		summary := catalogueElement("summary", "a", fmt.Sprintf("Item %d.1", p.Count), "href", fmt.Sprintf("/product/p%d-1", p.Count))
		c.Probes = append(c.Probes, semanticProbe("/catalog-summary", responseContract(map[string][]string{"title": {"Catalog summary"}}, summary)))
	}
	addCatalogueDraft(c)
	if c.Target == catalogueRebuild && c.Index%2 == 0 {
		c.Features = append(c.Features, "catalog-precomputed-destinations")
		c.Files["pkg/catalog-item/v2/item.go"] = strings.Replace(catalogueTypes, "Code Code; Label string;", "Code Code; Label, SKU, Href string;", 1)
		c.Files["pkg/shelf/loader.go"] = strings.Replace(c.Files["pkg/shelf/loader.go"], "Label:parts[1]", `Label:parts[1], SKU:parts[0], Href:"/product/"+parts[0]`, 1)
		cardEdits := strings.NewReplacer("props.Item.Code.Path()", "props.Item.Href", "props.Item.Code", "props.Item.SKU")
		c.Files["partials/catalog/card.pk"] = cardEdits.Replace(c.Files["partials/catalog/card.pk"])
		c.Files["pages/catalog-summary.pk"] = strings.Replace(c.Files["pages/catalog-summary.pk"], "state.Item.Code.Path()", "state.Item.Href", 1)
	}
}

// addCatalogueShelf adds the catalogue model and its file-system loader.
//
// A rebuild case on an even position opens the file system through a function per call;
// both variants keep independent content and link expectations when the component graph
// reloads.
//
// Takes c (*casefile.Case) which receives the model and loader sources.
func addCatalogueShelf(c *casefile.Case) {
	c.Files["pkg/catalog-item/v2/item.go"] = catalogueTypes
	c.Files["pkg/shelf/loader.go"] = catalogueLoader
	c.Files["pkg/shelf/files_native.go"] = "//go:build !interp\n\npackage shelf\nimport (\"embed\";\"io/fs\")\n//go:embed items.txt\nvar embedded embed.FS\nvar Files fs.FS=embedded\n"
	c.Files["pkg/shelf/files_interp.go"] = "//go:build interp\n\npackage shelf\nimport (\"os\";\"io/fs\")\nvar Files fs.FS=os.DirFS(\"pkg/shelf\")\n"
	if c.Target != catalogueRebuild || c.Index%2 != 0 {
		c.Features = append(c.Features, "catalog-filesystem-package-variable")
		return
	}
	c.Features = append(c.Features, "catalog-filesystem-function")
	c.Files["pkg/shelf/loader.go"] = strings.Replace(catalogueLoader, "fs.ReadFile(Files,", "fs.ReadFile(Files(),", 1)
	c.Files["pkg/shelf/files_native.go"] = strings.Replace(c.Files["pkg/shelf/files_native.go"], "var Files fs.FS=embedded", "func Files() fs.FS {return embedded}", 1)
	c.Files["pkg/shelf/files_interp.go"] = strings.Replace(c.Files["pkg/shelf/files_interp.go"],
		"var Files fs.FS=os.DirFS(\"pkg/shelf\")", "func Files() fs.FS {return os.DirFS(\"pkg/shelf\")}", 1)
}

// addCatalogueListing adds the catalogue page, its branch panels, items and icons.
//
// Takes c (*casefile.Case) which receives generated sources and asset probes.
// Takes p (Project) which holds the drawn application model.
// Takes width (int) which is the number of catalogue items.
//
// Returns []casefile.PageElement which lists the expected item card landmarks.
func addCatalogueListing(c *casefile.Case, p Project, width int) []casefile.PageElement {
	var rows, body, imports strings.Builder
	fmt.Fprintf(&imports, "records %q\nitem %q\n", catalogueData, catalogueModel)
	fmt.Fprintf(&imports, "layout %q\n", casefile.Module+"/partials/catalog/layout.pk")
	body.WriteString(catalogueListingOpen)
	var cards []casefile.PageElement
	for position := range width {
		branch := fmt.Sprintf("branch%d", position)
		code := fmt.Sprintf("p%d-%d", p.Count, position+1)
		label := fmt.Sprintf("Item %d.%d", p.Count, position+1)
		full := (int(c.Index%2)+position)%2 == 0
		kind := "bare"
		if full {
			kind = "full"
		}
		fmt.Fprintf(&rows, "%s|%s|%s\n", code, label, kind)
		path := "partials/catalog/" + branch + "/panel.pk"
		fmt.Fprintf(&imports, "%s %q\n", branch, casefile.Module+"/"+path)
		fmt.Fprintf(&body, `<piko:partial is=%q :server.item="state.Items[%d]"></piko:partial>`, branch, position)
		c.Files[path] = cataloguePage(`<div><piko:partial is="shared" :server.item="props.Item"></piko:partial></div>`,
			fmt.Sprintf("item %q\nshared %q", catalogueModel, casefile.Module+"/partials/catalog/card.pk"), catalogueBranchScript)
		cards = append(cards, catalogueElement("buy", "a", "Buy "+label, "href", "/product/"+code, "data-sku", code))
		if full {
			cards = append(cards, catalogueElement("feature", "strong", "Featured "+label),
				catalogueElement("photo", "img", label+" front", "src", "/_piko/assets/example.org/pikofuzzcase/lib/catalog-one.svg"),
				catalogueElement("photo", "img", label+" back", "src", "/_piko/assets/example.org/pikofuzzcase/lib/catalog-two.svg"))
		}
	}
	c.Files["pkg/shelf/items.txt"] = rows.String()
	for _, name := range []string{"one", "two"} {
		file := "lib/catalog-" + name + ".svg"
		icon := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="2" height="2"><title>Catalog %d %s</title><rect width="2" height="2"/></svg>`, p.Count, name)
		c.Files[file] = icon
		asset := newProbe("/_piko/assets/"+casefile.Module+"/"+file, http.StatusOK)
		asset.Asset = &casefile.ResponseAsset{Revalidate: true, MediaType: "image/svg+xml", SHA256: fmt.Sprintf("%x", sha256.Sum256([]byte(icon)))}
		c.Probes = append(c.Probes, asset)
	}
	body.WriteString("</main></piko:partial>")
	c.Files["pages/catalog.pk"] = cataloguePage(body.String(), imports.String(), catalogueListingScript)
	return cards
}

// addCatalogueVisits adds each actor's catalogue visits and their themed contracts.
//
// Takes c (*casefile.Case) which receives the visit probes.
// Takes p (Project) which holds the drawn application model.
// Takes cards ([]casefile.PageElement) which lists the expected item card landmarks.
func addCatalogueVisits(c *casefile.Case, p Project, cards []casefile.PageElement) {
	for visit, actor := range []string{"A", "B", "A"} {
		theme := "light"
		if actor == "B" {
			theme = "dark"
		}
		title := "Catalog " + theme
		elements := make([]casefile.PageElement, 0, catalogueFixedLandmarks+len(cards))
		elements = append(elements,
			catalogueElement("layout", "header", fmt.Sprintf("Item %d.1", p.Count), "data-theme", theme),
			catalogueElement("heading", headingTag, title, "data-theme", theme),
			catalogueElement("search", "form", "Find items", "action", "/catalog", "method", "get"),
			catalogueElement("filter", "input", "Search catalog", "name", "q", "type", "search"),
			catalogueElement("submit", "button", "Search", "type", "submit"))
		probe := semanticProbe("/catalog", responseContract(catalogueHead(title, theme), append(elements, cards...)...))
		probe.Actor = actor
		probe.Stage = "read-persisted"
		if visit < 2 {
			probe.Stage = "set-cookie"
			probe.Cookies = map[string]string{"catalog_theme": theme}
		}
		c.Probes = append(c.Probes, probe)
	}
}

// addCatalogueDraft adds the draft page, the catalogue not-found page and their
// mode-specific contracts.
//
// Takes c (*casefile.Case) which receives the pages and updated probes.
func addCatalogueDraft(c *casefile.Case) {
	c.Files["pages/draft.pk"] = cataloguePage(`<h1 data-fuzz-element="draft">Draft preview</h1>`, "", catalogueDraftScript)
	c.Files["pages/!404.pk"] = cataloguePage(`<h1 data-fuzz-element="missing">`+catalogueUnavailable+`</h1>`, "", catalogueMissingScript)
	for index := range c.Probes {
		probe := &c.Probes[index]
		if (probe.Path == "/missing" || probe.Path == "/missing-route") && probe.Status == http.StatusNotFound {
			probe.Program = false
			probe.Output = nil
			probe.Text = catalogueUnavailable
			probe.Semantic = responseContract(map[string][]string{"title": {catalogueUnavailable}}, catalogueElement("missing", headingTag, catalogueUnavailable))
		}
	}
	draft := newProbe("/draft", 0)
	draft.ByMode = map[string]casefile.ModeResponse{
		"interp": {
			Status:   http.StatusOK,
			Semantic: *responseContract(map[string][]string{"title": {"Draft preview"}, "canonical": {"http://fuzz.test/draft"}}, catalogueElement("draft", headingTag, "Draft preview")),
		},
		"compiled": {
			Status:   http.StatusNotFound,
			Semantic: *responseContract(map[string][]string{"title": {catalogueUnavailable}, "canonical": {}}, catalogueElement("missing", headingTag, catalogueUnavailable)),
		},
	}
	c.Probes = append(c.Probes, draft)
}

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
	"fmt"
	"net/http"
	"slices"

	"piko.sh/piko/tests/fuzz/internal/casefile"
)

const (
	// collectionDetail holds the external collection's detail page.
	collectionDetail = `<template p-collection="catalog" p-provider="markdown" p-collection-source="catalog">
<article id="fuzz-program"><output data-fuzz-output="doc" :data-value="state.URL" p-text="state.Title"></output>` +
		`<output data-fuzz-output="origin" p-text="state.Origin"></output><a href="/catalog-list" aria-label="Catalog index">All items</a><piko:content /></article>
</template>
<script type="application/x-go">
package main
import (
 "piko.sh/piko"
 catalog "` + casefile.CollectionModule + `/entries"
)
type Document struct { Title string; URL string }
type Response struct { Title string; URL string; Origin string }
func Render(r *piko.RequestData, props piko.NoProps) (Response,piko.Metadata,error) {
 d := piko.GetData[Document](r)
 return Response{Title:d.Title, URL:d.URL, Origin:catalog.Origin},piko.Metadata{},nil
}
</script>`

	// collectionIndex holds the external collection's index page.
	collectionIndex = `<template>
<section id="fuzz-program"><div p-for="doc in state.Docs"><output data-fuzz-output="listing" :data-value="doc.URL" p-text="doc.Title"></output>` +
		`<a :href="doc.URL" :aria-label="doc.Title" p-text="doc.Title"></a></div></section>
</template>
<script type="application/x-go">
package main
import "piko.sh/piko"
type Document struct { Title string; URL string }
type Response struct { Docs []Document }
func Render(r *piko.RequestData, props piko.NoProps) (Response,piko.Metadata,error) {
 items,err := piko.GetAllCollectionItems("catalog")
 if err != nil { return Response{},piko.Metadata{},err }
 docs := make([]Document,0,len(items))
 for _,item := range items {
  title,_ := item["Title"].(string)
  url,_ := item["URL"].(string)
  docs=append(docs,Document{Title:title,URL:url})
 }
 return Response{Docs:docs},piko.Metadata{},nil
}
</script>`
)

// addCollectionRoutes adds source and expected observations for collection routes.
//
// Takes c (*casefile.Case) which receives generated sources and expected results.
// Takes p (Project) which holds the drawn application model.
func addCollectionRoutes(c *casefile.Case, p Project) {
	const origin = "external catalog"
	c.Features = append(c.Features, "external-markdown-collection", "ordered-collection-list", "collection-origin-decoy")
	c.Files[casefile.CollectionDir+"/go.mod"] = "module " + casefile.CollectionModule + "\n\ngo 1.27.0\n"
	c.Files[casefile.CollectionDir+"/entries/source.go"] = "package entries\nconst Origin = \"" + origin + "\"\n"
	c.Files["pages/catalog/{slug}.pk"] = collectionDetail
	c.Files["pages/catalog-list.pk"] = collectionIndex
	markdown := func(title, body string) string {
		return fmt.Sprintf("---\ntitle: %s\ndate: 2026-01-01\n---\n\n%s\n", title, body)
	}
	list := programProbe("/catalog-list")
	list.Semantic = responseContract(nil)
	entries := 2 + p.Count%2
	details := make([]casefile.Probe, 0, entries)
	for position := range entries {
		slug := fmt.Sprintf("card-%02d-%c", p.Count, 'a'+position)
		path := "/catalog/" + slug
		title := fmt.Sprintf("Catalogue %d item %d", p.Count, position+1)
		body := fmt.Sprintf("Distributed document number %d has checksum %d.", position+1, (position+2)*(p.Count+1))
		c.Files[casefile.CollectionDir+"/entries/"+slug+".md"] = markdown(title, body)
		c.Files["content/catalog/"+slug+".md"] = markdown("Local substitute "+slug, "Local-only conflicting body.")
		list.Semantic.Ordinary = append(list.Semantic.Ordinary, catalogueElement("", "a", title, "href", path, "aria-label", title))
		list.Output = append(list.Output, valuedOutput("listing", title, path))
		detail := programProbe(path, valuedOutput("doc", title, path), programOutput("origin", origin))
		detail.Text = body
		detail.Semantic = responseContract(nil)
		detail.Semantic.Ordinary = []casefile.PageElement{catalogueElement("", "a", "Catalog index", "href", "/catalog-list", "aria-label", "Catalog index")}
		details = append(details, detail)
	}
	c.Files["content/catalog/local-only.md"] = markdown("Local only", "This document is present in the app alone.")
	c.Probes = append(c.Probes, list)
	slices.Reverse(details)
	c.Probes = append(c.Probes, details...)
	c.Probes = append(c.Probes, newProbe("/catalog/local-only", http.StatusNotFound))
}

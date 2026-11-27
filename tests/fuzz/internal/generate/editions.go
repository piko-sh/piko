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
	"slices"
	"strings"

	"piko.sh/piko/tests/fuzz/internal/casefile"
)

const (
	// editionAsset names the source-edition asset.
	editionAsset = "lib/edition.svg"

	// editionReferenceOffset moves the revised body's reference number away from the drawn
	// count shown in its title.
	editionReferenceOffset = 99
)

// addContentEditions adds source and expected observations for source-edition histories.
//
// Takes c (*casefile.Case) which receives generated sources and expected results.
// Takes p (Project) which holds the drawn application model.
//
// Returns error which reports a collection probe without the outputs or semantic contract
// the edition history rewrites, or nil on success.
func addContentEditions(c *casefile.Case, p Project) error {
	svg := func(fill string) string {
		return fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 12"><rect width="20" height="12" fill=%q/></svg>`, fill)
	}
	initial, updated := svg("#13452f"), svg("#fe3231")
	asset := func(content string) casefile.Probe {
		probe := newProbe("/_piko/assets/"+casefile.Module+"/"+editionAsset, http.StatusOK)
		probe.Asset = &casefile.ResponseAsset{MediaType: "image/svg+xml", Revalidate: true, SHA256: fmt.Sprintf("%x", sha256.Sum256([]byte(content)))}
		return probe
	}
	c.Files[editionAsset] = initial
	c.Probes = append(c.Probes, asset(initial))
	baseline := casefile.Edition{ID: "original", Write: nil, Remove: nil, Probes: c.Probes}
	revised := casefile.Edition{ID: "replace", Write: map[string]string{editionAsset: updated}, Remove: nil, Probes: nil}
	removed := casefile.Edition{ID: "remove", Write: map[string]string{editionAsset: updated}, Remove: nil, Probes: nil}
	slug := fmt.Sprintf("card-%02d-a", p.Count)
	target := "/catalog/" + slug
	name := casefile.CollectionDir + "/entries/" + slug + ".md"
	title := fmt.Sprintf("Revised edition %d", p.Count)
	body := fmt.Sprintf("Edited publication with reference %d.", p.Count+editionReferenceOffset)
	revised.Write[name] = fmt.Sprintf("---\ntitle: %s\ndate: 2026-01-01\n---\n\n%s\n", title, body)
	removed.Remove = []string{name}
	for index := range c.Probes {
		probe := &c.Probes[index]
		edited, deleted := *probe, *probe
		switch probe.Path {
		case target:
			if len(probe.Output) == 0 {
				return fmt.Errorf("collection probe %s has no title output to revise", probe.Path)
			}
			edited.Text = body
			edited.Output = slices.Clone(probe.Output)
			edited.Output[0].Text = title
			deleted = newProbe(target, http.StatusNotFound)
		case "/catalog-list":
			var err error
			edited, deleted, err = reviseCollectionList(probe, target, title)
			if err != nil {
				return err
			}
		default:
			if strings.HasSuffix(probe.Path, "/"+editionAsset) {
				edited, deleted = asset(updated), asset(updated)
			}
		}
		revised.Probes = append(revised.Probes, edited)
		removed.Probes = append(removed.Probes, deleted)
	}
	c.Expected.Editions = []casefile.Edition{baseline, revised, removed, {ID: "restore", Write: nil, Remove: nil, Probes: c.Probes}}
	c.Features = append(c.Features, "collection-edition-history", "asset-validator-history", "edition-rollback")
	return nil
}

// reviseCollectionList models the collection index after retitling and after removing one
// entry.
//
// Takes probe (*casefile.Probe) which is the original collection index probe.
// Takes target (string) which is the edited entry's route.
// Takes title (string) which is the edited entry's new title.
//
// Returns edited (casefile.Probe) which is the index probe after retitling the entry.
// Returns deleted (casefile.Probe) which is the index probe after removing the entry.
// Returns error which reports a missing semantic contract or an output list that does not
// align with its semantic elements, or nil on success.
func reviseCollectionList(probe *casefile.Probe, target, title string) (edited, deleted casefile.Probe, err error) {
	if probe.Semantic == nil {
		return casefile.Probe{}, casefile.Probe{}, fmt.Errorf("collection index %s has no semantic contract", probe.Path)
	}
	if len(probe.Semantic.Ordinary) != len(probe.Output) {
		return casefile.Probe{}, casefile.Probe{}, fmt.Errorf("collection index %s has %d outputs but %d semantic elements", probe.Path, len(probe.Output), len(probe.Semantic.Ordinary))
	}
	edited, deleted = *probe, *probe
	edited.Output = slices.Clone(probe.Output)
	deleted.Output = nil
	editedSemantic, deletedSemantic := *probe.Semantic, *probe.Semantic
	editedSemantic.Ordinary = slices.Clone(probe.Semantic.Ordinary)
	deletedSemantic.Ordinary = nil
	for position, output := range edited.Output {
		if output.Value != target {
			deleted.Output = append(deleted.Output, output)
			deletedSemantic.Ordinary = append(deletedSemantic.Ordinary, editedSemantic.Ordinary[position])
			continue
		}
		edited.Output[position].Text = title
		editedSemantic.Ordinary[position].Name = title
		editedSemantic.Ordinary[position].Attributes = map[string]string{"href": target, "aria-label": title}
	}
	edited.Semantic, deleted.Semantic = &editedSemantic, &deletedSemantic
	return edited, deleted, nil
}

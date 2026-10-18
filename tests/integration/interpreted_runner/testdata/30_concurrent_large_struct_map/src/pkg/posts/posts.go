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

package posts

import (
	"fmt"
	"sort"
)

type Post struct {
	Title, Description, Slug, Date, Story, Label, Summary, Updated   string
	Author, Image, Series, ArticleDate, URL, DateLabel, Year, Byline string
	Tags, Related                                                    []string
	Order, Minutes                                                   int
	Draft                                                            bool
}

var tagPool = []string{"go", "interpreters", "maps", "memory", "parsing", "testing", "web"}

var catalogue = func() map[string]Post {
	out := make(map[string]Post, 14)
	for i := range 14 {
		slug := fmt.Sprintf("post-%02d", i)
		out[slug] = Post{
			Title:   "Title of " + slug,
			Slug:    slug,
			URL:     "/posts/" + slug,
			Tags:    []string{tagPool[i%7], tagPool[(i*3+1)%7]},
			Related: []string{fmt.Sprintf("post-%02d", (i+1)%14), fmt.Sprintf("post-%02d", (i+6)%14)},
			Order:   i,
			Minutes: 3 + i%9,
			Draft:   i%5 == 4,
		}
	}
	return out
}()

func All() []Post {
	out := make([]Post, 0, len(catalogue))
	for _, post := range catalogue {
		out = append(out, post)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Order < out[j].Order })
	return out
}

func Related(slug string, n int) []Post {
	visible := make(map[string]Post, len(catalogue))
	for _, post := range All() {
		if !post.Draft {
			visible[post.Slug] = post
		}
	}
	self, ok := visible[slug]
	if !ok {
		return nil
	}
	chosen := map[string]bool{slug: true}
	var out []Post
	for _, link := range self.Related {
		if post, found := visible[link]; found && !chosen[link] {
			chosen[link] = true
			out = append(out, post)
		}
	}
	var candidates []Post
	for candidateSlug, post := range visible {
		if !chosen[candidateSlug] && sharesTag(post, self) {
			candidates = append(candidates, post)
		}
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].Order < candidates[j].Order })
	out = append(out, candidates...)
	if len(out) > n {
		out = out[:n]
	}
	return out
}

func sharesTag(a, b Post) bool {
	for _, tag := range a.Tags {
		for _, other := range b.Tags {
			if tag == other {
				return true
			}
		}
	}
	return false
}

func Lines() []string {
	var lines []string
	var trail []string
	for _, post := range All() {
		if post.Draft {
			continue
		}
		trail = append(trail, post.Slug)
		path := trail
		path = append(path, "end")
		var titles []string
		for _, related := range Related(post.Slug, 3) {
			titles = append(titles, related.Title)
		}
		lines = append(lines, fmt.Sprintf("%s (%d min, depth %d of %d): %v", post.Title, post.Minutes, len(trail), len(path), titles))
	}
	return lines
}

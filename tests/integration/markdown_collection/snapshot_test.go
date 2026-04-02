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

//go:build integration

package markdown_collection_test

import (
	"cmp"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"piko.sh/piko/wdk/json"

	"piko.sh/piko/internal/collection/collection_dto"
)

type contentItemSnapshot struct {
	Metadata       map[string]any `json:"metadata"`
	ID             string         `json:"id"`
	Slug           string         `json:"slug"`
	Locale         string         `json:"locale"`
	TranslationKey string         `json:"translationKey"`
	URL            string         `json:"url"`
	CreatedAt      string         `json:"createdAt,omitempty"`
	UpdatedAt      string         `json:"updatedAt,omitempty"`
	PublishedAt    string         `json:"publishedAt,omitempty"`
	ReadingTime    int            `json:"readingTime"`
	HasContentAST  bool           `json:"hasContentAST"`
	HasExcerptAST  bool           `json:"hasExcerptAST"`
}

type navigationNodeSnapshot struct {
	Children   []navigationNodeSnapshot `json:"children,omitempty"`
	ID         string                   `json:"id"`
	Title      string                   `json:"title"`
	Section    string                   `json:"section"`
	Subsection string                   `json:"subsection,omitempty"`
	URL        string                   `json:"url,omitempty"`
	Icon       string                   `json:"icon,omitempty"`
	Level      int                      `json:"level"`
	Order      int                      `json:"order"`
	Hidden     bool                     `json:"hidden,omitempty"`
}

type navigationTreeSnapshot struct {
	Sections []navigationNodeSnapshot `json:"sections"`
	Locale   string                   `json:"locale"`
}

type navigationGroupsSnapshot struct {
	Groups map[string]navigationTreeSnapshot `json:"groups"`
}

func snapshotItems(items []collection_dto.ContentItem) []contentItemSnapshot {
	snapshots := make([]contentItemSnapshot, 0, len(items))

	for i := range items {
		item := &items[i]

		metadata := make(map[string]any, len(item.Metadata))
		for k, v := range item.Metadata {
			if k == collection_dto.MetaKeyNavigation {
				continue
			}

			if t, ok := v.(time.Time); ok {
				if t.IsZero() {
					continue
				}
				metadata[k] = t.Format(time.RFC3339)
				continue
			}

			metadata[k] = v
		}

		snapshots = append(snapshots, contentItemSnapshot{
			ID:             item.ID,
			Slug:           item.Slug,
			Locale:         item.Locale,
			TranslationKey: item.TranslationKey,
			URL:            item.URL,
			ReadingTime:    item.ReadingTime,
			CreatedAt:      item.CreatedAt,
			UpdatedAt:      item.UpdatedAt,
			PublishedAt:    item.PublishedAt,
			HasContentAST:  item.ContentAST != nil && len(item.ContentAST.RootNodes) > 0,
			HasExcerptAST:  item.ExcerptAST != nil && len(item.ExcerptAST.RootNodes) > 0,
			Metadata:       metadata,
		})
	}

	slices.SortFunc(snapshots, func(a, b contentItemSnapshot) int {
		return cmp.Or(cmp.Compare(a.Slug, b.Slug), cmp.Compare(a.Locale, b.Locale))
	})

	return snapshots
}

func snapshotNavigation(groups *collection_dto.NavigationGroups) navigationGroupsSnapshot {
	result := navigationGroupsSnapshot{
		Groups: make(map[string]navigationTreeSnapshot, len(groups.Groups)),
	}

	for name, tree := range groups.Groups {
		result.Groups[name] = navigationTreeSnapshot{
			Locale:   tree.Locale,
			Sections: snapshotNodes(tree.Sections),
		}
	}

	return result
}

func snapshotNodes(nodes []*collection_dto.NavigationNode) []navigationNodeSnapshot {
	if len(nodes) == 0 {
		return nil
	}

	result := make([]navigationNodeSnapshot, 0, len(nodes))
	for _, node := range nodes {
		result = append(result, navigationNodeSnapshot{
			ID:         node.ID,
			Title:      node.Title,
			Section:    node.Section,
			Subsection: node.Subsection,
			URL:        node.URL,
			Icon:       node.Icon,
			Level:      node.Level,
			Order:      node.Order,
			Hidden:     node.Hidden,
			Children:   snapshotNodes(node.Children),
		})
	}

	return result
}

func marshalJSON(t *testing.T, v any) []byte {
	t.Helper()
	data, err := json.StdConfig().MarshalIndent(v, "", "  ")
	require.NoError(t, err, "Failed to marshal snapshot to JSON")
	return data
}

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

package lsp_domain

import (
	"context"
	"fmt"

	"go.lsp.dev/protocol"
)

// GetRenameEdits generates workspace edits to rename a symbol across the
// document. This builds on GetReferences to find all occurrences and create
// text edits for each.
//
// Takes position (protocol.Position) which specifies the location of the symbol to
// rename.
// Takes newName (string) which provides the new name for the symbol.
//
// Returns map[protocol.DocumentURI][]protocol.TextEdit which contains the text
// edits grouped by document URI.
// Returns error when finding references fails.
func (d *document) GetRenameEdits(ctx context.Context, position protocol.Position, newName string) (map[protocol.DocumentURI][]protocol.TextEdit, error) {
	locations, err := d.GetReferences(ctx, position)
	if err != nil {
		return nil, fmt.Errorf("finding references for rename: %w", err)
	}

	if len(locations) == 0 {
		return nil, nil
	}

	edits := make([]protocol.TextEdit, 0, len(locations))
	for _, location := range locations {
		edits = append(edits, protocol.TextEdit{
			Range:   location.Range,
			NewText: newName,
		})
	}

	result := make(map[protocol.DocumentURI][]protocol.TextEdit)
	if len(edits) > 0 {
		result[d.URI] = edits
	}

	return result, nil
}

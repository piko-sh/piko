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
package item

import (
	"piko.sh/piko"
)

type DeleteAction struct {
	piko.ActionMetadata
}

type DeleteInput struct {
	ItemID     string `json:"item_id"`
	CategoryID string `json:"category_id"`
	Label      string `json:"label"`
}

type DeleteResponse struct {
	ReceivedItemID     string `json:"received_item_id"`
	ReceivedCategoryID string `json:"received_category_id"`
	ReceivedLabel      string `json:"received_label"`
}

func (a DeleteAction) Call(input DeleteInput) (DeleteResponse, error) {
	a.Response().AddHelper("showResult", input.ItemID, input.CategoryID, input.Label)

	return DeleteResponse{
		ReceivedItemID:     input.ItemID,
		ReceivedCategoryID: input.CategoryID,
		ReceivedLabel:      input.Label,
	}, nil
}

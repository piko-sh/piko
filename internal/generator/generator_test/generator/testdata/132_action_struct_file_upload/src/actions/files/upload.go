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

package files

import "piko.sh/piko"

type UploadInput struct {
	Title  string          `json:"title" validate:"required"`
	Avatar piko.FileUpload `json:"avatar"`
}

type UploadOutput struct {
	FileName string `json:"file_name"`
	Success  bool   `json:"success"`
}

type UploadAction struct {
	piko.ActionMetadata
}

func (a UploadAction) Call(input UploadInput) (UploadOutput, error) {
	name := ""
	if header := input.Avatar.Header(); header != nil {
		name = header.Filename
	}
	return UploadOutput{FileName: name, Success: input.Title != ""}, nil
}

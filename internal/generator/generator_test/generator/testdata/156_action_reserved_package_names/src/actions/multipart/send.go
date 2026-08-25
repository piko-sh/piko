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

package multipart

import "piko.sh/piko"

type SendOutput struct {
	FileName string `json:"file_name"`
}

// SendAction lives in a package called "multipart", which the generated wrapper file also
// imports from the standard library whenever any action takes a file upload.
type SendAction struct {
	piko.ActionMetadata
}

func (a *SendAction) Call(attachment piko.FileUpload) (SendOutput, error) {
	name := ""
	if header := attachment.Header(); header != nil {
		name = header.Filename
	}
	return SendOutput{FileName: name}, nil
}

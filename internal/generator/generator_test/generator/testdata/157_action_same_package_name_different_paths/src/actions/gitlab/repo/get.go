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

package repo

import "piko.sh/piko"

type GitlabGetInput struct {
	Group   string `json:"group" validate:"required"`
	Project string `json:"project" validate:"required"`
}

type GitlabGetOutput struct {
	Path string `json:"path"`
}

// GitlabGetAction shares its package name, "repo", with the action in
// actions/github/repo: the two packages differ only by import path.
type GitlabGetAction struct {
	piko.ActionMetadata
}

func (a *GitlabGetAction) Call(input GitlabGetInput) (GitlabGetOutput, error) {
	return GitlabGetOutput{Path: input.Group + "/" + input.Project}, nil
}

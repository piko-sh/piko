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

type GithubGetInput struct {
	Owner string `json:"owner" validate:"required"`
	Name  string `json:"name" validate:"required"`
}

type GithubGetOutput struct {
	FullName string `json:"full_name"`
}

// GithubGetAction shares its package name, "repo", with the action in
// actions/gitlab/repo: the two packages differ only by import path.
type GithubGetAction struct {
	piko.ActionMetadata
}

func (a *GithubGetAction) Call(input GithubGetInput) (GithubGetOutput, error) {
	return GithubGetOutput{FullName: input.Owner + "/" + input.Name}, nil
}

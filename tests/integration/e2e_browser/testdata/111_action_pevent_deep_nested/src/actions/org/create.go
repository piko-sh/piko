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
package org

import (
	"strconv"

	"piko.sh/piko"
)

type CreateAction struct {
	piko.ActionMetadata
}

type Team struct {
	Name string `json:"name"`
	Size int    `json:"size"`
}

type Department struct {
	Name string `json:"name"`
	Team Team   `json:"team"`
}

type CreateInput struct {
	Name       string     `json:"name"`
	Department Department `json:"department"`
}

type CreateResponse struct {
	OrgName  string `json:"org_name"`
	DeptName string `json:"dept_name"`
	TeamName string `json:"team_name"`
	TeamSize int    `json:"team_size"`
}

func (a CreateAction) Call(input CreateInput) (CreateResponse, error) {
	a.Response().AddHelper("showResult",
		input.Name,
		input.Department.Name,
		input.Department.Team.Name,
		strconv.Itoa(input.Department.Team.Size),
	)

	return CreateResponse{
		OrgName:  input.Name,
		DeptName: input.Department.Name,
		TeamName: input.Department.Team.Name,
		TeamSize: input.Department.Team.Size,
	}, nil
}

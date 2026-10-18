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

package main

import (
	"net/http"
	apiv1 "testproject_madness/pkg/api"
	"testproject_madness/pkg/api/v2"
	_ "testproject_madness/pkg/blank"
	. "testproject_madness/pkg/dot"
	myhttp "testproject_madness/pkg/http"
	localhelpers "testproject_madness/pkg/utils"
	"testproject_madness/third_party/github.com/external/helpers"
)

type Response struct {
	DotInfo DotImported

	APIUser    apiv1.User
	APIProduct api.Product

	Local    localhelpers.UtilHelper
	External helpers.ExternalHelper
	Boxed    localhelpers.Box[apiv1.User]
	Embedded

	StdlibClient http.Client
}

type Embedded struct {
	myhttp.Client
}

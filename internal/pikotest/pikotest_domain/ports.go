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

package pikotest_domain

import (
	"context"
	"io"
	"net/http"

	"piko.sh/piko/internal/render/render_domain"
)

// RenderPort defines the HTML rendering capability needed by TestView.
// This is a narrow interface extracted from render_domain.RenderService,
// keeping only what the testing framework requires.
type RenderPort interface {
	// RenderAST renders an abstract syntax tree to the provided writer.
	//
	// Takes w (io.Writer) which receives the rendered output.
	// Takes response (http.ResponseWriter) which provides the HTTP response.
	// Takes request (*http.Request) which contains the incoming HTTP request.
	// Takes opts (RenderASTOptions) which configures the rendering behaviour.
	//
	// Returns error when rendering fails.
	RenderAST(
		ctx context.Context,
		w io.Writer,
		response http.ResponseWriter,
		request *http.Request,
		opts render_domain.RenderASTOptions,
	) error
}

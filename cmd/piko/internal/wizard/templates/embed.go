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

package templates

import "embed"

var (
	// ReadmesFS embeds the readme files from the readmes directory.
	//
	//go:embed readmes/*
	ReadmesFS embed.FS

	// ConfigsFS holds the embedded file system containing configuration files.
	//
	//go:embed configs/*
	ConfigsFS embed.FS

	// GoTmplFS holds the embedded Go template files from the go/*.tmpl directory.
	//
	//go:embed go/*
	GoTmplFS embed.FS

	// PikoTmplFS holds the embedded Piko template files.
	//
	//go:embed piko/*
	PikoTmplFS embed.FS

	// E2ETmplFS embeds the e2e template files for test generation.
	//
	//go:embed e2e/*
	E2ETmplFS embed.FS

	// IconsFS holds the embedded SVG icon files for the scaffold project.
	//
	//go:embed icons/*
	IconsFS embed.FS
)

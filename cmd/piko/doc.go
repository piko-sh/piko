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

// Package piko is the command-line tool for creating and managing Piko
// projects.
//
// This package is a thin entry point that dispatches to subcommands implemented
// in internal packages.
//
// # Subcommands
//
// The CLI supports the following subcommands:
//
//   - new: Launches the interactive project creation wizard
//   - fmt: Formats Piko template files (.pk)
//   - inspect: Inspects FlatBuffers binary files
//   - get/describe/info/watch/diagnostics/tui: Monitoring commands via gRPC
//   - version: Displays the CLI version
//   - help: Displays usage information
//
// When invoked without a subcommand it displays a welcome message with
// available commands and version information.
package main

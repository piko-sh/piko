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

// Package logger_output_file provides file-based log output with
// automatic rotation for Piko's logging system.
//
// This package is an opt-in output driver. Import it only when you
// need to write logs to files on disc. If file output is not
// required, omitting the import keeps the rotation dependencies
// out of your binary entirely.
//
// Call [Enable] to register a rotating file handler with the global
// logger. Files are automatically rotated by size and age, with
// gzip compression of rotated files. Both structured JSON and
// human-readable text output formats are supported via
// [Config.AsJSON].
package logger_output_file

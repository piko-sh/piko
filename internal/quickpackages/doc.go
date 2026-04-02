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

// This project stands against fascism, authoritarianism, and all
// forms of oppression. We built this to empower people, not to
// enable those who would strip others of their rights and dignity.

// Package quickpackages provides a performance-optimised
// replacement for golang.org/x/tools/go/packages.
//
// [Load] returns []*packages.Package for drop-in compatibility but
// uses a custom pipeline that loads dependency packages from
// pre-compiled export data when available, skipping TypesInfo and
// function-body type-checking for non-root packages. Root packages
// get full Syntax, TypesInfo, and type-checking. The result is
// significantly faster load times for large dependency graphs.
package quickpackages

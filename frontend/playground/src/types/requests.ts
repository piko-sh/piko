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

/** Options for analysing Go source code. */
export interface AnalyseRequest {
  /**
   * Maps virtual file paths to their content.
   * At minimum, include a main.go file.
   */
  sources: Record<string, string>;

  /**
   * Go module name for the user's code.
   * Defaults to "playground" if not specified.
   */
  moduleName?: string;
}

/** Options for requesting code completions at a cursor position. */
export interface CompletionRequest {
  /** Go source code to analyse. */
  source: string;

  /** Virtual file path (for multi-file scenarios). */
  filePath?: string;

  /** 1-indexed line number for the cursor position. */
  line: number;

  /** 1-indexed column number for the cursor position. */
  column: number;

  /** Go module name for context. */
  moduleName?: string;
}

/** Options for requesting hover information at a cursor position. */
export interface HoverRequest {
  /** Go source code to analyse. */
  source: string;

  /** Virtual file path. */
  filePath?: string;

  /** 1-indexed line number. */
  line: number;

  /** 1-indexed column number. */
  column: number;

  /** Go module name for context. */
  moduleName?: string;
}

/** Options for parsing a PK template into an AST. */
export interface ParseTemplateRequest {
  /** PK template content. */
  template: string;

  /** Go script block content (optional, can be embedded in template). */
  script?: string;

  /** Go module name for context. */
  moduleName?: string;
}

/** Options for rendering a PK template preview to HTML. */
export interface RenderPreviewRequest {
  /** PK template content. */
  template: string;

  /** Go script block content. */
  script?: string;

  /** JSON-encoded props to pass to the template. */
  propsJson?: string;

  /** Go module name for context. */
  moduleName?: string;
}

/** Options for validating code without performing a full analysis. */
export interface ValidateRequest {
  /** Go source code to validate. */
  source: string;

  /** Virtual file path. */
  filePath?: string;
}

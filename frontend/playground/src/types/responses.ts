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

import type { Diagnostic, Location, Range } from './common';

/** Holds details about a named type found in Go source code. */
export interface TypeInfo {
  /** Type name. */
  name: string;

  /** Type kind (struct, interface, alias, etc.). */
  kind: string;

  /** Struct fields (if applicable). */
  fields?: FieldInfo[];

  /** Methods defined on this type. */
  methods?: MethodInfo[];

  /** Whether the type is exported. */
  isExported: boolean;

  /** Type's doc comment. */
  documentation?: string;

  /** Where the type is defined. */
  location: Location;
}

/** Holds details about a single struct field. */
export interface FieldInfo {
  /** Field name. */
  name: string;

  /** String representation of the field type. */
  typeString: string;

  /** Struct tag (if present). */
  tag?: string;

  /** Whether this is an embedded field. */
  isEmbedded?: boolean;

  /** Field's doc comment. */
  documentation?: string;
}

/** Holds details about a method defined on a type. */
export interface MethodInfo {
  /** Method name. */
  name: string;

  /** Method signature string. */
  signature: string;

  /** Whether the receiver is a pointer. */
  isPointerReceiver?: boolean;

  /** Method's doc comment. */
  documentation?: string;
}

/** Holds details about a package-level function. */
export interface FunctionInfo {
  /** Function name. */
  name: string;

  /** Function signature string. */
  signature: string;

  /** Whether the function is exported. */
  isExported: boolean;

  /** Function's doc comment. */
  documentation?: string;

  /** Where the function is defined. */
  location: Location;
}

/** Holds details about a Go import statement. */
export interface ImportInfo {
  /** Import path. */
  path: string;

  /** Import alias (empty if none). */
  alias?: string;

  /** Whether the import is used in code. */
  isUsed: boolean;
}

/** Holds the results of analysing Go source code. */
export interface AnalyseResponse {
  /** Whether the analysis completed without errors. */
  success: boolean;

  /** Extracted type information. */
  types?: TypeInfo[];

  /** Extracted function information. */
  functions?: FunctionInfo[];

  /** Detected imports. */
  imports?: ImportInfo[];

  /** Warnings or errors from analysis. */
  diagnostics?: Diagnostic[];

  /** Error message if success is false. */
  error?: string;
}

/** The kind of a completion item. */
export type CompletionKind =
  | 'function'
  | 'type'
  | 'variable'
  | 'constant'
  | 'keyword'
  | 'package'
  | 'field'
  | 'method';

/** Holds a single code completion suggestion. */
export interface CompletionItem {
  /** Text shown in the completion list. */
  label: string;

  /** Completion kind (function, type, variable, etc.). */
  kind: CompletionKind | string;

  /** Additional information about the item. */
  detail?: string;

  /** Item's documentation. */
  documentation?: string;

  /** Text to insert (if different from label). */
  insertText?: string;

  /** Used for sorting (if different from label). */
  sortText?: string;
}

/** Holds the code completion results. */
export interface CompletionResponse {
  /** Whether completions were generated. */
  success: boolean;

  /** Completion items. */
  items?: CompletionItem[];

  /** Error message if success is false. */
  error?: string;
}

/** Holds the hover information for a source position. */
export interface HoverResponse {
  /** Whether hover info was found. */
  success: boolean;

  /** Hover content (markdown). */
  content?: string;

  /** Range the hover applies to. */
  range?: Range;

  /** Error message if success is false. */
  error?: string;
}

/** Holds a single node in a parsed PK template AST. */
export interface TemplateNode {
  /** Node type (element, text, expression, etc.). */
  type: string;

  /** Element/component name (for elements). */
  name?: string;

  /** Text content (for text nodes). */
  content?: string;

  /** Element attributes. */
  attributes?: Record<string, string>;

  /** Child nodes. */
  children?: TemplateNode[];

  /** Where this node appears. */
  location: Location;
}

/** Holds metadata about a PK template's script block. */
export interface ScriptBlockInfo {
  /** Name of the props type (if defined). */
  propsType?: string;

  /** Types defined in the script. */
  types?: string[];

  /** Whether there's an init function. */
  hasInit?: boolean;
}

/** Simplified representation of a parsed PK template. */
export interface TemplateAST {
  /** Top-level template nodes. */
  nodes: TemplateNode[];

  /** Parsed script block info. */
  scriptBlock?: ScriptBlockInfo;
}

/** Holds the results of parsing a PK template. */
export interface ParseTemplateResponse {
  /** Whether parsing succeeded. */
  success: boolean;

  /** Simplified representation of the template AST. */
  ast?: TemplateAST;

  /** Warnings or errors from parsing. */
  diagnostics?: Diagnostic[];

  /** Error message if success is false. */
  error?: string;
}

/** Holds the rendered template preview output. */
export interface RenderPreviewResponse {
  /** Whether rendering succeeded. */
  success: boolean;

  /** Rendered HTML output. */
  html?: string;

  /** Extracted CSS (if applicable). */
  css?: string;

  /** Warnings from rendering. */
  diagnostics?: Diagnostic[];

  /** Error message if success is false. */
  error?: string;
}

/** Holds the validation results for source code. */
export interface ValidateResponse {
  /** Whether the code is valid. */
  valid: boolean;

  /** Validation errors and warnings. */
  diagnostics?: Diagnostic[];
}

/** Holds metadata about the WASM runtime environment. */
export interface RuntimeInfo {
  /** Piko version. */
  version: string;

  /** Go version used to build. */
  goVersion: string;

  /** Available stdlib packages. */
  stdlibPackages: string[];

  /** Available features. */
  capabilities: string[];
}

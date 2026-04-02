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

export { PikoPlayground } from './client';
export type { PikoPlaygroundOptions } from './client';

export {
  ApiNotAvailableError,
  GoRuntimeError,
  LoaderError,
  TimeoutError,
  WasmBinaryLoadError,
  WasmExecLoadError,
  WasmLoader,
} from './loader';
export type {
  GoConstructor,
  GoInstance,
  LoaderState,
  PikoWasmApi,
  PikoWindow,
  WasmLoaderOptions,
} from './loader';

export type {
  Diagnostic,
  DiagnosticSeverity,
  Location,
  Position,
  Range,
  AnalyseRequest,
  CompletionRequest,
  HoverRequest,
  ParseTemplateRequest,
  RenderPreviewRequest,
  ValidateRequest,
  AnalyseResponse,
  CompletionItem,
  CompletionKind,
  CompletionResponse,
  FieldInfo,
  FunctionInfo,
  HoverResponse,
  ImportInfo,
  MethodInfo,
  ParseTemplateResponse,
  RenderPreviewResponse,
  RuntimeInfo,
  ScriptBlockInfo,
  TemplateAST,
  TemplateNode,
  TypeInfo,
  ValidateResponse,
} from './types';

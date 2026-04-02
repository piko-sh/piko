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

import type {
  AnalyseRequest,
  AnalyseResponse,
  CompletionRequest,
  CompletionResponse,
  HoverRequest,
  HoverResponse,
  ParseTemplateRequest,
  ParseTemplateResponse,
  RenderPreviewRequest,
  RenderPreviewResponse,
  RuntimeInfo,
  ValidateRequest,
  ValidateResponse,
} from '../types';

/** Loader state machine states. */
export type LoaderState =
  | 'idle'
  | 'loading-runtime'
  | 'loading-wasm'
  | 'starting'
  | 'initialising'
  | 'ready'
  | 'error';

/** Configuration options for the WASM loader. */
export interface WasmLoaderOptions {
  /**
   * URL to the wasm_exec.js file from the Go SDK.
   * Defaults to './assets/wasm_exec.js'.
   */
  wasmExecUrl?: string;

  /**
   * URL to the piko.wasm file.
   * Defaults to './assets/piko.wasm'.
   */
  wasmUrl?: string;

  /**
   * Timeout in milliseconds for loading and initialisation.
   * Defaults to 30000 (30 seconds).
   */
  timeout?: number;

  /**
   * Polling interval in milliseconds when waiting for the piko global.
   * Defaults to 50.
   */
  pollInterval?: number;

  /**
   * Callback invoked whenever the loader transitions to a new state.
   */
  onStateChange?: (state: LoaderState) => void;

  /**
   * Callback invoked when a loading or initialisation error occurs.
   */
  onError?: (error: Error) => void;
}

/** Piko global API exposed by the WASM module after initialisation. */
export interface PikoWasmApi {
  /** Initialise the WASM module. */
  init(): Promise<void>;

  /** Get runtime information. */
  getRuntimeInfo(): RuntimeInfo;

  /** Analyse Go source code. */
  analyse(request: AnalyseRequest): Promise<AnalyseResponse>;

  /** Get code completions at a position. */
  getCompletions(request: CompletionRequest): Promise<CompletionResponse>;

  /** Get hover information at a position. */
  getHover(request: HoverRequest): Promise<HoverResponse>;

  /** Validate code without full analysis. */
  validate(request: ValidateRequest): Promise<ValidateResponse>;

  /** Parse a PK template. */
  parseTemplate(request: ParseTemplateRequest): Promise<ParseTemplateResponse>;

  /** Render a template preview. */
  renderPreview(
    request: RenderPreviewRequest
  ): Promise<RenderPreviewResponse>;
}

/** Go constructor exposed by `wasm_exec.js`. */
export interface GoConstructor {
  /** Creates a new Go runtime instance. */
  new (): GoInstance;
}

/** Go runtime instance created by the `Go` constructor. */
export interface GoInstance {
  /** Import object for WebAssembly instantiation. */
  importObject: WebAssembly.Imports;

  /** Run the Go program. Returns when the program exits. */
  run(instance: WebAssembly.Instance): Promise<void>;
}

/** Extended Window interface with Go and piko globals. */
export interface PikoWindow extends Window {
  /** The Go constructor injected by wasm_exec.js. */
  Go?: GoConstructor;
  /** The piko API object set by the WASM module after initialisation. */
  piko?: PikoWasmApi;
}

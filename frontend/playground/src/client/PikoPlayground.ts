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

import { WasmLoader } from '../loader/WasmLoader';
import { ApiNotAvailableError } from '../loader/errors';
import type { LoaderState, PikoWasmApi, WasmLoaderOptions } from '../loader/types';
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

/** Configuration options for `PikoPlayground`. */
export interface PikoPlaygroundOptions extends WasmLoaderOptions {
  /** Whether to initialise automatically on construction. Defaults to `false`. */
  autoInit?: boolean;
}

/** High-level client for the Piko WASM playground. */
export class PikoPlayground {
  /** Underlying WASM loader instance. */
  private readonly loader: WasmLoader;
  /** Cached WASM API, set once initialisation succeeds. */
  private api: PikoWasmApi | null = null;
  /** In-flight init promise for deduplication. */
  private initPromise: Promise<void> | null = null;

  /**
   * Creates a new PikoPlayground with the given configuration.
   *
   * @param options - The playground configuration.
   */
  constructor(options: PikoPlaygroundOptions = {}) {
    this.loader = new WasmLoader(options);

    if (options.autoInit) {
      this.initPromise = this.init();
    }
  }

  /**
   * Returns the current loader state.
   *
   * @returns The current state of the loading state machine.
   */
  getState(): LoaderState {
    return this.loader.getState();
  }

  /**
   * Checks whether the playground is initialised and ready.
   *
   * @returns `true` when the API is available.
   */
  isReady(): boolean {
    return this.loader.isReady() && this.api !== null;
  }

  /**
   * Initialises the playground by loading the WASM module.
   *
   * Safe to call multiple times; subsequent calls return the same promise.
   */
  async init(): Promise<void> {
    if (this.api) {
      return;
    }

    if (this.initPromise) {
      return this.initPromise;
    }

    this.initPromise = this.doInit();
    return this.initPromise;
  }

  /**
   * Resets the playground to its initial state so that initialisation can be retried.
   */
  reset(): void {
    this.loader.reset();
    this.api = null;
    this.initPromise = null;
  }

  /**
   * Returns runtime information from the WASM module.
   *
   * @returns The runtime metadata including version and available packages.
   * @throws {ApiNotAvailableError} If the playground is not initialised.
   */
  getRuntimeInfo(): RuntimeInfo {
    this.ensureReady();
    return this.api!.getRuntimeInfo();
  }

  /**
   * Analyses Go source code for types, functions, imports, and diagnostics.
   *
   * @param request - The analysis request containing source files.
   * @returns The analysis results.
   * @throws {ApiNotAvailableError} If the playground is not initialised.
   */
  async analyse(request: AnalyseRequest): Promise<AnalyseResponse> {
    this.ensureReady();
    return this.api!.analyse(request);
  }

  /**
   * Returns code completions at a cursor position.
   *
   * @param request - The completion request with source and position.
   * @returns The completion items.
   * @throws {ApiNotAvailableError} If the playground is not initialised.
   */
  async getCompletions(request: CompletionRequest): Promise<CompletionResponse> {
    this.ensureReady();
    return this.api!.getCompletions(request);
  }

  /**
   * Returns hover information (type and documentation) at a position.
   *
   * @param request - The hover request with source and position.
   * @returns The hover content and range.
   * @throws {ApiNotAvailableError} If the playground is not initialised.
   */
  async getHover(request: HoverRequest): Promise<HoverResponse> {
    this.ensureReady();
    return this.api!.getHover(request);
  }

  /**
   * Validates code without performing a full analysis.
   *
   * @param request - The validation request with source code.
   * @returns The validation result with diagnostics.
   * @throws {ApiNotAvailableError} If the playground is not initialised.
   */
  async validate(request: ValidateRequest): Promise<ValidateResponse> {
    this.ensureReady();
    return this.api!.validate(request);
  }

  /**
   * Parses a PK template into an AST with diagnostics.
   *
   * @param request - The parse request with template content.
   * @returns The parsed AST and diagnostics.
   * @throws {ApiNotAvailableError} If the playground is not initialised.
   */
  async parseTemplate(
    request: ParseTemplateRequest
  ): Promise<ParseTemplateResponse> {
    this.ensureReady();
    return this.api!.parseTemplate(request);
  }

  /**
   * Renders a template preview to HTML.
   *
   * @param request - The render request with template content and optional props.
   * @returns The rendered HTML output and diagnostics.
   * @throws {ApiNotAvailableError} If the playground is not initialised.
   */
  async renderPreview(
    request: RenderPreviewRequest
  ): Promise<RenderPreviewResponse> {
    this.ensureReady();
    return this.api!.renderPreview(request);
  }

  /**
   * Analyses a single Go source file.
   *
   * @param source - The Go source code.
   * @param filePath - The virtual file path. Defaults to `"main.go"`.
   * @param moduleName - The Go module name. Defaults to `"playground"`.
   * @returns The analysis results.
   * @throws {ApiNotAvailableError} If the playground is not initialised.
   */
  async analyseSingleFile(
    source: string,
    filePath = 'main.go',
    moduleName = 'playground'
  ): Promise<AnalyseResponse> {
    return this.analyse({
      sources: { [filePath]: source },
      moduleName,
    });
  }

  /**
   * Validates a single Go source file.
   *
   * @param source - The Go source code.
   * @param filePath - The virtual file path. Defaults to `"main.go"`.
   * @returns The validation result.
   * @throws {ApiNotAvailableError} If the playground is not initialised.
   */
  async validateSingleFile(
    source: string,
    filePath = 'main.go'
  ): Promise<ValidateResponse> {
    return this.validate({ source, filePath });
  }

  /**
   * Loads the WASM module and caches the API reference.
   */
  private async doInit(): Promise<void> {
    this.api = await this.loader.load();
  }

  /**
   * Throws if the API has not been initialised.
   *
   * @throws {ApiNotAvailableError} When `init()` has not been called.
   */
  private ensureReady(): void {
    if (!this.api) {
      throw new ApiNotAvailableError(
        'Playground not initialised. Call init() first.'
      );
    }
  }
}

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

import {
  ApiNotAvailableError,
  GoRuntimeError,
  TimeoutError,
  WasmBinaryLoadError,
  WasmExecLoadError,
} from './errors';
import type {
  GoInstance,
  LoaderState,
  PikoWasmApi,
  PikoWindow,
  WasmLoaderOptions,
} from './types';

/** Default URL for the Go `wasm_exec.js` support script. */
const DEFAULT_WASM_EXEC_URL = './assets/wasm_exec.js';
/** Default URL for the compiled Piko WASM binary. */
const DEFAULT_WASM_URL = './assets/piko.wasm';
/** Default timeout in milliseconds for loading and initialisation. */
const DEFAULT_TIMEOUT = 30000;
/** Default polling interval in milliseconds when waiting for the piko global. */
const DEFAULT_POLL_INTERVAL = 50;

/**
 * Loads and initialises the Piko WASM module.
 */
export class WasmLoader {
  /** Resolved configuration with defaults applied. */
  private readonly options: Required<
    Omit<WasmLoaderOptions, 'onStateChange' | 'onError'>
  > &
    Pick<WasmLoaderOptions, 'onStateChange' | 'onError'>;

  /** Current loader state. */
  private state: LoaderState = 'idle';
  /** Cached WASM API instance, set once loading succeeds. */
  private api: PikoWasmApi | null = null;
  /** In-flight load promise for deduplication. */
  private loadPromise: Promise<PikoWasmApi> | null = null;

  /**
   * Creates a new WasmLoader with the given configuration.
   *
   * @param options - The loader configuration.
   */
  constructor(options: WasmLoaderOptions = {}) {
    this.options = {
      wasmExecUrl: options.wasmExecUrl ?? DEFAULT_WASM_EXEC_URL,
      wasmUrl: options.wasmUrl ?? DEFAULT_WASM_URL,
      timeout: options.timeout ?? DEFAULT_TIMEOUT,
      pollInterval: options.pollInterval ?? DEFAULT_POLL_INTERVAL,
      onStateChange: options.onStateChange,
      onError: options.onError,
    };
  }

  /**
   * Returns the current loader state.
   *
   * @returns The current state of the loading state machine.
   */
  getState(): LoaderState {
    return this.state;
  }

  /**
   * Checks whether the WASM module is loaded and ready.
   *
   * @returns `true` when the API is available.
   */
  isReady(): boolean {
    return this.state === 'ready' && this.api !== null;
  }

  /**
   * Returns the WASM API if loaded, or `null` if not ready.
   *
   * @returns The API instance, or `null` if not ready.
   */
  getApi(): PikoWasmApi | null {
    return this.api;
  }

  /**
   * Loads and initialises the WASM module.
   *
   * Returns the cached API when already loaded, or deduplicates concurrent
   * calls by returning the same in-flight promise.
   *
   * @returns The initialised WASM API.
   */
  async load(): Promise<PikoWasmApi> {
    if (this.api) {
      return this.api;
    }

    if (this.loadPromise) {
      return this.loadPromise;
    }

    this.loadPromise = this.doLoad();
    return this.loadPromise;
  }

  /**
   * Resets the loader to idle state so that loading can be retried.
   */
  reset(): void {
    this.state = 'idle';
    this.api = null;
    this.loadPromise = null;

    const win = window as PikoWindow;
    delete win.piko;
  }

  /**
   * Transitions to a new loader state and notifies the listener.
   *
   * @param newState - The state to transition to.
   */
  private setState(newState: LoaderState): void {
    this.state = newState;
    this.options.onStateChange?.(newState);
  }

  /**
   * Transitions to the error state, notifies the error listener, and rethrows.
   *
   * @param error - The error to propagate.
   */
  private handleError(error: Error): never {
    this.setState('error');
    this.options.onError?.(error);
    throw error;
  }

  /**
   * Executes the full loading sequence: loads `wasm_exec.js`, fetches and
   * instantiates the WASM binary, starts the Go runtime, and waits for the
   * piko API to become available.
   *
   * @returns The initialised WASM API.
   */
  private async doLoad(): Promise<PikoWasmApi> {
    try {
      this.setState('loading-runtime');
      await this.loadWasmExec();

      this.setState('loading-wasm');
      const instance = await this.loadWasm();

      this.setState('starting');
      await this.startGoRuntime(instance);

      this.setState('initialising');
      this.api = await this.waitForPikoAndInit();

      this.setState('ready');
      return this.api;
    } catch (error) {
      this.loadPromise = null;
      this.handleError(error instanceof Error ? error : new Error(String(error)));
    }
  }

  /**
   * Loads `wasm_exec.js` by injecting a script tag into the document head.
   *
   * Skips injection when the `Go` constructor is already available on `window`.
   */
  private async loadWasmExec(): Promise<void> {
    const win = window as PikoWindow;

    if (win.Go) {
      return;
    }

    return new Promise<void>((resolve, reject) => {
      const script = document.createElement('script');
      script.src = this.options.wasmExecUrl;
      script.async = true;

      const timeout = setTimeout(() => {
        script.remove();
        reject(
          new TimeoutError(
            `Loading wasm_exec.js timed out after ${this.options.timeout}ms`
          )
        );
      }, this.options.timeout);

      script.onload = () => {
        clearTimeout(timeout);
        if (win.Go) {
          resolve();
        } else {
          reject(
            new WasmExecLoadError(
              'wasm_exec.js loaded but Go constructor not found'
            )
          );
        }
      };

      script.onerror = () => {
        clearTimeout(timeout);
        script.remove();
        reject(
          new WasmExecLoadError(
            `Failed to load wasm_exec.js from ${this.options.wasmExecUrl}`
          )
        );
      };

      document.head.appendChild(script);
    });
  }

  /**
   * Fetches the WASM binary and instantiates it with streaming compilation.
   *
   * @returns An object containing the Go instance and WebAssembly instance.
   */
  private async loadWasm(): Promise<{ go: GoInstance; instance: WebAssembly.Instance }> {
    const win = window as PikoWindow;

    if (!win.Go) {
      throw new WasmExecLoadError('Go constructor not available');
    }

    const go = new win.Go();

    try {
      const controller = new AbortController();
      const timeoutId = setTimeout(() => controller.abort(), this.options.timeout);

      const response = await fetch(this.options.wasmUrl, {
        signal: controller.signal,
      });

      clearTimeout(timeoutId);

      if (!response.ok) {
        throw new WasmBinaryLoadError(
          `Failed to fetch WASM binary: ${response.status} ${response.statusText}`
        );
      }

      const result = await WebAssembly.instantiateStreaming(
        response,
        go.importObject
      );

      return { go, instance: result.instance };
    } catch (error) {
      if (error instanceof WasmBinaryLoadError) {
        throw error;
      }
      if (error instanceof DOMException && error.name === 'AbortError') {
        throw new TimeoutError(
          `Loading WASM binary timed out after ${this.options.timeout}ms`
        );
      }
      throw new WasmBinaryLoadError(
        `Failed to load WASM binary: ${error instanceof Error ? error.message : String(error)}`,
        error instanceof Error ? error : undefined
      );
    }
  }

  /**
   * Starts the Go runtime without waiting for the program to exit.
   *
   * `go.run()` resolves when the Go program exits; the program itself sets
   * up `window.piko` and then blocks on a channel receive. This method
   * resolves after a brief delay to give the runtime time to initialise.
   *
   * @param wasmResult - An object holding the Go and WebAssembly instances to run.
   */
  private startGoRuntime(wasmResult: {
    go: GoInstance;
    instance: WebAssembly.Instance;
  }): Promise<void> {
    return new Promise<void>((resolve, reject) => {
      const { go, instance } = wasmResult;

      go.run(instance).catch((error: Error) => {
        reject(
          new GoRuntimeError(
            `Go runtime exited unexpectedly: ${error.message}`,
            error
          )
        );
      });

      setTimeout(resolve, 10);
    });
  }

  /**
   * Polls for the `window.piko` global and calls `piko.init()` once available.
   *
   * @returns The initialised WASM API.
   */
  private async waitForPikoAndInit(): Promise<PikoWasmApi> {
    const win = window as PikoWindow;
    const startTime = Date.now();

    return new Promise<PikoWasmApi>((resolve, reject) => {
      const poll = () => {
        if (Date.now() - startTime > this.options.timeout) {
          reject(
            new TimeoutError(
              `Waiting for piko API timed out after ${this.options.timeout}ms`
            )
          );
          return;
        }

        if (!win.piko) {
          setTimeout(poll, this.options.pollInterval);
          return;
        }

        win.piko
          .init()
          .then(() => {
            if (!win.piko) {
              reject(new ApiNotAvailableError('piko API disappeared after init'));
              return;
            }
            resolve(win.piko);
          })
          .catch((error: Error) => {
            reject(
              new GoRuntimeError(
                `piko.init() failed: ${error.message}`,
                error
              )
            );
          });
      };

      poll();
    });
  }
}

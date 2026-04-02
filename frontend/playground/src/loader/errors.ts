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

/** Base error class for all loader errors. */
export class LoaderError extends Error {
  /**
   * Creates a new loader error.
   *
   * @param message - A human-readable error description.
   * @param code - A machine-readable error code.
   */
  constructor(
    message: string,
    public readonly code: string
  ) {
    super(message);
    this.name = 'LoaderError';
  }
}

/** Thrown when loading `wasm_exec.js` fails. */
export class WasmExecLoadError extends LoaderError {
  /**
   * Creates a new WASM exec load error.
   *
   * @param message - A human-readable error description.
   * @param cause - The underlying error, if any.
   */
  constructor(
    message: string,
    public readonly cause?: Error
  ) {
    super(message, 'WASM_EXEC_LOAD_ERROR');
    this.name = 'WasmExecLoadError';
  }
}

/** Thrown when loading the WASM binary fails. */
export class WasmBinaryLoadError extends LoaderError {
  /**
   * Creates a new WASM binary load error.
   *
   * @param message - A human-readable error description.
   * @param cause - The underlying error, if any.
   */
  constructor(
    message: string,
    public readonly cause?: Error
  ) {
    super(message, 'WASM_BINARY_LOAD_ERROR');
    this.name = 'WasmBinaryLoadError';
  }
}

/** Thrown when the Go runtime fails to start. */
export class GoRuntimeError extends LoaderError {
  /**
   * Creates a new Go runtime error.
   *
   * @param message - A human-readable error description.
   * @param cause - The underlying error, if any.
   */
  constructor(
    message: string,
    public readonly cause?: Error
  ) {
    super(message, 'GO_RUNTIME_ERROR');
    this.name = 'GoRuntimeError';
  }
}

/** Thrown when initialisation times out. */
export class TimeoutError extends LoaderError {
  /**
   * Creates a new timeout error.
   *
   * @param message - A human-readable error description.
   */
  constructor(message: string) {
    super(message, 'TIMEOUT_ERROR');
    this.name = 'TimeoutError';
  }
}

/** Thrown when the piko API is not available after initialisation. */
export class ApiNotAvailableError extends LoaderError {
  /**
   * Creates a new API not available error.
   *
   * @param message - A human-readable error description.
   */
  constructor(message: string) {
    super(message, 'API_NOT_AVAILABLE');
    this.name = 'ApiNotAvailableError';
  }
}

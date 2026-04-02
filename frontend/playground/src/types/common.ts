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

/** Describes a position in source code. */
export interface Location {
  /** File path. */
  filePath: string;

  /** 1-indexed line number. */
  line: number;

  /** 1-indexed column number. */
  column: number;
}

/** Holds a line and column pair within source code. */
export interface Position {
  /** 1-indexed line number. */
  line: number;

  /** 1-indexed column number. */
  column: number;
}

/** Describes a span between two positions in source code. */
export interface Range {
  /** Range start position. */
  start: Position;

  /** Range end position. */
  end: Position;
}

/** Diagnostic severity levels. */
export type DiagnosticSeverity = 'error' | 'warning' | 'info' | 'hint';

/** Holds a warning or error produced during analysis. */
export interface Diagnostic {
  /** Diagnostic severity. */
  severity: DiagnosticSeverity;

  /** Diagnostic message. */
  message: string;

  /** Where the diagnostic applies. */
  location?: Location;

  /** Optional diagnostic code. */
  code?: string;
}

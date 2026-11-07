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

import {parseSfcBlocks, SfcBlockKind} from './sfcBlocks';

/**
 * A content-oriented view of the blocks in a Piko (.pk or .pkc) file.
 *
 * Thin adapter over the shared block parser in sfcBlocks.ts, which mirrors
 * the server's internal/sfcparser.
 */

/**
 * Holds the data for a single block in a Piko file.
 */
export interface SfcBlock {
    /** The type of block. */
    type: SfcBlockKind;
    /** The text content inside the block tags. */
    content: string;
    /** The start offset of the block in the file. */
    start: number;
    /** The end offset of the block in the file. */
    end: number;
}

/**
 * Parses a Piko file and extracts all blocks in source order.
 *
 * @param text - The full text of the Piko file.
 * @returns The list of blocks found in the file.
 */
export function parseSfc(text: string): SfcBlock[] {
    return parseSfcBlocks(text).map(block => ({
        type: block.kind,
        content: block.content,
        start: block.start,
        end: block.end,
    }));
}

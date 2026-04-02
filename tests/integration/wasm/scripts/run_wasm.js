#!/usr/bin/env node
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

/**
 * run_wasm.js - Node.js WASM test runner for Piko
 *
 * This script loads the Piko WASM module and executes functions via the JavaScript API.
 * It reads a JSON request from stdin and writes the JSON response to stdout.
 *
 * Usage:
 *   echo '{"function":"generate","args":{...}}' | node run_wasm.js
 *
 * The script expects the following files in the same directory:
 *   - wasm_exec.js (Go WASM runtime)
 *   - piko.wasm (compiled Piko WASM binary)
 */

const fs = require('fs');
const path = require('path');

// Load Go WASM support - this creates the global Go class
require('./wasm_exec.js');

/**
 * Waits for the piko global object to become available.
 * The Go WASM module sets up this global asynchronously after go.run() is called.
 *
 * @param {number} timeout - Maximum time to wait in milliseconds
 * @returns {Promise<void>} - Resolves when piko is available, rejects on timeout
 */
function waitForPiko(timeout = 10000) {
    return new Promise((resolve, reject) => {
        const start = Date.now();
        const check = () => {
            if (typeof piko !== 'undefined') {
                resolve();
            } else if (Date.now() - start > timeout) {
                reject(new Error('Timeout waiting for piko global to be defined'));
            } else {
                setTimeout(check, 10);
            }
        };
        check();
    });
}

/**
 * Reads all data from stdin as a string.
 *
 * @returns {Promise<string>} - The complete stdin content
 */
function readStdin() {
    return new Promise((resolve, reject) => {
        let data = '';
        process.stdin.setEncoding('utf8');
        process.stdin.on('data', chunk => {
            data += chunk;
        });
        process.stdin.on('end', () => {
            resolve(data);
        });
        process.stdin.on('error', reject);
    });
}

/**
 * Main entry point for the WASM test runner.
 */
async function main() {
    // Read request JSON from stdin
    const input = await readStdin();
    let request;
    try {
        request = JSON.parse(input);
    } catch (err) {
        console.log(JSON.stringify({
            success: false,
            error: `Invalid JSON input: ${err.message}`
        }));
        process.exit(1);
    }

    // Load WASM binary
    const go = new Go();
    const wasmPath = path.join(__dirname, 'piko.wasm');

    if (!fs.existsSync(wasmPath)) {
        console.log(JSON.stringify({
            success: false,
            error: `WASM binary not found at: ${wasmPath}`
        }));
        process.exit(1);
    }

    const wasmBuffer = fs.readFileSync(wasmPath);

    let result;
    try {
        result = await WebAssembly.instantiate(wasmBuffer, go.importObject);
    } catch (err) {
        console.log(JSON.stringify({
            success: false,
            error: `Failed to instantiate WASM: ${err.message}`
        }));
        process.exit(1);
    }

    // Start Go runtime (non-blocking - it runs in the background)
    // Do not await this - the Go main() function runs forever (select {})
    go.run(result.instance);

    // Wait for piko global to be available
    try {
        await waitForPiko();
    } catch (err) {
        console.log(JSON.stringify({
            success: false,
            error: err.message
        }));
        process.exit(1);
    }

    // Initialise WASM module
    try {
        await piko.init();
    } catch (err) {
        console.log(JSON.stringify({
            success: false,
            error: `Failed to initialise piko: ${err.message || err}`
        }));
        process.exit(1);
    }

    // Call the requested function
    let response;
    try {
        switch (request.function) {
            case 'generate':
                response = await piko.generate(request.args);
                break;
            case 'render':
                response = await piko.render(request.args);
                break;
            case 'dynamicRender':
                response = await piko.dynamicRender(request.args);
                break;
            case 'analyse':
                response = await piko.analyse(request.args);
                break;
            case 'validate':
                response = await piko.validate(request.args);
                break;
            case 'getCompletions':
                response = await piko.getCompletions(request.args);
                break;
            case 'getHover':
                response = await piko.getHover(request.args);
                break;
            case 'getRuntimeInfo':
                response = piko.getRuntimeInfo();
                break;
            default:
                response = {
                    success: false,
                    error: `Unknown function: ${request.function}`
                };
        }
    } catch (err) {
        response = {
            success: false,
            error: `Function execution failed: ${err.message || err}`
        };
    }

    // Output response as JSON to stdout
    console.log(JSON.stringify(response));

    // Exit cleanly (the Go runtime is still running in the background)
    process.exit(0);
}

main().catch(err => {
    console.log(JSON.stringify({
        success: false,
        error: `Unexpected error: ${err.message}`
    }));
    process.exit(1);
});

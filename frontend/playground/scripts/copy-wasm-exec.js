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

#!/usr/bin/env node
/**
 * Copies wasm_exec.js from the Go SDK to the assets directory.
 * This file is required to run Go WASM in the browser.
 */

const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');

// Get GOROOT from the environment
let goroot;
try {
  goroot = execSync('go env GOROOT', { encoding: 'utf8' }).trim();
} catch (error) {
  console.error('Error: Could not determine GOROOT. Is Go installed?');
  process.exit(1);
}

// wasm_exec.js location changed in Go 1.24+
// - Go 1.24+: lib/wasm/wasm_exec.js
// - Go <1.24: misc/wasm/wasm_exec.js
const possiblePaths = [
  path.join(goroot, 'lib', 'wasm', 'wasm_exec.js'),
  path.join(goroot, 'misc', 'wasm', 'wasm_exec.js'),
];

let source = null;
for (const p of possiblePaths) {
  if (fs.existsSync(p)) {
    source = p;
    break;
  }
}

if (!source) {
  console.error('Error: Could not find wasm_exec.js in Go SDK.');
  console.error('Searched paths:');
  possiblePaths.forEach(p => console.error(`  - ${p}`));
  process.exit(1);
}

const dest = path.join(__dirname, '..', 'assets', 'wasm_exec.js');

// Ensure assets directory exists
const assetsDir = path.dirname(dest);
if (!fs.existsSync(assetsDir)) {
  fs.mkdirSync(assetsDir, { recursive: true });
}

// Copy the file
try {
  fs.copyFileSync(source, dest);
  console.log(`Copied wasm_exec.js from ${source}`);
  console.log(`                      to ${dest}`);
} catch (error) {
  console.error(`Error copying wasm_exec.js: ${error.message}`);
  process.exit(1);
}

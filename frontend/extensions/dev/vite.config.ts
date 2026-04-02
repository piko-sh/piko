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

import { defineConfig } from 'vite';
import terser from '@rollup/plugin-terser';
import { compression } from 'vite-plugin-compression2';
import etagPlugin from '../../core/vite-plugin-etag';
import sriPlugin from '../../core/vite-plugin-sri';
import * as path from 'path';

export default defineConfig({
  plugins: [
    etagPlugin(),
    sriPlugin(),
    compression({
      include: /\.(js|mjs)$/,
      threshold: 512,
      skipIfLargerOrEqual: true,
      algorithms: ['gzip'],
      filename: (pathStr) => {
        const ext = path.extname(pathStr);
        const base = path.basename(pathStr, ext);
        const dir = path.dirname(pathStr);
        return path.join(dir, `${base}${ext}.gz`);
      },
      compressionOptions: { level: 9 }
    }),
    compression({
      include: /\.(js|mjs)$/,
      threshold: 512,
      skipIfLargerOrEqual: true,
      algorithms: ['brotliCompress'],
      filename: (pathStr) => {
        const ext = path.extname(pathStr);
        const base = path.basename(pathStr, ext);
        const dir = path.dirname(pathStr);
        return path.join(dir, `${base}${ext}.br`);
      },
      compressionOptions: {}
    }),
  ],

  test: {
    globals: true,
    environment: 'jsdom',
    include: ['src/**/*.spec.ts'],
    coverage: {
      provider: 'v8',
      all: true,
      include: ['src/**/*.ts'],
      exclude: ['**/*.spec.ts', '**/index.ts'],
    },
  },

  build: {
    outDir: '../../../internal/daemon/daemon_frontend/built',
    emptyOutDir: false,
    sourcemap: true,
    minify: false,

    lib: {
      formats: ['es', 'iife'],
      entry: './src/main.ts',
      name: 'PPDev',
      fileName: (format) => `ppframework.dev.${format}.js`
    },

    rollupOptions: {
      external: [],
      output: [
        {
          format: 'es',
          entryFileNames: 'ppframework.dev.es.js',
          plugins: [],
        },
        {
          format: 'es',
          entryFileNames: 'ppframework.dev.min.es.js',
          plugins: [terser({
            compress: {
              defaults: true,
              passes: 2,
              pure_funcs: ['console.log', 'console.info', 'console.debug'],
            },
            mangle: true,
            format: { comments: false }
          })],
        },
        {
          format: 'iife',
          entryFileNames: 'ppframework.dev.min.js',
          name: 'PPDev',
          plugins: [terser({
            compress: {
              defaults: true,
              passes: 2,
              pure_funcs: ['console.log', 'console.info', 'console.debug'],
            },
            mangle: true,
            format: { comments: false }
          })],
        }
      ]
    }
  },
});

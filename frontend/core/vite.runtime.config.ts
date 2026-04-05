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
import etagPlugin from './vite-plugin-etag';
import sriPlugin from './vite-plugin-sri';
import { compression } from 'vite-plugin-compression2';
import * as path from 'path';

export default defineConfig({
  define: {
    __DEV__: 'true',
  },

  resolve: {
    alias: {
      '@': path.resolve(__dirname, 'src')
    }
  },

  plugins: [
    compression({
      include: /\.(html|xml|css|json|js|mjs|svg|yaml|yml|toml|ico)$/,
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
      include: /\.(html|xml|css|json|js|mjs|svg|yaml|yml|toml|ico)$/,
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

    etagPlugin(),
    sriPlugin(),
  ],

  build: {
    outDir: '../../internal/daemon/daemon_frontend/built',
    emptyOutDir: false,
    sourcemap: true,
    minify: false,

    lib: {
      formats: ['es'],
      entry: './src/runtime.ts',
      name: 'PPRuntime',
      fileName: (format) => `ppframework.runtime.${format}.js`
    },

    rollupOptions: {
      output: [
        {
          format: 'es',
          entryFileNames: 'ppframework.runtime.es.js',
          plugins: [],
        },
        {
          format: 'es',
          entryFileNames: 'ppframework.runtime.min.es.js',
          plugins: [terser({
            compress: {
              defaults: true,
              passes: 4,
              pure_funcs: ['console.log', 'console.warn', 'console.info', 'console.debug'],
              pure_getters: true,
              unsafe: true,
              unsafe_arrows: true,
              unsafe_comps: true,
              unsafe_math: true,
              unsafe_methods: true,
              unsafe_proto: true,
              unsafe_undefined: true,
              reduce_vars: true,
              reduce_funcs: true,
              booleans_as_integers: true,
              toplevel: true,
              hoist_props: true,
              join_vars: true,
              negate_iife: true,
              sequences: 500,
              inline: 3,
              global_defs: { __DEV__: false }
            },
            mangle: {
              toplevel: true,
              properties: {
                regex: /^_/,
                reserved: ['_csrf_ephemeral_token', '_f', '_class', '_style', '_k', '_c', '_s', '_ref', '_type', '_registerCapability', '_emitHook', '_markReady', '__pikoShimData__']
              }
            },
            format: {
              comments: false
            }
          })],
        },
      ]
    }
  },
});

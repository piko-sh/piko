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
import dts from 'vite-plugin-dts';
import * as path from 'path';

export default defineConfig({
  resolve: {
    alias: {
      '@': path.resolve(__dirname, 'src'),
    },
  },

  plugins: [
    dts({
      insertTypesEntry: true,
      rollupTypes: true,
      outDir: 'dist',
    }),
  ],

  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: ['./test/test-setup.ts'],
    include: ['src/**/*.spec.ts', 'test/**/*.spec.ts'],
    exclude: ['test/integration/**'],
    coverage: {
      provider: 'v8',
      reporter: ['text', 'json', 'html'],
      exclude: ['test/**', '**/*.spec.ts', '**/index.ts'],
    },
  },

  build: {
    outDir: 'dist',
    emptyOutDir: true,
    sourcemap: true,
    minify: false,

    lib: {
      formats: ['es', 'umd'],
      entry: './src/main.ts',
      name: 'PikoPlayground',
      fileName: (format) => `piko.playground.${format}.js`,
    },

    rollupOptions: {
      output: [
        {
          format: 'es',
          entryFileNames: 'piko.playground.es.js',
          plugins: [],
        },
        {
          format: 'es',
          entryFileNames: 'piko.playground.min.es.js',
          plugins: [
            terser({
              compress: {
                defaults: true,
                passes: 2,
                pure_funcs: ['console.log', 'console.info', 'console.debug'],
              },
              mangle: true,
              format: { comments: false },
            }),
          ],
        },
        {
          format: 'umd',
          entryFileNames: 'piko.playground.umd.js',
          name: 'PikoPlayground',
          plugins: [],
        },
        {
          format: 'umd',
          entryFileNames: 'piko.playground.min.umd.js',
          name: 'PikoPlayground',
          plugins: [
            terser({
              compress: {
                defaults: true,
                passes: 2,
                pure_funcs: ['console.log', 'console.info', 'console.debug'],
              },
              mangle: true,
              format: { comments: false },
            }),
          ],
        },
      ],
    },
  },
});

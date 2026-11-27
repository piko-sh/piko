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

import { defineConfig, transformWithEsbuild, type Plugin, type PluginOption } from 'vite';
import { promises as fs } from 'node:fs';
import terser from '@rollup/plugin-terser';
import { visualizer } from 'rollup-plugin-visualizer';
import etagPlugin from './vite-plugin-etag';
import sriPlugin from './vite-plugin-sri';
import { compression } from 'vite-plugin-compression2';
import * as path from 'node:path';

const isAnalyze = process.env.ANALYZE === 'true';

/**
 * The bundle exports whose names start with an underscore. Generated page and partial
 * scripts import them by name, and the UMD build exposes exports as properties, so the
 * property mangling below must keep these names.
 */
const UNDERSCORE_EXPORTS = [
  '_createPKContext',
  '_createPagePKContext',
  '_createRefs',
  '_currentPageVisit',
  '_initCleanupObserver',
  '_registerLifecycle',
  '_runPageCleanup',
];

/**
 * Property names the compiler writes into component code and that the component runtime
 * reads, or the other way round. The runtime must use the same names as compiled code.
 */
const COMPILED_COMPONENT_PROPERTIES = [
  '_f', '_class', '_style', '_k', '_c', '_s', '_ref', '_type', '_updateFormState',
];

/** Property names in server requests and responses. */
const SERVER_PROPERTIES = ['_csrf_ephemeral_token', '_helpers'];

/**
 * Property names the runtime reads on the namespace of a generated page or partial module.
 */
const MODULE_PROTOCOL_PROPERTIES = ['__reinit__', '__PARTIAL_ID__'];

/**
 * Window properties that other bundles, the server's pages or any other outside code
 * read or write.
 */
const WINDOW_PROPERTIES = [
  '__pikoRuntime', '__pikoPageContext', '__pikoDebug', '__PP_HOOKS_QUEUE__', '__piko_animation',
];

/**
 * Names terser mangles and keeps in the minified builds.
 *
 * Only names with one leading underscore are renamed. Names with two leading underscores
 * belong to the protocol between the runtime and other code, so they always keep their
 * names, even when a new one is not in the lists above.
 */
const MANGLE_OPTIONS = {
  toplevel: true,
  properties: {
    regex: /^_(?!_)/,
    reserved: [
      ...COMPILED_COMPONENT_PROPERTIES,
      ...SERVER_PROPERTIES,
      ...MODULE_PROTOCOL_PROPERTIES,
      ...WINDOW_PROPERTIES,
      ...UNDERSCORE_EXPORTS,
    ],
  },
};

/** Source of the runtime recovery script, which is built on its own. */
const RUNTIME_RECOVERY_SOURCE = path.resolve(__dirname, 'src/runtimeRecovery.ts');

/** File name of the built runtime recovery script in the output directory. */
const RUNTIME_RECOVERY_FILE = 'runtime-recovery.js';

/**
 * Builds src/runtimeRecovery.ts into a small classic script next to the bundles.
 *
 * Pages load it with a plain `<script defer>` at a fixed URL, so it cannot be an ES
 * module and must not import anything. It is emitted once per build, with the first
 * output, so the compression, ETag and SRI plugins treat it like any other bundle file.
 *
 * @returns The plugin.
 */
function runtimeRecoveryPlugin(): Plugin {
  let emitted = false;

  return {
    name: 'piko-runtime-recovery',
    apply: 'build',

    buildStart() {
      emitted = false;
      this.addWatchFile(RUNTIME_RECOVERY_SOURCE);
    },

    async generateBundle() {
      if (emitted) {
        return;
      }
      emitted = true;
      const source = await fs.readFile(RUNTIME_RECOVERY_SOURCE, 'utf8');
      const result = await transformWithEsbuild(source, RUNTIME_RECOVERY_SOURCE, {
        format: 'iife',
        target: 'es2017',
        legalComments: 'none',
        sourcemap: false,
      });
      this.emitFile({ type: 'asset', fileName: RUNTIME_RECOVERY_FILE, source: result.code });
    },
  };
}

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
    runtimeRecoveryPlugin(),

    compression({
      include: /\.(html|xml|css|json|js|mjs|svg|yaml|yml|toml|ico)$/,
      threshold: 1024,
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
      threshold: 1024,
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

    ...(isAnalyze ? [visualizer({
      filename: 'bundle-analysis.html',
      open: true,
      gzipSize: true,
      brotliSize: true,
      template: 'treemap'
    }) as PluginOption] : [])
  ],

  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: ['./test/test-setup.ts'],
    alias: {
      /**
       * Generated page scripts import the runtime from the URL the server serves the
       * bundle at. Here that URL resolves to a shim that re-exports the runtime source.
       */
      '/_piko/dist/ppframework.core.es.js': path.resolve(__dirname, 'test/pkRuntimeShim.ts'),
      '@generator-goldens': path.resolve(__dirname, '../../internal/generator/generator_test/generator/testdata'),
    },
    include: [
      'src/**/*.spec.ts',
      'test/**/*.spec.ts'
    ],
    coverage: {
      provider: 'v8',
      reporter: ['text', 'json', 'html'],
      exclude: [
        'test/**',
        '**/*.spec.ts',
        '**/index.ts'
      ]
    },
  },

  build: {
    outDir: '../../internal/daemon/daemon_frontend/built',
    emptyOutDir: false,
    sourcemap: true,
    minify: false,

    lib: {
      formats: ['es', 'umd'],
      entry: './src/main.ts',
      name: 'PPFramework',
      fileName: (format) => `ppframework.core.${format}.js`
    },

    rollupOptions: {
      output: [
        {
          format: 'es',
          entryFileNames: 'ppframework.core.es.js',
          plugins: [],
        },
        {
          format: 'es',
          entryFileNames: 'ppframework.core.min.es.js',
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
              toplevel: true,
              hoist_props: true,
              join_vars: true,
              negate_iife: true,
              sequences: 500,
              inline: 3,
              global_defs: { __DEV__: false }
            },
            mangle: MANGLE_OPTIONS,
            format: {
              comments: false
            }
          })],
        },
        {
          format: 'umd',
          entryFileNames: 'ppframework.core.min.umd.js',
          name: 'PPFramework',
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
              hoist_props: true,
              join_vars: true,
              negate_iife: true,
              sequences: 500,
              inline: 3,
              global_defs: { __DEV__: false }
            },
            mangle: MANGLE_OPTIONS,
            format: {
              comments: false
            }
          })],
        }
      ]
    }
  },
});

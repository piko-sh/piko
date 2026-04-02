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

import {describe, expect, it} from 'vitest';

import {
    buildLspArgsFromConfig,
    getBinaryName,
    isValidFileName,
    LspConfig,
    nodeArchToGoArch,
    resolvePlatformBinaryPath,
    validateFileName,
} from './extensionUtils';

describe('extensionUtils', () => {
    describe('validateFileName', () => {
        describe('valid names', () => {
            it('should accept simple alphanumeric names', () => {
                expect(validateFileName('component')).toBeNull();
                expect(validateFileName('MyComponent')).toBeNull();
                expect(validateFileName('Component123')).toBeNull();
            });

            it('should accept names with hyphens', () => {
                expect(validateFileName('my-component')).toBeNull();
                expect(validateFileName('header-nav')).toBeNull();
                expect(validateFileName('a-b-c')).toBeNull();
            });

            it('should accept names with underscores', () => {
                expect(validateFileName('my_component')).toBeNull();
                expect(validateFileName('header_nav')).toBeNull();
                expect(validateFileName('a_b_c')).toBeNull();
            });

            it('should accept mixed valid characters', () => {
                expect(validateFileName('my-component_v2')).toBeNull();
                expect(validateFileName('Header_Nav-Item')).toBeNull();
            });

            it('should accept single character names', () => {
                expect(validateFileName('a')).toBeNull();
                expect(validateFileName('Z')).toBeNull();
                expect(validateFileName('1')).toBeNull();
            });
        });

        describe('invalid names', () => {
            it('should reject empty string', () => {
                expect(validateFileName('')).toBe('File name cannot be empty');
            });

            it('should reject whitespace-only string', () => {
                expect(validateFileName('   ')).toBe('File name cannot be empty');
                expect(validateFileName('\t')).toBe('File name cannot be empty');
                expect(validateFileName('\n')).toBe('File name cannot be empty');
            });

            it('should reject names with spaces', () => {
                expect(validateFileName('my component')).toContain('only contain');
            });

            it('should reject names with dots', () => {
                expect(validateFileName('my.component')).toContain('only contain');
                expect(validateFileName('file.pk')).toContain('only contain');
            });

            it('should reject names with special characters', () => {
                expect(validateFileName('my@component')).toContain('only contain');
                expect(validateFileName('my#component')).toContain('only contain');
                expect(validateFileName('my$component')).toContain('only contain');
                expect(validateFileName('my!component')).toContain('only contain');
            });

            it('should reject path traversal attempts', () => {
                expect(validateFileName('../etc/passwd')).toContain('only contain');
                expect(validateFileName('..\\windows')).toContain('only contain');
            });

            it('should reject names with slashes', () => {
                expect(validateFileName('path/to/file')).toContain('only contain');
                expect(validateFileName('path\\to\\file')).toContain('only contain');
            });
        });
    });

    describe('resolvePlatformBinaryPath', () => {
        describe('Linux', () => {
            it('should resolve Linux x64 path', () => {
                expect(resolvePlatformBinaryPath('linux', 'x64')).toBe('bin/linux-amd64/piko-lsp');
            });

            it('should resolve Linux arm64 path', () => {
                expect(resolvePlatformBinaryPath('linux', 'arm64')).toBe('bin/linux-arm64/piko-lsp');
            });
        });

        describe('macOS', () => {
            it('should resolve macOS x64 path', () => {
                expect(resolvePlatformBinaryPath('darwin', 'x64')).toBe('bin/darwin-amd64/piko-lsp');
            });

            it('should resolve macOS arm64 path (Apple Silicon)', () => {
                expect(resolvePlatformBinaryPath('darwin', 'arm64')).toBe('bin/darwin-arm64/piko-lsp');
            });
        });

        describe('Windows', () => {
            it('should resolve Windows x64 path with .exe extension', () => {
                expect(resolvePlatformBinaryPath('win32', 'x64')).toBe('bin/win32-amd64/piko-lsp.exe');
            });

            it('should resolve Windows arm64 path with .exe extension', () => {
                expect(resolvePlatformBinaryPath('win32', 'arm64')).toBe('bin/win32-arm64/piko-lsp.exe');
            });
        });
    });

    describe('buildLspArgsFromConfig', () => {
        it('should include basic TCP args', () => {
            const config: LspConfig = {
                enablePprof: false,
                pprofPort: 6060,
                enableFormatting: false,
                enableFileLogging: false,
            };

            const args = buildLspArgsFromConfig(4389, config);

            expect(args).toEqual(['--tcp', '--port=4389']);
        });

        it('should include pprof when enabled', () => {
            const config: LspConfig = {
                enablePprof: true,
                pprofPort: 6060,
                enableFormatting: false,
                enableFileLogging: false,
            };

            const args = buildLspArgsFromConfig(4389, config);

            expect(args).toContain('--pprof');
            expect(args).toContain('--pprof-port=6060');
        });

        it('should use custom pprof port', () => {
            const config: LspConfig = {
                enablePprof: true,
                pprofPort: 7070,
                enableFormatting: false,
                enableFileLogging: false,
            };

            const args = buildLspArgsFromConfig(4389, config);

            expect(args).toContain('--pprof-port=7070');
        });

        it('should include formatting when enabled', () => {
            const config: LspConfig = {
                enablePprof: false,
                pprofPort: 6060,
                enableFormatting: true,
                enableFileLogging: false,
            };

            const args = buildLspArgsFromConfig(4389, config);

            expect(args).toContain('--formatting');
        });

        it('should include file-logging when enabled', () => {
            const config: LspConfig = {
                enablePprof: false,
                pprofPort: 6060,
                enableFormatting: false,
                enableFileLogging: true,
            };

            const args = buildLspArgsFromConfig(4389, config);

            expect(args).toContain('--file-logging');
        });

        it('should include all flags when all enabled', () => {
            const config: LspConfig = {
                enablePprof: true,
                pprofPort: 6060,
                enableFormatting: true,
                enableFileLogging: true,
            };

            const args = buildLspArgsFromConfig(5000, config);

            expect(args).toContain('--tcp');
            expect(args).toContain('--port=5000');
            expect(args).toContain('--pprof');
            expect(args).toContain('--pprof-port=6060');
            expect(args).toContain('--formatting');
            expect(args).toContain('--file-logging');
        });

        it('should use different ports correctly', () => {
            const config: LspConfig = {
                enablePprof: false,
                pprofPort: 6060,
                enableFormatting: false,
                enableFileLogging: false,
            };

            expect(buildLspArgsFromConfig(1234, config)).toContain('--port=1234');
            expect(buildLspArgsFromConfig(8080, config)).toContain('--port=8080');
            expect(buildLspArgsFromConfig(0, config)).toContain('--port=0');
        });
    });

    describe('isValidFileName', () => {
        it('should return true for valid names', () => {
            expect(isValidFileName('valid-name')).toBe(true);
            expect(isValidFileName('ValidName')).toBe(true);
            expect(isValidFileName('valid_name_123')).toBe(true);
        });

        it('should return false for invalid names', () => {
            expect(isValidFileName('')).toBe(false);
            expect(isValidFileName('invalid name')).toBe(false);
            expect(isValidFileName('invalid.name')).toBe(false);
        });
    });

    describe('getBinaryName', () => {
        it('should return piko-lsp for Linux', () => {
            expect(getBinaryName('linux')).toBe('piko-lsp');
        });

        it('should return piko-lsp for macOS', () => {
            expect(getBinaryName('darwin')).toBe('piko-lsp');
        });

        it('should return piko-lsp.exe for Windows', () => {
            expect(getBinaryName('win32')).toBe('piko-lsp.exe');
        });

        it('should return piko-lsp for unknown platforms', () => {
            expect(getBinaryName('freebsd')).toBe('piko-lsp');
            expect(getBinaryName('unknown')).toBe('piko-lsp');
        });
    });

    describe('nodeArchToGoArch', () => {
        it('should convert x64 to amd64', () => {
            expect(nodeArchToGoArch('x64')).toBe('amd64');
        });

        it('should preserve arm64', () => {
            expect(nodeArchToGoArch('arm64')).toBe('arm64');
        });

        it('should preserve other architectures', () => {
            expect(nodeArchToGoArch('arm')).toBe('arm');
            expect(nodeArchToGoArch('ia32')).toBe('ia32');
        });
    });
});

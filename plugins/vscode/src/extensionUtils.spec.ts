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
    buildServerArgs,
    nodePlatformToGoOs,
    getBinaryName,
    isValidFileName,
    LspConfig,
    MAX_FILE_NAME_LENGTH,
    MAX_TCP_PORT,
    nodeArchToGoArch,
    parseLimit,
    parsePort,
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
                expect(resolvePlatformBinaryPath('linux', 'x64')).toBe('bin/linux-amd64/pikopls');
            });

            it('should resolve Linux arm64 path', () => {
                expect(resolvePlatformBinaryPath('linux', 'arm64')).toBe('bin/linux-arm64/pikopls');
            });
        });

        describe('macOS', () => {
            it('should resolve macOS x64 path', () => {
                expect(resolvePlatformBinaryPath('darwin', 'x64')).toBe('bin/darwin-amd64/pikopls');
            });

            it('should resolve macOS arm64 path (Apple Silicon)', () => {
                expect(resolvePlatformBinaryPath('darwin', 'arm64')).toBe('bin/darwin-arm64/pikopls');
            });
        });

        describe('Windows', () => {
            it('should resolve Windows x64 path with .exe extension', () => {
                expect(resolvePlatformBinaryPath('win32', 'x64')).toBe('bin/windows-amd64/pikopls.exe');
            });

            it('should resolve Windows arm64 path with .exe extension', () => {
                expect(resolvePlatformBinaryPath('win32', 'arm64')).toBe('bin/windows-arm64/pikopls.exe');
            });
        });
    });

    describe('buildServerArgs', () => {
        const baseConfig: LspConfig = {
            enablePprof: false,
            pprofPort: 6060,
            enableFormatting: false,
            enableFileLogging: false,
        };

        it('should pass no transport flags for stdio', () => {
            expect(buildServerArgs({kind: 'stdio'}, baseConfig)).toEqual([]);
        });

        it('should limit TCP to a single connection', () => {
            expect(buildServerArgs({kind: 'tcp', port: 4389}, baseConfig))
                .toEqual(['--tcp', '--port=4389', '--max-connections=1']);
        });

        it('should use different ports correctly', () => {
            expect(buildServerArgs({kind: 'tcp', port: 1234}, baseConfig)).toContain('--port=1234');
            expect(buildServerArgs({kind: 'tcp', port: 0}, baseConfig)).toContain('--port=0');
        });

        it.each<[string, Partial<LspConfig>, string[]]>([
            ['pprof', {enablePprof: true}, ['--pprof', '--pprof-port=6060']],
            ['custom pprof port', {enablePprof: true, pprofPort: 7070}, ['--pprof', '--pprof-port=7070']],
            ['formatting', {enableFormatting: true}, ['--formatting']],
            ['file logging without a path', {enableFileLogging: true}, ['--file-logging']],
            ['file logging with a path', {enableFileLogging: true, logFilePath: '/logs/pikopls.log'}, ['--file-logging', '--log-file=/logs/pikopls.log']],
            ['log path ignored when logging is off', {logFilePath: '/logs/pikopls.log'}, []],
            ['gopls path', {goplsPath: ' /usr/bin/gopls '}, ['--gopls-path=/usr/bin/gopls']],
            ['blank gopls path', {goplsPath: '   '}, []],
            ['gopls disable', {goplsDisable: true}, ['--gopls-disable']],
            ['gopls limits', {goplsMaxChildren: 4, goplsMaxOverlays: 32}, ['--gopls-max-children=4', '--gopls-max-overlays=32']],
            ['zero gopls limits use server defaults', {goplsMaxChildren: 0, goplsMaxOverlays: 0}, []],
        ])('should map %s identically for both transports', (_name, overrides, expected) => {
            const config = {...baseConfig, ...overrides};

            expect(buildServerArgs({kind: 'stdio'}, config)).toEqual(expected);
            expect(buildServerArgs({kind: 'tcp', port: 9}, config).slice(3)).toEqual(expected);
        });
    });

    describe('nodePlatformToGoOs', () => {
        it.each([
            ['win32', 'windows'],
            ['darwin', 'darwin'],
            ['linux', 'linux'],
            ['freebsd', 'freebsd'],
        ])('should map %s to %s', (platform, expected) => {
            expect(nodePlatformToGoOs(platform)).toBe(expected);
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
        it('should return pikopls for Linux', () => {
            expect(getBinaryName('linux')).toBe('pikopls');
        });

        it('should return pikopls for macOS', () => {
            expect(getBinaryName('darwin')).toBe('pikopls');
        });

        it('should return pikopls.exe for Windows', () => {
            expect(getBinaryName('win32')).toBe('pikopls.exe');
        });

        it('should return pikopls for unknown platforms', () => {
            expect(getBinaryName('freebsd')).toBe('pikopls');
            expect(getBinaryName('unknown')).toBe('pikopls');
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

    describe('validateFileName length', () => {
        it('accepts names up to the limit and rejects longer ones', () => {
            expect(validateFileName('a'.repeat(MAX_FILE_NAME_LENGTH))).toBeNull();
            expect(validateFileName('a'.repeat(MAX_FILE_NAME_LENGTH + 1))).toContain('longer than');
        });
    });

    describe('parsePort', () => {
        it.each([1, 4389, MAX_TCP_PORT])('accepts %s', value => {
            expect(parsePort(value)).toBe(value);
        });

        it.each([0, -1, MAX_TCP_PORT + 1, 80.5, Number.NaN, '4389', undefined, null])('rejects %s', value => {
            expect(parsePort(value)).toBeUndefined();
        });
    });

    describe('parseLimit', () => {
        it.each([0, 1, 8])('accepts %s', value => {
            expect(parseLimit(value, 8)).toBe(value);
        });

        it.each([-1, 9, 1.5, Number.POSITIVE_INFINITY, '2', undefined])('rejects %s', value => {
            expect(parseLimit(value, 8)).toBeUndefined();
        });
    });
});

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
 * Templates for the "Piko: New Piko File" command.
 *
 * Kept as string constants (rather than bundled files) so they carry no
 * licence header into the user's project.
 */

/** Content for a new Piko page (uses piko.NoProps and an i18n block). */
export const DEFAULT_PAGE_TEMPLATE = `<template>
  <div>
    <h1 p-text="state.Title"></h1>
  </div>
</template>

<script type="application/x-go">
package main

import "piko.sh/piko"

type Response struct {
  Title string
}

func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
  return Response{
    Title: "Page Title",
  }, piko.Metadata{}, nil
}
</script>

<style>
</style>

<i18n lang="json">
{
  "en": {
  }
}
</i18n>
`;

/** Content for a new Piko partial (has a Props struct). */
export const DEFAULT_PARTIAL_TEMPLATE = `<template>
  <div>

  </div>
</template>

<script type="application/x-go">
package main

import "piko.sh/piko"

type Props struct {
}

type Response struct {
}

func Render(r *piko.RequestData, props Props) (Response, piko.Metadata, error) {
  return Response{}, piko.Metadata{}, nil
}
</script>

<style>
</style>
`;

/**
 * Builds the content for a new Piko client component (.pkc).
 *
 * @param elementName - The custom element name (the file name).
 * @returns The component content.
 */
export function buildComponentTemplate(elementName: string): string {
    return `<template name="${elementName}">
  <p p-text="state.message"></p>
</template>

<script lang="ts">
const state = {
  message: "Hello from ${elementName}" as string,
};
</script>

<style>
</style>
`;
}

/**
 * Describes one entry in the new-file quick pick.
 */
export interface PikoFileTemplate {
    /** The quick pick label. */
    label: string;
    /** The quick pick description. */
    description: string;
    /** The file extension, including the dot. */
    extension: '.pk' | '.pkc';
    /** Builds the file content from the file name (without extension). */
    render: (name: string) => string;
}

/** The templates offered by the new-file command. */
export const PIKO_FILE_TEMPLATES: readonly PikoFileTemplate[] = [
    {label: 'Page', description: 'Piko page (.pk) with NoProps and i18n', extension: '.pk', render: () => DEFAULT_PAGE_TEMPLATE},
    {label: 'Partial', description: 'Piko partial (.pk) with a Props struct', extension: '.pk', render: () => DEFAULT_PARTIAL_TEMPLATE},
    {label: 'Component', description: 'Piko client component (.pkc) with TypeScript', extension: '.pkc', render: buildComponentTemplate},
];

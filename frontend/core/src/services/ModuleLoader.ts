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

/** Handles loading ES modules during SPA navigation. */
export interface ModuleLoader {
    /**
     * Loads module scripts from a parsed document by appending script elements to the DOM.
     * @param doc - The parsed document to extract module scripts from.
     */
    loadFromDocument(doc: Document): void;

    /**
     * Loads module scripts from a parsed document and awaits their execution via dynamic import.
     * @param doc - The parsed document to extract module scripts from.
     */
    loadFromDocumentAsync(doc: Document): Promise<void>;

    /**
     * Checks whether a module has already been loaded.
     * @param src - The module URL to check.
     * @returns True if the module has been loaded.
     */
    hasLoaded(src: string): boolean;

    /**
     * Gets the set of loaded module URLs.
     * @returns The set of URLs for all loaded modules.
     */
    getLoadedModules(): Set<string>;
}

/**
 * Creates a ModuleLoader for loading ES modules during SPA navigation.
 * Tracks loaded modules to prevent duplicate loading.
 * @returns A new ModuleLoader instance.
 */
export function createModuleLoader(): ModuleLoader {
    const loadedModuleScripts = new Set<string>();

    /**
     * Loads a single script by appending a module script element to the document body.
     * @param src - The module URL to load.
     */
    function loadScript(src: string): void {
        if (loadedModuleScripts.has(src)) {
            return;
        }
        loadedModuleScripts.add(src);
        const newScript = document.createElement('script');
        newScript.type = 'module';
        newScript.src = src;
        document.body.appendChild(newScript);
    }

    return {
        loadFromDocument(doc: Document) {
            const moduleScripts = doc.querySelectorAll('script[type="module"]');
            moduleScripts.forEach(scriptEl => {
                const src = scriptEl.getAttribute('src');
                if (src) {
                    loadScript(src);
                }
            });
        },

        async loadFromDocumentAsync(doc: Document) {
            const loadPromises: Promise<unknown>[] = [];
            let failCount = 0;

            const moduleScripts = doc.querySelectorAll('script[type="module"]');
            moduleScripts.forEach(scriptEl => {
                const src = scriptEl.getAttribute('src');
                if (src && !loadedModuleScripts.has(src)) {
                    loadedModuleScripts.add(src);
                    loadPromises.push(
                        import(/* webpackIgnore: true */ src).catch(err => {
                            failCount++;
                            console.error(`ModuleLoader: Failed to load module ${src}:`, err);
                        })
                    );
                }
            });

            await Promise.all(loadPromises);

            if (failCount > 0) {
                console.error(`ModuleLoader: ${failCount}/${loadPromises.length} module(s) failed to load`);
            }
        },

        hasLoaded(src: string) {
            return loadedModuleScripts.has(src);
        },

        getLoadedModules() {
            return loadedModuleScripts;
        }
    };
}

/**
 * Initialises a module loader with existing scripts already present on the page.
 * @param loader - The module loader to initialise.
 */
export function initModuleLoaderFromPage(loader: ModuleLoader): void {
    const initialScripts = document.querySelectorAll('script[type="module"]');
    const loadedModules = loader.getLoadedModules();

    initialScripts.forEach(scriptEl => {
        const src = scriptEl.getAttribute('src');
        if (src) {
            loadedModules.add(src);
        }
    });
}

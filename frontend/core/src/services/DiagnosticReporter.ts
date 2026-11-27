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

import {_currentPageVisit} from '@/pk/lifecycle';
import type {DiagnosticKind, DiagnosticPayload} from './HookManager';

/** A lookup that found nothing, as reported by the code that made it. */
export interface DiagnosticReport {
    /** What kind of lookup found nothing. */
    kind: DiagnosticKind;
    /** The name, selector or target that was looked up. */
    name: string;
    /** The API that made the lookup, such as reloadPartial or pk.refs. */
    source: string;
    /** The partial id the lookup ran in. Leave it out to use the page path. */
    context?: string;
}

/** Sends a diagnostic payload to the hook system. */
export type DiagnosticEmitter = (payload: DiagnosticPayload) => void;

/**
 * The most distinct diagnostics sent in one page visit. This stops a loop
 * that looks up many different missing names from flooding listeners.
 */
const MAX_DIAGNOSTICS_PER_PAGE_VISIT = 100;

/** Separates the kind from the name in a deduplication key. */
const KEY_SEPARATOR = '\u0000';

/** The registered emitter, or null until PPFramework registers one. */
let emitDiagnostic: DiagnosticEmitter | null = null;

/**
 * Diagnostics raised before an emitter is registered, in the order they were
 * raised. Its size is bounded by the per-visit limit, because only new reports
 * are held.
 */
const heldPayloads: DiagnosticPayload[] = [];

/** The page visit that the reported keys belong to. */
let reportedVisit: object | null = null;

/** The kind and name pairs already sent in the current page visit. */
const reportedKeys = new Set<string>();

/**
 * Registers the function that sends diagnostics to the hook system, then sends
 * any diagnostics held until now, in order. PPFramework calls this during
 * init(), after the queued hook listeners are registered.
 *
 * @param emitter - The function that sends a diagnostic payload.
 */
export function registerDiagnosticEmitter(emitter: DiagnosticEmitter): void {
    emitDiagnostic = emitter;
    const held = heldPayloads.splice(0);
    for (const payload of held) {
        emitter(payload);
    }
}

/**
 * Records a report for the current page visit if it has not been sent yet.
 *
 * @param report - The diagnostic report.
 * @returns True if the report is new and should be sent.
 */
function claimReport(report: DiagnosticReport): boolean {
    const visit = _currentPageVisit();
    if (visit !== reportedVisit) {
        reportedVisit = visit;
        reportedKeys.clear();
    }

    const key = `${report.kind}${KEY_SEPARATOR}${report.name}`;
    if (reportedKeys.has(key) || reportedKeys.size >= MAX_DIAGNOSTICS_PER_PAGE_VISIT) {
        return false;
    }
    reportedKeys.add(key);
    return true;
}

/**
 * Sends a diagnostic event for a lookup that found nothing. Each kind and name
 * is sent at most once per page visit. Diagnostics raised before an emitter is
 * registered are held and sent when it is. This never throws, and it still runs in
 * production builds, where console messages are removed.
 *
 * @param report - The diagnostic report.
 */
export function reportDiagnostic(report: DiagnosticReport): void {
    if (!claimReport(report)) {
        return;
    }
    const payload: DiagnosticPayload = {
        kind: report.kind,
        name: report.name,
        context: report.context ?? window.location.pathname,
        source: report.source,
        timestamp: Date.now()
    };
    if (!emitDiagnostic) {
        heldPayloads.push(payload);
        return;
    }
    emitDiagnostic(payload);
}

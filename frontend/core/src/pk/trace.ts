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

/** Configuration options for the tracer. */
export interface TraceConfig {
    /** Traces partial reloads (default: true). */
    partialReloads: boolean;
    /** Traces event emissions and listeners (default: true). */
    events: boolean;
    /** Traces handler executions (default: true). */
    handlers: boolean;
    /** Traces SSE connections (default: true). */
    sse: boolean;
}

/** A single trace log entry. */
export interface TraceEntry {
    /** Entry type. */
    type: 'partial' | 'event' | 'handler' | 'sse';
    /** Entry name/identifier. */
    name: string;
    /** Duration in milliseconds (if applicable). */
    duration?: number;
    /** Timestamp when the entry was created. */
    timestamp: number;
    /** Additional metadata. */
    metadata?: Record<string, unknown>;
}

/** Aggregated metrics for a traced item. */
export interface TraceMetrics {
    /** Number of times this item was traced. */
    count: number;
    /** Average duration in milliseconds. */
    avgDuration: number;
    /** Maximum duration in milliseconds. */
    maxDuration: number;
    /** Minimum duration in milliseconds. */
    minDuration: number;
}

/** Maximum number of trace entries to keep in memory. */
const MAX_TRACE_ENTRIES = 1000;

/** Opt-in tracer for debugging partial and island interactions. */
class PKTracer {
    /** Whether tracing is currently enabled. */
    private enabled = false;
    /** Current trace configuration. */
    private config: TraceConfig = {
        partialReloads: true,
        events: true,
        handlers: true,
        sse: true
    };
    /** Collected trace entries. */
    private entries: TraceEntry[] = [];
    /** Maximum number of entries to retain. */
    private maxEntries = MAX_TRACE_ENTRIES;

    /**
     * Enables tracing with optional configuration.
     *
     * @param config - Partial configuration to merge with defaults.
     */
    enable(config?: Partial<TraceConfig>): void {
        this.enabled = true;
        if (config) {
            this.config = {...this.config, ...config};
        }
        console.log('%c[Piko] Tracing enabled', 'color: #29e; font-weight: bold', this.config);
    }

    /** Disables tracing. */
    disable(): void {
        this.enabled = false;
        console.log('%c[Piko] Tracing disabled', 'color: #999');
    }

    /**
     * Checks if tracing is enabled.
     *
     * @returns True if tracing is currently enabled.
     */
    isEnabled(): boolean {
        return this.enabled;
    }

    /** Clears all trace entries. */
    clear(): void {
        this.entries = [];
    }

    /**
     * Returns all trace entries.
     *
     * @returns A shallow copy of the entries array.
     */
    getEntries(): TraceEntry[] {
        return [...this.entries];
    }

    /**
     * Returns aggregated metrics grouped by name.
     *
     * @returns Record mapping names to their aggregated metrics.
     */
    getMetrics(): Record<string, TraceMetrics> {
        const metrics = new Map<string, {count: number; totalDuration: number; maxDuration: number; minDuration: number}>();

        for (const entry of this.entries) {
            let m = metrics.get(entry.name);
            if (!m) {
                m = {count: 0, totalDuration: 0, maxDuration: 0, minDuration: Infinity};
                metrics.set(entry.name, m);
            }
            m.count++;
            if (entry.duration !== undefined) {
                m.totalDuration += entry.duration;
                m.maxDuration = Math.max(m.maxDuration, entry.duration);
                m.minDuration = Math.min(m.minDuration, entry.duration);
            }
        }

        return Object.fromEntries(
            Array.from(metrics.entries()).map(([name, m]) => [
                name,
                {
                    count: m.count,
                    avgDuration: m.count > 0 ? m.totalDuration / m.count : 0,
                    maxDuration: m.maxDuration,
                    minDuration: m.minDuration === Infinity ? 0 : m.minDuration
                }
            ])
        );
    }

    /**
     * Adds a trace entry, trimming old entries if the buffer exceeds capacity.
     *
     * @param entry - Trace entry to add.
     */
    private addEntry(entry: TraceEntry): void {
        this.entries.push(entry);

        if (this.entries.length > this.maxEntries) {
            this.entries = this.entries.slice(-this.maxEntries);
        }
    }

    /**
     * Traces a partial reload.
     *
     * @param name - Partial name.
     * @param duration - Duration in milliseconds.
     * @param args - Optional metadata.
     */
    tracePartialReload(name: string, duration: number, args?: Record<string, unknown>): void {
        if (!this.enabled || !this.config.partialReloads) {
            return;
        }

        this.addEntry({
            type: 'partial',
            name,
            duration,
            timestamp: Date.now(),
            metadata: args
        });

        console.log(
            `%c[Piko] Partial Reload: "${name}" (${duration.toFixed(1)}ms)`,
            'color: #29e; font-weight: bold',
            args ? `\n  Args: ${JSON.stringify(args)}` : ''
        );
    }

    /**
     * Traces an event emission.
     *
     * @param eventName - Event name.
     * @param source - Source of the event.
     * @param payload - Optional event payload.
     */
    traceEvent(eventName: string, source: string, payload?: unknown): void {
        if (!this.enabled || !this.config.events) {
            return;
        }

        this.addEntry({
            type: 'event',
            name: eventName,
            timestamp: Date.now(),
            metadata: {source, payload}
        });

        console.log(
            `%c[Piko] Event: "${eventName}"`,
            'color: #4a4; font-weight: bold',
            `\n  Source: ${source}`,
            payload !== undefined ? `\n  Payload: ${JSON.stringify(payload)}` : ''
        );
    }

    /**
     * Traces a handler execution.
     *
     * @param handlerName - Handler name.
     * @param duration - Duration in milliseconds.
     * @param result - Optional result value.
     */
    traceHandler(handlerName: string, duration: number, result?: unknown): void {
        if (!this.enabled || !this.config.handlers) {
            return;
        }

        this.addEntry({
            type: 'handler',
            name: handlerName,
            duration,
            timestamp: Date.now(),
            metadata: result !== undefined ? {result} : undefined
        });

        console.log(
            `%c[Piko] Handler: "${handlerName}" (${duration.toFixed(1)}ms)`,
            'color: #a4a; font-weight: bold'
        );
    }

    /**
     * Traces an SSE connection event.
     *
     * @param url - SSE endpoint URL.
     * @param event - Connection event type.
     * @param data - Optional event data.
     */
    traceSSE(url: string, event: 'connect' | 'disconnect' | 'message' | 'error', data?: unknown): void {
        if (!this.enabled || !this.config.sse) {
            return;
        }

        this.addEntry({
            type: 'sse',
            name: url,
            timestamp: Date.now(),
            metadata: {event, data}
        });

        const colours: Record<string, string> = {
            connect: 'color: #4a4',
            disconnect: 'color: #a44',
            message: 'color: #29e',
            error: 'color: #f44'
        };

        console.log(
            `%c[Piko] SSE ${event}: "${url}"`,
            `${colours[event]}; font-weight: bold`,
            data !== undefined ? `\n  Data: ${JSON.stringify(data)}` : ''
        );
    }
}

/** Global tracer instance. */
export const trace = new PKTracer();

/**
 * Logs a trace entry via the global tracer.
 *
 * Only logs if tracing is enabled.
 *
 * @param name - Entry name.
 * @param data - Optional data payload.
 */
export function traceLog(name: string, data?: unknown): void {
    if (!trace.isEnabled()) {
        return;
    }
    trace.traceEvent(name, 'manual', data);
}

/**
 * Creates a traced version of a function.
 *
 * Measures execution time and logs via the tracer.
 *
 * @param name - Name for the trace entry.
 * @param operation - Async function to trace.
 * @returns Wrapped function that traces execution.
 */
export function traceAsync<T>(
    name: string,
    operation: () => Promise<T>
): () => Promise<T> {
    return async () => {
        const start = performance.now();
        try {
            return await operation();
        } finally {
            const duration = performance.now() - start;
            trace.traceHandler(name, duration);
        }
    };
}

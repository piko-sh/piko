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

import { describe, it, expect, vi, beforeEach } from 'vitest';
import { createEventBus } from '@/services/EventBus';
import type { EventBus } from '@/services/EventBus';

describe('EventBus', () => {
  let bus: EventBus;

  beforeEach(() => {
    bus = createEventBus();
  });

  describe('on()', () => {
    it('should subscribe to events', () => {
      const callback = vi.fn();
      bus.on('test-event', callback);
      bus.emit('test-event', { data: 'hello' });
      expect(callback).toHaveBeenCalledWith({ data: 'hello' });
    });

    it('should return an unsubscribe function', () => {
      const callback = vi.fn();
      const unsubscribe = bus.on('test-event', callback);

      bus.emit('test-event', 'first');
      expect(callback).toHaveBeenCalledTimes(1);

      unsubscribe();
      bus.emit('test-event', 'second');
      expect(callback).toHaveBeenCalledTimes(1);
    });

    it('should allow multiple subscribers for the same event', () => {
      const callback1 = vi.fn();
      const callback2 = vi.fn();

      bus.on('test-event', callback1);
      bus.on('test-event', callback2);

      bus.emit('test-event', 'data');

      expect(callback1).toHaveBeenCalledWith('data');
      expect(callback2).toHaveBeenCalledWith('data');
    });
  });

  describe('off()', () => {
    it('should unsubscribe from events', () => {
      const callback = vi.fn();
      bus.on('test-event', callback);

      bus.emit('test-event', 'first');
      expect(callback).toHaveBeenCalledTimes(1);

      bus.off('test-event', callback);
      bus.emit('test-event', 'second');
      expect(callback).toHaveBeenCalledTimes(1);
    });

    it('should handle unsubscribing non-existent callback gracefully', () => {
      const callback = vi.fn();
      expect(() => bus.off('non-existent', callback)).not.toThrow();
    });
  });

  describe('emit()', () => {
    it('should emit events with data', () => {
      const callback = vi.fn();
      bus.on('test-event', callback);

      bus.emit('test-event', { id: 123, name: 'test' });

      expect(callback).toHaveBeenCalledWith({ id: 123, name: 'test' });
    });

    it('should emit events without data', () => {
      const callback = vi.fn();
      bus.on('test-event', callback);

      bus.emit('test-event');

      expect(callback).toHaveBeenCalledWith(undefined);
    });

    it('should not throw when emitting to non-existent event', () => {
      expect(() => bus.emit('non-existent')).not.toThrow();
    });

    it('should catch errors in callbacks and continue', () => {
      const errorCallback = vi.fn(() => {
        throw new Error('Test error');
      });
      const normalCallback = vi.fn();
      const consoleErrorSpy = vi.spyOn(console, 'error').mockImplementation(() => {});

      bus.on('test-event', errorCallback);
      bus.on('test-event', normalCallback);

      bus.emit('test-event', 'data');

      expect(errorCallback).toHaveBeenCalled();
      expect(normalCallback).toHaveBeenCalled();
      expect(consoleErrorSpy).toHaveBeenCalled();

      consoleErrorSpy.mockRestore();
    });
  });

  describe('clear()', () => {
    it('should clear all listeners for a specific event', () => {
      const callback1 = vi.fn();
      const callback2 = vi.fn();

      bus.on('event1', callback1);
      bus.on('event2', callback2);

      bus.clear('event1');

      bus.emit('event1');
      bus.emit('event2');

      expect(callback1).not.toHaveBeenCalled();
      expect(callback2).toHaveBeenCalled();
    });

    it('should clear all listeners when no event specified', () => {
      const callback1 = vi.fn();
      const callback2 = vi.fn();

      bus.on('event1', callback1);
      bus.on('event2', callback2);

      bus.clear();

      bus.emit('event1');
      bus.emit('event2');

      expect(callback1).not.toHaveBeenCalled();
      expect(callback2).not.toHaveBeenCalled();
    });
  });
});

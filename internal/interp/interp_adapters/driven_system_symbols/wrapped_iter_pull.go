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

package driven_system_symbols

import (
	"reflect"
)

// drainValues receives and discards every remaining element from ch until it is closed.
// The producer goroutines spawned by wrappedIterPull and wrappedIterPull2 only exit once
// their send either succeeds or the done channel is observed; draining guarantees a
// blocked send unblocks so the goroutine can return and close ch.
//
// Takes ch (<-chan T) the channel to drain.
func drainValues[T any](ch <-chan T) {
	for {
		if _, ok := <-ch; !ok {
			return
		}
	}
}

// wrappedIterPull wraps iter.Pull[V any] for interpreted code.
//
// iter.Pull converts a push-style iter.Seq[V] into a pull-style pair of (next, stop)
// functions. The returned next yields V values untyped (as any) because the interpreter
// dispatches on the runtime type of the yielded value, not the static type of V. The
// strategy: spawn a goroutine that consumes the input Seq and pushes values onto an
// unbuffered channel; next reads from the channel, stop closes a done signal so the
// producer can exit.
//
// Takes sequence (any) which is an iter.Seq[V] expressed as a func(yield func(V) bool);
// reflect.Value.Seq supplies the iteration.
//
// Returns next (any) which is func() (any, bool) reading the next yielded value.
// Returns stop (any) which is func() that signals the producer goroutine to exit.
//
// Concurrency: the producer goroutine runs until the Seq completes or stop is invoked;
// callers must invoke stop to release the goroutine when iteration ends early.
func wrappedIterPull(sequence any) (next any, stop any) {
	sequenceVal := reflect.ValueOf(sequence)
	values := make(chan reflect.Value)
	done := make(chan struct{})
	stopped := false
	stopFunction := func() {
		if stopped {
			return
		}
		stopped = true
		close(done)
		drainValues(values)
	}
	go func() {
		defer close(values)
		for v := range sequenceVal.Seq() {
			select {
			case values <- v:
			case <-done:
				return
			}
		}
	}()
	nextFunction := func() (any, bool) {
		v, ok := <-values
		if !ok {
			return nil, false
		}
		return v.Interface(), true
	}
	return nextFunction, stopFunction
}

// wrappedIterPull2 wraps iter.Pull2[K, V any]. Like wrappedIterPull but for Seq2
// sequences yielding two values per iteration.
//
// Takes sequence (any) which is an iter.Seq2[K, V] expressed as a func(yield func(K, V)
// bool).
//
// Returns next (any) which is func() (any, any, bool) reading the next yielded pair.
// Returns stop (any) which is func() that signals the producer goroutine to exit.
//
// Concurrency: the producer goroutine runs until the Seq2 completes or stop is invoked;
// callers must invoke stop to release the goroutine when iteration ends early.
func wrappedIterPull2(sequence any) (next any, stop any) {
	sequenceVal := reflect.ValueOf(sequence)
	type pair struct {
		k reflect.Value
		v reflect.Value
	}
	values := make(chan pair)
	done := make(chan struct{})
	stopped := false
	stopFunction := func() {
		if stopped {
			return
		}
		stopped = true
		close(done)
		drainValues(values)
	}
	go func() {
		defer close(values)
		for k, v := range sequenceVal.Seq2() {
			select {
			case values <- pair{k: k, v: v}:
			case <-done:
				return
			}
		}
	}()
	nextFunction := func() (any, any, bool) {
		p, ok := <-values
		if !ok {
			return nil, nil, false
		}
		return p.k.Interface(), p.v.Interface(), true
	}
	return nextFunction, stopFunction
}

func init() {
	if _, ok := Symbols["iter"]; ok {
		Symbols["iter"]["Pull"] = reflect.ValueOf(wrappedIterPull)
		Symbols["iter"]["Pull2"] = reflect.ValueOf(wrappedIterPull2)
	}
}

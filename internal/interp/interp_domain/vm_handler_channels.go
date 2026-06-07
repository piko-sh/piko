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

package interp_domain

import (
	"fmt"
	"reflect"
)

// blockWake reports how a ctx-aware blocking channel or select operation completed: with
// a value, via context cancellation, or because a sibling goroutine panicked.
type blockWake uint8

const (
	// blockWakeValue indicates the channel or select operation completed normally and
	// produced a value.
	blockWakeValue blockWake = iota

	// blockWakeCancelled indicates the VM's context was cancelled while the operation was
	// parked.
	blockWakeCancelled

	// blockWakePanic indicates a sibling goroutine panicked while the operation was parked.
	blockWakePanic
)

var (
	// reflectTypeChanElemInt is the canonical reflect.Type for an int channel element; used
	// by trySendTypedChannel pointer-equality dispatch.
	//
	//nolint:gochecknoglobals // immutable type cache for switch-case dispatch
	reflectTypeChanElemInt = reflect.TypeFor[int]()

	// reflectTypeChanElemInt64 is the canonical reflect.Type for an int64 channel element.
	//
	reflectTypeChanElemInt64 = reflect.TypeFor[int64]()

	// reflectTypeChanElemString is the canonical reflect.Type for a string channel element.
	//
	reflectTypeChanElemString = reflect.TypeFor[string]()

	// reflectTypeChanElemBool is the canonical reflect.Type for a bool channel element.
	//
	reflectTypeChanElemBool = reflect.TypeFor[bool]()

	// reflectTypeChanElemFloat64 is the canonical reflect.Type for a float64 channel
	// element.
	//
	reflectTypeChanElemFloat64 = reflect.TypeFor[float64]()
)

// surfaceBlockingWake converts a non-value wake reason into the terminal opResult that
// aborts dispatch with the right error.
//
// Takes wake (blockWake) which is the wake reason to surface.
//
// Returns opContinue for blockWakeValue, otherwise opPanicError after recording the
// cause.
func (vm *VM) surfaceBlockingWake(wake blockWake) opResult {
	switch wake {
	case blockWakeCancelled:
		return vm.surfaceContextCancellation()
	case blockWakePanic:
		return vm.surfaceGoroutinePanicAbort()
	default:
		return opContinue
	}
}

// surfaceContextCancellation aborts dispatch because the VM's context was cancelled while
// a blocking channel or select operation was parked. It records the cancelled flag so
// subsequent periodic checks agree and stores the context cause as the evaluation error.
//
// Returns opPanicError so the dispatch loop unwinds.
func (vm *VM) surfaceContextCancellation() opResult {
	vm.cancelled.Store(1)
	if err := vm.ctx.Err(); err != nil {
		vm.evalError = err
	} else {
		vm.evalError = errBlockingOpUnblocked
	}
	return opPanicError
}

// surfaceGoroutinePanicAbort aborts dispatch because a sibling goroutine panicked while
// this VM was parked on a blocking channel or select operation, surfacing the recorded
// panic so it propagates instead of deadlocking.
//
// Returns opPanicError so the dispatch loop unwinds.
func (vm *VM) surfaceGoroutinePanicAbort() opResult {
	if info := vm.globals.goroutinePanic.Load(); info != nil {
		vm.evalError = fmt.Errorf("goroutine panicked: %v", info.value)
	} else {
		vm.evalError = errBlockingOpUnblocked
	}
	return opPanicError
}

// handleChannelSend sends a value on a channel by reading the value from the appropriate
// register and converting it to the channel's element type.
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes registers (*Registers) which holds the channel and value.
// Takes instruction (instruction) which encodes the channel and value regs.
//
// Returns opResult indicating the next execution step.
func handleChannelSend(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	channel := registers.general[instruction.a]
	if !channel.IsValid() {
		vmPanicInvalidRegister("handleChannelSend", "channel", instruction.a, instruction, frame, registers)
	}

	done := vm.ctx.Done()
	panicWake := vm.globals.goroutinePanicWakeChan()
	wake := blockWakeValue
	released := vm.releaseAroundBlock()
	recovered, panicked := guardChannelOp(func() {
		if done == nil && panicWake == nil && trySendTypedChannel(vm, channel, registers, instruction.b, registerKind(instruction.c)) {
			return
		}
		ext := makeInstruction(0, instruction.b, instruction.c, 0)
		value := materialiseArenaValueUnconditional(vm.arena, buildSelectSendValue(vm, registers, ext, channel.Type().Elem()))
		if done == nil && panicWake == nil {
			channel.Send(value)
			return
		}
		wake = selectChannelSend(channel, value, done, panicWake)
	})
	vm.reacquireAfterBlock(released)
	if panicked {
		return raiseNativePanicAsInterpreted(vm, recovered)
	}
	if wake != blockWakeValue {
		return vm.surfaceBlockingWake(wake)
	}
	return opContinue
}

// trySendTypedChannel sends a typed-bank value via the cached channel handle.
//
// Uses the Go-typed channel handle cached on vm.typedHandleCache, avoiding the
// reflect.Value boxing + reflect.Value.Send overhead that the slow path pays. Returns
// true on a successful typed send; false when the channel's element type isn't one of the
// supported primitive types or when the source register kind doesn't align with the
// channel's element type.
//
// Takes vm (*VM) which owns the typed handle cache.
// Takes channel (reflect.Value) which is the channel value.
// Takes sourceRegister (uint8) which is the source register index in the typed bank
// identified by sourceKind.
// Takes sourceKind (registerKind) which selects the source bank.
//
// Returns true when the typed send fired, false when the slow reflect path must run.
func trySendTypedChannel(vm *VM, channel reflect.Value, registers *Registers, sourceRegister uint8, sourceKind registerKind) bool {
	switch channel.Type().Elem() {
	case reflectTypeChanElemInt:
		return sendTypedChannel[chan int](vm, channel, sourceKind, registerInt, func(typed chan int) {
			typed <- int(registers.ints[sourceRegister])
		})
	case reflectTypeChanElemInt64:
		return sendTypedChannel[chan int64](vm, channel, sourceKind, registerInt, func(typed chan int64) {
			typed <- registers.ints[sourceRegister]
		})
	case reflectTypeChanElemString:
		return sendTypedChannel[chan string](vm, channel, sourceKind, registerString, func(typed chan string) {
			typed <- registers.strings[sourceRegister]
		})
	case reflectTypeChanElemBool:
		return sendTypedChannel[chan bool](vm, channel, sourceKind, registerBool, func(typed chan bool) {
			typed <- registers.bools[sourceRegister]
		})
	case reflectTypeChanElemFloat64:
		return sendTypedChannel[chan float64](vm, channel, sourceKind, registerFloat, func(typed chan float64) {
			typed <- registers.floats[sourceRegister]
		})
	default:
		return false
	}
}

// sendTypedChannel performs a kind-checked typed-channel send via the per-VM cache.
// Returns false when the source register kind doesn't match the channel's element kind,
// or when the cache lookup fails because the underlying channel isn't actually of type T.
//
// Takes vm (*VM) which owns the typed handle cache.
// Takes channel (reflect.Value) which is the channel value.
// Takes sourceKind (registerKind) which is the source register's reported kind at the
// call site.
// Takes expectedKind (registerKind) which is the kind required for the channel's element
// type.
// Takes send (func(T)) which is the callback that performs the actual typed send once the
// cache hit is confirmed.
//
// Returns true when the typed send completed, false otherwise.
func sendTypedChannel[T any](vm *VM, channel reflect.Value, sourceKind, expectedKind registerKind, send func(T)) bool {
	if sourceKind != expectedKind {
		return false
	}
	typed, ok := lookupTypedChannel[T](vm, channel)
	if !ok {
		return false
	}
	send(typed)
	return true
}

// lookupTypedChannel returns the cached typed-handle for the channel.
//
// On cache hit the stored handle is returned; on first encounter the handle is extracted
// from the reflect.Value and cached. The bool reports whether the underlying channel
// actually has the expected element type. The bool is false when channel.Type() reports a
// named type such as `type Named chan int` rather than the bare `chan int` the type
// assertion targets.
//
// Takes vm (*VM) which owns the cache.
// Takes channel (reflect.Value) which is the channel value.
//
// Returns the typed handle.
// Returns a bool indicating whether the type assertion succeeded.
func lookupTypedChannel[T any](vm *VM, channel reflect.Value) (T, bool) {
	var zero T
	pointer := channel.Pointer()
	if pointer == 0 {
		return zero, false
	}
	if cached, ok := vm.typedHandleCache[pointer]; ok {
		typed, ok := cached.(T)
		return typed, ok
	}
	extracted := channel.Interface()
	typed, ok := extracted.(T)
	if !ok {
		return zero, false
	}
	if vm.typedHandleCache == nil {
		vm.typedHandleCache = make(map[uintptr]any, initialTypedHandleCacheCapacity)
	}
	if len(vm.typedHandleCache) >= maxTypedHandleCacheEntries {
		return typed, true
	}
	vm.typedHandleCache[pointer] = extracted
	return typed, true
}

// handleChannelReceive receives a value from a channel and stores it in the destination
// register along with a boolean indicating success.
//
// Takes frame (*callFrame) which provides the destination extension word.
// Takes registers (*Registers) which holds the channel and destination.
// Takes instruction (instruction) which encodes the channel register.
//
// Returns opResult indicating the next execution step.
func handleChannelReceive(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	extensionWord := frame.function.body[frame.programCounter]
	frame.programCounter++
	channel := registers.general[instruction.a]
	if !channel.IsValid() {
		vmPanicInvalidRegister("handleChannelReceive", "channel", instruction.a, instruction, frame, registers)
	}

	done := vm.ctx.Done()
	panicWake := vm.globals.goroutinePanicWakeChan()
	if done == nil && panicWake == nil {
		if tryRecvTypedChannel(vm, channel, registers, instruction.b, extensionWord.a, registerKind(extensionWord.b)) {
			return opContinue
		}
		result, ok := channel.Recv()
		writeChannelReceiveResult(registers, instruction.b, extensionWord, result, ok)
		return opContinue
	}

	released := vm.releaseAroundBlock()
	result, ok, wake := selectChannelReceive(channel, done, panicWake)
	vm.reacquireAfterBlock(released)
	if wake != blockWakeValue {
		return vm.surfaceBlockingWake(wake)
	}
	writeChannelReceiveResult(registers, instruction.b, extensionWord, result, ok)
	return opContinue
}

// writeChannelReceiveResult stores a received value and its comma-ok flag into the
// destination registers. Interface results are unwrapped to their dynamic value first so
// the destination bank receives the concrete type.
//
// Takes registers (*Registers) which receives the value and ok flag.
// Takes okRegister (uint8) which is the int-bank register for the comma-ok bool.
// Takes extensionWord (instruction) whose a/b fields carry the destination register index
// and kind.
// Takes result (reflect.Value) which is the received value.
// Takes ok (bool) which is true when the channel produced a value (false when closed).
func writeChannelReceiveResult(registers *Registers, okRegister uint8, extensionWord instruction, result reflect.Value, ok bool) {
	registers.ints[okRegister] = boolToInt64(ok)
	if result.IsValid() && result.Kind() == reflect.Interface {
		result = result.Elem()
	}
	writeRegisterValue(registers, extensionWord.a, registerKind(extensionWord.b), result)
}

// selectChannelReceive performs a receive with wake cancellation.
//
// Also wakes on context cancellation or a sibling goroutine panic. done and panicWake are
// each nil when their wake source is inactive; at least one is non-nil (the caller takes
// a plain receive otherwise). The fixed-size case array avoids a per-receive heap
// allocation.
//
// Takes channel (reflect.Value) which is the channel to receive from.
// Takes done (<-chan struct{}) which is the context Done channel, or nil.
// Takes panicWake (<-chan struct{}) which is the goroutine-panic signal, or nil.
//
// Returns the received value, its comma-ok flag, and the wake reason.
func selectChannelReceive(channel reflect.Value, done, panicWake <-chan struct{}) (reflect.Value, bool, blockWake) {
	var storage [3]reflect.SelectCase
	storage[0] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: channel}
	count, cancelIndex, panicIndex := appendWakeCases(storage[:], 1, done, panicWake)
	chosen, received, recvOK := reflect.Select(storage[:count])
	if chosen == cancelIndex {
		return reflect.Value{}, false, blockWakeCancelled
	}
	if chosen == panicIndex {
		return reflect.Value{}, false, blockWakePanic
	}
	return received, recvOK, blockWakeValue
}

// selectChannelSend performs a send with wake cancellation.
//
// Also wakes on context cancellation or a sibling goroutine panic. Semantics mirror
// selectChannelReceive. A send on a closed channel still panics; the caller guards the
// call with guardChannelOp.
//
// Takes channel (reflect.Value) which is the channel to send on.
// Takes value (reflect.Value) which is the value to send.
// Takes done (<-chan struct{}) which is the context Done channel, or nil.
// Takes panicWake (<-chan struct{}) which is the goroutine-panic signal, or nil.
//
// Returns the wake reason.
func selectChannelSend(channel, value reflect.Value, done, panicWake <-chan struct{}) blockWake {
	var storage [3]reflect.SelectCase
	storage[0] = reflect.SelectCase{Dir: reflect.SelectSend, Chan: channel, Send: value}
	count, cancelIndex, panicIndex := appendWakeCases(storage[:], 1, done, panicWake)
	chosen, _, _ := reflect.Select(storage[:count])
	if chosen == cancelIndex {
		return blockWakeCancelled
	}
	if chosen == panicIndex {
		return blockWakePanic
	}
	return blockWakeValue
}

// appendWakeCases fills the context-cancellation and goroutine-panic receive arms into
// cases starting at index start, skipping each arm whose channel is nil.
//
// Takes cases ([]reflect.SelectCase) which is the backing array to populate.
// Takes start (int) which is the first free index (after the user's own cases).
// Takes done (<-chan struct{}) which is the context Done channel, or nil.
// Takes panicWake (<-chan struct{}) which is the goroutine-panic signal, or nil.
//
// Returns the total case count, the cancel-arm index (-1 when absent), and the panic-arm
// index (-1 when absent).
func appendWakeCases(cases []reflect.SelectCase, start int, done, panicWake <-chan struct{}) (count, cancelIndex, panicIndex int) {
	count, cancelIndex, panicIndex = start, -1, -1
	if done != nil {
		cancelIndex = count
		cases[count] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(done)}
		count++
	}
	if panicWake != nil {
		panicIndex = count
		cases[count] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(panicWake)}
		count++
	}
	return count, cancelIndex, panicIndex
}

// tryRecvTypedChannel attempts to receive a value from a Go-typed channel handle cached
// on vm.typedHandleCache, dispatching directly into the destination register's typed bank
// without paying for reflect.Value.Recv's per-call result allocation.
//
// Takes vm (*VM) which owns the typed handle cache.
// Takes channel (reflect.Value) which is the channel value.
// Takes okRegister (uint8) which receives the comma-ok bool as int64.
// Takes destinationRegister (uint8) which is the result register index in the typed bank
// identified by destinationKind.
// Takes destinationKind (registerKind) which selects the destination bank.
//
// Returns true when the typed receive fired, false when the slow reflect path must run.
func tryRecvTypedChannel(vm *VM, channel reflect.Value, registers *Registers, okRegister, destinationRegister uint8, destinationKind registerKind) bool {
	switch channel.Type().Elem() {
	case reflectTypeChanElemInt:
		return recvTypedChannel[chan int](vm, channel, destinationKind, registerInt, func(typed chan int) bool {
			value, channelOpen := <-typed
			registers.ints[destinationRegister] = int64(value)
			registers.ints[okRegister] = boolToInt64(channelOpen)
			return true
		})
	case reflectTypeChanElemInt64:
		return recvTypedChannel[chan int64](vm, channel, destinationKind, registerInt, func(typed chan int64) bool {
			value, channelOpen := <-typed
			registers.ints[destinationRegister] = value
			registers.ints[okRegister] = boolToInt64(channelOpen)
			return true
		})
	case reflectTypeChanElemString:
		return recvTypedChannel[chan string](vm, channel, destinationKind, registerString, func(typed chan string) bool {
			value, channelOpen := <-typed
			registers.strings[destinationRegister] = value
			registers.ints[okRegister] = boolToInt64(channelOpen)
			return true
		})
	case reflectTypeChanElemBool:
		return recvTypedChannel[chan bool](vm, channel, destinationKind, registerBool, func(typed chan bool) bool {
			value, channelOpen := <-typed
			registers.bools[destinationRegister] = value
			registers.ints[okRegister] = boolToInt64(channelOpen)
			return true
		})
	case reflectTypeChanElemFloat64:
		return recvTypedChannel[chan float64](vm, channel, destinationKind, registerFloat, func(typed chan float64) bool {
			value, channelOpen := <-typed
			registers.floats[destinationRegister] = value
			registers.ints[okRegister] = boolToInt64(channelOpen)
			return true
		})
	default:
		return false
	}
}

// recvTypedChannel performs a kind-checked typed-channel receive via the per-VM cache,
// returning false when the destination register kind doesn't match the channel's element
// kind or when the cached typed-handle assertion fails.
//
// Takes vm (*VM) which owns the typed handle cache.
// Takes channel (reflect.Value) which is the channel value.
// Takes destinationKind (registerKind) which is the register kind the caller expects to
// write into.
// Takes expectedKind (registerKind) which is the kind required for the channel's element
// type.
// Takes recv (func(T) bool) which performs the receive and returns true when complete.
//
// Returns true when the typed receive completed, false otherwise.
func recvTypedChannel[T any](vm *VM, channel reflect.Value, destinationKind, expectedKind registerKind, recv func(T) bool) bool {
	if destinationKind != expectedKind {
		return false
	}
	typed, ok := lookupTypedChannel[T](vm, channel)
	if !ok {
		return false
	}
	return recv(typed)
}

// handleChannelClose closes the channel stored in the general register bank at the index
// specified by the instruction.
//
// Takes registers (*Registers) which holds the channel to close.
// Takes instruction (instruction) which encodes the channel register index.
//
// Returns opResult indicating the next execution step.
func handleChannelClose(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	channel := registers.general[instruction.a]
	if !channel.IsValid() {
		vmPanicInvalidRegister("handleChannelClose", "channel", instruction.a, instruction, frame, registers)
	}
	if recovered, panicked := guardChannelOp(func() { channel.Close() }); panicked {
		return raiseNativePanicAsInterpreted(vm, recovered)
	}
	return opContinue
}

// guardChannelOp runs op and converts any panic into a returned recovery value.
//
// Used to catch runtime panics from channel send/receive/close operations (e.g. "send on
// closed channel") so handlers can re-raise them as interpreted panics rather than
// terminate the host process.
//
// Takes op (func()) which is the channel operation to guard.
//
// Returns recovered (any) which is the panic value (nil when op completed normally).
// Returns panicked (bool) which is true when op panicked, false otherwise.
func guardChannelOp(op func()) (recovered any, panicked bool) {
	defer func() {
		if r := recover(); r != nil {
			recovered = r
			panicked = true
		}
	}()
	op()
	return nil, false
}

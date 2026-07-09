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

// Package telemetry_grpcfb is piko's reusable telemetry-over-gRPC streaming base: a
// client-streaming transport that ships a piko site's native observability (analytics,
// watchdog events, profiles, logs, traces, metrics, errors) to a remote sink.
//
// piko owns the wire schema (telemetry.fbs -> telemetryfb) because piko emits these
// shapes. The package provides:
//
//   - a FlatBuffers gRPC Codec (codec.go) and a hardened, allocation-free structural
//     verifier (verify.go): the Go FlatBuffers runtime ships none, and every frame a sink
//     reads crosses a trust boundary;
//   - the plain-Go wire structs and their (de)serialisation (fb.go);
//   - a hand-written client-streaming gRPC ServiceDesc plus RegisterServer / NewServer
//     helpers for the sink side (service.go);
//   - a non-blocking batching Client with periodic flush, a bounded drop-on-full send
//     queue, automatic stream reconnect, and a circuit breaker (client.go).
//
// It is a separate Go module (like the db engines and monitoring_api) so the gRPC
// dependency stays out of piko's core module.
package telemetry_grpcfb

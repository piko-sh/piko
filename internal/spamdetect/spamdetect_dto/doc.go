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

// Package spamdetect_dto defines data transfer objects for the spam detection module.
//
// It contains the schema builder for declaring form field signals ([NewSchema],
// [TextField], [Honeypot], [Timing], [FieldGroup]), the submission type that carries form
// data for analysis ([Submission]), analysis result types with per-detector and per-field
// breakdowns ([AnalysisResult], [DetectorResult], [FieldResult], [SubmissionRecord]),
// signal and priority constants, configuration types ([ServiceConfig]), and sentinel
// errors used across architectural boundaries.
//
// Schemas built via [NewSchema] are immutable after construction and safe for concurrent
// use. Configuration types such as [ServiceConfig] are intended to be set once at
// initialisation and read thereafter. Sentinel errors are exported so callers can branch
// on specific failure modes using [errors.Is].
package spamdetect_dto

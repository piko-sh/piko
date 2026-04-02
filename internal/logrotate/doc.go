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

// Package logrotate manages size-based log file rotation,
// implementing [io.WriteCloser].
//
// It automatically rotates the log file when it exceeds a configured
// size. Rotated files can optionally be compressed with gzip and
// pruned by count or age.
//
// # Usage
//
//	w, err := logrotate.New(ctx, logrotate.Config{
//	    Directory:  "/var/log",
//	    Filename:   "app.log",
//	    MaxSize:    10,   // megabytes
//	    MaxBackups: 3,
//	    MaxAge:     7,    // days
//	    Compress:   true,
//	})
//	if err != nil {
//	    return err
//	}
//	defer w.Close()
//
//	handler := slog.NewJSONHandler(w, nil)
//
// # Thread safety
//
// A single [Writer] is safe for concurrent use by multiple goroutines.
// All write and rotation operations are serialised by an internal mutex.
// Background compression and cleanup run in a dedicated goroutine.
package logrotate

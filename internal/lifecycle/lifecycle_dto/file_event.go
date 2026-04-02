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

package lifecycle_dto

// FileEventType represents a type of file system event.
type FileEventType int

const (
	// FileEventTypeUnknown is an unknown file system event type.
	FileEventTypeUnknown FileEventType = iota

	// FileEventTypeCreate indicates a new file has been created.
	FileEventTypeCreate

	// FileEventTypeWrite indicates that a file was modified.
	FileEventTypeWrite

	// FileEventTypeRemove indicates a file was deleted from the filesystem.
	FileEventTypeRemove

	// FileEventTypeRename indicates a file was renamed or moved.
	FileEventTypeRename
)

// FileEvent represents a file system change notification.
type FileEvent struct {
	// Path is the absolute file system path of the changed file.
	Path string

	// Type indicates the kind of file system event (create, write, remove, or rename).
	Type FileEventType
}

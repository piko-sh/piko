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

// Package storage_provider_disk provides a filesystem-based storage
// provider for the Piko storage system.
//
// This provider stores objects on the local filesystem using atomic
// writes to prevent data corruption. It is suitable for development,
// testing, and single-server deployments where distributed storage
// is not required. All file operations are sandboxed for security.
//
// # Usage
//
//	config := storage_provider_disk.Config{
//	    BaseDirectory: "/var/data/storage",
//	}
//	provider, err := storage_provider_disk.NewDiskProvider(config)
//	if err != nil {
//	    return err
//	}
//
//	service := storage.NewService("disk")
//	service.RegisterProvider(ctx, "disk", provider)
//
// # Limitations
//
// The disk provider does not support presigned URLs or multipart
// uploads. The storage service layer provides fallback mechanisms
// for presigned URL generation when using this provider.
package storage_provider_disk

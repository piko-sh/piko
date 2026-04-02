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

package persistence

// DatabaseType identifies the database backend in use.
type DatabaseType string

const (
	// DatabaseTypeOtter indicates an in-memory otter cache backend.
	DatabaseTypeOtter DatabaseType = "otter"
)

// RegistryDALFactory creates Registry data access layers from the otter
// backend.
type RegistryDALFactory interface {
	// NewRegistryDAL creates a new Registry data access layer instance.
	//
	// Returns any which should be type-asserted to
	// registry_dal.RegistryDALWithTx. The any type avoids import cycles between
	// persistence and registry packages.
	//
	// Returns error when the DAL cannot be created.
	NewRegistryDAL() (any, error)
}

// OrchestratorDALFactory creates Orchestrator data access layers from the
// otter backend.
type OrchestratorDALFactory interface {
	// NewOrchestratorDAL creates a new Orchestrator data access layer instance.
	//
	// Returns any which should be type-asserted to
	// orchestrator_domain.TaskStore. The any type avoids import cycles between
	// persistence and orchestrator packages.
	//
	// Returns error when the DAL cannot be created.
	NewOrchestratorDAL() (any, error)
}

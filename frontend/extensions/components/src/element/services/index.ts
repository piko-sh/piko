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

export {createPropTypeRegistry, propertyToAttributeName} from "./PropTypeRegistry";
export type {PropTypeRegistry, PropTypeRegistryOptions} from "./PropTypeRegistry";

export {createStateManager} from "./StateManager";
export type {StateManager, StateManagerOptions} from "./StateManager";

export {createFormAssociation} from "./FormAssociation";
export type {FormAssociation, FormAssociationOptions} from "./FormAssociation";

export {createBehaviourApplicator} from "./BehaviourApplicator";
export type {
    BehaviourApplicator,
    BehaviourApplicatorOptions,
    PPBehaviour,
} from "./BehaviourApplicator";

export {createShadowDOMService, RESET_CSS} from "./ShadowDOMService";
export type {ShadowDOMService, ShadowDOMServiceOptions} from "./ShadowDOMService";

export {createSlotManager} from "./SlotManager";
export type {SlotManager, SlotManagerOptions} from "./SlotManager";

export {createLifecycleManager} from "./LifecycleManager";
export type {LifecycleManager} from "./LifecycleManager";

export {createAttributeSyncService} from "./AttributeSyncService";
export type {AttributeSyncService, AttributeSyncServiceOptions} from "./AttributeSyncService";

export {createRenderScheduler} from "./RenderScheduler";
export type {RenderScheduler, RenderSchedulerOptions} from "./RenderScheduler";

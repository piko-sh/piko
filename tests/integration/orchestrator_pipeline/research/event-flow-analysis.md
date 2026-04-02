# Event Flow Analysis

## Primary Event Flow

```
lifecycle (build runner)
  -> registry.UpsertArtefact()
    -> publishEvent(EventArtefactCreated) -- artefactEventsPublished++
    -> event bus delivers to bridge
      -> bridge.handleEventWithAck()
        -> bridge.handleEvent()
          -> bridge.processArtefactEvent()
            -> bridge.dispatchProfileTask() (for each profile)
              -> dispatcher.Dispatch()
          -> bridge.finaliseEventHandling() -- artefactEventsProcessed++
```

## Secondary Event Flow (On Task Success)

When a task SUCCEEDS, the executor calls `registry.AddVariant()`:

```
executor completes successfully
  -> storeAndCreateVariant()
    -> registry.AddVariant()
      -> publishEvent(EventArtefactUpdated) -- artefactEventsPublished++
      -> event bus delivers to bridge
        -> bridge.handleEventWithAck()
          -> bridge.handleEvent() (with EventArtefactUpdated)
            -> bridge.finaliseEventHandling() -- artefactEventsProcessed++
```

**CRITICAL**: This secondary event increments `artefactEventsPublished` AGAIN.
The bridge must process it for Phase 1 (flush) to complete.

## Counter Balance

For a single artefact with ONE profile that SUCCEEDS:
- Published: 2 (UpsertArtefact + AddVariant)
- Processed: 2 (initial event + secondary event)
- Phase 1 passes: 2 >= 2

For a single artefact with ONE profile that FAILS:
- Published: 1 (UpsertArtefact only, no AddVariant)
- Processed: 1 (initial event only)
- Phase 1 passes: 1 >= 1

## Profile Chain Events

For a `.pkc` file with CHAINED profiles (compile -> minify -> compress):
- Each successful task triggers AddVariant -> EventArtefactUpdated
- Each EventArtefactUpdated is processed by the bridge
- The bridge may dispatch NEW tasks for downstream profiles
- Published count grows with each successful variant registration

## Key Files

- Event publishing: `internal/registry/registry_domain/artefact_lifecycle.go` (publishEvent)
- Bridge event handling: `internal/orchestrator/orchestrator_adapters/driving_artefact_workflow_bridge.go`
- Counter: `finaliseEventHandling()` at line ~757
- AddVariant: `internal/registry/registry_domain/artefact_lifecycle.go:1038-1085`

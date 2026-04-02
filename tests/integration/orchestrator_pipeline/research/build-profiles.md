# Build Profiles (GetProfilesForFile)

## Location

`internal/lifecycle/lifecycle_domain/build_profiles.go`

## How It Works

`GetProfilesForFile(artefactID, ignoreExt)` routes by file extension:
- `.pkc` -> component profile chain
- `.ico`, `.png`, `.webmanifest` -> copy-only profiles
- `.svg` -> SVG optimisation chain
- `.css` -> CSS minification chain
- `.js` -> JS minification chain (priority varies by prefix)

## Profile Chain for .pkc Files

Need to read the exact `extensionProfiles[".pkc"]` builder function to see the
full chain. Expected chain based on analysis:

1. **compiled** (compile-component): depends on "source", priority=NEED
2. **minified** (minify): depends on "compiled"
3. **compressed-gz** (compress-gzip): depends on "minified"
4. **compressed-br** (compress-brotli): depends on "minified"

Each profile in the chain:
- Has a `DependsOn` referencing the upstream profile/variant
- Has `Params` (e.g., sourcePath)
- Has `ResultingTags` (e.g., storageBackendId, fileExtension, mimeType)

## Why This Matters

In our integration tests, we only assign ONE profile per artefact.
The real system assigns 4+ profiles in a dependency chain.
When the first task completes and calls AddVariant:
- EventArtefactUpdated is published
- The bridge processes this event
- If the bridge re-evaluates the artefact's profiles, it may dispatch
  the NEXT profile in the chain (whose dependency is now satisfied)
- This cascading dispatch is never tested

## TODO

- Read the exact `.pkc` builder function
- Read `processArtefactEvent` for EventArtefactUpdated handling
- Create integration test with chained profiles

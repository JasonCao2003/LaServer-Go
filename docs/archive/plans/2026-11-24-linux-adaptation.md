# Linux adaptation implementation plan

**Goal:** Run the existing API and room service on Linux without changing their network contracts or installing dependencies automatically.

**Architecture:** Keep the Windows DLL adapter. Add a Linux cgo adapter that loads the same ABI v1 bridge through libdl; share the real server entry point. Build SkyENet separately from the exact sky-enet 0.1.5 source. No new Go module dependencies.

**Constraints:** Independent service modules; legacy repository read only; no real database migration, commit or push. Generated build state and logs stay in ignored project directories. Missing native sources must be reported rather than replaced with stock ENet.

## Tasks

- [x] Add failing Linux transport tests for validation, dynamic loader errors, ABI compatibility, packet ownership and lifecycle using a compiled C ABI fixture. This fixture tests the adapter boundary, not ENet compatibility.
- [x] Implement Linux cgo loading, symbol resolution, cleanup, polling, send and disconnect; use C-owned loader state and never retain Go pointers in C.
- [x] Share the Windows/Linux entry point; test help and invalid arguments, and handle Linux SIGTERM.
- [x] Make existing optional SkyENet integration tests available on Linux without breaking Windows environment variables.
- [x] Add root verification and native build scripts; update README and handoff with exact commands and evidence boundaries.
- [x] Run room tests, race detector, vet, Linux build and Windows cross-build. Run API checks only with already available module dependencies (GOPROXY=off). Review the diff independently; record skipped external integration explicitly.

## Review focus

Wrong/missing ABI symbols must return errors with all handles released; no calls after Close; received data must survive later polls; negative/excessive poll timeout must be rejected; no Linux dependency may break Windows builds. CGO_ENABLED=0 on Linux is unsupported and documented. Real UDP/client verification requires the separately built SkyENet library, and is not implied by fixture tests.

## Final evidence

Room tests, race, vet, Linux build and Windows amd64 cross-build passed. Independent read-only review passed after making verification scripts strictly offline. API verification stopped at missing cached modernc.org/sqlite; no dependency was downloaded. Real SkyENet library build and network compatibility remain unverified because source/library dependencies are absent. See [Linux development](../../linux-development.md) for commands and log paths.

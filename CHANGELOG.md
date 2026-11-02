# Changelog

## 1.0.1

- Pinned Go 1.26.10 (previously 1.26.9) and updated source-map-js to 1.2.2; added weekly Go/npm security scans.
- Detect truncated streams without fabricating successful completion, preserve length-limited Responses semantics, and bound streaming JSON/error-body reads.
- Require an exclusive gateway lock for offline restore/key rotation; bound same-origin redirect chains and synchronize admin configuration snapshots.
- Record actual working-tree source hashes and the embedded Vue runtime in packages; align local packaging with all three release platforms.
- Add regression coverage for stream completion, timeouts, maintenance locking, configuration concurrency, and release provenance.

## 1.0.0

- Unified release metadata and versioned artifact names.
- Required the complete CI quality gate before release publication.
- Added explicit portable and installed runtime directory modes.
- Added draining readiness and configurable graceful shutdown.
- Added browser-level administration tests and keyboard-accessible dialogs.
- Limited the documented release platforms to Windows and Linux.

## 0.9.0

- Improved readiness, model routing, administration workflows, backup recovery, and release metadata.

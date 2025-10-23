nixOS-immich-webui/docs/claude/architecture.md
# Architecture & Package Overview

This document describes the high-level architecture of the `nixOS-immich-webui` project and summarizes the responsibilities of each internal package. It is intended as a companion to the trimmed top-level `CLAUDE.md` (high-level overview) and to the more focused documents in `docs/claude/` (backups, HTMX, security, etc.).

Goals
- Provide a clear mental model of the system for contributors and reviewers.
- Show where to make changes for common tasks (config, handlers, backup logic).
- Explain build modes and the location of environment-specific paths.

High-level system view
- Single Go binary serving a progressive-enhancement web UI.
- Core responsibilities:
  - Manage NixOS configuration (JSON-based config → `.nix` modules).
  - Control and monitor Immich containers.
  - Provide USB backup tooling for config and photo library.
  - Expose a small admin UI (works without JavaScript; HTMX for enhancement).
  - Execute privileged system operations (reboot, poweroff, apply NixOS).
- Intended deployment: NixOS host with a ZFS pool named `tank`. The binary binds to `localhost:8000` and is typically fronted by `Caddy`.

Project layout (short)
The full tree is available at the repository root; a representative subset:

```nixOS-immich-webui/CLAUDE.md#L1-60
nixOS-immich-webui/
├── main.go
├── internal/
│   ├── config/
│   ├── handlers/
│   ├── services/
│   ├── system/
│   └── templates/
├── example/etc/nixos/
├── docs/
└── test/
```

Principles and patterns
- Progressive enhancement: server-side-rendered HTML is functional without JS; HTMX adds partial updates and polling.
- Small surface area for privileged actions: only a few endpoints perform system operations; these are isolated in the `system` package.
- Thread-safety: shared runtime state (configs, backup state) is protected with package-level mutexes and RWMutex where appropriate.
- Atomic writes: config saves use `.tmp` + `os.Rename` to switch and `.old` backups for rollback.
- Minimal runtime dependencies: standard library HTTP server, `html/template`, `embed` for templates, `slog` for structured logging.

Package summaries
Below are the main `internal/` packages and their responsibilities.

- `internal/config`
  - Purpose: Centralized configuration management and validation.
  - Key responsibilities:
    - Load/save `nixconfig.json` and `immich-config.json`.
    - Provide thread-safe getters/setters for active configuration.
    - Validate inputs (timezone, time format, email, Tailscale auth keys).
    - Expose helper functions for ML model validation and display names.
    - Path separation for dev vs. prod via `paths_dev.go` and `paths_prod.go`.
    - Atomic switching and rollback helpers (`SwitchConfigJSON`, `RollbackConfigJSON`).
  - Where to change: updates to accepted config fields, validation rules, or file layout.

- `internal/handlers`
  - Purpose: HTTP request handling and templating glue.
  - Key responsibilities:
    - Route handlers for system, Immich, backup features (split into `SystemHandler`, `ImmichHandler`, `BackupHandler`).
    - Parse form data, call `config`/`services`/`system` APIs, and render templates or HTMX fragments.
    - Maintain progressive-enhancement compatibility (handle both full-page and partial responses).
  - Where to change: new endpoints, routing, form parsing, or UI fragment rendering.

- `internal/services`
  - Purpose: Business logic and higher-level workflows.
  - Key responsibilities:
    - Implement backup orchestration (async execution, state tracking, history management).
    - Provide reusable operations for complex tasks that coordinate `system` calls and file operations.
    - Keep HTTP handlers thin by centralizing logic here.
  - Where to change: backup algorithm, history retention policy, or new complex workflows.

- `internal/system`
  - Purpose: Low-level system interactions requiring elevated privileges.
  - Key responsibilities:
    - Execute `nixos-rebuild`/apply changes and provide rollback behavior.
    - Control Docker/Immich lifecycle (start/stop/update).
    - Power management (`PowerOff`, `Reboot`) and disk operations (mount/unmount detection).
    - Discover eligible USB disks for backups.
  - Safety notes: This package runs system commands and must be audited carefully before change.

- `internal/templates`
  - Purpose: HTML templates embedded into the binary.
  - Key responsibilities:
    - Organize templates into full pages and HTMX fragments.
    - Provide small reusable partials (status panels, forms, progress fragments).
  - Where to change: UI text, fragments used for HTMX partial updates.

Build modes and paths
- Two build modes are supported via Go build tags:
  - Development (`-tags dev`): uses `test/` directories and avoids touching system paths.
  - Production (default): uses `/etc/nixos/`, `/tank/immich-config/`, and `./backup-history.json` (next to the binary).
- Path constants live in `internal/config/paths_dev.go` and `internal/config/paths_prod.go`. Update these in one place to change deployment layout.

State & concurrency
- Config state: guarded by mutexes in the `config` package (`nixConfigMu`, `immichConfigMu`).
- Backup state: stored in a `BackupState` structure with an RWMutex and exposed through service accessors (`GetState`, `setState`, `resetState`).
- File-based history: history is persisted as a JSON file and pruned automatically; operations that write history perform file-IO and are serialized to avoid corruption.

Integration points and extension points
- Templates: add fragments to `internal/templates/web/` and include in handlers.
- New config properties: extend `internal/config/types.go`, add validation in `validation.go`, update templates and handlers to expose the fields.
- New services or workflows: implement in `internal/services/` and keep handlers small.
- External integration: Immich API, Tailscale controls, and Caddy configuration are all entry points for future features.

Operational considerations
- Security: current model is local-only (binds to `localhost:8000`) and relies on a reverse proxy for external access. See `docs/claude/security.md` for the planned enhancements (basic auth, OIDC/Tailscale).
- Backups: detailed backup flow, rsync parsing, and history management are documented separately in `docs/claude/backups.md`.
- Troubleshooting: logging is via `slog`; enable debug logs with the `--debug` flag or runtime config. See `docs/claude/troubleshooting.md`.

Where to look next
- Top-level trimmed overview: `CLAUDE.md`.
- Backup deep-dive: `docs/claude/backups.md`.
- HTMX and frontend guidelines: `docs/claude/htmx.md`.
- Security notes and deployment guidance: `docs/claude/security.md`.
- Example configs: `example/etc/nixos/`.
- Source code: `internal/` (config, handlers, services, system, templates).

Contributing and making changes
- Follow the progressive enhancement and thread-safety patterns above.
- Prefer adding logic to `services/` rather than `handlers/`.
- Keep `system/` changes minimal and audit commands that affect the host.
- Add unit tests where feasible; manual testing is required for system-level features.

This document is a living summary. For API-level details, function signatures, and the complete backup/state models, consult the files under `internal/` and the dedicated documents in `docs/claude/`.
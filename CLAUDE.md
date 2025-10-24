# CLAUDE.md — NixOS Immich WebUI

Purpose
-------
`nixOS-immich-webui` is a small Go-based appliance UI that helps you manage a NixOS host running Immich (self-hosted photo/video backup). The binary serves a progressive-enhancement web UI, backup tooling, and a small set of system-management utilities. This file is a concise, high-level entrypoint — detailed technical docs live under `docs/claude/`.

Current status: **v0.1.0-alpha.3 complete**

Quick start (developer)
-----------------------
- Run in safe development mode (uses `test/` fixtures):
```nixOS-immich-webui/CLAUDE.md#L1-6
go run -tags dev .
```

- Build production binary:
```nixOS-immich-webui/CLAUDE.md#L1-6
go build -o nixos-immich-webui .
```

- Default server: binds to `localhost:8000`. In production you should front it with a reverse proxy (recommended: `Caddy`).

High-level features
-------------------
- JSON-based NixOS configuration management
- Immich container lifecycle controls (start/stop/update)
- USB backup workflow (configs + photo library) with progress & history
- Progressive-enhancement UI (works without JavaScript; HTMX for UX)
- Two-page UI: home/dashboard (backup & system controls) and configuration
- OAuth authentication support (Cloudflare integration)
- Cloudflare Tunnel and Tailscale integration for remote access

Where to find things
--------------------
- High-level architecture and package responsibilities:
  - `docs/claude/architecture.md`
- Backup system deep-dive:
  - `docs/claude/backups.md`
- HTMX usage & frontend guidelines:
  - `docs/claude/htmx.md`
- Security guidance and deployment notes:
  - `docs/claude/security.md`
- Troubleshooting and debug tips:
  - `docs/claude/troubleshooting.md`
- Example NixOS configs:
  - `example/etc/nixos/`
- Source code (main packages):
  - `internal/config/` — config handling & validation
  - `internal/handlers/` — HTTP handlers
  - `internal/services/` — backup & business logic
  - `internal/system/` — system commands and privileged operations
  - `internal/templates/web/` — HTML templates & HTMX fragments

Minimal project tree (representative)
-------------------------------------
```nixOS-immich-webui/CLAUDE.md#L1-32
nixOS-immich-webui/
├── main.go
├── go.mod
├── internal/
│   ├── config/
│   ├── handlers/
│   ├── services/
│   ├── system/
│   └── templates/web/
├── example/etc/nixos/
├── docs/claude/
└── test/
```

Security & environment (short)
------------------------------
- Default binding is `localhost:8000`. Use a reverse proxy (Caddy) for TLS and external access.
- The service currently performs privileged operations and runs as root in production — treat this as an operational risk. See `docs/claude/security.md` for recommended hardening (reverse-proxy auth, OIDC, Tailscale gating, privilege separation).
- Runtime assumptions: NixOS host, ZFS pool named `tank`, Immich docker-compose in `/tank/immich-config/` in production (dev mode uses `test/` paths).

Guidance for contributors
-------------------------
- Keep `main.go` small — add logic to `internal/services/` and keep handlers thin.
- Follow progressive-enhancement: always implement server-side fallback for UI interactions before adding HTMX attributes.
- Use `-tags dev` while developing to avoid touching live system paths.
- When changing backup, system, or config code, update the corresponding document under `docs/claude/` and add unit tests for parsing/validation where feasible.

If you want me to
------------------
- I can split additional large sections from the old `CLAUDE.md` into more focused docs, or prepare a PR that replaces the top-level file with this trimmed version and moves the deep-dive content into `docs/claude/`. Tell me which you prefer and I’ll proceed.

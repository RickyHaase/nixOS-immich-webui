# Ente Server Integration Plan

## Overview

Add Ente (self-hosted, E2E-encrypted photo platform) as a parallel service to Immich,
managed through the existing NixOS admin WebUI. Uses the native NixOS `services.ente`
module (available in nixpkgs) with Garage for S3 storage instead of MinIO.

## Architecture Summary

```
                  ┌─────────────────────────────────┐
                  │        Admin WebUI (Go)          │
                  │  :8000                           │
                  │  /        → Immich dashboard     │
                  │  /config  → Immich config        │
                  │  /ente    → Ente dashboard (NEW)  │
                  └────┬──────────────┬──────────────┘
                       │              │
              ┌────────▼──┐    ┌──────▼──────┐
              │  Immich    │    │  Ente Stack  │
              │  (Docker)  │    │  (NixOS svc) │
              │  :2283     │    │  :8080       │
              └────────────┘    │  + Postgres  │
                                │  + Garage    │
                                │    :3900     │
                                └─────────────┘
```

Key points:
- Ente Museum (API server) runs as a native NixOS systemd service via `services.ente`
- Garage runs as a native NixOS systemd service via `services.garage`
- PostgreSQL is managed by `services.ente.api.enableLocalDB = true`
- Ente web frontends served via nginx (bundled with the module)
- Storage lives on a ZFS dataset: `/tank/ente/`

---

## Phase 1: NixOS Module Configuration

### 1a. Garage S3 Storage (`example/etc/nixos/garage.nix`)

Create a NixOS module that configures Garage for local-only S3:

```nix
{ config, pkgs, ... }:
let
  vars = builtins.fromJSON (builtins.readFile ./nixconfig.json);
in
{
  services.garage = {
    enable = true;
    package = pkgs.garage;
    settings = {
      # Single-node, local-only setup
      replication_mode = "none";

      # Metadata on fast storage (SSD/ZFS)
      metadata_dir = "/tank/ente/garage/meta";

      # Data on main pool
      data_dir = "/tank/ente/garage/data";

      # Bind to localhost only — no external access needed
      rpc_bind_addr = "127.0.0.1:3901";
      rpc_public_addr = "127.0.0.1:3901";

      s3_api = {
        s3_region = "garage";
        api_bind_addr = "127.0.0.1:3900";
        root_domain = ".s3.garage.localhost";
      };

      # Admin API for bucket/key management
      admin = {
        api_bind_addr = "127.0.0.1:3903";
      };
    };
  };

  # ZFS dataset for Ente/Garage data
  # Created manually: zfs create tank/ente
  #                    zfs create tank/ente/garage
}
```

**Automated setup** (run once after first boot or via activation script):
1. `garage layout assign` — assign the local node
2. `garage layout apply`
3. `garage key create ente-museum-key`
4. `garage bucket create b2-eu-cen` (hardcoded bucket name Ente expects)
5. `garage bucket allow --read --write --owner b2-eu-cen --key ente-museum-key`

This will be handled by a oneshot systemd service or an activation script
in the NixOS module so the admin doesn't need to run manual commands.

### 1b. Ente Server (`example/etc/nixos/ente.nix`)

```nix
{ config, pkgs, ... }:
let
  vars = builtins.fromJSON (builtins.readFile ./nixconfig.json);
in
{
  services.ente = {
    api = {
      enable = true;
      enableLocalDB = true;  # Auto-provisions PostgreSQL

      settings = {
        # Garage S3 configuration
        s3 = {
          are_local_buckets = true;       # HTTP, not HTTPS
          use_path_style_urls = true;     # Required for Garage
          b2-eu-cen = {
            key = { _secret = "/tank/ente/secrets/garage-key"; };
            secret = { _secret = "/tank/ente/secrets/garage-secret"; };
            endpoint = "http://127.0.0.1:3900";
            region = "garage";
            bucket = "b2-eu-cen";
          };
        };

        key = {
          encryption = { _secret = "/tank/ente/secrets/encryption-key"; };
          hash = { _secret = "/tank/ente/secrets/hash-key"; };
        };

        jwt = {
          secret = { _secret = "/tank/ente/secrets/jwt-secret"; };
        };
      };
    };

    web = {
      enable = true;
      # Domain config depends on deployment
      # For local/Tailscale access, these can be localhost subpaths
      # or Caddy-proxied subdomains
    };
  };

  # Ensure garage starts before ente
  systemd.services.ente.after = [ "garage.service" ];
  systemd.services.ente.requires = [ "garage.service" ];
}
```

### 1c. Secrets Generation Script

Create a helper script (or oneshot service) that generates secrets on first run:

- `/tank/ente/secrets/encryption-key` — `openssl rand -hex 32`
- `/tank/ente/secrets/hash-key` — `openssl rand -hex 32`
- `/tank/ente/secrets/jwt-secret` — `openssl rand -hex 32`
- `/tank/ente/secrets/garage-key` — extracted from `garage key create` output
- `/tank/ente/secrets/garage-secret` — extracted from `garage key create` output

### 1d. Config JSON Extension

Add Ente section to `nixconfig.json`:

```json
{
  "system": { ... },
  "remoteAccess": { ... },
  "ente": {
    "enable": false
  }
}
```

Kept minimal for now — most Ente config lives in `museum.yaml` via the NixOS
module settings, not in our JSON. The `enable` flag controls whether the
NixOS module is activated.

### 1e. Update `configuration.nix` Import

Add `./garage.nix` and `./ente.nix` to the imports list (conditionally based
on `vars.ente.enable`).

---

## Phase 2: Go Backend — Config & System Layer

### 2a. Config Types (`internal/config/types.go`)

Add:

```go
type EnteConfig struct {
    Enable bool `json:"enable"`
}
```

Add `Ente EnteConfig` field to `ConfigVariables`.

### 2b. Config Parser (`internal/config/parser.go`)

- Extend `GetConfig()` / `SaveConfig()` to handle the new `ente` field
- No separate JSON file needed — Ente config is minimal and fits in `nixconfig.json`

### 2c. System Commands (`internal/system/commands.go`)

Add Ente service management functions:

```go
func GetEnteStatus() string
    // systemctl show -p ActiveState --value ente.service

func EnteService(command string) error
    // systemctl <start|stop|restart> ente.service

func GetGarageStatus() string
    // systemctl show -p ActiveState --value garage.service

func GarageService(command string) error
    // systemctl <start|stop|restart> garage.service
```

### 2d. Dev Mode Paths (`internal/config/paths_dev.go`)

Add dev-mode paths/stubs so `go run -tags dev .` works without Ente installed.

---

## Phase 3: Go Backend — Ente Handler

### 3a. Handler (`internal/handlers/ente.go`)

New `EnteHandler` struct following the existing Immich pattern:

```go
type EnteHandler struct {
    templates embed.FS
}

func NewEnteHandler(templates embed.FS) *EnteHandler

// Page
func (h *EnteHandler) HandleEntePage(w, r)     // GET /ente — full page

// Service controls
func (h *EnteHandler) HandleEnteStatus(w, r)   // GET /ente-status — HTMX fragment
func (h *EnteHandler) HandleEnteStart(w, r)    // POST /ente-start
func (h *EnteHandler) HandleEnteStop(w, r)     // POST /ente-stop
func (h *EnteHandler) HandleEnteRestart(w, r)  // POST /ente-restart

// Garage controls (minimal)
func (h *EnteHandler) HandleGarageStatus(w, r) // GET /garage-status — HTMX fragment
```

### 3b. Routes (`main.go`)

Register new routes:

```go
enteHandler := handlers.NewEnteHandler(templates.FS)

mux.HandleFunc("GET /ente",          enteHandler.HandleEntePage)
mux.HandleFunc("GET /ente-status",   enteHandler.HandleEnteStatus)
mux.HandleFunc("POST /ente-start",   enteHandler.HandleEnteStart)
mux.HandleFunc("POST /ente-stop",    enteHandler.HandleEnteStop)
mux.HandleFunc("POST /ente-restart", enteHandler.HandleEnteRestart)
mux.HandleFunc("GET /garage-status", enteHandler.HandleGarageStatus)
```

---

## Phase 4: Templates & UI

### 4a. Ente Dashboard Page (`internal/templates/web/ente.html`)

New full page at `/ente` with:

1. **Header/nav** — links back to `/` (Immich dashboard) and `/config`
2. **Service status panel** — Ente + Garage status with start/stop/restart buttons
   - HTMX polling on `/ente-status` (same pattern as Immich status)
   - Garage status shown as a sub-indicator
3. **Admin controls section** — placeholder panels for future features:
   - "User Management" — placeholder with "Coming soon" message
   - "Storage Usage" — placeholder for S3 bucket stats
   - "Registration Settings" — placeholder for enable/disable registration
4. **Setup section** — shows whether Ente is enabled in NixOS config,
   with toggle + apply button

### 4b. HTMX Fragments

- `ente_status.html` — service status + control buttons
- `garage_status.html` — Garage status indicator
- `ente_toggle.html` — enable/disable Ente in NixOS config

### 4c. Navigation Update

Add an "Ente" link/tab to the existing `index.html` and `config.html` nav bars
so users can navigate between Immich and Ente dashboards.

### 4d. Progressive Enhancement

All controls work as standard HTML forms (POST + redirect).
HTMX adds inline updates without full page reload.

---

## Phase 5: Documentation

### 5a. `docs/claude/ente.md`

New deep-dive doc covering:
- Ente integration architecture
- Garage S3 setup details
- NixOS module options used
- Secrets management
- Troubleshooting

### 5b. Update `CLAUDE.md`

Add Ente to the features list, project tree, and "where to find things" section.

### 5c. Update `docs/claude/architecture.md`

Add Ente handler/service to the architecture overview.

---

## Implementation Order

| Step | Description | Depends on |
|------|-------------|------------|
| 1 | NixOS modules (garage.nix, ente.nix, secrets script) | — |
| 2 | Config types + parser extension | — |
| 3 | System commands (service control) | — |
| 4 | Dev mode stubs | 2, 3 |
| 5 | Ente handler | 2, 3 |
| 6 | Templates (ente.html + fragments) | 5 |
| 7 | Route registration in main.go | 5 |
| 8 | Nav updates to existing templates | 6 |
| 9 | Documentation | 1-8 |

Steps 1, 2, and 3 can be done in parallel.
Steps 5-8 can be done as a batch once the foundation is ready.

---

## Open Questions / Future Work

- **Ente web frontends**: The NixOS module can serve Photos, Albums, etc. via nginx.
  Domain/subdomain strategy depends on whether we use Caddy (current) or let the
  module's nginx handle it. Recommend: let nginx handle Ente subdomains, Caddy
  proxies everything.
- **Backup integration**: Ente data lives on `/tank/ente/` (ZFS). Backup to USB
  could be extended to include Ente's Garage data + PostgreSQL dump. Deferred
  per user request.
- **User management API**: Museum has admin endpoints for user management.
  The placeholder panels will be filled in with actual API calls in a future phase.
- **CORS for Garage**: Ente web clients need CORS configured on the S3 buckets.
  This should be part of the automated Garage setup script.
- **Replication**: Single-bucket setup for now. Ente supports 3-bucket replication
  (hot/cold/glacier) if needed later.

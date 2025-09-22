# Design Decisions: Storage Locations for Immich Appliance

## Overview

This document outlines the rationale and decisions for file and backup storage locations in the Easy Immich Server appliance, specifically tailored for NixOS. The goal is to maximize reliability, ease of management, and disaster recovery, while respecting NixOS’s unique filesystem conventions.

---

## 1. Immich Config Files

**Files:**  
- `immich-config.json`
- `docker-compose.yml`
- `.env`

**Location:**  
- **Media Drive (`tank`)**: `/tank/immich-config/`

**Rationale:**  
- Keeps all Immich application state and deployment files together with user data (photos, DB).
- Survives OS upgrades, boot drive failures, and NixOS rebuilds.
- Simplifies backup and restore workflows—media drive contains everything needed for Immich.

---

## 2. Go Web UI Binary and Config

**Files:**  
- `nixos-immich-webui` (Go binary)
- `config.json` (Web UI config)

**Location:**  
- **Boot Drive:** `/root/nixos-immich-webui/` and `/root/nixos-immich-webui/config.json`

**Rationale:**  
- `/root/` is private to the admin user and not managed by NixOS or the package manager.
- Ensures the binary and config are isolated from user/media data.
- Simplifies upgrades and manual management during alpha/dev stages.

---

## 3. OS Config Backups

**Files:**  
- Backups of `/etc/nixos/configuration.nix` and related system config files.

**Location:**  
- **Media Drive:** `/tank/config-backup/`

**Rationale:**  
- Ensures system config backups persist across boot drive failures or OS reinstalls.
- Centralizes all critical backup data on the media drive for disaster recovery.

---

## 4. Media Backups (Photos, Videos, etc.)

**Files:**  
- Backups of Immich-managed media library.

**Location:**  
- **Boot Drive:** `/root/immich-media-backup/`

**Rationale:**  
- Keeps media backups isolated from the main media drive, useful for redundancy or migration.
- `/root/` is private, not managed by NixOS, and safe from system rebuilds.
- Consistent with the binary/config location for simplicity.

---

## 5. General Principle

- **Stash all appliance-managed files and backups in `/root/[sub-folder]` on the boot drive for privacy, simplicity, and safety from NixOS management.**
- **Store persistent application state and user data on the media drive for durability and disaster recovery.**

---

## Summary Table

| Item                      | Location                        | Rationale                                 |
|---------------------------|---------------------------------|-------------------------------------------|
| Immich config files       | `/tank/immich-config/`          | Persistent, survives OS changes           |
| Go binary & config        | `/root/nixos-immich-webui/`     | Private, isolated, easy to manage         |
| OS config backups         | `/tank/config-backup/`          | Durable, disaster recovery                |
| Media backups             | `/root/immich-media-backup/`    | Redundant, private, safe from NixOS mgmt  |

---

*This design ensures robust separation of concerns, maximizes reliability, and simplifies both backup and restore operations for the Immich appliance on NixOS.*

---

## 6. Configuration Management Architecture

**Approach:** JSON with `builtins.fromJSON`

**Files:**
- `nixconfig.json` (user-configurable settings)
- Modular `.nix` files (system configuration modules)

**Location:**
- **System Config Directory:** `/etc/nixos/`

**Rationale:**
The project evolved from Go templates with regex parsing to a JSON-based approach using NixOS's native `builtins.fromJSON` functionality.

### Why JSON over Go Templates?

| Aspect | Go Templates + Regex | JSON + builtins.fromJSON |
|--------|---------------------|-------------------------|
| **Parsing reliability** | Brittle regex patterns | Bulletproof JSON unmarshaling |
| **Generation complexity** | Template execution + embed.FS | Simple `json.Marshal()` |
| **Backup strategy** | Multiple template files | Single JSON file |
| **Rollback process** | Complex template restoration | Simple `.old` file copy |
| **NixOS integration** | External template system | Native NixOS built-ins |
| **Debugging** | Template syntax errors | Standard JSON validation |
| **Maintainability** | Template-struct sync required | Single source of truth |

### Key Benefits

1. **Structured Reliability**: JSON provides type safety and standard validation
2. **NixOS-Native**: Uses `builtins.fromJSON` and `builtins.readFile` - no external dependencies
3. **Consistent Pattern**: Every module uses the same 2-line JSON import pattern
4. **Simple Workflow**: Integrates with existing `switchConfig()` and `applyChanges()` functions
5. **Modular Organization**: Clear separation of concerns across configuration modules

### Implementation Pattern

Every NixOS module follows this consistent pattern:

```nix
{ config, pkgs, ... }:
let
  vars = builtins.fromJSON (builtins.readFile ./nixconfig.json);
in
{
  # Use vars.section.setting throughout
}
```

This approach eliminates complex template parsing while providing the structured configuration management needed for reliable system state parsing, generation, and rollback capabilities.
# Test Folder Structure

This document describes the organization and purpose of the `test` directory used for development and safe testing in the NixOS Immich WebUI project.

## Directory Layout

```
test/
├── immich-app/                # Test instance of Immich docker-compose setup
├── nixos/                     # NixOS configuration files for testing
│   ├── configuration.nix      # Main NixOS config (test version)
│   ├── configuration.old      # Previous config backup
│   └── configuration.tmp      # Temporary config during apply/save
└── tank/
    └── immich/                # Simulated ZFS dataset for Immich data/config
        ├── immich-config.json # Immich config JSON (test version)
        └── immich-config.tmp  # Temporary Immich config during operations
```

## Purpose

- **Safe Development:**  
  The `test` directory allows developers to experiment and validate changes without impacting the production system or real data.

- **Mirrored Structure:**  
  The layout closely mirrors the actual deployment structure (`/etc/nixos/`, `/root/immich-app/`, ZFS datasets) to ensure realistic testing and smooth transitions to production.

- **Config & Data Management:**  
  Temporary and backup files (`*.old`, `*.tmp`) are used to support atomic operations, rollback, and safe config switching during development.

## Component Descriptions

- **immich-app/**  
  Contains a test instance of the Immich docker-compose setup, used for container lifecycle management and integration testing.

- **nixos/**  
  Holds NixOS configuration files for testing.  
  - `configuration.nix`: The main NixOS config used in tests.
  - `configuration.old`: Backup of the previous config for rollback.
  - `configuration.tmp`: Temporary config file used during save/apply operations.

- **tank/immich/**  
  Simulates the ZFS dataset for Immich data and configuration.  
  - `immich-config.json`: Test version of the Immich config JSON.
  - `immich-config.tmp`: Temporary config file used during operations.

## Usage Notes

- All development and testing should use the `test/` directory to avoid unintended changes to the live system.
- The structure supports automated and manual testing, backup/restore workflows, and safe experimentation with configuration changes.

---

*For more details on development workflow and environment setup, see the other docs in this directory.*
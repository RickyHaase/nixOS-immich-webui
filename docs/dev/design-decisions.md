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
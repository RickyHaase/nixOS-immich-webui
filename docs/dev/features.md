# Feature Roadmap

This document outlines planned features and enhancements for the NixOS Immich WebUI project.
## NixOS Server Management
Full NixOS server configuration with web UI for changing any options that need user input (what the final configuration will be and what will be configurable through the web UI is still TBD).
  - [ ] Avahi (mDNS)
  - [ ] Sanoid (ZFS Snapshot management)
  - [ ] Caddy (Reverse Proxy)
  - [ ] Docker
  - [ ] Firewall Config
  - [ ] Updates (enable unattended)
  - [ ] And I'm sure a few more to be added

## Immich Container Management
  - [ ] Local access via mDNS hostname immich.local
  - [ ] Docker Management - Pull, Start, Stop, Update, and Roll-back containers
  - [ ] Roll back DB (using the in-built DB dump. maybe ZFS snapshots to attempt more incremental rollbacks - not ideal but could be serviceable and easier), Roll back library state (using ZFS snapshots)
  - [ ] Reset full Immich instance (with undo thanks to ZFS - need to think about this one and retention on rollback snapshots)
## Immich Admin Interface
Could make this optional if admins want to manage Immich from within Immich. Need to decide to manage the config via API or JSON. Most likely use API - at least to pull user management out of the Immich web UI and keep all admin tasks together.
  - [ ] Automate the creation of the initial admin user (not sure how), store the creds (not sure how/where), generate admin user API token and store for use with other admin tasks (again, not sure how yet).
  - [ ] Configure email settings
  - [ ] Configure library settings (probably no interface, just pick what's "best" and lock it in with the JSON file. Need to at least know where photos are going for backups (lib vs upload)).
  - [ ] User creation (with storage labels)
  - [ ] Auto kick-off jobs based on external actions (storage migration)
  - [ ] Sync up backup schedule with configured system backups (future feature)
## Backup and Restore System
Most critical and most complicated function to build - UX will be key.
  - [ ] Email Alerts (likely failure alerts and weekly summaries)
    - Need alert for detected corruption from ZFS. Will need a KB doc for how to fix this... this will need to be pushed back to after core functionality is built out. Also need to consider DB issues/corruption and repair/recovery procedures
    - Also admin password reset for Caddy basic auth to webUI
  - [ ] Internal restore points (NixOS configs, Immich Config, Immich DB, ZFS snapshots)
  - [ ] External USB Backups (Photo Libraries, NixOS configs, Immich Config, Immich DB)
    - File System will be exFAT for user access to library regardless of the system used
    - Not sure which scheduling options to provide - Scheduled (keep plugged in), Automatic (when plugged in), Manual (kick off in webUI)
    - (maybe - maybe optional) Encrypt .zip with all configs & DB to protect sensitive credentials
  - [ ] Syncthing - Need to look into this one quite a bit more. Hope to not have to introduce another UI to implement. Regardless, will likely require detailed guide for setting up target destination (receive only, versions, encryption/decryption).
    - Enables reliable same-network or remote one-way syncing to any NAS or PC.
    - Need to sync library as well as a folder for config files and DB dumps.
    - Has a rudimentary "versioning" system but still not really a "backup" solution. Mainly protects against system failure or does a good off-site sync for disasters that may take out the USB backup as well as the main system.
      - If syncing to a NAS or PC with FS snapshots, can be an effective versioned backup.
    - Has encrypted option for using an "untrusted" backup target (i.e., the self-hoster's NAS that recommended the project).
## Remote Access
LAST FEATURE - Still have a good bit of work and planning to do here but below is the idea for the feature.
  - [ ] Cloudflared - Very guide-heavy but local setup will be as simple as entering the tunnel token. Requires domain in Cloudflare.
    - Strongly advise use of OIDC when publicly exposed - needs advanced guide.
    - Guide for split-addresses in Immich app (local backups = faster backups).
    - Option to disable local access to admin page IF routing through Cloudflare WITH ZeroTrust auth in front.
  - [ ] Tailscale - Can automate more of the setup but requires client to be installed on each device.
  - [ ] Port forward, DuckDNS, Caddy, Let's Encrypt.
    - Implementation would likely be easy enough IF user is not behind CG NAT (can be checked I think) and can figure out port-forwarding.
    - Not sure I want to do this due to potential security concerns just having people expose their photos to the web and then relying on Caddy + Immich + Auto-updates + whatever insecure password they set up for their Immich accounts.
    - Would NOT allow for remote access to admin panel through this method (Caddy only responding to immich.local:8080 so even if they port forward, they can't expose it this way if they tried).
  - [ ] WireGuard, Port Forward, DuckDNS.
    - More secure, most complicated, relies on public IP address.
## Setup Script
  - Instructions to get script/installer onto machine
  - Installation either via flag on binary `[binary] --install` or via shell script `curl | sudo sh`

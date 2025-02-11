# NixOS-Immich-WebUI
A server management tool with a web UI designed to manage a NixOS system with the sole purpose of running and maintaining Immich on the host.

NOTE: If compiling as-is, this will not change any NixOS configs. This was intentional because I don't want anyone who may download this release in the future to accidentally overwrite their NixOS config.
To have this change the configuration.nix file stored in /etc/nixos/, you'll need to change the value on line 14 (see comment).
To apply changes and run a `nixos-rebuild switch`, lines 267-270 will need to be uncommented before compiling.

This repo currently houses the Proof of Concept where I learned about how to do the basic things necessary to build the project in Go. It currently does nothing useful. See the post on https://notes.rickyhaase.com for more details on this.

Also to note, as this is just a PoC, it was thrown together with different components from ChatGPT (also see https://notes.rickyhaase.com to see my thoughts on this (eventually)). I doubt that any of this will remain as I learn more about Go and re-write things as I see best.
Things like error handling and config re-loading are all a mess and mostly non-functional, and the entire structure is likely going to change as this is my first project getting back into things and my first time learning a compiled language (for however much that may change things).

## Environment Assumptions
These are the assumptions that are made about the environment in which the program will be run. For now, they are manually configured but a setup/installation script that configures all of these at install will need to be made.

1. There is a ZFS pool separate from the boot drive called "tank" with datasets tank/postgres and tank/immich which are where Immich data will be stored.
2. This program is stored in /root/ (until I find what best-practice is for storing a binary on NixOS that's not bundled in the package manager - it looks like /opt is used by NixOS, not sure if I can put things there not used by the package manager).
3. There has been no other configuration done on the system after installing it via the normal GUI installer.

## Roadmap/Future Feature List
- [ ] NixOS Server Management - Full NixOS server configuration with web UI for changing any options that need user input (what the final configuration will be and what will be configurable through the web UI is still TBD).
  - [ ] Avahi (mDNS)
  - [ ] Sanoid (ZFS Snapshot management)
  - [ ] Caddy (Reverse Proxy)
  - [ ] Docker
  - [ ] Firewall Config
  - [ ] Updates (enable unattended)
  - [ ] And I'm sure a few more to be added
- [ ] Immich container management
  - [ ] Local access via mDNS hostname immich.local
  - [ ] Docker Management - Pull, Start, Stop, Update, and Roll-back containers
  - [ ] Roll back DB (using the in-built DB dump. maybe ZFS snapshots to attempt more incremental rollbacks - not ideal but could be serviceable and easier), Roll back library state (using ZFS snapshots)
  - [ ] Reset full Immich instance (with undo thanks to ZFS - need to think about this one and retention on rollback snapshots)
- [ ] Immich Admin (could make this optional if admins want to manage Immich from within Immich) - need to decide to manage the config via API or JSON. Most likely use API - at least to pull user management out of the Immich web UI and keep all admin tasks together.
  - [ ] Automate the creation of the initial admin user (not sure how), store the creds (not sure how/where), generate admin user API token and store for use with other admin tasks (again, not sure how yet).
  - [ ] Configure email settings
  - [ ] Configure library settings (probably no interface, just pick what's "best" and lock it in with the JSON file. Need to at least know where photos are going for backups (lib vs upload)).
  - [ ] User creation (with storage labels)
  - [ ] Auto kick-off jobs based on external actions (storage migration)
  - [ ] Sync up backup schedule with configured system backups (future feature)
- [ ] Backup and Restore - Most critical and most complicated function to build - UX will be key.
  - [ ] Email Alerts (likely failure alerts and weekly summaries)
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
- [ ] Remote Access - LAST FEATURE - Still have a good bit of work and planning to do here but below is the idea for the feature.
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


## To-Dos
### NixOS Server Management
- [ ] Build out a set of .nix files that define the desired state of the server and are organized and compartmentalized in a logical manner.
  - [ ] configuration.nix with nothing but the imports and default timezone, language, and regional settings configured during install.
  - [ ] nixGo.nix with the systemd configuration for the web server and anything else that must remain and not be touched by the application.
  - [ ] Additional .nix files as needed to hold the configs that will be modifiable via the web interface.
  - [ ] admin.nix for an advanced admin to have a file that won't be touched, allowing them to modify with any additional configs they may want.
- [ ] Convert those .nix files into templates within a .go file that is imported into the program and can be called from the writer function.
- [ ] Build web pages that correspond to each .nix file and contain inputs to modify the parts of the server config that are necessary.
- [ ] Structure the web server to read the existing config corresponding to each webpage on load, save the .tmp file on save, alert when leaving without applying, and copy to .nix and run a rebuild on reload.
- [ ] Auto-rollback if nixos-rebuild fails. Auto-rollback if no web requests are accepted by the server within 60 seconds of new config being applied. Optional rollback to previously applied config if the admin is unhappy with any resulting changes.

## Implementation Notes
- I am not planning to implement any authentication for the admin interface at this time. Authentication will be handled via whatever methods are offered through the proxy being used (Caddy basic auth or for remote access, Cloudflare Zero Trust's auth mechanism).
- Not really sure if I need to manage or care about the root and user passwords on the system. The admin will choose those when installing NixOS on the metal, and I could reset those when taking over the config as root. I guess it's just something I haven't yet put much thought into and will need to make a decision at some point along the way (likely deployment script).

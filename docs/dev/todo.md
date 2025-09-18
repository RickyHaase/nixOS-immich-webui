# Development To-Do Items

This document tracks development tasks that are either in progress or planned for future releases. Items marked with checkboxes indicate completion status.

## Completed Items
- [x] Split README into different pages in /docs/dev/ to keep things organized.
- [x] Although the work has been done, only a monolithic Nix config file has been committed to the repo as it was manageable in one template and one page in the UI (so far).
  - ~~Create a set of .nix files that define the desired state of the server, organized and compartmentalized logically.~~
    - ~~configuration.nix with only the imports and default timezone, language, and regional settings configured during install.~~
    - ~~system.nix with the systemd service for the web server.~~
    - ~~Additional .nix files as needed to hold the configs that will be modifiable via the web interface.~~
    - ~~admin.nix for an advanced admin to have a file that won't be touched, allowing them to modify with any additional configs they may want.~~
      - ~~This could be modifiable in an advanced option to enable/disable SSH without overwriting the entire file, just updating the relevant string.~~
- [x] Convert the .nix file into a template.
- [x] Build a web page that contains inputs to modify the necessary parts of the server config.
- [x] Structure the web server to read the existing config corresponding to each webpage on load, save the .tmp file on save, alert when leaving without applying, and copy to .nix and run a rebuild on reload.
## Pending Items

### Core System
- [ ] Implement auto-rollback if nixos-rebuild fails
  - Auto-rollback if no web requests are accepted by the server within 60 seconds of new config being applied
  - Provide an optional rollback to the previously applied config if the admin is unhappy with any resulting changes
- [ ] Figure out how to embed templates into the binary
- [ ] Parse templates at initialization instead of at runtime after core development of templates (thanks to YouTube comment @iskariotski)
- [ ] Re-organize code into multiple files
- [ ] Create [Unit Tests](https://youtu.be/W4njY-VzkUU)
- [ ] Internal Backups as a failsafe incase one internal disk dies and the users have not performed proper backups to external media (backup server config to data disk and photos to boot disk (configurable with re-encoded files based on boot disk side))

### Frontend & UI
- [ ] Add HTMX and CSS libraries into the source instead of calling from CDNs (eventually, no rush now)
- [ ] Need to do something about email notifications and admin interface password reset
- [ ] Caddy basic Auth
- [ ] Add a check for the correct ZFS datasets and display a message along the lines of "please setup storage - check guide [here]" if the correct storage configuration is not found. At this time, this is an intentional design decision as a barrier to entry to make sure there is someone involved in the initial setup who can at least follow written instructions to setup/manage ZFS pools. This means that if everything gets toasted, there's atleast a chance of knowing someone who can mount the pool and pull photos (even if with written instructions)

### Container & Infrastructure
- [ ] Test Podman as an alternative to Docker
- [ ] Consider including Cockpit as a read-only server status admin page... might just be something to include in the example admin.nix file

### Remote Access
- [ ] Function to start/stop tailscale
- [ ] Function to sign out of tailscale
- [ ] Function to use tailscale serve for immich
- [ ] Cloudflare Tunnel integration - need docs on how to setup with OIDC and split connection in app
- [ ] Pangolin integration - basic w/ docs for self-hosted VPS

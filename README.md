# Easy Immich Server
A set of configuration files and a web application that compile into a single binary to manage a NixOS host, designed to provide an appliance-like experience for running Immich.

Currently under development. Building out templates, pages, docs, routes, and learning Go. Some resources will not be committed even if they're marked as "done" in the To-Do because while they've been created, they haven't been sanitized which will happen in an upcoming to-do.

NOTE: The main.go that is currently being committed to main has the file paths set to the test directory for all read/write operations. This will change in the beta releases, once I begin embedding the templates into the binary.

## Functionality Status
Have a "functional" web UI that can display the currently applied NixOS config (currently just reading from test/configuration.nix).

## Roadmap
- [x] v0.1.0-alpha.1
  - [x] Single configuration.nix file with complete and stable configuration for running an Immich server
  - [x] Immich accessible at http://immich.local and admin interface accessible at http://immich.local:8080
  - [x] Parse out and render modifiable settings in web UI
  - [x] Modify config and write new config to file system from web UI
  - [x] Apply Nix config and issue `nixos-rebuild switch` from web UI (error catching and rollback will NOT be included in this release)
  - [x] Minimal Immich container controls - start/stop, update
- [x] v0.1.0-alpha.2
  - [x] Add functionality to update Immich Email config from web UI
  - [x] Embed template files in the binary
  - [x] Add GUI interfaces to power off and restart the server
  - [x] Make sure that error handling is actually working as expected (unit testing not yet setup)
  - [x] Configured Logging levels (Info, Error, and Debug) - currently requires hard-coded switch
  - [x] Basic USB-drive backup (a restoration option will NOT be included in this release (all photos, the config files, and the latest DB Dump will be copied)). See [backups.md](/docs/dev/backups.md)
- [ ] v0.1.0-alpha.3
  - [ ] Include documentation and config files necessary to get a working server running
  - [ ] Refactor single main.go into seperate files/modules/componets/whatever Go calls them for better organization and easier maintainability
- [ ] v0.1.0-beta.1
  - [ ] Get some CSS and make a usable mobile-first UI
  - [ ] Enhance the web UI to be more responsive by using HTMX and modals to minimize page reloads. Ensure this is implemented with progressive enhancement and graceful degradation for clients without JavaScript
  - [ ] Add an update button for the host system
  - [ ] Sort out GitHub binary releases
<!-- - [ ] 0.1.0-beta.2 -->
  <!-- - [ ] Basic deployment mechanism -->
  <!-- - [ ] Make sure Immich is installed in the expected location before allowing a configuration update to be applied -->
<!-- - [ ] v0.1.0-rc.1 -->

## Environment Assumptions
[link](docs/dev/environment.md)
### Setup
This is currently much more manual than I hope it to be in the future but at this stage in development, this is what is required:
1. Most any hardware will do that meets NixOS and Immich's compatibility requirements.
2. Requires separate boot and storage drives and at this time, an SSD is recommended for storage.
3. Install NixOS to the boot drive.
4. Configure the data drive as a ZFS pool called "tank" and create the datasets as defined in [storage.md](/docs/setup/storage.md)
5. (WIP) Place the files (and binary) from the release page into their [respective folders](/docs/setup/environment.md) (create folders as needed)

## Features
- [ ] Contains instructions to set up a functional NixOS server running and serving Immich at http://immich.local/
- [ ] Make changes to the configuration of a server that was pre-configured per the setup guide above
- [ ] Add the server to Tailscale using the web UI (only for SSH access at this time)
- [ ] Start/Stop, Update, and configure sending Gmail for installed Immich instance

[planned](docs/dev/features.md)

## To-Do
[link](docs/dev/todo.md)

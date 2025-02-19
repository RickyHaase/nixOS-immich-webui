# Easy Immich Server
A server management tool with a web UI designed to manage a NixOS system with the sole purpose of running and maintaining Immich on the host.

Currently under development. Building out templates, pages, docs, routes, and learning Go. Some resources will not be committed even if they're marked as "done" in the To-Do because while they've been created, they haven't been santized which will happen in an upcoming to-do

## Functionality Status
Have a "functional" webui that can display the currently applied nixos config (currently just reading from test/configuration.nix).

## Roadmap
- [ ] 0.1.0-Alpha.1
  - [x] Single configuration.nix file with complete and stable configuration for running an Immich server
  - [x] Immich accessible at http://immich.local and admin interface accessible at http://immich.local:8080
  - [x] Parse out and render modifiable settings in webui
  - [ ] Modify config and write new config to file system from webui
  - [ ] Apply nix config and issue `nixos-rebuild switch` from webui (error catching and rollback will NOT be included in this release)
  - [ ] Install Immich with minimal controls (outside of default settings) - start/stop, update, and use gmail account
  - [ ] Basic USB-drive backup (a restoration option will NOT be included in this release (all photos will be there on the file system tho))
  - [ ] Include documenation and config files necessary to get a working server running

## Environment Assumptions
[link](docs/dev/environment.md)

## Features
- [ ] nothin yet

[planned](docs/dev/features.md)

## To-Do
[link](docs/dev/todo.md)

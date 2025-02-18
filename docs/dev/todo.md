# To-Dos
## NixOS Server Management
- [x] Split readme out into different pages in /docs/dev/ to keep things organized
- [x] Build out a set of .nix files that define the desired state of the server and are organized and compartmentalized in a logical manner.
  - [x] configuration.nix with nothing but the imports and default timezone, language, and regional settings configured during install.
  - [x] system.nix with the systemd service for the web server
  - [x] Additional .nix files as needed to hold the configs that will be modifiable via the web interface.
  - [x] admin.nix for an advanced admin to have a file that won't be touched, allowing them to modify with any additional configs they may want.
    - Perhaps this will be modifiable in an advanced option to enable/disable ssh but it will not overwrie the whole file, instead just updating the one relivant string
- [ ] Convert those .nix files into templates within a .go file that is imported into the program and can be called from the writer function.
- [ ] Build web pages that correspond to each .nix file and contain inputs to modify the parts of the server config that are necessary.
- [ ] Structure the web server to read the existing config corresponding to each webpage on load, save the .tmp file on save, alert when leaving without applying, and copy to .nix and run a rebuild on reload.
- [ ] Auto-rollback if nixos-rebuild fails. Auto-rollback if no web requests are accepted by the server within 60 seconds of new config being applied. Optional rollback to previously applied config if the admin is unhappy with any resulting changes.

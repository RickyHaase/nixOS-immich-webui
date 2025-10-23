## Preface

This doc is a full step-by-step walkthrough of my first deployment and setup (from the near-final alpha3 branch). This deployment will go live as the first test deployment and I will be manually updating it incrementally from the repository.

Once in beta with pre-built binaries and a proper setup script then I'll update the ture deployment.md with "official" setup instructions. In the meantime, this will serve as the complete working deployment guide for testing.

There will also be a full [Google ODIC/Cloudflare Tunnel guide](./google+cloudflare) created as part of this setup process (manual - no web UI (yet))

## Step-by-step

1. [Download nixos](https://nixos.org/download/#nix-install-linux) and flash to USB (I used Balena Etcher)
2. Got an old 8th gen i7 Dell micro PC and installed an old 1TB SATA SSD for boot and a 2TB NVME for media storage
3. Installed nixos onto the SATA (with GUI, since I'll be deploying at a friend's house and I want him to be able to plug into a monitor and be able orient himself in the OS to follow instructions over the phone incase of issue)
  - BIOS settings: SATA Operation: AHCI, AC Power Recover: on, Secure Boot: off
  - Boot to USB - requires internet I guess... don't love that, maybe I pulled the minimal installer - doesn't matter
  - created nixos user in setup wizard - used same pass for root (keeping things simple right now... may change for final setup)
  - chose to include GNOME desktop - general server installs may skip this (especailly if on lower power machines)
  - Allow unfree software (makes life easier - not strictly required)
  - Partitions: select internal SATA and chose Erase disk (with swap - keeps system stable during unexpectedly high memory usage). No encryption (keeps recovery and admin simple).
  - Install -> Restart Now -> Done -> Unplug Installer USB -> Boot to internal nixos install
4. Queued up nixos config
    - with monitor and internet still connected, login to the account you setup during the install
    - open terminal and switch to root with `sudo su`
    - change to nixos config dir `cd etc/nixos`
    - create `admin.nix` with the `git` package ([see example admin.nix](../../example/etc/nixos/admin.nix)) (NOTE: if not using a Desktop Environment and you only have the console, including `tmux` at this time is a good idea)
    - edit `configuration.nix` to import `admin.nix` ([see example configuration.nix](../../example/etc/nixos/configuration.nix.md))
    - run `nixos-rebuild switch` to get the new package
    - open a new terminal and clone the git repo into the standard user's home folder `git clone https://github.com/RickyHaase/nixOS-immich-webui`
    - cd into the newly created project directory. In my case, I needed to pull a different branch so I did so `git checkout -b v0.1.0-alpha.3-dev origin/v0.1.0-alpha.3-dev`
    - switch to root `sudo su`
    - copy the example .nix files into the nixos dir `cp -r example/etc/nixos/* /etc/nixos`
    - switch back to the terminal in /etc/nixos (or navigate there if it's closed)
    - modify the new admin.nix to only include what is needed/wanted (git, go, tmux, etc.)
    - add ZFS import in `configuration.nix` file (there should be `./hardware-configuration.nix`, `./admin.nix`, and `./zfs.nix` in imports). No other imports or changes should be applied to `configuration.nix` just yet.
    - comment out the line is zfs.nix that imports pool "tank": `# boot.zfs.extraPools = [ "tank" ];`
    - run `nixos-rebuild switch` to enable the new configurations/packages (go and ZFS)
5. Prepared nixmich binary
  - switch back to the terminal in the git repo (or navigate there if only using one terminal)
  - run `go build .`
  - create dir in /root for nixmich to live: `mkdir ~/nixmich-prod`
  - move the newly made binary into the diectory and rename it `nixmich`: `mv nixOS-immich-webui ~/nixmich-prod/nixmich`
  - confirm binary runs in new location: `~/nixmich-prod/nixmich`
  - there should be a log indicating that the server started. It won't be reachable yet (firewall not configured) so just `ctrl + c` to stop process
6. Prepared ZFS datasets
  - NOTE: I had to reboot the system before `zpool create` would play nice. No issues after that.
  - run `lsblk` to identify the name of the disk(s) being used in the ZFS pool. In this case, it's `nvme0n1`
  - create zpool tank: `zpool create -o ashift=12 -o autotrim=on -f tank /dev/nvme0n1` (this command is device-specific so if you're not running a Samsung 990 EVO Plus, do some research to verify the correct "-o" args)
  - modify `zfs.nix` to auto-mount "tank" in the future by uncommenting the line `boot.zfs.extraPools = [ "tank" ];` (no necessary to rebuild at this time)
  - NOTE: at this time it's a good idea to open up the `storage.md` file from the setup docs in another terminal (or wherever on the same machine) to copy-paste commands over from it
  - create postgres dataset:
  ```
  zfs create \
    -o recordsize=8K \
    -o compression=lz4 \
    -o atime=off \
    -o relatime=on \
    -o sync=standard \
    -o prefetch=metadata \
    tank/pgdata
  ```
  - create Immich datasets:
  ```
  zfs create -o recordsize=128K -o compression=lz4 -o atime=off tank/immich
  zfs create -o recordsize=512K -o compression=lz4 -o atime=off tank/immich/library
  ```
  - create config datasets:
  ```
  zfs create -o compression=lz4 -o copies=2 tank/config-backups
  zfs create -o compression=lz4 -o copies=2 tank/immich-compose
  ```
7. Prepared immich config, compose, and env
  - return to the `nixOS-immich-webui` project folder
  - copy the immich config files to their correct directories:
  ```
  cp example/immich/immich-config.json /tank/immich/immich-config.json
  cp example/immich/immich-compose/example.env /immich/immich-compose/.env
  cp example/immich/immich-compose/docker-compose.yml /immich/immich-compose/docker-compose.yml
  ```
8. Applied remaining nixos config
  - modify configuration.nix per the remaining instruction in configuration.nix.md (include the remaining imports and comment out hostname and timezone)
  - run `nixos-rebuild switch`
  - this will apply all configurations and start all services to permit the use of the nixmich web UI AND start the immich server
9. Went to http://immich.local:8080 to complete server and immich configuration
  - enable auto-updates (ensure nixos-rebuild functions)
  - enable tailscale with valid authkey
  - add gmail as SMTP server for immich notifications
10. go to http://immich.local to setup immich admin account
  - tested email notifications
11. configure remote access via the [google + cloudflare](./google+cloudflare.md) doc
  - this requires some dev and will be completed tonight (hopefully)

NOTE: I also had to disable "automatic suspend" in the gnome settings page... not sure if this would actually cause any issues but I have disabled it just in case.

Settings page did not persist reboot. If including Gnome DE, it is recommended to include the below in the configuration.nix file (along with lib in the config, pkgs, ... at the top). That said, I disabled in system.nix via another method since so it may not be required.

```
  # Set GNOME power settings via dconf (applies system-wide, including GDM)
  programs.dconf.profiles.gdm.databases = [{
    settings = {
      "org/gnome/settings-daemon/plugins/power" = {
        sleep-inactive-ac-type = "nothing";
        sleep-inactive-battery-type = "nothing";
        sleep-inactive-ac-timeout = lib.gvariant.mkUint32 0;
        sleep-inactive-battery-timeout = lib.gvariant.mkUint32 0;
      };
    };
  }];

  # Also set defaults for user sessions
  programs.dconf.profiles.user.databases = [{
    settings = {
      "org/gnome/settings-daemon/plugins/power" = {
        sleep-inactive-ac-type = "nothing";
        sleep-inactive-battery-type = "nothing";
        sleep-inactive-ac-timeout = lib.gvariant.mkUint32 0;
        sleep-inactive-battery-timeout = lib.gvariant.mkUint32 0;
      };
      "org/gnome/desktop/session" = {
        idle-delay = lib.gvariant.mkUint32 0;
      };
    };
  }];
```

## Binary update
1. pull whatever branch/tag/whatever you want to update to from github
2. build the binary
3. stop systemd service `systemctl stop nixmich`
4. replace the binary  `/root/nixmich-prod/nixmich` with a new one (same name, same location)
5. start the systemd service `systemct start nixmich`

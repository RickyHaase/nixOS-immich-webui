# ZFS Setup
1. Create `/etc/nixos/zfs.nix`
  ```
  networking.hostId = "12345678";

  boot.supportedFilesystems = [ "zfs" ];
  boot.zfs.forceImportRoot = false;
  # boot.zfs.extraPools = [ "tank" ];

  services.zfs.autoScrub.enable = true;
  ```
2. In configuration.nix, add ./zfs.nix to the list of imports
3. Reboot machine
4. Create ZFS pool tank
  ```
  # in my case, this is the first and only sata disk - /dev/sda
  zpool create -o ashift=9 -o autotrim=on -f tank /dev/sda
  ```
5. uncomment `boot.zfs.extraPools = [ "tank" ];` in zfs.nix and rebuild
6. Create postgres dataset
  ```
  zfs create \
    -o recordsize=8K \
    -o logbias=latency \
    -o compression=lz4 \
    -o atime=off \
    -o relatime=on \
    -o sync=standard \
    tank/pgdata
  ```
6. Create Immich datasets
  ```
  # Create the parent dataset (for thumbnails and re-encoded videos)
  zfs create -o recordsize=128K -o compression=lz4 -o atime=off tank/immich

  # Create the child dataset (for original, high-quality uploads)
  # copies=2 will use double space but enable healing of detected corruption
  # May need to be tank/immich/upload if not using storage template
  zfs create -o recordsize=512K -o compression=lz4 -o copies=2 -o atime=off tank/immich/library
  ```


MAYBE: Create config dataset - compose, .env,  Perhaps a config-backup dataset would make more sense - .zips containing compose, env, .nix, and DB backups
zfs create -o recordsize=16K -o compression=lz4 -o copies=2 -o atime=off tank/config

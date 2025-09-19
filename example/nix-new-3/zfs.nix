# zfs.nix - ZFS filesystem and snapshot configuration using JSON variables
{ config, pkgs, ... }:

let
  # Consistent builtins.fromJSON pattern across all modules
  vars = builtins.fromJSON (builtins.readFile ./variables.json);
in
{
  # ZFS support and configuration from JSON
  boot.supportedFilesystems = [ "zfs" ];
  boot.zfs = {
    forceImportRoot = false;
    extraPools = [ vars.storage.zfs.poolName ];
  };

  # ZFS maintenance services using JSON configuration
  services.zfs.autoScrub = {
    enable = vars.storage.zfs.autoScrub;
    pools = [ vars.storage.zfs.poolName ];
  };

  # Sanoid snapshot management with JSON retention settings
  services.sanoid = {
    enable = true;
    interval = "hourly";
    datasets = {
      "${vars.storage.zfs.poolName}" = {
        recursive = true;
        autoprune = true;
        autosnap = true;
        
        # Snapshot retention from JSON configuration
        hourly = vars.storage.zfs.snapshots.hourly;
        daily = vars.storage.zfs.snapshots.daily;
        weekly = vars.storage.zfs.snapshots.weekly;
        monthly = vars.storage.zfs.snapshots.monthly;
        yearly = vars.storage.zfs.snapshots.yearly;
      };
    };
  };
}
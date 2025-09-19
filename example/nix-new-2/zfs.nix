# zfs.nix - ZFS filesystem and snapshot configuration
{ config, pkgs, ... }:

let
  # Import variables from variables.nix
  vars = import ./variables.nix;
in
{
  # ZFS support and configuration
  boot.supportedFilesystems = [ "zfs" ];
  boot.zfs = {
    forceImportRoot = false;
    extraPools = [ vars.zfsPoolName ];
  };

  # ZFS maintenance services
  services.zfs.autoScrub = {
    enable = true;
    pools = [ vars.zfsPoolName ];
  };

  # Sanoid snapshot management
  services.sanoid = {
    enable = true;
    interval = "hourly";
    datasets = {
      "${vars.zfsPoolName}" = {
        recursive = true;
        autoprune = true;
        autosnap = true;
        
        # Snapshot retention settings
        hourly = 24;
        daily = 7;
        weekly = 0;
        monthly = 0;
        yearly = 0;
      };
    };
  };
}
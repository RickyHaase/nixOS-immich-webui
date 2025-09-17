{ config, pkgs, ... }:

{
  networking.hostId = "814357d3";

  boot.supportedFilesystems = [ "zfs" ];
  boot.zfs.forceImportRoot = false;
  boot.zfs.extraPools = [ "tank" ];

  services.zfs.autoScrub.enable = true;

  services.sanoid.enable = true;
  services.sanoid  = {
    interval = "hourly";
    datasets = {
      "tank" = {
        recursive = true;
        autoprune = true;
        autosnap = true;
        hourly = 24;
        daily = 7;
        weekly = 1;
        monthly = 0;
        yearly = 0;
      };
    };
  };
}

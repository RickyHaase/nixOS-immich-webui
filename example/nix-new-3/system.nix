# system.nix - System-level configuration using JSON variables
{ config, pkgs, ... }:

let
  # Simple pattern repeated in every module - read JSON directly
  vars = builtins.fromJSON (builtins.readFile ./variables.json);
in
{
  # Timezone configuration from JSON
  time.timeZone = vars.system.timeZone;

  # Automatic system upgrades from JSON
  system.autoUpgrade = {
    enable = vars.system.autoUpgrade;
    dates = vars.system.upgradeTime;
    flags = [
      "--update-input"
      "nixpkgs"
      "-L" # print build logs
    ];
    randomizedDelaySec = "15min";
    allowReboot = vars.system.autoUpgrade;
    rebootWindow = {
      lower = vars.system.upgradeLower;
      upper = vars.system.upgradeUpper;
    };
  };

  # USB device support for backups
  services.udisks2.enable = true;
  
  # Essential system packages
  environment.systemPackages = with pkgs; [
    zip  # Required for backup functionality
  ];
}
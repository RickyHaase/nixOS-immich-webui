# system.nix - System-level configuration using imported variables
{ config, pkgs, ... }:

let
  # Import variables from variables.nix
  vars = import ./variables.nix;
in
{
  # Timezone configuration
  time.timeZone = vars.timeZone;

  # Automatic system upgrades
  system.autoUpgrade = {
    enable = vars.autoUpgrade;
    dates = vars.upgradeTime;
    flags = [
      "--update-input"
      "nixpkgs"
      "-L" # print build logs
    ];
    randomizedDelaySec = "15min";
    allowReboot = vars.autoUpgrade;
    rebootWindow = {
      lower = vars.upgradeLower;
      upper = vars.upgradeUpper;
    };
  };

  # USB device support for backups
  services.udisks2.enable = true;
  
  # Essential system packages
  environment.systemPackages = with pkgs; [
    zip  # Required for backup functionality
  ];
}
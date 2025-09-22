{ config, pkgs, ... }:

let
  # Read JSON variables using builtins.fromJSON
  vars = builtins.fromJSON (builtins.readFile ./nixconfig.json);
in
{
  # Timezone from JSON
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
    randomizedDelaySec = "45min";
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

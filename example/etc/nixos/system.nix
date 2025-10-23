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

  # Completely disable suspend/hibernate at the systemd level
  systemd.targets.sleep.enable = false;
  systemd.targets.suspend.enable = false;
  systemd.targets.hibernate.enable = false;
  systemd.targets.hybrid-sleep.enable = false;

  # Systemd service for go app stored in /root
  systemd.services.nixmich = {
    description = "nixmich web UI";
    after = [ "network.target" ];
    wantedBy = [ "multi-user.target" ];
    serviceConfig = {
      ExecStart = pkgs.writeShellScript "nixmich-start" ''
        source /etc/profile
        exec /root/nixmich-prod/nixmich
      '';
      Restart = "always";
      User = "root";
      WorkingDirectory = "/root/nixmich-prod";
      StandardOutput = "journal";
      StandardError = "journal";
    };
  };
}

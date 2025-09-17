{ config, pkgs, ... }:

{
# #Systemd service for go app stored in /root
    # systemd.services.webui = {
    #     description = "NixOS-Immich WebUI Service";
    #     after = [ "network.target" ];
    #     wantedBy = [ "multi-user.target" ];
    #     serviceConfig = {
    #         ExecStart = "/root/ezimmich";
    #         Restart = "always";
    #         User = "root";
    #         WorkingDirectory = "/root";
    #         StandardOutput = "journal";
    #         StandardError = "journal";
    #     };
    # };

# #Enable Unattended Upgrades
  system.autoUpgrade.enable = true;
  system.autoUpgrade.dates = "02:00";
  system.autoUpgrade = {
    # flake = inputs.self.outPath;
    flags = [
      "--update-input"
      "nixpkgs"
      "-L" # print build logs
    ];
    randomizedDelaySec = "45min";
  };
  system.autoUpgrade.allowReboot = true;
  system.autoUpgrade.rebootWindow.lower = "03:00";
  system.autoUpgrade.rebootWindow.upper = "04:00";
}

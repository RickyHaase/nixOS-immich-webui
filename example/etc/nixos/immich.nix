{ config, pkgs, ... }:

{
  virtualisation.docker.enable = true;
  virtualisation.docker.autoPrune.enable = true;
  virtualisation.docker.autoPrune.dates = "monthly";

  environment.systemPackages = with pkgs; [
    wget
  ];

  users.groups.immich = {};
  users.users.immich = {
    isSystemUser = true;
    group = "immich";
    description = "Immich";
    extraGroups = [ "docker" ];
  };

  systemd.services.immich-app = {
    description = "Manage Immich Compose Stack";
    requires = [ "docker.service" ];
    after = [ "docker.service" ];
    wantedBy = [ "multi-user.target" ];

    serviceConfig = {
      Type = "simple";
      ExecStart = "${pkgs.docker}/bin/docker compose up";
      ExecStop = "${pkgs.docker}/bin/docker compose down";
      Restart = "always";
      WorkingDirectory = "/tank/immich-config";
      TimeoutStopSec = "90";
    };
  };

  # Need to define a cron job to update Immich
  # Enable/disable and configure time in webGUI

}

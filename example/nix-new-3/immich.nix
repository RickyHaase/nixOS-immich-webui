# immich.nix - Immich application and Docker configuration using JSON variables
{ config, pkgs, ... }:

let
  # Consistent pattern - builtins.fromJSON reads the configuration
  vars = builtins.fromJSON (builtins.readFile ./variables.json);
in
{
  # Docker virtualization with JSON configuration
  virtualisation.docker = {
    enable = true;
    autoPrune = {
      enable = true;
      dates = vars.immich.autoPruneSchedule;
    };
  };

  # Required packages for Immich operations
  environment.systemPackages = with pkgs; [
    wget  # Required for docker-compose and other operations
  ];

  # Immich user and group configuration (optional - currently using root)
  users.groups.immich = {};
  users.users.immich = {
    isSystemUser = true;
    group = "immich";
    description = "Immich Application User";
    extraGroups = [ "docker" ];
  };

  # Immich systemd service using JSON variables
  systemd.services.immich-app = {
    description = "Manage Immich Docker Compose Stack";
    requires = [ "docker.service" ];
    after = [ "docker.service" ];
    wantedBy = [ "multi-user.target" ];

    serviceConfig = {
      Type = "simple";
      ExecStart = "${pkgs.docker}/bin/docker compose up";
      ExecStop = "${pkgs.docker}/bin/docker compose down";
      Restart = "always";
      WorkingDirectory = vars.immich.workingDirectory;
      TimeoutStopSec = vars.immich.dockerTimeout;
      # Note: Currently runs as root, could be changed to immich user
    };
  };
}
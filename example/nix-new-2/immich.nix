# immich.nix - Immich application and Docker configuration
{ config, pkgs, ... }:

let
  # Import variables from variables.nix
  vars = import ./variables.nix;
in
{
  # Docker virtualization
  virtualisation.docker = {
    enable = true;
    autoPrune = {
      enable = true;
      dates = "monthly";
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

  # Immich systemd service
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
      WorkingDirectory = vars.immichWorkingDir;
      TimeoutStopSec = vars.immichTimeout;
      # Note: Currently runs as root, could be changed to immich user
    };
  };
}
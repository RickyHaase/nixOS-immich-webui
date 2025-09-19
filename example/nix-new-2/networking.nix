# networking.nix - Network configuration using imported variables
{ config, pkgs, ... }:

let
  # Import variables from variables.nix
  vars = import ./variables.nix;
in
{
  # Host configuration
  networking.hostName = vars.hostName;
  networking.hostId = vars.hostId;  # Required for ZFS

  # Avahi (mDNS) service discovery
  services.avahi = {
    enable = true;
    openFirewall = true;
    hostName = vars.hostName;
    publish = {
      enable = true;
      addresses = true;
    };
    extraServiceFiles = {
      immich = ''
        <service-group>
          <name replace-wildcards="yes">%h</name>
          <service>
           <type>_http._tcp</type>
           <port>${toString vars.webPublicPort}</port>
          </service>
        </service-group>
      '';
    };
  };

  # Caddy reverse proxy
  services.caddy = {
    enable = true;
    virtualHosts = {
      # Public Immich access
      "${vars.hostName}.local:${toString vars.webPublicPort}" = {
        extraConfig = ''
          reverse_proxy http://localhost:${toString vars.immichInternalPort}
        '';
      };
      # Admin panel access
      ":${toString vars.adminPanelPort}" = {
        extraConfig = ''
          reverse_proxy http://localhost:8000
        '';
      };
    };
  };

  # Firewall configuration
  networking.firewall = {
    allowPing = true;
    allowedTCPPorts = [ vars.webPublicPort vars.adminPanelPort ];
  };
}
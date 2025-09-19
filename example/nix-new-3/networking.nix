# networking.nix - Network configuration using JSON variables
{ config, pkgs, ... }:

let
  # Same simple pattern - read JSON directly with builtins
  vars = builtins.fromJSON (builtins.readFile ./variables.json);
in
{
  # Host configuration from JSON
  networking.hostName = vars.networking.hostName;
  networking.hostId = vars.networking.hostId;  # Required for ZFS

  # Avahi (mDNS) service discovery
  services.avahi = {
    enable = true;
    openFirewall = true;
    hostName = vars.networking.hostName;
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
           <port>${toString vars.ports.webPublic}</port>
          </service>
        </service-group>
      '';
    };
  };

  # Caddy reverse proxy using JSON port configuration
  services.caddy = {
    enable = true;
    virtualHosts = {
      # Public Immich access
      "${vars.networking.hostName}.local:${toString vars.ports.webPublic}" = {
        extraConfig = ''
          reverse_proxy http://localhost:${toString vars.ports.immichInternal}
        '';
      };
      # Admin panel access
      ":${toString vars.ports.adminPanel}" = {
        extraConfig = ''
          reverse_proxy http://localhost:8000
        '';
      };
    };
  };

  # Firewall configuration from JSON
  networking.firewall = {
    allowPing = vars.firewall.allowPing;
    allowedTCPPorts = vars.firewall.allowedTCPPorts;
  };
}
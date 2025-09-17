{ config, pkgs, ... }:

let
  hostName = "immich-dev-vm";
in
{
  networking.hostName = hostName;

  # networking.domain = ".local";
  # networking.enableIPv6 = false;
  # boot.kernel.sysctl."net.ipv6.conf.all.disable_ipv6" = true;

  # Avahi (mDNS)
  services.avahi = {
    enable = true;
    openFirewall = true;
    hostName = hostName;
    publish = {
      #userServices = true;
      enable = true;
      #domain = true;
      addresses = true;
      #workstation = true;
    };
    extraServiceFiles = {
      immich = ''
        <service-group>
          <name replace-wildcards="yes">%h</name>
          <service>
           <type>_http._tcp</type>
           <port>80</port>
          </service>
        </service-group>
      '';
    };
  };

  # Caddy (reverse proxy)
  services.caddy.enable = true;
  services.caddy = {
    virtualHosts."${hostName}.local:80".extraConfig = ''
      reverse_proxy http://localhost:2283
    '';
    virtualHosts.":8080".extraConfig = ''
      reverse_proxy http://localhost:8000
    '';
  };

  networking.firewall.allowPing = true;
  networking.firewall.allowedTCPPorts = [ 80 8080 ];

}

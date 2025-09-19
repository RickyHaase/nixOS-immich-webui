# remoteaccess.nix - Remote access configuration using JSON variables
{ config, pkgs, ... }:

let
  # Same pattern in every file - simple and consistent
  vars = builtins.fromJSON (builtins.readFile ./variables.json);
in
{
  # Tailscale VPN service from JSON configuration
  services.tailscale.enable = vars.remoteAccess.tailscale.enable;

  # Automatic Tailscale connection service
  systemd.services.tailscale-autoconnect = {
    description = "Automatic connection to Tailscale";

    # Service dependencies
    after = [ "network-pre.target" "tailscale.service" ];
    wants = [ "network-pre.target" "tailscale.service" ];
    wantedBy = [ "multi-user.target" ];

    # Run as oneshot service
    serviceConfig.Type = "oneshot";

    # Connection script using JSON auth key
    script = with pkgs; ''
      # Wait for tailscaled to settle
      sleep 2

      # Check if already authenticated to tailscale
      status="$(${tailscale}/bin/tailscale status -json | ${jq}/bin/jq -r .BackendState)"
      if [ $status = "Running" ]; then
        # Already connected, exit successfully
        exit 0
      fi

      # Authenticate with tailscale using auth key from JSON
      ${tailscale}/bin/tailscale up -authkey ${vars.remoteAccess.tailscale.authKey} --ssh
    '';
  };
}
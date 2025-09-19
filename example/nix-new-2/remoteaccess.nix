# remoteaccess.nix - Remote access configuration using imported variables
{ config, pkgs, ... }:

let
  # Import variables from variables.nix
  vars = import ./variables.nix;
in
{
  # Tailscale VPN service
  services.tailscale.enable = vars.tailscaleEnable;

  # Automatic Tailscale connection service
  systemd.services.tailscale-autoconnect = {
    description = "Automatic connection to Tailscale";

    # Service dependencies
    after = [ "network-pre.target" "tailscale.service" ];
    wants = [ "network-pre.target" "tailscale.service" ];
    wantedBy = [ "multi-user.target" ];

    # Run as oneshot service
    serviceConfig.Type = "oneshot";

    # Connection script
    script = with pkgs; ''
      # Wait for tailscaled to settle
      sleep 2

      # Check if already authenticated to tailscale
      status="$(${tailscale}/bin/tailscale status -json | ${jq}/bin/jq -r .BackendState)"
      if [ $status = "Running" ]; then
        # Already connected, exit successfully
        exit 0
      fi

      # Authenticate with tailscale using provided auth key
      ${tailscale}/bin/tailscale up -authkey ${vars.tsAuthKey} --ssh
    '';
  };
}
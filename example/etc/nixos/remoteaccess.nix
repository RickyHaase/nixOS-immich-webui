{ config, pkgs, ... }:

let
  # Read JSON variables using builtins.fromJSON
  vars = builtins.fromJSON (builtins.readFile ./variables.json);
in
{
  # cloudflared
  # Must also allow unfree
  # Not sure if it will be better to run on host or in docker

  # Tailscale VPN service from JSON configuration
  services.tailscale.enable = vars.remoteAccess.tailscale.enable;

  # create a oneshot job to authenticate to Tailscale
  systemd.services.tailscale-autoconnect = {
    description = "Automatic connection to Tailscale";

    # make sure tailscale is running before trying to connect to tailscale
    after = [ "network-pre.target" "tailscale.service" ];
    wants = [ "network-pre.target" "tailscale.service" ];
    wantedBy = [ "multi-user.target" ];

    # set this service as a oneshot job
    serviceConfig.Type = "oneshot";

    # have the job run this shell script using JSON auth key
    script = with pkgs; ''
    # wait for tailscaled to settle
    sleep 2

    # check if we are already authenticated to tailscale
    status="$(${tailscale}/bin/tailscale status -json | ${jq}/bin/jq -r .BackendState)"
    if [ $status = "Running" ]; then # if so, then do nothing
        exit 0
    fi

    # otherwise authenticate with tailscale using auth key from JSON
    ${tailscale}/bin/tailscale up -authkey ${vars.remoteAccess.tailscale.authKey} --ssh
    '';
    # Maybe add ssh and/or serve options

  };
}

{ config, pkgs, ... }:

let
  vars = builtins.fromJSON (builtins.readFile ./nixconfig.json);
in
{
  services.garage = {
    enable = vars.ente.enable;
    package = pkgs.garage;

    settings = {
      # Single-node deployment — no replication needed
      replication_factor = 1;

      metadata_dir = "/tank/ente/garage/meta";
      data_dir = "/tank/ente/garage/data";

      # Bind to localhost only — Ente Museum accesses S3 locally
      rpc_bind_addr = "127.0.0.1:3901";
      rpc_public_addr = "127.0.0.1:3901";

      s3_api = {
        s3_region = "garage";
        api_bind_addr = "127.0.0.1:3900";
        root_domain = ".s3.garage.localhost";
      };

      # Admin API for bucket/key management (local only)
      admin = {
        api_bind_addr = "127.0.0.1:3903";
      };
    };
  };

  # ZFS datasets must be created manually before first use:
  #   zfs create tank/ente
  #   zfs create tank/ente/garage
  #   zfs create tank/ente/garage/meta
  #   zfs create tank/ente/garage/data
}

{ config, pkgs, ... }:

let
  vars = builtins.fromJSON (builtins.readFile ./nixconfig.json);
in
{
  services.ente = {
    api = {
      enable = vars.ente.enable;
      enableLocalDB = true; # Auto-provisions PostgreSQL database

      settings = {
        # Garage S3 configuration
        s3 = {
          are_local_buckets = true;      # HTTP, not HTTPS (Garage is local)
          use_path_style_urls = true;    # Required for Garage

          # The bucket key names are hardcoded by Ente (b2-eu-cen, etc.)
          # but they can point to any S3-compatible provider
          b2-eu-cen = {
            key = { _secret = "/tank/ente/secrets/garage-key"; };
            secret = { _secret = "/tank/ente/secrets/garage-secret"; };
            endpoint = "http://127.0.0.1:3900";
            region = "garage";
            bucket = "b2-eu-cen";
          };
        };

        key = {
          encryption = { _secret = "/tank/ente/secrets/encryption-key"; };
          hash = { _secret = "/tank/ente/secrets/hash-key"; };
        };

        jwt = {
          secret = { _secret = "/tank/ente/secrets/jwt-secret"; };
        };
      };
    };

    web = {
      enable = vars.ente.enable;
    };
  };

  # Ensure Garage starts before Ente Museum
  systemd.services.ente = {
    after = [ "garage.service" ];
    requires = [ "garage.service" ];
  };
}

{ config, pkgs, ... }:

let
  vars = builtins.fromJSON (builtins.readFile ./nixconfig.json);

  # Script that initializes Garage layout, creates the Ente bucket and key.
  # Idempotent — safe to run on every boot. Skips steps that are already done.
  garageSetupScript = pkgs.writeShellScript "garage-setup" ''
    set -euo pipefail

    GARAGE="${pkgs.garage}/bin/garage"
    SECRETS_DIR="/tank/ente/secrets"
    mkdir -p "$SECRETS_DIR"
    chmod 700 "$SECRETS_DIR"

    # Wait for Garage API to be ready
    for i in $(seq 1 30); do
      if $GARAGE status >/dev/null 2>&1; then
        break
      fi
      sleep 1
    done

    # Apply layout if no layout is currently applied
    NODE_ID=$($GARAGE node id --quiet 2>/dev/null | cut -c1-16)
    if [ -n "$NODE_ID" ]; then
      LAYOUT_STATUS=$($GARAGE layout show 2>&1 || true)
      if echo "$LAYOUT_STATUS" | grep -q "no role assigned"; then
        $GARAGE layout assign "$NODE_ID" -z dc1 -c 1G
        $GARAGE layout apply --version 1 2>/dev/null || $GARAGE layout apply
      fi
    fi

    # Create bucket if it doesn't exist
    if ! $GARAGE bucket info b2-eu-cen >/dev/null 2>&1; then
      $GARAGE bucket create b2-eu-cen
    fi

    # Create key if secrets don't already exist
    if [ ! -f "$SECRETS_DIR/garage-key" ] || [ ! -f "$SECRETS_DIR/garage-secret" ]; then
      KEY_OUTPUT=$($GARAGE key create ente-museum-key 2>&1 || true)

      # If key already exists, get its info instead
      if echo "$KEY_OUTPUT" | grep -q "already exists"; then
        KEY_OUTPUT=$($GARAGE key info ente-museum-key 2>&1)
      fi

      # Extract key ID and secret from output
      KEY_ID=$(echo "$KEY_OUTPUT" | grep -oP 'Key ID: \K\S+' || true)
      KEY_SECRET=$(echo "$KEY_OUTPUT" | grep -oP 'Secret key: \K\S+' || true)

      if [ -n "$KEY_ID" ] && [ -n "$KEY_SECRET" ]; then
        echo -n "$KEY_ID" > "$SECRETS_DIR/garage-key"
        echo -n "$KEY_SECRET" > "$SECRETS_DIR/garage-secret"
        chmod 600 "$SECRETS_DIR/garage-key" "$SECRETS_DIR/garage-secret"
      fi
    fi

    # Grant the key full access to the bucket
    KEY_ID=$(cat "$SECRETS_DIR/garage-key" 2>/dev/null || true)
    if [ -n "$KEY_ID" ]; then
      $GARAGE bucket allow --read --write --owner b2-eu-cen --key ente-museum-key 2>/dev/null || true
    fi

    # Generate Ente application secrets if they don't exist
    if [ ! -f "$SECRETS_DIR/encryption-key" ]; then
      ${pkgs.openssl}/bin/openssl rand -hex 32 > "$SECRETS_DIR/encryption-key"
      chmod 600 "$SECRETS_DIR/encryption-key"
    fi

    if [ ! -f "$SECRETS_DIR/hash-key" ]; then
      ${pkgs.openssl}/bin/openssl rand -hex 32 > "$SECRETS_DIR/hash-key"
      chmod 600 "$SECRETS_DIR/hash-key"
    fi

    if [ ! -f "$SECRETS_DIR/jwt-secret" ]; then
      ${pkgs.openssl}/bin/openssl rand -hex 32 > "$SECRETS_DIR/jwt-secret"
      chmod 600 "$SECRETS_DIR/jwt-secret"
    fi

    echo "Garage setup complete."
  '';
in
{
  # Oneshot service that runs after Garage starts to ensure bucket + keys exist
  systemd.services.garage-setup = {
    description = "Initialize Garage buckets and keys for Ente";
    enable = vars.ente.enable;
    after = [ "garage.service" ];
    requires = [ "garage.service" ];
    before = [ "ente.service" ];
    wantedBy = [ "multi-user.target" ];

    serviceConfig = {
      Type = "oneshot";
      ExecStart = garageSetupScript;
      RemainAfterExit = true;
    };
  };
}

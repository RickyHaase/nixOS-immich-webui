# configuration-imports-example.nix
# This shows what users need to add to their existing configuration.nix
# Matches the process described in configuration.nix.md

{ config, pkgs, ... }:

{
  imports = [
    # Existing imports (user keeps these)
    ./hardware-configuration.nix
    
    # NEW: Add these imports for Immich configuration
    ./variables.nix      # Contains all user-configurable variables
    ./system.nix         # System settings (timezone, auto-upgrade)
    ./networking.nix     # Network, firewall, Caddy proxy
    ./zfs.nix           # ZFS and snapshot configuration
    ./immich.nix        # Docker and Immich service
    ./remoteaccess.nix  # Tailscale remote access
    ./admin.nix         # User-managed packages
  ];

  # EXISTING: User's current configuration remains untouched
  boot.loader.systemd-boot.enable = true;
  boot.loader.efi.canTouchEfiVariables = true;
  networking.networkmanager.enable = true;
  
  # REMOVE OR COMMENT: networking.hostName (now in networking.nix)
  # networking.hostName = "nixos"; # Define your hostname.
  
  # User's other existing configuration...
  i18n.defaultLocale = "en_US.UTF-8";
  
  users.users.mainuser = {
    isNormalUser = true;
    description = "Main User";
    extraGroups = [ "networkmanager" "wheel" ];
  };
  
  system.stateVersion = "24.11";
}

# What this approach provides:
# 1. Minimal changes to user's existing configuration.nix
# 2. Clear separation - Immich config in separate modules
# 3. User retains full control over their base system
# 4. Only hostname needs to be moved (clear documentation)
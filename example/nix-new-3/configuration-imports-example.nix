# configuration-imports-example.nix
# Shows what users add to their existing configuration.nix
# Simple import process - no complex interface files needed

{ config, pkgs, ... }:

{
  imports = [
    # User's existing imports (keep these)
    ./hardware-configuration.nix
    
    # NEW: Add these Immich module imports
    # Each module uses: builtins.fromJSON (builtins.readFile ./variables.json)
    ./system.nix         # System settings (timezone, auto-upgrade)
    ./networking.nix     # Network, firewall, Caddy proxy, hostname
    ./zfs.nix           # ZFS pools and snapshot configuration
    ./immich.nix        # Docker and Immich service management
    ./remoteaccess.nix  # Tailscale remote access
    ./admin.nix         # User-managed packages and configurations
  ];

  # User's existing configuration remains completely untouched
  boot.loader.systemd-boot.enable = true;
  boot.loader.efi.canTouchEfiVariables = true;
  networking.networkmanager.enable = true;
  
  # ONLY CHANGE: Remove/comment hostname (now in networking.nix)
  # networking.hostName = "nixos"; # Define your hostname.
  
  # All other user configuration stays the same
  i18n.defaultLocale = "en_US.UTF-8";
  
  users.users.mainuser = {
    isNormalUser = true;
    description = "Main User";
    extraGroups = [ "networkmanager" "wheel" ];
  };
  
  # User's custom packages, services, etc. - all unchanged
  environment.systemPackages = with pkgs; [
    # User's existing packages
  ];
  
  system.stateVersion = "24.11";
}

# Benefits of this approach:
# 1. Minimal changes to user's existing configuration
# 2. No complex interface files to understand
# 3. Every module follows same simple pattern
# 4. JSON file completely separate from Nix logic
# 5. Easy to enable/disable individual modules
# 6. Clear separation between user config and Immich config
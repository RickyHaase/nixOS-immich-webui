# configuration.nix setup

The program does not touch the `configuration.nix` file. For this reason, you will need to make a couple of changes AFTER ~~running the installation script from the binary~~ copying over the various .nix files but BEFORE running the `nixos-configuration switch` command

## Required Changes to configuration.nix

### 1. Add Module Imports

In your `/etc/nixos/configuration.nix` file, add the following imports to the `imports` section:

```nix
{
  imports = [
    #  existing imports
    ./hardware-configuration.nix

    # Add these:
    ./system.nix
    ./zfs.nix
    ./admin.nix
    ./networking.nix
    ./immich.nix
    # ./remoteaccess.nix
  ];
}
```

### 2.  Hostname Configuration

Comment out the below line:
```nix
networking.hostName = "nixos"; # Define your hostname.
```

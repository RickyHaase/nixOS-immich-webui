# Deployment Guide

This guide walks you through the complete setup process for the NixOS Immich WebUI system, from initial hardware setup to a fully functional photo backup server.

## Prerequisites

### Hardware Requirements
- Separate boot and storage drives recommended
  - Boot drive: SSD (minimum 32GB, recommended will be as big as you can afford up to the size of the library once I have the internal backups feature working)
  - Storage drive: HDD or SSD for photos and database (ZFS supports multi-disk pooling if you have multiple disks you want to use for redundancy and error correction)
- Minimum system requirements meeting both NixOS and [Immich](https://immich.app/docs/install/requirements/#hardware) specifications
- Network connectivity during installation

### Before You Begin
- Ensure your system meets the hardware requirements
- Have a USB drive ready for NixOS installation
- Know your target hostname (default: `immich`)

## Installation Overview

The deployment process follows these main steps:
1. **System Setup**: Install NixOS with basic configuration
2. **Storage Setup**: Configure ZFS pool for data storage
3. **NixOS Configuration**: Install modular configuration files
4. **Immich Setup**: Configure Docker containers and services
5. **WebUI Deployment**: Install and run the management interface

NOTE: Steps 3-5 should be replaced by a single step that involves downloading the binary and running it with an arguement for "install/setup"

## Step 1: System Setup

Follow the [System Setup Guide](system.md) to:
- Configure BIOS settings
- Install NixOS with appropriate partitioning
- Complete initial system configuration

**Important**: Choose "No desktop" during installation unless you need GUI access.

## Step 2: Storage Setup

Follow the [Storage Setup Guide](storage.md) to:
- Configure ZFS support in NixOS
- Create the `tank` ZFS pool
- Set up required datasets for Immich and PostgreSQL

After completing storage setup, verify your datasets:
```bash
zfs list
# Should show: tank, tank/pgdata, tank/immich, tank/immich/library, tank/config-backups, tank/immich-config
```

## Step 3: NixOS Configuration

### Copy Configuration Files

Copy the modular NixOS configuration files to your system:

```bash
# Copy all .nix files from the project's example directory
curl -L https://github.com/immich-app/nixOS-immich-webui/archive/refs/heads/main.tar.gz | tar -xz --strip-components=2 nixOS-immich-webui-main/example/etc/nixos/*.nix -C /etc/nixos/
```

NOTE: this is an untested placehoder method of getting the files where they need to be

### Update configuration.nix

Edit `/etc/nixos/configuration.nix` to add the required imports. Add these lines to the `imports` section:

```nix
{
  imports = [
    ./hardware-configuration.nix

    # Add these modules:
    ./system.nix
    ./zfs.nix
    ./admin.nix
    ./networking.nix
    ./immich.nix
    # ./remoteaccess.nix  # Uncomment later if needed
  ];

}
```

Comment out the default hostname

```nix
  # networking.hostName = "nixos";
```

### Configure Hostname

Edit `/etc/nixos/networking.nix` and update the hostname:

```nix
let
  hostName = "immich";  # Change this to your desired hostname
in
```

### Apply Configuration

Test and apply the new configuration:

```bash
# Test the configuration
nixos-rebuild test

# If successful, apply permanently
nixos-rebuild switch
```

> **⚠️ WARNING: INCOMPLETE DOCUMENTATION**
>
> The content below (Steps 4-5) has not yet been reviewed and is placeholder boilerplate code that may be incorrect. Additionally, Steps 3 & 4 may need to be reordered. Please proceed with caution and verify all commands before executing.

## Step 4: Immich Setup

### Create Immich Directory

```bash
mkdir -p /root/immich-app
```

### Copy Docker Configuration

Copy the Immich Docker configuration files:

```bash
cp /path/to/nixOS-immich-webui/example/tank/immich-config/docker-compose.yml /root/immich-app/
```

### Configure Environment Variables

Create `/root/immich-app/.env` with your configuration:

```bash
# Database
DB_PASSWORD=postgres
DB_USERNAME=postgres
DB_DATABASE_NAME=immich

# Upload Locations
UPLOAD_LOCATION=/tank/immich/library
DB_DATA_LOCATION=/tank/pgdata

# Immich Version
IMMICH_VERSION=release

# If using GPU acceleration, uncomment and configure:
# CUDA_VISIBLE_DEVICES=all
```

### Copy Immich Configuration

Copy the Immich application configuration:

```bash
cp /path/to/nixOS-immich-webui/example/tank/immich-config/immich-config.json /tank/immich-config/
```

### Start Immich Services

```bash
# Start the Immich service
systemctl start immich-app

# Enable it to start on boot
systemctl enable immich-app

# Check status
systemctl status immich-app
```

## Step 5: WebUI Deployment

### Install the WebUI Binary

Copy the nixOS-immich-webui binary to the root directory:

```bash
cp /path/to/nixOS-immich-webui/nixos-immich-webui /root/
chmod +x /root/nixos-immich-webui
```

### Configure as System Service (Optional)

To run the WebUI as a systemd service, uncomment the service definition in `/etc/nixos/system.nix`:

```nix
systemd.services.webui = {
    description = "NixOS-Immich WebUI Service";
    after = [ "network.target" ];
    wantedBy = [ "multi-user.target" ];
    serviceConfig = {
        ExecStart = "/root/nixos-immich-webui";  # Update binary name if different
        Restart = "always";
        User = "root";
        WorkingDirectory = "/root";
        StandardOutput = "journal";
        StandardError = "journal";
    };
};
```

Then rebuild and start the service:

```bash
nixos-rebuild switch
systemctl start webui
systemctl enable webui
```

### Manual Startup

Alternatively, run the WebUI manually:

```bash
cd /root
./nixos-immich-webui
```

## Step 6: First Access and Configuration

### Access the Services

Once everything is running, you can access:

- **Immich**: `http://immich.local` (or `http://your-hostname.local`)
- **WebUI Admin**: `http://immich.local:8080` (or `http://your-hostname.local:8080`)

### Initial Immich Setup

1. Open `http://immich.local` in your browser
2. Create your admin account
3. Configure your library settings
4. Start uploading photos

### Initial WebUI Configuration

1. Open `http://immich.local:8080` in your browser
2. Configure system settings through the web interface
3. Set up email notifications (optional)
4. Configure backup settings

## Validation

Verify your installation by checking:

```bash
# ZFS pools and datasets
zfs list

# Docker containers
docker ps

# Service status
systemctl status immich-app
systemctl status webui  # If configured as service

# Network connectivity
ping immich.local  # From another device on the network
```

## Troubleshooting

### Common Issues

**Immich not accessible at hostname.local**
- Check Avahi service: `systemctl status avahi-daemon`
- Verify firewall ports: `systemctl status firewall`

**Docker containers not starting**
- Check ZFS datasets are mounted: `zfs mount -a`
- Verify file permissions on data directories
- Check Docker service: `systemctl status docker`

**WebUI admin panel not accessible**
- Verify Caddy is running: `systemctl status caddy`
- Check port 8080 is open: `ss -tlnp | grep 8080`

**Configuration changes not applying**
- Check NixOS rebuild output for errors
- Verify all .nix files have correct syntax: `nixos-rebuild test`

### Log Files

Check these logs for debugging:
- Immich: `journalctl -u immich-app`
- WebUI: `journalctl -u webui` (if service) or check terminal output
- Caddy: `journalctl -u caddy`
- Docker: `journalctl -u docker`

## Next Steps

After successful deployment:

1. **Setup Backups**: Configure USB backup through the WebUI
2. **Remote Access**: Configure Tailscale or other remote access methods
3. **Email Notifications**: Set up email alerts for system events
4. **System Monitoring**: Review logs and set up monitoring as needed

## File Locations Reference

- **NixOS Config**: `/etc/nixos/`
- **Immich App**: `/root/immich-app/`
- **WebUI Binary**: `/root/nixos-immich-webui`
- **Photo Storage**: `/tank/immich/library/`
- **Database**: `/tank/pgdata/`
- **Config Backup**: `/tank/immich-config/`

Your NixOS Immich WebUI system is now ready for use!

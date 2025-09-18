## Compatibility
Most any system that meets both the NixOS compatibility and the Immich minimum specifications should work.

For now, it's assumed that any system used will have separate data and boot disks.

## Dell Micro PC BIOS Configuration
- [ ] Secure Boot (disable - NixOS does not have a pre-installed cert on most systems)
- [ ] If Dell - System Config->SATA Operation->select AHCI
- [ ] Power Management->AC Recovery->Power On

## NixOS Installer
1. Download Gnome Installer.
2. Flash to USB.
3. Boot to USB (assuming BIOS settings are already configured).
4. Run NixOS installer (make sure you have an internet connection).
5. Select the relevant settings for location and keyboard.
6. For Users, enter whatever username and password you want - choose "use same password for admin".
7. For Desktop, choose "No desktop" (unless you want to have a GUI for checking on your files when plugged into a monitor or whatnot).
8. No need to enable unfree software at this time.
9. Partitions - select boot disk from menu, choose erase disk, and, if it's an SSD, enable swap (with hibernate). DO NOT enable if boot storage is an SD card, EMMC, or USB storage - in these cases, choose no swap.
10. Summary -> Install.
11. Restart Now -> Done -> Unplug USB.

> **⚠️ Warning**
>
> I don't think I really like these steps here... it's getting late tho so I'm going to leave it here as a "boilerplate"
## Post-Installation Setup

After the initial NixOS installation, you'll need to configure the system for Immich. This involves setting up the modular configuration system that the WebUI will manage.

### Initial System Check

First, verify your basic installation:
```bash
# Check that you can access the system
sudo systemctl status

# Verify network connectivity
ping google.com
```

### Next Steps

1. **Storage Configuration**: Follow the [Storage Setup Guide](storage.md) to configure ZFS
2. **Complete Configuration**: Follow the [Deployment Guide](deployment.md) for full system setup

The deployment guide will walk you through copying and configuring the modular NixOS configuration files that enable the WebUI to manage your system.

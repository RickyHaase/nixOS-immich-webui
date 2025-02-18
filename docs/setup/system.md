## Compatability
Most any system that meets both the nixOS compatability and the Immich minimum specifications should work

For now, it's assumed that any system used will have seperate data and boot disks.

## Dell Micro PC BIOS Configuration
- [ ] Secure Boot (disable - nixOS does not have a pre-installed cert on most systems)
- [ ] If Dell - System Config->SATA Operation->select AHCI
- [ ] Power Management->AC Recovery->Power On

## NixOS Installer
1. Download Gnome Installer.
2. Flash to USB.
3. Boot to USB (assuming BIOS settings are already configured).
4. Run NixOS installer (make sure you have an internet connection).
5. Select the relevant settings for location and keyboard.
6. For Users, enter whatever username and password you want - choose “use same password for admin”.
7. For Desktop, choose “No desktop” (unless you want to have a GUI for checking on your files when plugged into a monitor or whatnot).
8. No need to enable unfree software at this time.
9. Partitions - select boot disk from menu, choose erase disk, and, if it’s an SSD, enable swap (with hibernate). DO NOT enable if boot storage is an SD card, EMMC, or USB storage, then choose no swap.
10. Summary -> Install.
11. Restart Now -> Done -> Unplug USB.

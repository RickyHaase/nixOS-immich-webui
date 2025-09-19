{ config, pkgs, ... }:

{ # Fill this .nix file with anything needed for administration external to the web UI. This file will not be touched by the program.
  environment.systemPackages = with pkgs; [
    tmux
    tree
    go
    git
    gh
    htop
    neofetch
    claude-code
    zip
  ];

  services.openssh.enable = true;
  services.openssh.settings.PasswordAuthentication = true;
  services.openssh.settings.PermitRootLogin = "yes";

  # https://www.reddit.com/r/NixOS/comments/185f0x4/how_to_mount_a_usb_drive/
  #services.devmon.enable = true;
  #services.gvfs.enable = true; 
  services.udisks2.enable = true;

}
